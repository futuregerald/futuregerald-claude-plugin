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

func post(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
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
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
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
