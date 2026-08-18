package sqlstore

// Regression guard for the CRITICAL: the documented wiring used
// SetMaxOpenConns(25) against SQLite with no pragmas, and 48 of 50 concurrent
// writes failed with SQLITE_BUSY -- which is not a unique violation, so it
// surfaced to the client as a 500.

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	_ "modernc.org/sqlite"

	"example.com/user-service/internal/accounts"
)

// onDisk mirrors production shape: a real file, not an in-memory database.
// The bug does not reproduce in memory with cache=shared.
func onDisk(t *testing.T, pragmas string) *sql.DB {
	t.Helper()
	dir, err := os.MkdirTemp("", "sqlstore")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "app.db")+pragmas)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(25) // the documented pool size

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return db
}

func concurrentWrites(t *testing.T, db *sql.DB, n int) (ok int, firstErr error) {
	t.Helper()
	s := NewAccountStore(db)

	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			u := user("w" + string(rune('a'+i%26)) + string(rune('a'+i/26)) + "@example.com")
			u.ID = u.Email
			errs[i] = s.Create(context.Background(), u)
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		if err == nil {
			ok++
		} else if firstErr == nil {
			firstErr = err
		}
	}
	return ok, firstErr
}

// The fix: busy_timeout makes a blocked writer wait instead of failing, and
// WAL lets readers proceed during a write.
func TestConcurrentWritesSucceedWithDocumentedPragmas(t *testing.T) {
	const n = 50
	db := onDisk(t, "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")

	ok, err := concurrentWrites(t, db, n)
	if ok != n {
		t.Fatalf("%d/%d concurrent writes succeeded; first error: %v", ok, n, err)
	}
}

// A SQLITE_BUSY must never be mistaken for a duplicate email, or a transient
// lock would be reported to the client as a 409.
func TestBusyErrorIsNotMistakenForDuplicate(t *testing.T) {
	db := onDisk(t, "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	_, err := concurrentWrites(t, db, 30)
	if errors.Is(err, accounts.ErrEmailTaken) {
		t.Errorf("a lock error was translated to ErrEmailTaken: %v", err)
	}
}
