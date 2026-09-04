package httpapi

import (
	"context"
	"errors"
	"net/http"

	"example.com/user-service/internal/accounts"
)

// Declared here, unexported, because this is what the handler actually uses.
// It keeps the handler testable without constructing a real Service.
type registrar interface {
	Register(ctx context.Context, email, password string) (accounts.User, error)
}

func handleRegister(svc registrar) http.HandlerFunc {
	// Declared inside the handler so no other handler can couple to a shape
	// this one has not promised to keep.
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}

	// Validation is a handler-local closure, not a Validator interface method.
	// Go forbids declaring methods on function-local types -- `func (r request)
	// Valid() ...` inside a function is a syntax error, verified -- so a
	// Validator interface would force `request` back to package scope and give
	// up the encapsulation that declaring it here buys.
	// Transport validates SHAPE AND PRESENCE only. A password-strength rule is a
	// business rule and belongs in internal/accounts -- putting it here would
	// have the transport layer owning domain policy, which is the exact
	// inversion this skill exists to prevent. It would also be unenforceable:
	// any other entry point (a CLI, a seed script, a gRPC handler) would bypass
	// it entirely.
	// This is also the only thing standing between a body of `null` and a 201.
	// `null` into a struct is a documented no-op: it decodes successfully and
	// leaves the zero value, so it arrives here indistinguishable from `{}`.
	// The decoder cannot catch either -- both are valid JSON -- so the presence
	// check is what turns them into a 422.
	valid := func(r request) map[string]string {
		problems := map[string]string{}
		if r.Email == "" {
			problems["email"] = "is required"
		}
		if r.Password == "" {
			problems["password"] = "is required"
		}
		return problems
	}

	return func(w http.ResponseWriter, r *http.Request) {
		req, err := decode[request](w, r)
		if err != nil {
			status, msg := decodeStatus(err)
			writeError(w, status, msg)
			return
		}
		if problems := valid(req); len(problems) > 0 {
			_ = encode(w, http.StatusUnprocessableEntity,
				map[string]any{"error": "invalid request", "problems": problems})
			return
		}

		u, err := svc.Register(r.Context(), req.Email, req.Password)
		switch {
		case errors.Is(err, accounts.ErrEmailTaken):
			writeError(w, http.StatusConflict, "email already registered")
			return
		case errors.Is(err, context.Canceled):
			// The client hung up. Nobody will read a response body, and this
			// is not a server fault -- do not log it as one.
			return
		case errors.Is(err, context.DeadlineExceeded):
			// We were too slow. Distinct from the 503 TimeoutHandler writes
			// when the whole request budget expires.
			writeError(w, http.StatusGatewayTimeout, "request timed out")
			return
		case err != nil:
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		_ = encode(w, http.StatusCreated, response{ID: u.ID, Email: u.Email})
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	_ = encode(w, status, map[string]string{"error": msg})
}
