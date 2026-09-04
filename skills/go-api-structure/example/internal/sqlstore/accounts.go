package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sqlitedriver "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"example.com/user-service/internal/accounts"
)

// Compile-time proof the contract still holds. Free, and it fails at build
// time rather than at wiring time in main.
var _ accounts.Store = (*AccountStore)(nil)

type AccountStore struct{ db *sql.DB }

func NewAccountStore(db *sql.DB) *AccountStore { return &AccountStore{db: db} }

func (s *AccountStore) ByEmail(ctx context.Context, email string) (accounts.User, error) {
	const query = `SELECT id, email, password_hash, created_at FROM users WHERE email = ?`

	var u accounts.User
	err := s.q(ctx).QueryRowContext(ctx, query, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Translate the driver's error into the domain's vocabulary here.
		// accounts must never learn what sql.ErrNoRows is.
		return accounts.User{}, accounts.ErrNotFound
	case err != nil:
		return accounts.User{}, fmt.Errorf("query user by email: %w", err)
	}
	return u, nil
}

func (s *AccountStore) Create(ctx context.Context, u accounts.User) error {
	const query = `INSERT INTO users (id, email, password_hash, created_at)
	               VALUES (?, ?, ?, ?)`

	_, err := s.q(ctx).ExecContext(ctx, query, u.ID, u.Email, u.PasswordHash, u.CreatedAt)
	if isUniqueViolation(err) {
		// The race the service pre-check cannot close is closed here.
		return accounts.ErrEmailTaken
	}
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// isUniqueViolation is the driver-specific error check. Swapping engines also
// means changing the placeholder syntax above (? vs $1) and possibly DSN/scan
// options -- all confined to this package. The equivalents:
//
//	Postgres (pgx): var e *pgconn.PgError;    errors.As(err, &e) && e.Code == "23505"
//	MySQL:          var e *mysql.MySQLError;  errors.As(err, &e) && e.Number == 1062
//
// Keeping it in one function is what makes "swap the database" a real claim
// rather than an aspiration.
func isUniqueViolation(err error) bool {
	var e *sqlitedriver.Error
	return errors.As(err, &e) && e.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
}
