package httpapi

import (
	"log/slog"
	"net/http"
)

// handleLive answers liveness: "should this process be restarted?"
//
// It consults nothing, and that is the whole design. An orchestrator kills a
// process that fails liveness, and killing a healthy process because its
// database is unreachable fixes nothing: it drops the in-flight requests, loses
// whatever was cached, and adds a reconnect storm to an outage that was already
// in progress. Because the dependency is shared, every replica fails the probe
// at the same instant, so the entire deployment enters a restart loop while the
// processes themselves were fine. This is the single most common way a health
// endpoint makes an incident worse.
//
// The only thing this endpoint can honestly report is that the process is
// running and still scheduling handlers -- which answering at all proves.
func handleLive() http.HandlerFunc {
	type response struct {
		Status string `json:"status"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		_ = encode(w, http.StatusOK, response{Status: "ok"})
	}
}

// handleReady answers readiness: "should this instance be sent traffic?"
//
// That question genuinely does depend on the dependencies, which is exactly why
// it is a second endpoint rather than a second opinion from the first. Failing
// readiness pulls the instance out of the load-balancer pool and LEAVES IT
// RUNNING, so it rejoins the moment its database comes back -- no restart, no
// lost warm state, no thundering herd.
func handleReady(log *slog.Logger, checks []ReadinessCheck) http.HandlerFunc {
	type checkResult struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	type response struct {
		Status string        `json:"status"`
		Checks []checkResult `json:"checks"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Allocated non-nil so an instance with no registered checks encodes
		// "checks":[] and not "checks":null. A nil slice marshals to null, and
		// every dashboard and alert that parses this then has to special-case
		// it -- or, more often, does not and breaks on the one instance that
		// has no dependencies.
		results := make([]checkResult, 0, len(checks))
		ready := true

		// Sequential, not concurrent. Fanning out would save a few milliseconds
		// on a probe nobody is waiting on, in exchange for several goroutines
		// writing to one result set and a synchronisation bug that only appears
		// when a dependency is already failing. The simpler shape is the
		// correct trade here.
		for _, c := range checks {
			// Re-tested before EVERY check rather than once at the top: if the
			// client hung up -- or the request budget expired -- while an
			// earlier check was dialling, probing the rest only piles load onto
			// dependencies that are already struggling, to produce a body
			// nobody is left to read. Nothing is written for the same reason
			// the register handler writes nothing to a cancelled client.
			if ctx.Err() != nil {
				return
			}

			status := "ok"
			if err := c.Check(ctx); err != nil {
				ready = false
				status = "failed"
				// The error goes to the log, never to the body. /readyz is
				// almost always unauthenticated, and a raw dependency error is
				// a free description of the internal topology: hostnames,
				// ports, driver names, schema names. The check's Name is what a
				// responder needs from the body -- which dependency to look at
				// -- and the log line carries the detail they need next.
				log.ErrorContext(ctx, "readiness check failed",
					"check", c.Name,
					"error", err,
				)
			}
			results = append(results, checkResult{Name: c.Name, Status: status})
		}

		// Every check has run before anything is written, because the loop does
		// not stop at the first failure. A body naming one broken dependency
		// while staying silent about the other two is worse than useless during
		// an incident: it reads as "only this one is broken" and sends the
		// responder down a single trail.
		status, code := "ready", http.StatusOK
		if !ready {
			status, code = "not ready", http.StatusServiceUnavailable
		}
		_ = encode(w, code, response{Status: status, Checks: results})
	}
}
