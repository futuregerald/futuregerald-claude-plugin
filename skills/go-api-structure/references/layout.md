# Layout, Wiring, and Growth

- [Per-directory contracts](#per-directory-contracts)
- [Growing through the tiers](#growing-through-the-tiers)
- [Config](#config)
- [Wiring in main](#wiring-in-main)
- [Graceful shutdown](#graceful-shutdown)
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
  `internal/postgres`, or anything else in the outer ring.
- Files: `<domain>.go` for types, errors, and interfaces; `service.go` for behaviour. Split
  `service.go` by cohesive group (`registration.go`, `authentication.go`) when it gets long —
  splitting into *packages* is what to avoid, not splitting into files.

### `internal/<adapter>/` — `postgres`, `redis`, `kafka`, `s3`, `stripe`

One package per external system. Implements interfaces the domain declared.

**When the technology name collides with the library's own package, name the adapter for the
capability instead.** An `internal/argon2` implementing `accounts.Hasher` would have to import
`golang.org/x/crypto/argon2` under an alias, and every reader then has to work out which
`argon2` a call refers to. `internal/pwhash`, exposing `NewArgon2Hasher`, avoids the collision
and leaves room for the second algorithm — which is the realistic case, since password hashing
gets migrated. Same for `internal/redis` wrapping `github.com/redis/go-redis` — prefer
`internal/cache` if it bites.

Default new services to **argon2id** (`golang.org/x/crypto/argon2`), which is the current
password-hashing recommendation. bcrypt remains perfectly serviceable and is not a bug to be
fixed on sight; the point of `accounts.Hasher` is that the choice stays swappable, and a
migration means adding `NewBcryptHasher` alongside and re-hashing on next successful login.

- **May import:** its driver/SDK and the domain packages whose interfaces it satisfies.
- **May not import:** `internal/httpapi`, or another adapter. Two adapters that need each
  other are being orchestrated in the wrong place — that belongs in a domain service.
- Include `var _ accounts.Store = (*AccountStore)(nil)` so a drifting contract fails the build.
- One workflow writing atomically through two stores is the case these rules do not solve on
  their own. See "Transactions across stores" in `interfaces.md` — the domain declares an
  `Atomic` interface, the adapter owns the transaction.

### `internal/httpapi/` (or `grpcapi`, `consumer`)

Everything that knows about the wire: routing, middleware, decoding, encoding, status codes,
auth extraction.

- **May import:** domain packages.
- **May not import:** adapters. A handler reaching for `postgres` has skipped the domain, and
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
//   _ "github.com/jackc/pgx/v5/stdlib"   // registers the "pgx" driver name
// Without that blank import, sql.Open("pgx", …) fails with: unknown driver "pgx".

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

	db, err := sql.Open("pgx", cfg.DB.DSN)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)

	// adapters — each takes its OWN params struct, so no package below cmd/
	// imports internal/config. main does the mapping.
	accountStore := postgres.NewAccountStore(db)
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
	ShutdownTimeout time.Duration
}

type Server struct {
	http            *http.Server
	shutdownTimeout time.Duration
}

func NewServer(cfg ServerConfig, svc registrar) *Server {
	mux := http.NewServeMux()
	mux.Handle("POST /users", handleRegister(svc))

	return &Server{
		http:            &http.Server{Addr: cfg.Addr, Handler: mux},
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
  service in CI). Testing a SQL adapter against a mock DB verifies the mock.
- **Transport:** `net/http/httptest` with a fake service — asserts status codes, decoding, and
  error mapping, not business rules.
- Guard slow tests with `testing.Short()` and `if testing.Short() { t.Skip(…) }`, so
  `go test -short ./...` stays a fast inner loop.

Shared fixtures go in `internal/<domain>/testdata/` (ignored by the toolchain) or a
`_test.go` helper — not a `testutil` package imported by production code.
