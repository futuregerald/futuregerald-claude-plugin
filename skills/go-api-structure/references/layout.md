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

### `internal/<adapter>/` — `sqlstore`, `cache`, `eventbus`, `objectstore`, `payments`

One package per external system. Implements interfaces the domain declared.

**The skill is not opinionated about which storage engine you pick.** The worked example uses
SQLite because it runs with no server; the same package serves Postgres or MySQL by changing
the driver import, the unique-violation check, the placeholder syntax and any DSN/scan
options — a bounded set, all inside this package. Name the package for what it is
(`sqlstore`), not for today's engine, when the engine is genuinely interchangeable.

**When the technology name collides with the library's own package, name the adapter for the
capability instead.** An `internal/sqlite` package would have to import `modernc.org/sqlite`
under an alias, and every reader then has to work out which `sqlite` a given call refers to —
hence `sqlstore`. The same applies to an `internal/argon2` wrapping
`golang.org/x/crypto/argon2` — though see below: prefer a vetted wrapper over that package
directly (use `pwhash`, which also leaves room for the second algorithm
you will eventually migrate to) and an `internal/redis` wrapping `github.com/redis/go-redis`
(use `cache`).

Default new services to **argon2id**, the current password-hashing recommendation — but use a
vetted wrapper such as `github.com/alexedwards/argon2id`, **not** `golang.org/x/crypto/argon2`
directly.

That distinction matters more than it looks. `x/crypto/argon2` is a bare key-derivation
function: it exposes `IDKey` (argon2id) and `Key` (argon2i) and nothing more, leaving you to generate the salt, encode the
result, store the parameters, and compare in constant time. Each is silently wrong when done
wrong — a fixed or missing salt, `==` instead of `subtle.ConstantTimeCompare`, or parameters
you never stored and therefore can never tune. bcrypt's `GenerateFromPassword` /
`CompareHashAndPassword` handle all of that for you, which is why bcrypt remains perfectly
serviceable and is not a bug to fix on sight.

The point of `accounts.Hasher` is that the choice stays swappable: migrating means adding a
second implementation and re-hashing on next successful login.

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

What goes inside the package — handlers, middleware, encoding, validation, health probes and
input hardening — is [`transport.md`](transport.md).

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
	// For SQLite the DSN must carry pragmas, e.g.
	//   file:app.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)
	// See "Pool size and DSN are engine-specific" below — without
	// busy_timeout, concurrent writes fail rather than wait.
	DSN string `env:"DATABASE_URL,required"`
	// 25 suits Postgres/MySQL. For SQLite, 25 is fine WITH the busy_timeout
	// pragma above; without it, use 1. Engine-specific, always.
	MaxOpenConns int `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
}

type AuthConfig struct {
	Argon2Memory      uint32 `env:"ARGON2_MEMORY_KIB" envDefault:"65536"`
	Argon2Iterations  uint32 `env:"ARGON2_ITERATIONS" envDefault:"3"`
	Argon2Parallelism uint8  `env:"ARGON2_PARALLELISM" envDefault:"2"`
}

func Load() (Config, error) { … }
```

### Pool size and DSN are engine-specific

Copying a connection-pool setting between engines is a real outage. Measured against the
example's own adapter, 50 concurrent writes through an on-disk SQLite database with
`SetMaxOpenConns(25)` and no pragmas:

```
as documented (pool 25, no pragmas)   FAILED 48/50 — database is locked (SQLITE_BUSY)
                                      FAILED 18/20 concurrent transactions
with _pragma=busy_timeout(5000)       50/50 OK, 20/20 OK
with SetMaxOpenConns(1)               50/50 OK, 20/20 OK
```

SQLite allows exactly one writer. Without `busy_timeout` a blocked writer fails immediately
instead of waiting, and `SQLITE_BUSY` is not a unique-violation, so it surfaces as a 500.
`journal_mode(WAL)` additionally lets readers proceed during a write.

So there are two working SQLite configurations, and the benchmark above shows both: keep the
pool at 25 **and** set `busy_timeout` (preferred — WAL still gives you concurrent readers), or
drop `SetMaxOpenConns` to 1 and serialise everything. What does not work is the default of one
without the other. Postgres and MySQL have none of these constraints and genuinely want ~25.

