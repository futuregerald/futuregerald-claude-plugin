package httpapi

import (
	"log/slog"
	"net/http"
)

// addRoutes is the one place that shows the whole API surface. Registering
// routes inline in NewServer works right up until the third endpoint, at which
// point the list is buried in a constructor that also defaults config, builds
// middleware and assembles an *http.Server. A reader asking "what does this
// service expose?" should get the answer from one screen, and a reviewer should
// see a new public endpoint appear in a diff without reading any plumbing.
//
// It takes the dependencies the handlers need rather than the *Server, so the
// route table cannot reach into the server's lifecycle -- and so a test can
// build a mux without building a server.
func addRoutes(mux *http.ServeMux, log *slog.Logger, svc registrar, ready []ReadinessCheck) {
	// Method patterns ("POST /users") require Go 1.22+. On 1.21 and earlier
	// this parses as a HOST pattern and silently never matches -- no error,
	// no panic, just 404s.
	mux.Handle("POST /users", handleRegister(svc))

	// Two probes rather than one, because they answer different questions with
	// different consequences: /healthz decides whether to RESTART the process,
	// /readyz decides whether to ROUTE TRAFFIC to it. See health.go.
	mux.Handle("GET /healthz", handleLive())
	mux.Handle("GET /readyz", handleReady(log, ready))
}
