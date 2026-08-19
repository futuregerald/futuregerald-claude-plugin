package httpapi

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// requestIDHeader is the wire name of the correlation ID. It is a constant
// because it is read on the way in and written on the way out, and a typo in
// one of those two places is invisible at compile time.
const requestIDHeader = "X-Request-Id"

// ctxKey is unexported so no other package can create the same key. A plain
// string key would collide with any other package that also stores "request_id"
// in a context, silently overwriting it.
type ctxKey int

const requestIDKey ctxKey = iota

// requestIDFrom returns the ID requestID stored, or "" if the middleware did
// not run. It stays unexported because everything that logs in this example
// lives in this package; a service that logs from its domain layer would export
// it (or, better, pass the ID as an argument rather than smuggle a context).
func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// chain wraps h so that the FIRST middleware listed is the OUTERMOST -- the
// order they are written is the order a request travels through them. It is
// applied back-to-front for that reason.
//
// This is the whole of the "middleware framework" this example needs. Anything
// more (named stacks, per-route groups, registration order resolution) buys
// indirection, not capability.
func chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

// recordingWriter remembers the status code and whether anything was written,
// which the ResponseWriter interface itself will not tell you: after the
// handler returns there is no way to ask "did this respond, and with what?".
// recoverPanic needs the second question answered and requestLog the first,
// so one wrapper serves both.
//
// It deliberately does NOT implement http.Flusher, http.Hijacker or
// http.Pusher. Wrapping a ResponseWriter silently drops any optional interface
// the original satisfied, so a production version needs explicit pass-through
// (or a library such as felixge/httpsnoop); streaming and WebSocket upgrades
// break without it. This example has no such handler, and the omission is
// clearer here than the twenty lines of forwarding would be.
type recordingWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (w *recordingWriter) WriteHeader(status int) {
	// net/http ignores a second WriteHeader and logs a "superfluous
	// WriteHeader" warning. Mirroring that here keeps the recorded status
	// equal to the one the client actually received.
	if w.written {
		return
	}
	w.status = status
	w.written = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *recordingWriter) Write(b []byte) (int, error) {
	// A handler that writes a body without calling WriteHeader has implicitly
	// sent a 200; record that rather than reporting the zero value.
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// recoverPanic turns a panicking handler into a 500 and a log line, instead of
// a killed process. It is the outermost middleware so that it also covers
// panics raised by the middleware inside it -- the cost of that placement is
// that the request ID is not yet in the context here, so the panic log cannot
// carry it.
func recoverPanic(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &recordingWriter{ResponseWriter: w, status: http.StatusOK}

			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				// ErrAbortHandler is net/http's documented way for a handler to
				// abandon a response deliberately; the server expects to catch
				// it and drop the connection silently. Swallowing it here would
				// convert an intentional abort into a bogus 500.
				//
				// recover() hands back an any, so `rec == http.ErrAbortHandler`
				// would compare interface values and miss a panic value that
				// WRAPS the sentinel. Assert to error first, then errors.Is;
				// re-panic with rec, not err, so the value the server catches
				// and the stack it unwinds are the original ones.
				if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}

				// Stack captured at recovery time, while the frames are still
				// the panicking ones. A log line without it names the URL but
				// not the line of code.
				log.ErrorContext(r.Context(), "panic recovered",
					"panic", rec,
					"method", r.Method,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)

				// The status is already on the wire and cannot be recalled, so
				// appending an error object would only corrupt a body the
				// client is mid-way through parsing. Logged, not written.
				if rw.written {
					return
				}
				writeError(rw, http.StatusInternalServerError, "internal error")
			}()

			next.ServeHTTP(rw, r)
		})
	}
}

// requestID gives every request a correlation ID, on the response header and in
// the context.
//
// An inbound ID is honoured rather than replaced: that is what lets one ID
// follow a request across services, which is the entire point of having one. A
// service exposed directly to the internet should validate the inbound value
// (length, character set) before echoing it into headers and logs.
func requestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(requestIDHeader)
			if id == "" {
				// crypto/rand.Text is documented never to return an error and
				// to carry at least 128 bits of randomness, so there is no
				// failure path to handle and no UUID dependency to take on.
				id = rand.Text()
			}

			// Set BEFORE delegating. http.TimeoutHandler buffers the inner
			// response and, on the success path, copies its buffered headers
			// over this map (maps.Copy, no clear) -- so a header set here
			// survives, while one set after next.ServeHTTP returns would be
			// written too late to reach the client.
			w.Header().Set(requestIDHeader, id)

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
		})
	}
}

// requestLog writes exactly one line per request, at the edge. Handlers below
// stay silent: log once where every request necessarily passes, and the volume
// is predictable and the fields are uniform.
//
// It sits outside TimeoutHandler so it reports the status the client actually
// received -- including the 503 the timeout itself writes -- and so its
// recordingWriter is only ever touched by this goroutine. Inside TimeoutHandler
// it would run on the handler goroutine, which may still be alive after the
// timeout has responded: a data race and a wrong status.
func requestLog(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &recordingWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rw, r)

			log.InfoContext(r.Context(), "http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", requestIDFrom(r.Context()),
			)
		})
	}
}
