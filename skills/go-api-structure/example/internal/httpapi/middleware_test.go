package httpapi

// Covers the two properties of the middleware chain that silently rot if
// nothing asserts them: that Handler() hands back the WRAPPED handler, and
// that a panicking handler becomes a 500 with a log line instead of a dead
// connection.

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Pins the contract that Handler() returns the WRAPPED handler. Without this,
// a future edit returning the bare mux would leave every functional test
// passing while middleware silently stopped being exercised.
//
// /healthz is deliberately an unrouted path: the request-ID middleware sits
// outside the mux, so the header must be set even on a 404.
func TestHandlerIncludesMiddleware(t *testing.T) {
	srv := NewServer(ServerConfig{Addr: ":0"}, slog.New(slog.DiscardHandler), fakeRegistrar{})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("Handler() did not include the request-ID middleware")
	}
}

func TestRecoverPanicReturns500AndLogs(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })

	rec := httptest.NewRecorder()
	recoverPanic(log)(boom).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got %d, want 500", rec.Code)
	}
	if !strings.Contains(buf.String(), "panic") {
		t.Fatalf("panic was not logged: %s", buf.String())
	}
}
