// Package functional wires the REAL stack — HTTP transport, domain service,
// SQL adapter, real database — and drives it through the front door.
//
// These are the tests that earn their keep. Every component in this repo has
// unit tests and they all pass; the defects that actually shipped were an
// adapter that bypassed its transaction (so writes survived rollback) and a
// worker pool that accepted jobs it never ran. Both were invisible to unit
// tests, because each function was individually correct. Only exercising the
// flow across components found them.
//
// So: mostly failure paths. The happy path is one test; the rest are the ways
// a real system goes wrong — duplicates, races, dead databases, timeouts,
// disconnects, and partial writes.
package functional

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"example.com/user-service/internal/accounts"
	"example.com/user-service/internal/httpapi"
	"example.com/user-service/internal/sqlstore"
)

const schema = `
CREATE TABLE users (
  id            TEXT PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at    TIMESTAMP NOT NULL
);`

// stubHasher stands in for internal/pwhash. Production uses argon2id via a
// vetted wrapper; reversing this one is trivial, which is the point of keeping
// hashing behind accounts.Hasher.
type stubHasher struct{}

func (stubHasher) Hash(plain string) (string, error) { return "hashed:" + plain, nil }
func (stubHasher) Compare(hash, plain string) error  { return nil }

type stack struct {
	db      *sql.DB
	handler http.Handler
}

// newStack builds the whole service: real SQLite, real adapter, real domain
// service, real router. Nothing is mocked below the HTTP boundary.
func newStack(t *testing.T) *stack {
	t.Helper()

	dir, err := os.MkdirTemp("", "functional")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	dsn := "file:" + filepath.Join(dir, "app.db") +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("schema: %v", err)
	}

	var seq int64
	var mu sync.Mutex
	newID := func() string {
		mu.Lock()
		defer mu.Unlock()
		seq++
		return fmt.Sprintf("user-%d", seq)
	}

	svc := accounts.NewService(sqlstore.NewAccountStore(db), stubHasher{}, time.Now, newID)
	srv := httpapi.NewServer(httpapi.ServerConfig{
		Addr:           "127.0.0.1:0",
		RequestTimeout: 5 * time.Second,
	}, slog.New(slog.DiscardHandler), svc)

	return &stack{db: db, handler: srv.Handler()}
}

func (s *stack) register(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	return rec
}

func (s *stack) countUsers(t *testing.T, email string) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE email = ?`, email).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// ---------------------------------------------------------------- happy path

func TestRegisterPersistsUserAndReturnsIt(t *testing.T) {
	s := newStack(t)

	rec := s.register(t, `{"email":"a@example.com","password":"correct-horse"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", rec.Code, rec.Body)
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["id"] == "" || got["id"] == nil {
		t.Errorf("response carried no id: %s", rec.Body)
	}

	// The response saying 201 is not evidence the row exists. Check the database.
	if n := s.countUsers(t, "a@example.com"); n != 1 {
		t.Errorf("rows for a@example.com = %d, want 1", n)
	}
}

// ------------------------------------------------------------- failure paths

func TestDuplicateEmailIsRejectedAndDoesNotCreateASecondRow(t *testing.T) {
	s := newStack(t)

	if rec := s.register(t, `{"email":"dup@example.com","password":"pw"}`); rec.Code != http.StatusCreated {
		t.Fatalf("first register: %d", rec.Code)
	}
	rec := s.register(t, `{"email":"dup@example.com","password":"different"}`)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
	if n := s.countUsers(t, "dup@example.com"); n != 1 {
		t.Errorf("rows = %d, want 1 — a duplicate slipped through", n)
	}
}

// The race the service's pre-check cannot close: many clients registering the
// same address at once. Exactly one must win, and the losers must see 409 —
// not 500, which is what a raw constraint error would produce.
func TestConcurrentDuplicateRegistrationsYieldExactlyOneUser(t *testing.T) {
	s := newStack(t)

	const n = 12
	codes := make([]int, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = s.register(t, `{"email":"race@example.com","password":"pw"}`).Code
		}(i)
	}
	wg.Wait()

	var created, conflict, other int
	for _, c := range codes {
		switch c {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflict++
		default:
			other++
		}
	}

	if created != 1 {
		t.Errorf("201s = %d, want exactly 1", created)
	}
	if other != 0 {
		t.Errorf("%d requests got neither 201 nor 409 — a lock or constraint error leaked", other)
	}
	if conflict != n-1 {
		t.Errorf("409s = %d, want %d", conflict, n-1)
	}
	if got := s.countUsers(t, "race@example.com"); got != 1 {
		t.Errorf("rows = %d, want 1", got)
	}
}

func TestMalformedBodyIsRejectedAndWritesNothing(t *testing.T) {
	s := newStack(t)

	rec := s.register(t, `{"email": "broken`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}

	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("%d rows written for a malformed request", n)
	}
}

