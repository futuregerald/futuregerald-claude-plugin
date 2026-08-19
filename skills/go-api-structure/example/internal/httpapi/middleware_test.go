package httpapi

// Covers the properties of the middleware chain that silently rot if nothing
// asserts them: that Handler() hands back the WRAPPED handler, that a panicking
// handler becomes a 500 with a log line instead of a dead connection, that a
// DELIBERATE abort is not laundered into a 500, that a response already on the
// wire is left alone, that an inbound correlation ID is honoured rather than
// replaced, and that the access log still emits its line when the request ends
// in a panic.
//
// Every one of those is argued at length in a comment in middleware.go, and
// before these tests existed every one of them could be deleted with the suite
// staying green.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/user-service/internal/accounts"
)

// logRecord finds the one log line whose message is msg and returns it decoded,
// so a test can ask for a FIELD rather than string-match a line that happens to
// contain the right digits somewhere. Fatal if there is no such line: an absent
// log line is the failure these tests exist to catch.
func logRecord(t *testing.T, logged *bytes.Buffer, msg string) map[string]any {
	t.Helper()
	for line := range strings.SplitSeq(strings.TrimSpace(logged.String()), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line is not JSON: %q (%v)", line, err)
		}
		if rec["msg"] == msg {
			return rec
		}
	}
	t.Fatalf("no %q line was logged: %s", msg, logged.String())
	return nil
}

// recovered runs h and hands back the value it panicked with, or nil if it
// returned normally. A `defer recover()` written in the test body would fire
// after the assertions instead of around the call, so the assertions would
// never run.
func recovered(h http.Handler, w http.ResponseWriter, r *http.Request) (value any) {
	defer func() { value = recover() }()
	h.ServeHTTP(w, r)
	return nil
}

// panicRegistrar is the service a handler cannot survive: it panics where a
// real one would return an error.
type panicRegistrar struct{}

func (panicRegistrar) Register(context.Context, string, string) (accounts.User, error) {
	panic("boom from the service")
}

// Pins the contract that Handler() returns the WRAPPED handler. Without this,
// a future edit returning the bare mux would leave every functional test
// passing while middleware silently stopped being exercised.
//
// The path is deliberately one the mux does not route: the request-ID
// middleware sits outside the mux, so the header must be set even on a 404.
// Using a real route would leave it ambiguous whether the middleware ran or
// the handler happened to set the header itself.
func TestHandlerIncludesMiddleware(t *testing.T) {
	srv := NewServer(ServerConfig{Addr: ":0"}, slog.New(slog.DiscardHandler), fakeRegistrar{})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/no-such-route", nil))
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("Handler() did not include the request-ID middleware")
	}
}

func TestRecoverPanicReturns500AndLogs(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })

	rec := httptest.NewRecorder()
	recoverPanic(log)(boom).ServeHTTP(rec,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got %d, want 500", rec.Code)
	}
	if !strings.Contains(buf.String(), "panic") {
		t.Fatalf("panic was not logged: %s", buf.String())
	}
}

// http.ErrAbortHandler is net/http's documented way for a handler to abandon a
// response deliberately: the server catches it and drops the connection without
// a stack trace. Converting it to a 500 invents a server error out of an
// intentional abort, and fills the log with panics nobody caused.
//
// The wrapped case is not padding. `rec == http.ErrAbortHandler` compares
// interface values and passes the bare sentinel while missing a value that
// WRAPS it, so a test that only sends the sentinel would stay green against
// exactly the comparison this code avoids.
func TestRecoverPanicRepanicsErrAbortHandler(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"the sentinel itself", http.ErrAbortHandler},
		{"a wrapped sentinel", fmt.Errorf("copying stream: %w", http.ErrAbortHandler)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logged bytes.Buffer
			abort := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(tt.value) })

			rec := httptest.NewRecorder()
			got := recovered(recoverPanic(slog.New(slog.NewJSONHandler(&logged, nil)))(abort), rec,
				httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

			err, ok := got.(error)
			if !ok || !errors.Is(err, http.ErrAbortHandler) {
				t.Fatalf("recovered %#v, want the panic re-raised carrying http.ErrAbortHandler", got)
			}
			if rec.Code == http.StatusInternalServerError {
				t.Error("a deliberate abort was reported to the client as a 500")
			}
			if rec.Body.Len() != 0 {
				t.Errorf("wrote a body for an abandoned response: %s", rec.Body)
			}
			if logged.Len() != 0 {
				t.Errorf("logged a deliberate abort as a panic: %s", logged.String())
			}
		})
	}
}

