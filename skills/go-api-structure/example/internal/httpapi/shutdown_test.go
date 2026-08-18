package httpapi

// Guards two fixes: Run(ctx) taking its cancellation from the caller (signals
// belong in main), and ShutdownTimeout being defaulted so an unset value does
// not produce an already-expired deadline that severs in-flight requests.

import (
	"context"
	"errors"
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
	srv := NewServer(ServerConfig{Addr: "127.0.0.1:0", RequestTimeout: 5 * time.Second}, svc)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	// The injectable listener is why this is testable at all: ListenAndServe
	// binds internally and never reveals the port it chose.
	srv.listen = func(string, string) (net.Listener, error) { return ln, nil }

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- srv.Run(ctx) }()

	respErr := make(chan error, 1)
	go func() {
		resp, err := http.Post("http://"+ln.Addr().String()+"/users",
			"application/json", strings.NewReader(`{"email":"a@example.com","password":"pw"}`))
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
	srv := NewServer(ServerConfig{Addr: "127.0.0.1:0"}, fakeRegistrar{})
	if srv.shutdownTimeout <= 0 {
		t.Fatalf("shutdownTimeout = %v; an unset value yields an already-expired "+
			"context and an instant, ungraceful shutdown", srv.shutdownTimeout)
	}
}
