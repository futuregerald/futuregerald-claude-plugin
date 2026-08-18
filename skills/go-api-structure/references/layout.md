# Layout, Wiring, and Growth

- [Per-directory contracts](#per-directory-contracts)
- [Growing through the tiers](#growing-through-the-tiers)
- [Config](#config)
- [Wiring in main](#wiring-in-main)
- [Graceful shutdown](#graceful-shutdown)
- [Context: deadlines, cancellation, values](#context-deadlines-cancellation-values)
- [Test file placement](#test-file-placement)

## Per-directory contracts

Each directory has one job and one import rule. The import rule is the part that matters.

### `cmd/<binary>/`

Package `main`, one directory per binary. Its only job: load config, construct dependencies,
start a transport, block, shut down cleanly.

- **May import:** everything.
- **Nothing may import it.**
- Keep it under ~150 lines. When it grows past that, extract a `newServer(cfg) (*server, error)`
  helper into the binary's own package — not a shared `dependencies` package that every binary
  imports and only some of it uses.

Name binaries for their role (`api`, `worker`, `migrate`), not their protocol.

### `internal/config/`

Env vars and flags into a typed struct, validated once, at startup.

- **May import:** stdlib and a config library.
- **May not import:** any other `internal/` package. Config is a leaf.
- **Nothing outside `cmd/` imports `config`.** Each component declares the settings struct it
  needs (`httpapi.ServerConfig`) and `main` maps the loaded config onto it. Same principle as
  consumer-declared interfaces: the component states its requirement, and the outer layer
  satisfies it. This is why `config` can be a leaf and still serve everyone.

### `internal/<domain>/`

The reason the service exists. Entities, invariants, workflows, and the interfaces the domain
needs to do its job.

- **May import:** stdlib, and other domain packages only when the dependency is genuinely
  one-directional.
- **May not import:** `net/http`, `database/sql`, drivers, SDKs, `internal/httpapi`,
  `internal/sqlstore`, or anything else in the outer ring.
- Files: `<domain>.go` for types, errors, and interfaces; `service.go` for behaviour. Split
  `service.go` by cohesive group (`registration.go`, `authentication.go`) when it gets long —
  splitting into *packages* is what to avoid, not splitting into files.

### `internal/<adapter>/` — `sqlstore`, `redis`, `kafka`, `s3`, `stripe`

One package per external system. Implements interfaces the domain declared.

**The skill is not opinionated about which storage engine you pick.** The worked example uses
SQLite because it runs with no server; the same package serves Postgres or MySQL with a
different driver import and a different unique-violation check. Name the package for what it
is (`sqlstore`), not for today's engine, when the engine is genuinely interchangeable.

**When the technology name collides with the library's own package, name the adapter for the
capability instead.** An `internal/sqlite` package would have to import `modernc.org/sqlite`
under an alias, and every reader then has to work out which `sqlite` a given call refers to —
hence `sqlstore`. The same applies to an `internal/argon2` wrapping
`golang.org/x/crypto/argon2` (use `pwhash`, which also leaves room for the second algorithm
you will eventually migrate to) and an `internal/redis` wrapping `github.com/redis/go-redis`
(use `cache`).

Default new services to **argon2id** (`golang.org/x/crypto/argon2`), which is the current
password-hashing recommendation. bcrypt remains perfectly serviceable and is not a bug to be
fixed on sight; the point of `accounts.Hasher` is that the choice stays swappable, and a
migration means adding `NewBcryptHasher` alongside and re-hashing on next successful login.

- **May import:** its driver/SDK and the domain packages whose interfaces it satisfies.
- **May not import:** `internal/httpapi`, or another adapter. Two adapters that need each
  other are being orchestrated in the wrong place — that belongs in a domain service.
- Include `var _ accounts.Store = (*AccountStore)(nil)` so a drifting contract fails the build.
- Confine driver-specific error decoding to one function, so "swap the database" stays true.
- One workflow writing atomically through two stores is the case these rules do not solve on
  their own. See "Transactions across stores" in `interfaces.md` — the domain declares an
  `Atomic` interface, the adapter owns the transaction.

### `internal/httpapi/` (or `grpcapi`, `consumer`)

Everything that knows about the wire: routing, middleware, decoding, encoding, status codes,
auth extraction.

- **May import:** domain packages.
- **May not import:** adapters. A handler reaching for `sqlstore` has skipped the domain, and
  the business rule it skipped will be reimplemented, differently, elsewhere.
- Request/response types live here with `json:` tags. Map to and from domain types explicitly.

### `migrations/`

Plain SQL, applied by tooling (`golang-migrate`, `atlas`, `goose`) as a deploy step or by a
dedicated `cmd/migrate/`. Do not auto-migrate from the API binary at startup: a failed
migration turns every replica into a crash loop, schema changes need different DB privileges
than request serving, and rollback becomes a deploy problem rather than an operator decision.
(Concurrency itself is usually handled — `golang-migrate` takes a Postgres advisory lock, and
goose supports locking — so serialisation is not the argument; deploy control is.)

## Growing through the tiers

**Tier 1 → 2.** Trigger: a second binary, a second author, or the first time a test needs a
real database. Create `cmd/api/`, move `main.go` into it, and split the domain out of the
handler file. Nothing else changes.

**Tier 2 → 3.** Trigger: two clusters of types that barely reference each other, or one
domain package past ~2k lines. Split by bounded context (`accounts`, `billing`, `catalog`),
not by layer. If the split leaves a cycle, the boundary is in the wrong place — see
`interfaces.md`.

**Do not** split into separate Go modules until deploy cadences actually diverge. Multi-module
repos add version-bump friction to every cross-cutting change and are hard to reverse.

## Config

```go
// fragment — struct tags assume caarlos0/env; Load elided
package config

type Config struct {
	HTTP HTTPConfig
	DB   DBConfig
	Auth AuthConfig
}

type HTTPConfig struct {
	Addr            string        `env:"HTTP_ADDR" envDefault:":8080"`
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" envDefault:"15s"`
}

type DBConfig struct {
	DSN          string `env:"DATABASE_URL,required"`
	MaxOpenConns int    `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
}

type AuthConfig struct {
	Argon2Memory      uint32 `env:"ARGON2_MEMORY_KIB" envDefault:"65536"`
	Argon2Iterations  uint32 `env:"ARGON2_ITERATIONS" envDefault:"3"`
	Argon2Parallelism uint8  `env:"ARGON2_PARALLELISM" envDefault:"2"`
}

func Load() (Config, error) { … }
```

Parse and validate everything at startup and fail loudly. A service that boots with a missing
setting and discovers it on the first request has turned a deploy failure into an incident.
Read env vars in exactly one place — `os.Getenv` scattered through packages is untestable and
undocumentable.

## Wiring in main

Construct dependencies in order, innermost last, and hand each one only what it needs.

```go
// fragment — cmd/api/main.go, imports elided except the one that is easy to miss:
//   _ "modernc.org/sqlite"   // registers the "sqlite" driver name
// A driver is registered by importing it for side effects. Without that blank
// import, sql.Open fails at runtime with: unknown driver "sqlite".
// Postgres would be _ "github.com/jackc/pgx/v5/stdlib" and the name "pgx".

func main() {
	if err := run(); err != nil {
		log.Fatalf("startup: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.DB.DSN)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)

	// adapters — each takes its OWN params struct, so no package below cmd/
	// imports internal/config. main does the mapping.
	accountStore := sqlstore.NewAccountStore(db)
	hasher := pwhash.NewArgon2Hasher(pwhash.Argon2Params{
		Memory:      cfg.Auth.Argon2Memory,
		Iterations:  cfg.Auth.Argon2Iterations,
		Parallelism: cfg.Auth.Argon2Parallelism,
	})

	// domain
	accountSvc := accounts.NewService(accountStore, hasher, time.Now, uuid.NewString)

	// transport
	srv := httpapi.NewServer(httpapi.ServerConfig{
		Addr:            cfg.HTTP.Addr,
		ShutdownTimeout: cfg.HTTP.ShutdownTimeout,
	}, accountSvc)
	return srv.Run()
}
```

The mapping is a few lines of boilerplate in exactly one place, and it buys a codebase where
every package below `cmd/` can be constructed in a test without touching environment
variables.

Note `sql.Open` does not connect — it validates arguments and returns lazily. Call
`db.PingContext` during startup if the process should fail fast on an unreachable database.

`main` returning through `run() error` keeps `defer` working — `log.Fatal` skips deferred
calls, so connections never close on the error path.

**On DI containers:** this hand-wiring is the right default and stays readable to roughly 30
dependencies. Past that, reach for `google/wire` (compile-time, generated, no reflection)
before a runtime container. A single `dependencies.go` holding every constructor is the
failure mode to avoid: it becomes a file every feature branch edits, and it hands each
component the whole dependency struct instead of its two real dependencies.

## Graceful shutdown

### `internal/httpapi/server.go` (complete file)

```go
package httpapi

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ServerConfig is declared here rather than imported from internal/config,
// for the same reason interfaces are declared by their consumer: this package
// states what it needs, and main maps the app config onto it. httpapi stays
// independent of how configuration happens to be loaded.
type ServerConfig struct {
	Addr            string
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
}

type Server struct {
	http            *http.Server
	shutdownTimeout time.Duration
}

const defaultRequestTimeout = 30 * time.Second

func NewServer(cfg ServerConfig, svc registrar) *Server {
	// The zero value of a timeout means "unset", but http.TimeoutHandler
	// reads 0 as "time out immediately" -- an unset field would make every
	// request 503 rather than simply not time out. http.Server's own
	// timeouts have the opposite convention, where 0 means no limit. That
	// asymmetry is worth defending against here rather than debugging later.
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = defaultRequestTimeout
	}

	mux := http.NewServeMux()
	mux.Handle("POST /users", handleRegister(svc))

	return &Server{
		http: &http.Server{
			Addr: cfg.Addr,
			// TimeoutHandler is what puts a deadline on every request
			// context, so handlers inherit cancellation without each one
			// remembering to set it.
			Handler: http.TimeoutHandler(mux, cfg.RequestTimeout,
				`{"error":"request timeout"}`),
			// Guards against a client that opens a connection and dribbles
			// headers forever. Not a context — the net/http server enforces
			// these itself, and no deadline reaches the handler from them.
			ReadHeaderTimeout: 5 * time.Second,
			// Must exceed RequestTimeout, or the connection dies before
			// TimeoutHandler can write its response.
			WriteTimeout: cfg.RequestTimeout + 5*time.Second,
			IdleTimeout:  60 * time.Second,
		},
		shutdownTimeout: cfg.ShutdownTimeout,
	}
}

func (s *Server) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	return s.http.Shutdown(shutdownCtx)
}
```

Use a fresh context for `Shutdown` — the signal context is already cancelled, so passing it
would abort in-flight requests immediately, which is the opposite of graceful. Keep the
timeout below the orchestrator's termination grace period.

Propagate `r.Context()` down every call so a disconnected client cancels its DB query.
`context.Context` is always the first parameter and is never stored in a struct.

## Context: deadlines, cancellation, values

Threading `ctx` through every signature is the easy half. These are the parts that are
usually missing, and each one is a real failure mode rather than a style preference.

**Zero means opposite things in the two timeout APIs.** `http.Server`'s `ReadTimeout`,
`WriteTimeout` and `IdleTimeout` treat `0` as *no limit*. `http.TimeoutHandler` treats `0` as
*expire immediately*, so an unset config field yields a server that 503s every request. Default
it explicitly in the constructor rather than trusting the zero value.

**The edge sets the deadline; leaves inherit it.** A handler should not invent its own
timeout, and neither should a store method. `http.TimeoutHandler` above puts one deadline on
the request context, and everything downstream — service, store, driver — is bounded by it
for free. A leaf that calls `context.WithTimeout` on its own is overriding a budget it cannot
see.

**Give outbound calls their own, shorter budget.** The exception to the rule above: when one
request fans out to a dependency that should not be allowed to consume the whole budget, bound
that call specifically. Always `defer cancel()` — skipping it leaks the timer until the parent
is done.

```go
// fragment — inside a service method
ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
defer cancel()

resp, err := s.payments.Charge(ctx, amount)
```

**Cancellation only works if you pass it on.** `QueryRowContext` and `ExecContext` exist so a
disconnected client actually stops the query; `Query`/`Exec` are the same call with the
cancellation silently discarded. Prefer the `…Context` variant of every library call. In a
loop that does not otherwise block, check explicitly:

```go
// fragment — a long-running worker loop
for _, item := range items {
	if err := ctx.Err(); err != nil {
		return err // cancelled or past deadline; stop cleanly
	}
	if err := s.process(ctx, item); err != nil {
		return err
	}
}
```

**Distinguish the two cancellation causes when it matters.** `errors.Is(err, context.Canceled)`
means the caller went away — usually not worth logging as an error. `context.DeadlineExceeded`
means you were too slow, which is. Mapping both to 500 hides a real signal; a cancelled
request is better logged at info and answered with 499/no response at all.

**Do not use the request context for work that outlives the request.** Firing a goroutine with
the request's `ctx` means it is cancelled the moment the response is written. Detach
deliberately:

```go
// fragment — keep the request's values (trace IDs, auth) but drop its deadline
bg := context.WithoutCancel(ctx)
go func() {
	ctx, cancel := context.WithTimeout(bg, 30*time.Second)
	defer cancel()
	if err := s.sendWelcomeEmail(ctx, u); err != nil {
		s.log.Error("welcome email failed", "err", err)
	}
}()
```

`context.WithoutCancel` requires Go 1.21+. Before that the idiom was to construct a fresh
`context.Background()` and copy the values across by hand.

**`context.Value` is for request-scoped data that middleware attaches and handlers read** —
trace ID, authenticated user, the transaction in `sqlstore`. It is not a dependency-injection
mechanism: anything a component needs in order to *function* is a constructor argument, so
that it is visible in the type system rather than discovered at runtime by a failed type
assertion.

When you do use it, the key must be an unexported named type, never a bare string:

```go
// fragment — the only correct shape for a context key
type userIDKey struct{}

func withUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey{}, id)
}

func userIDFrom(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey{}).(string)
	return id, ok
}
```

A `string` key collides silently across packages, because `ctx.Value("user_id")` set by your
middleware and by a library are the same key. An unexported struct type cannot collide,
because no other package can name it. Export the accessor functions, never the key.

**`ctx` is always the first parameter and is never stored in a struct.** A stored context is
frozen at construction time, so it carries the deadline of whatever call happened to build the
object — usually startup, meaning no deadline at all. The one legitimate long-lived context is
the process lifetime context in `main`, and it is passed, not stored.


## Test file placement

Tests live beside the code, in the same directory. There is no `test/` tree.

| File | Package | Use for |
|------|---------|---------|
| `service_test.go` | `accounts` | Unit tests needing unexported access |
| `service_test.go` | `accounts_test` | Tests written against the public API — the better default, since it catches an awkward exported surface |
| `export_test.go` | `accounts` | Deliberately exposing an internal to `accounts_test` |

Both `accounts` and `accounts_test` may coexist in the directory.

- **Domain packages:** table-driven tests with fakes. Fast, no Docker, run on every save.
- **Adapters:** integration tests against a real instance (`testcontainers-go`, or a compose
  service in CI). Testing a SQL adapter against a mock DB verifies the mock. An in-process engine such as
  SQLite makes this cheap enough to run on every save.
- **Transport:** `net/http/httptest` with a fake service — asserts status codes, decoding, and
  error mapping, not business rules.
- Guard slow tests with `testing.Short()` and `if testing.Short() { t.Skip(…) }`, so
  `go test -short ./...` stays a fast inner loop.

Shared fixtures go in `internal/<domain>/testdata/` (ignored by the toolchain) or a
`_test.go` helper — not a `testutil` package imported by production code.