The rule is that **pool size and DSN options travel with the engine, not with the code** — so treat
them as configuration, not as something to copy from another service.

One more trap, for tests: an in-memory SQLite DSN **without** `cache=shared` gives every
pooled connection its own private database, so a table created on one connection is missing on
the next. It fails with `no such table`, not a lock error, which sends you hunting in the wrong
place entirely.

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
	// Neutral label: run() returns config, database AND server errors, and each
	// is already wrapped with its own context. A prefix here could only mislabel.
	if err := run(); err != nil {
		log.Fatalf("%v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// DSN carries the SQLite pragmas; see "Pool size and DSN are engine-specific".
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

	// transport — main owns the logger for the same reason it owns signal
	// handling below: one process, one place deciding. Named logger rather than
	// log, because this file already imports log for log.Fatalf.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	srv := httpapi.NewServer(httpapi.ServerConfig{
		Addr:            cfg.HTTP.Addr,
		ShutdownTimeout: cfg.HTTP.ShutdownTimeout,
	}, logger, accountSvc, httpapi.ReadinessCheck{
		// main is the only place holding the *sql.DB, so it is the only place
		// that can hand httpapi a check without httpapi importing a driver.
		Name:  "database",
		Check: db.PingContext,
	})

	// main owns signal handling and hands the resulting context down.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return srv.Run(ctx)
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

### `internal/httpapi/server.go`

Fragment, not a complete file. It keeps the shutdown path and elides what this section does
not argue about — readiness validation, the logger default, the middleware bodies. The file as
it stands is [`example/internal/httpapi/server.go`](../example/internal/httpapi/server.go); it
forms `package httpapi` **together with** `accounts.go` from `interfaces.md`, where `registrar`
and `handleRegister` are declared, so neither is standalone.