// A dead database must surface as a 500 whose body tells the client nothing
// about the schema, the driver, or the query.
func TestDatabaseFailureDoesNotLeakInternals(t *testing.T) {
	s := newStack(t)
	_ = s.db.Close() // simulate the database going away mid-life

	rec := s.register(t, `{"email":"x@example.com","password":"pw"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	body := strings.ToLower(rec.Body.String())
	for _, leak := range []string{"sql", "sqlite", "database", "users", "insert", "select"} {
		if strings.Contains(body, leak) {
			t.Errorf("response leaked %q to the client: %s", leak, rec.Body)
		}
	}
}

// Whatever else happens, the plaintext password must never reach the database
// or the response.
func TestPasswordNeverStoredOrReturnedInPlaintext(t *testing.T) {
	const secret = "super-secret-value"
	s := newStack(t)

	rec := s.register(t, `{"email":"p@example.com","password":"`+secret+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), secret) {
		t.Errorf("password echoed in the response: %s", rec.Body)
	}

	var hash string
	if err := s.db.QueryRow(`SELECT password_hash FROM users WHERE email = ?`,
		"p@example.com").Scan(&hash); err != nil {
		t.Fatalf("select: %v", err)
	}
	if hash == secret {
		t.Error("password stored in plaintext")
	}
	if !strings.Contains(hash, "hashed:") {
		t.Errorf("password_hash = %q — not produced by the Hasher", hash)
	}
}

// Atomicity across two writes. If the second fails, the first must not survive
// — the failure mode that unit tests on either write cannot see.
func TestPartialWriteIsRolledBackEntirely(t *testing.T) {
	s := newStack(t)
	store := sqlstore.NewAccountStore(s.db)
	tm := sqlstore.NewTxManager(s.db)
	ctx := context.Background()

	first := accounts.User{
		ID: "tx-1", Email: "tx@example.com",
		PasswordHash: "h", CreatedAt: time.Now(),
	}
	boom := errors.New("second step failed")

	err := tm.InTx(ctx, func(ctx context.Context) error {
		if err := store.Create(ctx, first); err != nil {
			return err
		}
		return boom // e.g. the billing account could not be opened
	})
	if !errors.Is(err, boom) {
		t.Fatalf("InTx err = %v, want %v", err, boom)
	}

	if n := s.countUsers(t, "tx@example.com"); n != 0 {
		t.Errorf("%d rows survived a rolled-back transaction", n)
	}

	// And the address must still be free afterwards.
	if rec := s.register(t, `{"email":"tx@example.com","password":"pw"}`); rec.Code != http.StatusCreated {
		t.Errorf("re-register after rollback = %d, want 201", rec.Code)
	}
}

// A client that hangs up mid-request must not be reported as a server error,
// and must not leave a half-written row.
func TestCancelledRequestIsNotAServerError(t *testing.T) {
	s := newStack(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // client is already gone

	req := httptest.NewRequest(http.MethodPost, "/users",
		strings.NewReader(`{"email":"gone@example.com","password":"pw"}`)).WithContext(ctx)
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusInternalServerError {
		t.Error("a client disconnect was reported as a 500")
	}
	if n := s.countUsers(t, "gone@example.com"); n != 0 {
		t.Errorf("%d rows written for a cancelled request", n)
	}
}

// The pre-check in Register cannot close the duplicate-email race: another
// caller can insert between the lookup and the write. In production that
// window is opened by timing; here it is opened deterministically by a Hasher
// that inserts the conflicting row as a side effect, because it runs at
// exactly the right moment.
//
// This is the path the unique index exists to defend, and the one a purely
// concurrent test exercises only by luck — I wrote such a test first and it
// passed against a broken adapter, which is precisely the false confidence
// functional tests are supposed to eliminate.
type raceHasher struct {
	db   *sql.DB
	once sync.Once
}

func (h *raceHasher) Hash(plain string) (string, error) {
	h.once.Do(func() {
		_, _ = h.db.Exec(
			`INSERT INTO users (id, email, password_hash, created_at) VALUES (?, ?, ?, ?)`,
			"interloper", "race2@example.com", "h", time.Now())
	})
	return "hashed:" + plain, nil
}

func (h *raceHasher) Compare(hash, plain string) error { return nil }

func TestDuplicateLosingTheRaceIsA409NotA500(t *testing.T) {
	s := newStack(t)

	// Rebuild the service with a hasher that races us, then re-route.
	svc := accounts.NewService(
		sqlstore.NewAccountStore(s.db), &raceHasher{db: s.db},
		time.Now, func() string { return "user-race" })
	srv := httpapi.NewServer(httpapi.ServerConfig{
		Addr: "127.0.0.1:0", RequestTimeout: 5 * time.Second},
		slog.New(slog.DiscardHandler), svc)

	req := httptest.NewRequest(http.MethodPost, "/users",
		strings.NewReader(`{"email":"race2@example.com","password":"pw"}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code == http.StatusInternalServerError {
		t.Fatalf("losing the insert race produced a 500 — the adapter is not "+
			"translating the unique-constraint violation (body %s)", rec.Body)
	}
	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
	if n := s.countUsers(t, "race2@example.com"); n != 1 {
		t.Errorf("rows = %d, want 1", n)
	}
}
