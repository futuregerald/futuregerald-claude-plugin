package httpapi

// Covers the two probes an orchestrator actually calls, and the reason they are
// two endpoints rather than one: /healthz decides whether to RESTART this
// process, /readyz decides whether to SEND IT TRAFFIC. Wiring the dependencies
// into the first is how a fleet with one sick database restarts itself to
// death, so "liveness consults nothing" is asserted here rather than left as an
// intention in a comment.

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec
}

// Liveness must answer without touching a dependency, so the check registered
// here is both failing and instrumented: if /healthz consulted it, the endpoint
// would report the process as unhealthy for something restarting cannot fix.
func TestHealthzIsUnconditionalAndConsultsNoDependency(t *testing.T) {
	var consulted atomic.Bool
	srv := NewServer(ServerConfig{Addr: ":0"}, slog.New(slog.DiscardHandler), fakeRegistrar{},
		ReadinessCheck{Name: "db", Check: func(context.Context) error {
			consulted.Store(true)
			return errors.New("database is down")
		}})

	rec := get(t, srv.Handler(), "/healthz")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body)
	}
	if consulted.Load() {
		t.Error("/healthz ran a readiness check; a failing dependency will now trigger restarts")
	}
}

// An instance with no registered dependencies is ready by definition. The
// empty-array shape is pinned deliberately: a nil slice marshals to null, and
// every consumer then has to special-case it.
func TestReadyzWithNoChecksIs200(t *testing.T) {
	srv := NewServer(ServerConfig{Addr: ":0"}, slog.New(slog.DiscardHandler), fakeRegistrar{})

	rec := get(t, srv.Handler(), "/readyz")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"checks":[]`) {
		t.Errorf("body = %s, want an empty checks array rather than null", rec.Body)
	}
}

// Three properties in one request, because they are the same mistake seen from
// three sides: report EVERY check (no short-circuit on the first failure),
// name the failing ones, and keep the driver's error text out of a body that is
// usually unauthenticated.
func TestReadyzReportsEveryCheckAndHidesErrorDetail(t *testing.T) {
	var logged bytes.Buffer
	srv := NewServer(ServerConfig{Addr: ":0"}, slog.New(slog.NewJSONHandler(&logged, nil)), fakeRegistrar{},
		ReadinessCheck{Name: "primary-db", Check: func(context.Context) error {
			return errors.New("dial tcp 10.0.0.5:5432: connection refused")
		}},
		ReadinessCheck{Name: "cache", Check: func(context.Context) error { return nil }},
		ReadinessCheck{Name: "search-index", Check: func(context.Context) error {
			return errors.New("index is rebuilding")
		}},
	)

	rec := get(t, srv.Handler(), "/readyz")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body %s)", rec.Code, rec.Body)
	}
	// search-index is listed last: it can only appear if the handler kept going
	// after primary-db failed.
	for _, name := range []string{"primary-db", "cache", "search-index"} {
		if !strings.Contains(rec.Body.String(), name) {
			t.Errorf("body does not name %q: %s", name, rec.Body)
		}
	}
	for _, leak := range []string{"10.0.0.5", "connection refused", "rebuilding"} {
		if strings.Contains(rec.Body.String(), leak) {
			t.Errorf("body leaked %q to an unauthenticated probe: %s", leak, rec.Body)
		}
	}
	// Hidden from the client, but useless if hidden from the operator too.
	if !strings.Contains(logged.String(), "connection refused") {
		t.Errorf("the check error was not logged anywhere: %s", logged.String())
	}
}

// Driven through handleReady rather than the server, on purpose: TimeoutHandler
// abandons a cancelled request on its own, so the full chain would return
// promptly even if the checks below ran forever. This asserts the handler's own
// behaviour.
func TestReadyzStopsCheckingWhenTheRequestIsCancelled(t *testing.T) {
	// The request context is the fixture: this test cancels it by hand to prove
	// the handler stops probing. t.Context() is the parent, not a substitute.
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	dialing := make(chan struct{})
	var laterCheckRan atomic.Bool
	h := handleReady(slog.New(slog.DiscardHandler), []ReadinessCheck{
		{Name: "slow", Check: func(ctx context.Context) error {
			close(dialing)
			<-ctx.Done()
			return ctx.Err()
		}},
		{Name: "never-reached", Check: func(context.Context) error {
			laterCheckRan.Store(true)
			return nil
		}},
	})

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.ServeHTTP(rec, req)
	}()

	<-dialing
	cancel()

	// rec is only read after done is closed, so the handler goroutine's writes
	// to it happen-before this goroutine's reads.
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("/readyz ignored the request context and is still probing a dependency")
	}
	if laterCheckRan.Load() {
		t.Error("a check ran after the request was cancelled; nobody is left to read the answer")
	}
	// Stopping early is only half of it. A handler that returns without writing
	// sends the implicit 200 net/http produces for silence, which on a readiness
	// probe reads as "send me traffic" -- asserted from an unfinished sweep.
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 -- readiness was abandoned, not confirmed (body %s)",
			rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"status":"unknown"`) {
		t.Errorf("body = %s, want the inconclusive status rather than a verdict", rec.Body)
	}
}

// The instance with no registered checks is the one that never enters the loop,
// so it is the one a guard written inside the loop does not protect: before this
// existed, a cancelled request to such an instance was answered
// 200 {"checks":[]} -- "route traffic to me" -- with nothing consulted at all.
func TestReadyzWithNoChecksDoesNotAnswerReadyWhenCancelled(t *testing.T) {
	// The already-dead context IS the fixture, exactly as in the cancelled
	// register test: this is a request whose client left before it was served.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	rec := httptest.NewRecorder()
	handleReady(slog.New(slog.DiscardHandler), nil).ServeHTTP(rec,
		httptest.NewRequestWithContext(ctx, http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body %s)", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), `"status":"ready"`) {
		t.Errorf("body claims readiness for a request nothing was checked for: %s", rec.Body)
	}
}