// A panic AFTER the handler has committed its status is not recoverable into a
// 500: the status is already on the wire, and appending an error object would
// corrupt a body the client is mid-way through parsing. The failure is real and
// belongs in the log; the response belongs to the handler that started it.
func TestRecoverPanicLeavesAnAlreadyWrittenResponseAlone(t *testing.T) {
	var logged bytes.Buffer
	half := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"accepted":true}`)
		panic("boom after the status went out")
	})

	rec := httptest.NewRecorder()
	recoverPanic(slog.New(slog.NewJSONHandler(&logged, nil)))(half).ServeHTTP(rec,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202 -- the client was already told 202", rec.Code)
	}
	if got := rec.Body.String(); got != `{"accepted":true}` {
		t.Errorf("body = %q, want the handler's own body with nothing appended", got)
	}
	// Hidden from the client, but the operator still has to hear about it.
	logRecord(t, &logged, "panic recovered")
}

// An inbound ID is honoured rather than replaced. That is the entire point of a
// correlation ID: one value follows the request across every service it
// touches. Generating a fresh one at each hop breaks the trace into unrelated
// fragments while leaving every "is the header set?" assertion green -- which is
// why this asserts the VALUE, in both places the middleware puts it.
func TestRequestIDHonoursAnInboundID(t *testing.T) {
	const inbound = "inbound-correlation-id"

	var carried string
	h := requestID()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		carried = requestIDFrom(r.Context())
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Header.Set(requestIDHeader, inbound)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get(requestIDHeader); got != inbound {
		t.Errorf("response header = %q, want the inbound %q", got, inbound)
	}
	if carried != inbound {
		t.Errorf("context carried %q, want the inbound %q", carried, inbound)
	}
}

// The access log must survive a panicking request -- the one request where a
// missing line hurts most, because the panic log names the URL but not the
// status, the duration or the request ID.
//
// Driven through NewServer rather than requestLog alone, because the middleware
// ORDER is what breaks it: recoverPanic sits OUTSIDE requestLog, and
// http.TimeoutHandler re-raises a handler panic on the outer goroutine, so the
// panic unwinds straight through requestLog's call into the handler. A log
// statement written after that call is never reached.
func TestRequestLogEmitsALineForAPanickingRequest(t *testing.T) {
	var logged bytes.Buffer
	srv := NewServer(ServerConfig{Addr: ":0"}, slog.New(slog.NewJSONHandler(&logged, nil)),
		panicRegistrar{})

	rec := post(t, srv.Handler(), `{"email":"a@example.com","password":"pw"}`)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body %s)", rec.Code, rec.Body)
	}

	got := logRecord(t, &logged, "http request")
	// JSON numbers decode as float64; the comparison is against the status the
	// client actually received, which recoverPanic wrote from outside.
	if got["status"] != float64(http.StatusInternalServerError) {
		t.Errorf("status = %v, want 500 -- the log claimed a status the client never got", got["status"])
	}
	if got["panic"] != true {
		t.Errorf("panic = %v, want true -- the line does not say the request died", got["panic"])
	}
	if got["path"] != "/users" {
		t.Errorf("path = %v, want /users", got["path"])
	}
	if got["request_id"] == "" {
		t.Error("the line carries no request ID, so it cannot be joined to the panic line")
	}
}
