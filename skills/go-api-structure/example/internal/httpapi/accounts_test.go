package httpapi

// Drives the doc's handler through httptest with a fake registrar, then
// through the real ServeMux built by NewServer. Verifies the status-code
// mapping and that the response body carries a populated id.

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/user-service/internal/accounts"
)

type fakeRegistrar struct {
	user accounts.User
	err  error
}

func (f fakeRegistrar) Register(ctx context.Context, email, password string) (accounts.User, error) {
	if f.err != nil {
		return accounts.User{}, f.err
	}
	return f.user, nil
}

// post sends what a real client sends: a JSON body declared as JSON. Fixtures
// that omit the header do not exercise the handler at all -- they stop at the
// media-type guard -- so the default has to be the honest one.
func post(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	return postAs(t, h, "application/json", body)
}

// postAs is post with the Content-Type under test's control. An empty
// contentType sends no header at all, which is the bypass the guard has to
// refuse and which httptest.NewRequest produces by default.
func postAs(t *testing.T, h http.Handler, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/users",
		strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRegisterCreatedCarriesID(t *testing.T) {
	svc := fakeRegistrar{user: accounts.User{
		ID:        "user-1",
		Email:     "a@example.com",
		CreatedAt: time.Now(),
	}}
	rec := post(t, handleRegister(svc), `{"email":"a@example.com","password":"pw"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v (body %q)", err, rec.Body.String())
	}
	// The finding-1 regression, observed at the wire: this used to be "".
	if got["id"] != "user-1" {
		t.Errorf("id = %v, want user-1 (body %s)", got["id"], rec.Body.String())
	}
	if got["email"] != "a@example.com" {
		t.Errorf("email = %v", got["email"])
	}
	// password must never round-trip
	if _, leaked := got["password"]; leaked {
		t.Error("password leaked into the response")
	}
	if _, leaked := got["password_hash"]; leaked {
		t.Error("password_hash leaked into the response")
	}
}

func TestRegisterEmailTakenIs409(t *testing.T) {
	// Wrapped, exactly as the service returns it, to prove errors.Is matches
	// through the wrap rather than only on a bare sentinel.
	svc := fakeRegistrar{err: wrap(accounts.ErrEmailTaken)}
	rec := post(t, handleRegister(svc), `{"email":"a@example.com","password":"pw"}`)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

func TestRegisterUnknownErrorIs500(t *testing.T) {
	svc := fakeRegistrar{err: errors.New("disk on fire")}
	rec := post(t, handleRegister(svc), `{"email":"a@example.com","password":"pw"}`)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "disk on fire") {
		t.Errorf("internal error detail leaked to client: %s", rec.Body.String())
	}
}

func TestRegisterMalformedBodyIs400(t *testing.T) {
	rec := post(t, handleRegister(fakeRegistrar{}), `{not json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestRegisterRejectsOversizedBody(t *testing.T) {
	big := `{"email":"` + strings.Repeat("a", 2<<20) + `","password":"pw"}`
	rec := post(t, handleRegister(fakeRegistrar{}), big)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d, want 413", rec.Code)
	}
}

func TestRegisterRejectsUnknownFields(t *testing.T) {
	rec := post(t, handleRegister(fakeRegistrar{}),
		`{"email":"a@example.com","password":"pw","admin":true}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", rec.Code)
	}
}

func TestRegisterRejectsTrailingJSON(t *testing.T) {
	rec := post(t, handleRegister(fakeRegistrar{}),
		`{"email":"a@example.com","password":"pw"}{"email":"b@example.com"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", rec.Code)
	}
}

// The media-type guard, and why it is a 415 rather than a 400. Nothing is
// malformed about this request -- it is syntactically fine and the server
// simply does not speak the format it declares. Same distinction the 413 makes:
// the body was well-formed, just too big.
func TestRegisterRejectsNonJSONContentType(t *testing.T) {
	rec := postAs(t, handleRegister(fakeRegistrar{}), "text/plain",
		`{"email":"a@example.com","password":"pw"}`)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("got %d, want 415 (body %s)", rec.Code, rec.Body.String())
	}
}

// A guard that rejected only a WRONG Content-Type would be bypassed by sending
// none, which is the first thing anyone tries. So the header is required, not
// merely checked when present.
func TestRegisterRejectsMissingContentType(t *testing.T) {
	rec := postAs(t, handleRegister(fakeRegistrar{}), "",
		`{"email":"a@example.com","password":"pw"}`)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("got %d, want 415 (body %s)", rec.Code, rec.Body.String())
	}
}

// Parameters on the media type are legal and extremely common. Comparing the
// raw header against "application/json" would 415 every client that sends a
// charset -- which is why the guard parses instead of comparing strings.
func TestRegisterAcceptsContentTypeParameters(t *testing.T) {
	svc := fakeRegistrar{user: accounts.User{ID: "user-1", Email: "a@example.com"}}
	rec := postAs(t, handleRegister(svc), "application/json; charset=utf-8",
		`{"email":"a@example.com","password":"pw"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201 (body %s)", rec.Code, rec.Body.String())
	}
}

// Media types are case-insensitive (RFC 9110), so this is a valid JSON request
// and must be served, not refused.
func TestRegisterContentTypeIsCaseInsensitive(t *testing.T) {
	svc := fakeRegistrar{user: accounts.User{ID: "user-1", Email: "a@example.com"}}
	rec := postAs(t, handleRegister(svc), "APPLICATION/JSON",
		`{"email":"a@example.com","password":"pw"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201 (body %s)", rec.Code, rec.Body.String())
	}
}

// The single-value guard runs a second Decode, and that read can fail for
// reasons that have nothing to do with trailing data: the body can overflow
// the cap only after the first value has decoded. Reporting that as "malformed
// body" would be a lie -- the request was well-formed, just too big -- so the
// guard has to preserve the error rather than replace it.
func TestRegisterOversizedAfterFirstValueIsStill413(t *testing.T) {
	body := `{"email":"a@example.com","password":"pw"}` +
		`{"email":"` + strings.Repeat("a", 2<<20) + `"}`
	rec := post(t, handleRegister(fakeRegistrar{}), body)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d, want 413", rec.Code)
	}
}

// The two branches of the single-value guard, side by side. Only the cleanly
// decoding cases exercise the err == nil branch -- a trailing value the
// decoder rejects (the unknown-field entry) never reaches it, which is why
// both kinds are worth pinning.
func TestRegisterRejectsTrailingJSONValues(t *testing.T) {
	const first = `{"email":"a@example.com","password":"pw"}`
	tests := []struct {
		name     string
		trailing string
	}{
		{"object decodes cleanly", `{}`},
		{"null decodes cleanly", `null`},
		{"number decodes cleanly", `7`},
		{"decoder rejects the trailing value", `{"password":"pw"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := post(t, handleRegister(fakeRegistrar{}), first+tt.trailing)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("got %d, want 400 (body %s)", rec.Code, rec.Body.String())
			}
		})
	}
}

// A bare `null` is valid JSON and decodes into a struct as a documented no-op:
// it succeeds and leaves the zero value. So it reaches validation looking
// exactly like `{}`, and it is the presence check -- not the decoder -- that
// rejects it. An empty body is a different failure entirely: nothing to
// decode, so io.EOF, so 400. Pinning both keeps that boundary deliberate.
func TestRegisterNullBodyIs422(t *testing.T) {
	rec := post(t, handleRegister(fakeRegistrar{}), `null`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("got %d, want 422 (body %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "email") ||
		!strings.Contains(rec.Body.String(), "password") {
		t.Fatalf("want both problems reported, got %s", rec.Body.String())
	}
}

func TestRegisterEmptyBodyIs400(t *testing.T) {
	rec := post(t, handleRegister(fakeRegistrar{}), ``)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestRegisterRejectsMissingFields(t *testing.T) {
	rec := post(t, handleRegister(fakeRegistrar{}), `{"email":"","password":""}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("got %d, want 422", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "email") ||
		!strings.Contains(rec.Body.String(), "password") {
		t.Fatalf("want both problems reported, got %s", rec.Body.String())
	}
}

// Exercises NewServer's routing, not just the bare handler.
func TestNewServerRoutesPostUsers(t *testing.T) {
	svc := fakeRegistrar{user: accounts.User{ID: "user-9", Email: "r@example.com"}}
	srv := NewServer(ServerConfig{Addr: ":0", ShutdownTimeout: time.Second},
		slog.New(slog.DiscardHandler), svc)

	rec := post(t, srv.http.Handler, `{"email":"r@example.com","password":"pw"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}

	// A route the mux should not serve.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users", nil)
	rec2 := httptest.NewRecorder()
	srv.http.Handler.ServeHTTP(rec2, req)
	if rec2.Code == http.StatusCreated {
		t.Error("GET /users should not hit the register handler")
	}
}

func wrap(err error) error { return errWrap{err} }

type errWrap struct{ err error }

func (e errWrap) Error() string { return "create user: " + e.err.Error() }
func (e errWrap) Unwrap() error { return e.err }

// Guards the context-error mapping: a cancelled client and a blown deadline
// must not both become 500, which the context guidance calls a mistake.
func TestContextErrorsMapToDistinctStatuses(t *testing.T) {
	rec := post(t, handleRegister(fakeRegistrar{err: wrap(context.DeadlineExceeded)}),
		`{"email":"a@example.com","password":"pw"}`)
	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("DeadlineExceeded -> %d, want 504", rec.Code)
	}

	rec = post(t, handleRegister(fakeRegistrar{err: wrap(context.Canceled)}),
		`{"email":"a@example.com","password":"pw"}`)
	// The client is gone; writing an error payload serves nobody.
	if strings.Contains(rec.Body.String(), "error") {
		t.Errorf("wrote an error body for a cancelled client: %q", rec.Body.String())
	}
	if rec.Code == http.StatusInternalServerError {
		t.Error("a cancelled client was reported as a server error (500)")
	}
}
