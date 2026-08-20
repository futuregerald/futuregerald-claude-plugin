package httpapi

// Covers the server's timeout wiring: that an unset RequestTimeout does not
// silently 503 everything, and that a handler exceeding the budget is cut off.

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/user-service/internal/accounts"
)

func TestZeroRequestTimeoutDoesNotBreakEveryRequest(t *testing.T) {
	svc := fakeRegistrar{user: accounts.User{ID: "user-9", Email: "r@example.com"}}
	// RequestTimeout deliberately left unset.
	srv := NewServer(ServerConfig{Addr: ":0", ShutdownTimeout: time.Second},
		slog.New(slog.DiscardHandler), svc)

	rec := post(t, srv.http.Handler, `{"email":"r@example.com","password":"pw"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (zero RequestTimeout must default, not 503)", rec.Code)
	}
}

func TestServerAppliesDefaultTimeouts(t *testing.T) {
	srv := NewServer(ServerConfig{Addr: ":0", ShutdownTimeout: time.Second},
		slog.New(slog.DiscardHandler), fakeRegistrar{})

	if srv.http.ReadHeaderTimeout <= 0 {
		t.Error("ReadHeaderTimeout not set; server is exposed to slow-header clients")
	}
	// WriteTimeout must exceed the handler budget or the connection dies
	// before TimeoutHandler can write its 503.
	if srv.http.WriteTimeout <= defaultRequestTimeout {
		t.Errorf("WriteTimeout %v must exceed request timeout %v",
			srv.http.WriteTimeout, defaultRequestTimeout)
	}
}

// A variadic parameter is not a private copy: called with the spread form,
// Go hands the callee the caller's own backing array. Storing it as-is would
// let the caller keep mutating what /readyz reports.
func TestNewServerCopiesReadinessChecks(t *testing.T) {
	checks := []ReadinessCheck{{
		Name:  "db",
		Check: func(context.Context) error { return nil },
	}}
	srv := NewServer(ServerConfig{Addr: ":0"},
		slog.New(slog.DiscardHandler), fakeRegistrar{}, checks...)

	checks[0].Name = "mutated by the caller"
	if srv.ready[0].Name != "db" {
		t.Fatalf("ready[0].Name = %q; the server aliases the caller's slice", srv.ready[0].Name)
	}
}

// A nil Check func has no sane default, so it fails loudly at construction
// rather than panicking on the first /readyz request.
func TestNewServerPanicsOnNilReadinessCheckFunc(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("NewServer accepted a ReadinessCheck with a nil Check func")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "db") {
			t.Fatalf("panic %v does not name the offending check", r)
		}
	}()

	NewServer(ServerConfig{Addr: ":0"},
		slog.New(slog.DiscardHandler), fakeRegistrar{}, ReadinessCheck{Name: "db"})
}

// A handler that outruns its budget must be cut off by TimeoutHandler, and
// the request context must be cancelled so downstream work stops too.
func TestSlowHandlerIsTimedOutAndContextCancelled(t *testing.T) {
	released := make(chan struct{})
	var sawCancel bool

	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		sawCancel = r.Context().Err() == context.DeadlineExceeded
		close(released)
	})

	h := http.TimeoutHandler(slow, 30*time.Millisecond, `{"error":"request timeout"}`)
	// t.Context() is the right parent even here: TimeoutHandler derives its own
	// 30ms deadline from the request context, so the cancellation under test is
	// still produced by the handler chain, not by the fixture.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/slow", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "request timeout") {
		t.Errorf("body = %q, want the configured timeout message", rec.Body.String())
	}

	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("handler never observed cancellation")
	}
	if !sawCancel {
		t.Error("handler context was not DeadlineExceeded; cancellation did not propagate")
	}
}
