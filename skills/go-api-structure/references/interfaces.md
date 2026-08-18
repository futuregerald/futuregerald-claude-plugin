# Interfaces and Dependency Direction

Every block below marked **complete file** compiles as written. Blocks marked
**fragment** omit imports or surrounding code and are illustrative only.

- [The inversion, end to end](#the-inversion-end-to-end)
- [When an interface earns its place](#when-an-interface-earns-its-place)
- [Sizing an interface](#sizing-an-interface)
- [Fakes](#fakes)
- [Non-determinism: clock, IDs, randomness](#non-determinism-clock-ids-randomness)
- [Transactions across stores](#transactions-across-stores)
- [Breaking an import cycle](#breaking-an-import-cycle)
- [Errors across the boundary](#errors-across-the-boundary)

## The inversion, end to end

Three packages. Note the import arrows: `sqlstore` and `httpapi` both import `accounts`;
`accounts` imports neither.

### `internal/accounts/accounts.go` — the domain declares what it needs (complete file)

```go
package accounts

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound   = errors.New("accounts: user not found")
	ErrEmailTaken = errors.New("accounts: email already registered")
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// Store is declared here — in the package that calls it, not the one that
// implements it. That is the whole inversion.
type Store interface {
	ByEmail(ctx context.Context, email string) (User, error)
	Create(ctx context.Context, u User) error
}

// Hasher keeps the password-hashing library out of the domain, and keeps the
// algorithm swappable — argon2id today, whatever replaces it later.
type Hasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error
}
```

### `internal/accounts/service.go` — rules only, no I/O of its own (complete file)

```go
package accounts

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Service struct {
	store  Store
	hasher Hasher
	now    func() time.Time
	newID  func() string
}

func NewService(store Store, hasher Hasher, now func() time.Time, newID func() string) *Service {
	return &Service{store: store, hasher: hasher, now: now, newID: newID}
}

func (s *Service) Register(ctx context.Context, email, password string) (User, error) {
	// This pre-check exists to return a clean error in the common case. It is
	// NOT the uniqueness guarantee — two concurrent calls can both pass it.
	// The unique index on users.email is the guarantee, and Store.Create
	// reports a violation as ErrEmailTaken. Both paths are handled.
	if _, err := s.store.ByEmail(ctx, email); err == nil {
		return User{}, ErrEmailTaken
	} else if !errors.Is(err, ErrNotFound) {
		return User{}, fmt.Errorf("lookup existing: %w", err)
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	u := User{
		ID:           s.newID(),
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    s.now(),
	}
	if err := s.store.Create(ctx, u); err != nil {
		// Already wrapped as ErrEmailTaken by the adapter when the unique
		// index fires; errors.Is at the transport edge still matches.
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}
```

The service mints the ID rather than letting the database do it, so the value is known
before the write and the caller gets a populated `User` back. Injecting `newID` keeps that
deterministic under test.

This is testable with zero infrastructure — every dependency is an argument.

### `internal/sqlstore/accounts.go` — the adapter reaches inward

Complete file. With `tx.go` (below) it forms `package sqlstore`; the two compile together.

**The storage engine is incidental.** This example uses SQLite so it runs with no server, and
the package is named `sqlstore` rather than `sqlite` because an `internal/sqlite` package
would collide with the driver's own. Swapping in Postgres or MySQL changes exactly two things
— the driver import and `isUniqueViolation`. Nothing about the architecture moves.

Every query goes through `s.q(ctx)`, never `s.db` — see
[Transactions across stores](#transactions-across-stores). A store method that touches `s.db`
directly silently escapes any surrounding transaction, so its write survives a rollback.

```go
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

// isUniqueViolation is the ONLY driver-specific function in this package, and
// the only thing that changes when the storage engine does:
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
```

If a DB row needs tags or columns the domain type shouldn't carry, declare a private
`type userRow struct { … }` in `sqlstore` and map it. Keep `db:` tags out of `accounts.User`.

### `internal/httpapi/accounts.go` — transport owns the wire format (complete file)

```go
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
```

Status codes live here and nowhere else. The domain returns errors; transport decides they
mean 409. Because the service wraps with `%w`, `errors.Is` still matches through the wrap.

## When an interface earns its place

Introduce one when **any** of these is true:

- The call leaves the process — DB, HTTP, queue, cache, blob store, mail, shell.
- A second real implementation exists or is imminent.
- A test needs to force an error path that is impractical to trigger for real.

Do **not** introduce one for:

- A struct with one implementation, used in one place, tested by calling it.
- "Future-proofing" a swap nobody has asked for.
- Non-determinism — clock, randomness, IDs. Inject a function value instead; see
  [Non-determinism](#non-determinism-clock-ids-randomness).
- Every service, on principle. `*accounts.Service` is a fine parameter type — its
  dependencies are already injected, so it is already substitutable where it matters.

The tell for over-application: an interface named `FooInterface`, `IFoo`, or a `Foo`
interface sitting next to the only `foo` that implements it.

## Sizing an interface

Interfaces belong to the caller, so they should be as small as the caller's actual use.
Two callers needing different subsets get two interfaces, not one union.

```go
// fragment — in a package OTHER than accounts, which is why the type is
// qualified. Inside accounts itself it would be plain User.
type userReader interface {
	ByEmail(ctx context.Context, email string) (accounts.User, error)
}
```

A single `Repository` with 20 methods forces every fake to stub 20 methods and tells you
nothing about what the caller touches.

## Fakes

Hand-written fakes beat generated mocks for these narrow interfaces: no codegen step, no
mock-framework DSL, and they read as ordinary Go.

### `internal/accounts/fake_test.go` (complete file)

```go
package accounts_test

import (
	"context"

	"example.com/user-service/internal/accounts"
)

type fakeStore struct {
	users     map[string]accounts.User
	createErr error // force the failure path
}

func newFakeStore() *fakeStore {
	// The map must be initialised — assigning into a nil map panics.
	return &fakeStore{users: make(map[string]accounts.User)}
}

func (f *fakeStore) ByEmail(_ context.Context, email string) (accounts.User, error) {
	u, ok := f.users[email]
	if !ok {
		return accounts.User{}, accounts.ErrNotFound
	}
	return u, nil
}

func (f *fakeStore) Create(_ context.Context, u accounts.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.users[u.Email] = u
	return nil
}
```

Reach for a mock library only when call-order or call-count assertions genuinely matter.

## Non-determinism: clock, IDs, randomness

These do not leave the process, so they do **not** get an interface. Inject them as plain
function values:

```go
// fragment — field declarations only
type Service struct {
	now   func() time.Time
	newID func() string
}
```

Production passes `time.Now` and a UUID function; a test passes fixed values:

```go
// fragment — inside a test
svc := accounts.NewService(
	newFakeStore(),
	stubHasher{},
	func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	func() string { return "user-1" },
)
```

This is why `NewService` takes `now` and `newID` — asserting on `CreatedAt` or `ID`
requires it. A `Clock` interface would be the anti-pattern flagged above: one method, one
production implementation, one test implementation.

## Transactions across stores

The hard case the layout rules do not solve on their own: one workflow must write through
two stores atomically. The domain may not import `database/sql`, and adapters may not
import each other, so neither side can own the transaction.

Resolve it with an interface the domain declares and the adapter implements, carrying the
transaction in the context so store methods stay unchanged.

```go
// fragment — in internal/accounts, beside the other interfaces
type Atomic interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}
```

The service composes writes without knowing what a transaction is:

```go
// fragment — a method on *Service
func (s *Service) RegisterWithBilling(ctx context.Context, email, password string) error {
	return s.atomic.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.Register(ctx, email, password); err != nil {
			return err
		}
		return s.billing.OpenAccount(ctx, email)
	})
}
```

### `internal/sqlstore/tx.go` — the adapter side (complete file)

```go
package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
)

type txKey struct{}

// querier is satisfied by both *sql.DB and *sql.Tx, so store methods do not
// care whether they are inside a transaction.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type TxManager struct{ db *sql.DB }

func NewTxManager(db *sql.DB) *TxManager { return &TxManager{db: db} }

func (m *TxManager) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	// Rollback after a successful Commit is a no-op, so this is safe
	// unconditionally and covers panics and early returns.
	defer func() { _ = tx.Rollback() }()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// q returns the transaction bound to ctx if there is one, else the pool.
// Every store method uses this instead of touching s.db directly.
func (s *AccountStore) q(ctx context.Context) querier {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return s.db
}
```

Trade-off worth stating: the transaction travels implicitly in the context, so a store
method's behaviour depends on how it was called. That is the price of keeping `database/sql`
out of the domain, and it is the usual bargain in Go. The alternative — passing `*sql.Tx`
through domain signatures — leaks the driver into the layer this whole structure exists to
protect.

If the two writes span **services rather than stores**, a shared transaction is not
available. Use the transactional outbox: write the row and an event in one local
transaction, publish from the outbox, and make the consumer idempotent.

## Breaking an import cycle

A cycle is a design signal, not a packaging accident. `accounts` ⇄ `billing` means one of
them is calling the other concretely.

Fix in this order:

1. **Move the interface to the consumer.** If `billing` needs to look up a user, `billing`
   declares `type UserLookup interface { ByID(...) (billing.Customer, error) }` and something
   in `cmd/` adapts `accounts` to it. The cycle disappears without a new package.
2. **Communicate by event.** If the coupling is "when X happens, Y should react", publish
   from `accounts` and subscribe in `billing` — wired in `cmd/`.
3. **Split the genuinely shared concept** into its own domain package both depend on.
   Legitimate, but only after 1 and 2 fail.

Never resolve a cycle by creating `internal/types` or `internal/shared` to hold the structs.
That converts one cycle into a package every future change has to edit.

## Errors across the boundary

- Domain packages define sentinel errors (`accounts.ErrNotFound`) or typed errors.
- Adapters translate foreign errors into domain errors at their edge — `sql.ErrNoRows` and
  a unique-constraint code never escape `sqlstore`.
- Transport translates domain errors into status codes at its edge.
- Wrap with `%w` when adding context; check with `errors.Is`/`errors.As`, never string
  matching.

Each layer only knows the vocabulary of the layer directly inside it.
