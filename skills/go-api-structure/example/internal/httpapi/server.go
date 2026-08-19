package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// ServerConfig is declared here rather than imported from internal/config,
// for the same reason interfaces are declared by their consumer: this package
// states what it needs, and main maps the app config onto it. httpapi stays
// independent of how configuration happens to be loaded.
type ServerConfig struct {
	Addr            string
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
}

// ReadinessCheck reports whether one dependency is usable right now.
// Name appears in the /readyz body so a failing check is identifiable.
type ReadinessCheck struct {
	Name  string
	Check func(context.Context) error
}

type Server struct {
	http            *http.Server
	shutdownTimeout time.Duration
	listen          func(network, addr string) (net.Listener, error)
	log             *slog.Logger
	ready           []ReadinessCheck
}

const (
	defaultRequestTimeout  = 30 * time.Second
	defaultShutdownTimeout = 15 * time.Second
)

func NewServer(cfg ServerConfig, log *slog.Logger, svc registrar, ready ...ReadinessCheck) *Server {
	// A nil logger is tolerated rather than fatal: the alternative is a panic
	// on the first request instead of at construction, which is strictly worse
	// to debug.
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	// The zero value of a timeout means "unset", but http.TimeoutHandler
	// reads 0 as "time out immediately" -- an unset field would make every
	// request 503 rather than simply not time out. http.Server's own
	// timeouts have the opposite convention, where 0 means no limit. That
	// asymmetry is worth defending against here rather than debugging later.
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = defaultRequestTimeout
	}
	// Same defence: an unset ShutdownTimeout would make context.WithTimeout
	// produce an already-expired deadline, so Shutdown would abort instantly
	// and sever in-flight requests -- the opposite of graceful.
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = defaultShutdownTimeout
	}

	mux := http.NewServeMux()
	// Method patterns ("POST /users") require Go 1.22+. On 1.21 and earlier
	// this parses as a HOST pattern and silently never matches -- no error,
	// no panic, just 404s.
	mux.Handle("POST /users", handleRegister(svc))

	// Outermost first: a request travels recoverPanic -> requestID ->
	// requestLog -> TimeoutHandler -> mux, and a response travels back out the
	// same way. The order is not cosmetic:
	//
	//   - recoverPanic is outermost so it also catches a panic raised by the
	//     middleware inside it, not just by a handler.
	//   - requestID precedes requestLog because the log line carries the ID.
	//   - requestLog wraps TimeoutHandler rather than sitting under it, so it
	//     records the 503 the timeout writes and stays on one goroutine.
	//
	// TimeoutHandler is what puts a deadline on every request context, so
	// handlers inherit cancellation without each one remembering to set it.
	handler := chain(
		http.TimeoutHandler(mux, cfg.RequestTimeout, `{"error":"request timeout"}`),
		recoverPanic(log),
		requestID(),
		requestLog(log),
	)

	return &Server{
		http: &http.Server{
			Addr:    cfg.Addr,
			Handler: handler,
			// Guards against a client that opens a connection and dribbles
			// headers forever. Not a context — the net/http server enforces
			// these itself, and no deadline reaches the handler from them.
			ReadHeaderTimeout: 5 * time.Second,
			// Must exceed RequestTimeout, or the connection dies before
			// TimeoutHandler can write its response.
			WriteTimeout: cfg.RequestTimeout + 5*time.Second,
			IdleTimeout:  60 * time.Second,
		},
		shutdownTimeout: cfg.ShutdownTimeout,
		listen:          net.Listen,
		log:             log,
		ready:           ready,
	}
}

// Handler exposes the routed handler so functional tests can drive the whole
// stack through httptest without binding a port. Cheap, and it is what makes
// end-to-end tests of the real wiring possible.
func (s *Server) Handler() http.Handler { return s.http.Handler }

// Run serves until ctx is cancelled, then shuts down gracefully.
//
// The CALLER owns signal handling. Signals are a process-wide singleton, so a
// binary running an HTTP server plus a worker needs one place deciding the
// shutdown order -- and that place is main, not a transport package.
//
// listen is injectable so tests can bind 127.0.0.1:0 and learn the real port.
// A shutdown path that cannot be tested is exactly the kind that rots.
func (s *Server) Run(ctx context.Context) error {
	ln, err := s.listen("tcp", s.http.Addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.http.Addr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.http.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	return s.http.Shutdown(shutdownCtx)
}
