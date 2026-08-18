package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"example.com/user-service/internal/accounts"
)

// Declared here, unexported, because this is what the handler actually uses.
// It keeps the handler testable without constructing a real Service.
type registrar interface {
	Register(ctx context.Context, email, password string) (accounts.User, error)
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func handleRegister(svc registrar) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "malformed body")
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

		writeJSON(w, http.StatusCreated, userResponse{ID: u.ID, Email: u.Email})
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
