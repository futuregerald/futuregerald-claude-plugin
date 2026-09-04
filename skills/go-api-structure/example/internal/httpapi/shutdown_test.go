package httpapi

// Guards two fixes: Run(ctx) taking its cancellation from the caller (signals
// belong in main), and ShutdownTimeout being defaulted so an unset value does
// not produce an already-expired deadline that severs in-flight requests.

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"example.com/user-service/internal/accounts"
)

type blockingRegistrar struct {
	started chan struct{}
	release chan struct{}
}

func (b blockingRegistrar) Register(ctx context.Context, email, password string) (accounts.User, error) {
	close(b.started)
	<-b.release
	return accounts.User{ID: "user-1", Email: email}, nil
}

func TestRunDrainsInFlightRequestOnCancel(t *testing.T) {
	svc := blockingRegistrar{started: make(chan struct{}), release: make(chan struct{})}

	// ShutdownTimeout deliberately unset — the default must cover it.
	srv := NewServer(ServerConfig{Addr: "127.0.0.1:0", RequestTimeout: 5 * time.Second},
		slog.New(slog.DiscardHandler), svc)

	// ListenConfig's context bounds the BIND only -- once the listener exists it
	// is unaffected by that context -- so t.Context() here cannot interfere with
	// the shutdown sequence this test is actually about.
	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	// The injectable listener is why this is testable at all: ListenAndServe
	// binds internally and never reveals the port it chose.
	srv.listen = func(string, string) (net.Listener, error) { return ln, nil }

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- srv.Run(ctx) }()

	// Built here rather than inside the goroutine so a construction failure is a
	// t.Fatalf instead of an error on respErr, which the assertion below would
	// misreport as a severed connection.
	//
	// t.Context() is deliberate: this request must be DRAINED, not cancelled.
	// The server's own ctx (cancelled at line "simulates SIGTERM") must be the
	// only cancellation in play, and t.Context() outlives the <-respErr wait.
	// The Content-Type is load-bearing -- without it the media-type guard
	// answers 415, blockingRegistrar never runs, and <-svc.started deadlocks.
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		"http://"+ln.Addr().String()+"/users",
		strings.NewReader(`{"email":"a@example.com","password":"pw"}`))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	respErr := make(chan error, 1)
	go func() {
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
		respErr <- err
	}()

	<-svc.started // request is in flight
	cancel()      // simulates SIGTERM arriving

	select {
	case err := <-runErr:
		t.Fatalf("Run returned while a request was in flight: %v", err)
	case <-time.After(150 * time.Millisecond):
	}

	close(svc.release)

	select {
	case err := <-runErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Run: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run never returned after the request completed")
	}

	if err := <-respErr; err != nil {
		t.Errorf("in-flight request was severed rather than drained: %v", err)
	}
}

func TestUnsetShutdownTimeoutIsDefaulted(t *testing.T) {
	srv := NewServer(ServerConfig{Addr: "127.0.0.1:0"}, slog.New(slog.DiscardHandler), fakeRegistrar{})
	if srv.shutdownTimeout <= 0 {
		t.Fatalf("shutdownTimeout = %v; an unset value yields an already-expired "+
			"context and an instant, ungraceful shutdown", srv.shutdownTimeout)
	}
}
