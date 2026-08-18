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
- Pass the sub-struct a component needs (`cfg.DB`), not the whole `Config`, so packages don't
  gain access to unrelated settings.

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

- **May import:** its driver/SDK, and the domain packages whose interfaces it satisfies.
- **May not import:** `internal/httpapi`, or another adapter. Two adapters that need each
  other are being orchestrated in the wrong place — that belongs in a domain service.
- Include `var _ accounts.Store = (*AccountStore)(nil)` so a drifting contract fails the build.

### `internal/httpapi/` (or `grpcapi`, `consumer`)

Everything that knows about the wire: routing, middleware, decoding, encoding, status codes,
auth extraction.

- **May import:** domain packages.
- **May not import:** adapters. A handler reaching for `postgres` has skipped the domain, and
  the business rule it skipped will be reimplemented, differently, elsewhere.
- Request/response types live here with `json:` tags. Map to and from domain types explicitly.

### `migrations/`

Plain SQL, applied by tooling (`golang-migrate`, `atlas`, `goose`) as a deploy step or by
`cmd/migrate/`. Never auto-migrate from the API binary at startup — concurrent replicas will
race, and a failed migration takes the service down with it.

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
package config

type Config struct {
	HTTP HTTPConfig
	DB   DBConfig
}

type HTTPConfig struct {
	Addr            string        `env:"HTTP_ADDR" envDefault:":8080"`
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" envDefault:"15s"`
}

type DBConfig struct {
	DSN         string `env:"DATABASE_URL,required"`
	MaxOpenConns int   `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
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

	// adapters
	accountStore := postgres.NewAccountStore(db)
	hasher := bcrypt.NewHasher(cfg.Auth.Cost)

	// domain
	accountSvc := accounts.NewService(accountStore, hasher, time.Now)

	// transport
	srv := httpapi.NewServer(cfg.HTTP, accountSvc)
	return srv.Run()
}
```

`main` returning through `run() error` keeps `defer` working — `log.Fatal` skips deferred
calls, so connections never close on the error path.

**On DI containers:** this hand-wiring is the right default and stays readable to roughly 30
dependencies. Past that, reach for `google/wire` (compile-time, generated, no reflection)
before a runtime container. A single `dependencies.go` holding every constructor is the
failure mode to avoid: it becomes a file every feature branch edits, and it hands each
component the whole dependency struct instead of its two real dependencies.

## Graceful shutdown

```go
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
