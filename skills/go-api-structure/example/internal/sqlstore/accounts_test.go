package sqlstore

// Runs the doc's adapter against a real (in-process) SQL engine. No server,
// no fixtures beyond a schema. Covers the error-translation patterns and the
// transaction semantics, which are the two things compiling cannot prove.

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"example.com/user-service/internal/accounts"
)

const schema = `
CREATE TABLE users (
  id            TEXT PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at    TIMESTAMP NOT NULL
);`

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	// Shared cache + a single connection so the in-memory DB is visible to
	// every query and transactions behave deterministically.
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return db
}

func user(email string) accounts.User {
	return accounts.User{
		ID:           "id-" + email,
		Email:        email,
		PasswordHash: "hashed",
		CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestByEmailTranslatesNoRows(t *testing.T) {
	s := NewAccountStore(newDB(t))

	_, err := s.ByEmail(context.Background(), "nobody@example.com")
	if !errors.Is(err, accounts.ErrNotFound) {
		t.Errorf("err = %v, want accounts.ErrNotFound", err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		t.Errorf("sql.ErrNoRows leaked past the adapter boundary: %v", err)
	}
}

func TestCreateThenByEmailRoundTrip(t *testing.T) {
	s := NewAccountStore(newDB(t))
	ctx := context.Background()
	want := user("round@example.com")

	if err := s.Create(ctx, want); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.ByEmail(ctx, want.Email)
	if err != nil {
		t.Fatalf("ByEmail: %v", err)
	}
	if got.ID != want.ID || got.Email != want.Email || got.PasswordHash != want.PasswordHash {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("CreatedAt got %v, want %v", got.CreatedAt, want.CreatedAt)
	}
}

// The finding-4 fix, against a real engine: a duplicate must surface as
// ErrEmailTaken (-> 409), never as a raw driver error (-> 500).
func TestCreateDuplicateIsErrEmailTaken(t *testing.T) {
	s := NewAccountStore(newDB(t))
	ctx := context.Background()

	if err := s.Create(ctx, user("dup@example.com")); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	second := user("dup@example.com")
	second.ID = "id-second"

	err := s.Create(ctx, second)
	if !errors.Is(err, accounts.ErrEmailTaken) {
		t.Fatalf("duplicate Create err = %v (%T), want accounts.ErrEmailTaken", err, err)
	}
}

func TestInTxCommits(t *testing.T) {
	db := newDB(t)
	s, tm := NewAccountStore(db), NewTxManager(db)
	ctx := context.Background()

	if err := tm.InTx(ctx, func(ctx context.Context) error {
		return s.Create(ctx, user("commit@example.com"))
	}); err != nil {
		t.Fatalf("InTx: %v", err)
	}
	if _, err := s.ByEmail(ctx, "commit@example.com"); err != nil {
		t.Errorf("after commit, ByEmail: %v", err)
	}
}

// This is the test that caught the doc using s.db instead of s.q(ctx): if the
// store does not pick the transaction up off the context, the write escapes
// the transaction and survives the rollback.
func TestInTxRollsBackOnError(t *testing.T) {
	db := newDB(t)
	s, tm := NewAccountStore(db), NewTxManager(db)
	ctx := context.Background()
	boom := errors.New("business rule said no")

	err := tm.InTx(ctx, func(ctx context.Context) error {
		if err := s.Create(ctx, user("rollback@example.com")); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("InTx err = %v, want %v", err, boom)
	}

	if _, err := s.ByEmail(ctx, "rollback@example.com"); !errors.Is(err, accounts.ErrNotFound) {
		t.Errorf("row survived rollback: ByEmail err = %v, want ErrNotFound", err)
	}
}

// Cancellation must actually reach the driver, which is the entire reason the
// adapter uses QueryRowContext/ExecContext rather than QueryRow/Exec.
func TestCancelledContextStopsTheQuery(t *testing.T) {
	s := NewAccountStore(newDB(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before the call

	_, err := s.ByEmail(ctx, "anyone@example.com")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want it to wrap context.Canceled", err)
	}
}

func TestExpiredDeadlineStopsTheWrite(t *testing.T) {
	s := NewAccountStore(newDB(t))
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	err := s.Create(ctx, user("late@example.com"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want it to wrap context.DeadlineExceeded", err)
	}
}