```go
// fragment — internal/httpapi/server.go, trimmed to the shutdown path.
// Elisions are marked. The file as it actually stands, middleware and all,
// is example/internal/httpapi/server.go.

// ServerConfig is declared here rather than imported from internal/config,
// for the same reason interfaces are declared by their consumer: this package
// states what it needs, and main maps the app config onto it. httpapi stays
// independent of how configuration happens to be loaded.
type ServerConfig struct {
	Addr            string
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
}

// ReadinessCheck reports whether one dependency is usable right now.
// Name appears in the /readyz body so a failing check is identifiable.
type ReadinessCheck struct {
	Name  string
	Check func(context.Context) error
}

type Server struct {
	http            *http.Server
	shutdownTimeout time.Duration
	listen          func(network, addr string) (net.Listener, error)
	log             *slog.Logger
	ready           []ReadinessCheck
}

const (
	defaultRequestTimeout  = 30 * time.Second
	defaultShutdownTimeout = 15 * time.Second
)

func NewServer(cfg ServerConfig, log *slog.Logger, svc registrar, ready ...ReadinessCheck) *Server {
	// Elided: a nil logger falls back to slog.DiscardHandler, while a nil
	// ReadinessCheck.Check panics here rather than at the first probe.

	// The zero value of a timeout means "unset", but http.TimeoutHandler
	// reads 0 as "time out immediately" -- an unset field would make every
	// request 503 rather than simply not time out. http.Server's own
	// timeouts have the opposite convention, where 0 means no limit. That
	// asymmetry is worth defending against here rather than debugging later.
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = defaultRequestTimeout
	}
	// Same defence: an unset ShutdownTimeout would make context.WithTimeout
	// produce an already-expired deadline, so Shutdown would abort instantly
	// and sever in-flight requests -- the opposite of graceful.
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = defaultShutdownTimeout
	}

	// Cloned because a variadic parameter shares the caller's backing array,
	// so keeping it would let the caller mutate what /readyz reports.
	checks := slices.Clone(ready)

	mux := http.NewServeMux()
	addRoutes(mux, log, svc, checks)

	// Outermost first: a request travels recoverPanic -> requestID ->
	// requestLog -> TimeoutHandler -> mux. The order is load-bearing; the
	// reasoning is on chain in middleware.go.
	//
	// TimeoutHandler is what puts a deadline on every request context, so
	// handlers inherit cancellation without each one remembering to set it.
	handler := chain(
		http.TimeoutHandler(mux, cfg.RequestTimeout, `{"error":"request timeout"}`),
		recoverPanic(log),
		requestID(),
		requestLog(log),
	)

	return &Server{
		http: &http.Server{
			Addr:    cfg.Addr,
			Handler: handler,
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
		listen:          net.Listen,
		log:             log,
		ready:           checks,
	}
}

// Handler exposes the routed handler so functional tests can drive the whole
// stack through httptest without binding a port. Cheap, and it is what makes
// end-to-end tests of the real wiring possible.
func (s *Server) Handler() http.Handler { return s.http.Handler }

// Run serves until ctx is cancelled, then shuts down gracefully.
//
// The CALLER owns signal handling. Signals are a process-wide singleton, so a
// binary running an HTTP server plus a worker needs one place deciding the
// shutdown order -- and that place is main, not a transport package.
//
// listen is injectable so tests can bind 127.0.0.1:0 and learn the real port.
// A shutdown path that cannot be tested is exactly the kind that rots.
func (s *Server) Run(ctx context.Context) error {
	ln, err := s.listen("tcp", s.http.Addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.http.Addr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.http.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
`context.Context` is always the first parameter, and a **request** context is never stored in a
struct.

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
means you were too slow, which is. Mapping both to 500 hides a real signal.

For a cancelled request, return without writing: the client has gone, so there is nobody to
read a body. Be aware that Go then sends an implicit **200** — a handler that returns without
calling `WriteHeader` cannot produce "no status at all". If you need cancelled requests
excluded from success metrics, record 499 (nginx's client-closed convention) in your logging
or metrics middleware, not on the wire.

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

For one-off work this is fine. Once it happens on every request, bound it with a worker pool
instead — see `references/concurrency.md`.

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

**`ctx` is always the first parameter, and a request context is never stored in a struct.**
A stored request context is
frozen at construction time, so it carries the deadline of whatever call happened to build the
object — usually startup, meaning no deadline at all. Two long-lived contexts are legitimate: the process
lifetime context in `main` (passed, not stored), and a context a long-lived component creates
for *itself* at construction and cancels in its own `Shutdown` — see the worker pool in
`references/concurrency.md`. The rule bans borrowing a *request's* deadline for work that
outlives the request; it does not ban a component owning its own lifecycle.


## Test file placement

### Start with functional tests, not unit tests

The default instinct is a unit test per function. Resist it. **Ten functions that each pass
their own test still do not tell you the business flow works** — the interesting failures live
in the wiring between them, and that is exactly the ground unit tests do not cover.

Both real defects found while writing this skill make the point. An adapter queried the
connection pool instead of the active transaction, so writes survived a rollback. A worker pool
returned success for jobs it then dropped. Every individual function was correct; every unit
test passed. Only driving the whole stack found them.

So the ordering is:

1. **Functional / integration first.** Wire the real transport, real service, real adapter and
   a real database, and drive it through the front door. In-process SQLite makes this fast
   enough to run on every save — there is no excuse to defer it.
2. **Unit tests for genuinely tricky logic** — a pricing rule, a state machine, a parser.
   Things with many branches and no I/O.
3. **Mocks last, and rarely.** A test that mocks the thing it is testing tests the mock.

**Cover failure paths, not just the happy path.** One test for "it works"; the rest for
duplicates, races, dead dependencies, timeouts, cancellations and partial writes. That is where
production actually breaks, and where a green unit suite gives the most false confidence.

**And confirm each test can fail.** Break the code on purpose and watch it go red. A concurrent
duplicate-registration test written for the example passed against a deliberately broken
adapter — the database serialised the writes, the pre-check absorbed every duplicate, and the
constraint path was never reached. It asserted a real invariant and still proved nothing.
Replacing luck with a deterministic injection point fixed it. A test that has never failed has
not been shown to test anything.

A worked set lives in `example/functional/flow_test.go`, with the reasoning in
`example/README.md`.

### Placement

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
