# Interfaces and Dependency Direction

- [The inversion, end to end](#the-inversion-end-to-end)
- [When an interface earns its place](#when-an-interface-earns-its-place)
- [Sizing an interface](#sizing-an-interface)
- [Fakes](#fakes)
- [Non-determinism: clock, IDs, randomness](#non-determinism-clock-ids-randomness)
- [Breaking an import cycle](#breaking-an-import-cycle)
- [Errors across the boundary](#errors-across-the-boundary)

## The inversion, end to end

Three packages. Note the import arrows: `postgres` and `httpapi` both import `accounts`;
`accounts` imports neither.

### `internal/accounts/accounts.go` — the domain declares what it needs

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

// Hasher keeps bcrypt/argon2 out of the domain.
type Hasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error
}
```

### `internal/accounts/service.go` — rules only, no I/O of its own

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
}

func NewService(store Store, hasher Hasher, now func() time.Time) *Service {
	return &Service{store: store, hasher: hasher, now: now}
}

func (s *Service) Register(ctx context.Context, email, password string) (User, error) {
	if _, err := s.store.ByEmail(ctx, email); err == nil {
		return User{}, ErrEmailTaken
	} else if !errors.Is(err, ErrNotFound) {
		return User{}, fmt.Errorf("lookup existing: %w", err)
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	u := User{Email: email, PasswordHash: hash, CreatedAt: s.now()}
	if err := s.store.Create(ctx, u); err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}
```

This is testable with zero infrastructure — every dependency is an argument.

### `internal/postgres/accounts.go` — the adapter reaches inward

```go
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"example.com/user-service/internal/accounts"
)

// Compile-time proof the contract still holds. Free, and it fails at build
// time rather than at wiring time in main.
var _ accounts.Store = (*AccountStore)(nil)

type AccountStore struct{ db *sql.DB }

func NewAccountStore(db *sql.DB) *AccountStore { return &AccountStore{db: db} }

func (s *AccountStore) ByEmail(ctx context.Context, email string) (accounts.User, error) {
	const q = `SELECT id, email, password_hash, created_at FROM users WHERE email = $1`

	var u accounts.User
	err := s.db.QueryRowContext(ctx, q, email).
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
```

If a DB row needs tags or columns the domain type shouldn't carry, declare a private
`type userRow struct { … }` in `postgres` and map it. Keep `db:` tags out of `accounts.User`.

### `internal/httpapi/accounts.go` — transport owns the wire format

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
```

Status codes live here and nowhere else. The domain returns errors; transport decides they
mean 409.

## When an interface earns its place

Introduce one when **any** of these is true:

- The call leaves the process — DB, HTTP, queue, cache, blob store, mail, shell.
- The result is non-deterministic — time, randomness, UUIDs.
- A second real implementation exists or is imminent.
- A test needs to force an error path that is impractical to trigger for real.

Do **not** introduce one for:

- A struct with one implementation, used in one place, tested by calling it.
- "Future-proofing" a swap nobody has asked for.
- Every service, on principle. `*accounts.Service` is a fine parameter type — its
  dependencies are already injected, so it is already substitutable where it matters.

The tell for over-application: an interface named `FooInterface`, `IFoo`, or a `Foo`
interface sitting next to the only `foo` that implements it.

## Sizing an interface

Interfaces belong to the caller, so they should be as small as the caller's actual use.
Two callers needing different subsets get two interfaces, not one union.

```go
// In the package that only reads:
type userReader interface {
	ByEmail(ctx context.Context, email string) (User, error)
}
```

A single `Repository` with 20 methods forces every fake to stub 20 methods and tells you
nothing about what the caller touches.

## Fakes

Hand-written fakes beat generated mocks for these narrow interfaces: no codegen step, no
mock-framework DSL, and they read as ordinary Go.

```go
package accounts_test

type fakeStore struct {
	users     map[string]accounts.User
	createErr error // force the failure path
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

Inject them as plain function values rather than inventing a `Clock` interface:

```go
type Service struct {
	now   func() time.Time
	newID func() string
}
```

Production passes `time.Now` and a UUID function; tests pass fixed values. This is why
`service.go` above takes `now func() time.Time` — asserting on `CreatedAt` requires it.

## Breaking an import cycle

A cycle is a design signal, not a packaging accident. `accounts` ⇄ `billing` means one of
them is calling the other concretely.

Fix in this order:

1. **Move the interface to the consumer.** If `billing` needs to look up a user, `billing`
   declares `type UserLookup interface { ByID(...) (billing.Customer, error) }` and something
   in `cmd/` adapts `accounts` to it. The cycle disappears without a new package.
2. **Communicate by event.** If the coupling is "when X happens, Y should react", publish
   from `accounts` and subscribe in `billing` — wired in `cmd/`.
3. **Split the genuinely shared concept** into its own domain package it turns out both
   depend on. Legitimate, but only after 1 and 2 fail.

Never resolve a cycle by creating `internal/types` or `internal/shared` to hold the structs.
That converts one cycle into a package every future change has to edit.

## Errors across the boundary

- Domain packages define sentinel errors (`accounts.ErrNotFound`) or typed errors.
- Adapters translate foreign errors into domain errors at their edge.
- Transport translates domain errors into status codes at its edge.
- Wrap with `%w` when adding context; check with `errors.Is`/`errors.As`, never string
  matching.

Each layer only knows the vocabulary of the layer directly inside it.
