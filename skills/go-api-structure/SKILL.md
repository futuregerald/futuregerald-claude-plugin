---
name: go-api-structure
description: Structure Go API and service codebases — package layout, module boundaries, and interface-driven dependency direction. Use when starting a new Go service, adding a feature or endpoint to an existing Go service, deciding which package a file or type belongs in, resolving an import cycle, reviewing Go project layout in a PR, or answering "how should I structure my Go project". Also use before writing Go code that touches a database, HTTP client, cache, queue, clock, or any other external system, so it lands behind a consumer-defined interface; and for Go concurrency work — goroutines, channels, worker pools, job queues, bounding how many things run at once, backpressure, graceful shutdown, or hunting a goroutine leak.
tags: [go, architecture]
---

# Go API Structure

## The rule everything else follows

Directories are navigation. **The import graph is the architecture.**

Moving a file into `internal/core/` does not decouple it from Postgres; deleting its
`import "database/sql"` does. Decide which way imports point first, then decide where files
go. Get the first wrong and no layout rescues it.

## The words this skill uses

Defined here because the rest of the page leans on them. Each is described in terms of the
running example rather than in the abstract.

| Term | What it means here |
|---|---|
| **Import graph** | A picture of which package imports which: every package is a box, every `import` line an arrow. It is the only structure the Go compiler enforces — and a cycle in it is a hard build error, not a style complaint. |
| **Domain** | Code holding business rules — "an email must be unique", "a password is stored hashed". Knows nothing about HTTP or databases. Here: `internal/accounts`. |
| **Adapter** | Code that talks to something outside your process — a database, Stripe, Redis, S3. Here: `internal/sqlstore`. |
| **Transport** | Code that speaks the wire protocol — HTTP handlers, JSON, status codes. Here: `internal/httpapi`. |
| **Bounded context** | One coherent slice of the business (`accounts`, `billing`, `catalog`). Used to decide where one package should end and the next begin. |
| **Consumer-declared interface** | An interface written in the package that *calls* it, rather than the package that implements it. This is what lets you point an arrow the other way. |
| **DTO** | Data Transfer Object — a struct that exists only to shape data on the wire, with `json:` tags. Kept separate from domain types so the API's shape and the business model can change independently. |
| **Capability package** | In-process machinery that is neither business rules nor an external system — a worker pool, a password hasher. Named for what it does. |
| **Backpressure** | What happens when work arrives faster than you can process it: either the producer waits, or the work is refused. An unbounded queue is the one wrong answer. |

### Why "the import graph is the architecture" is a literal claim

Take a working service and rename the directory holding its business logic:

```
before:  httpapi ──▶ accounts              ◀── sqlstore
after:   httpapi ──▶ core/domain/entities  ◀── sqlstore
```

Every importing file had to be edited and the build broke until they were. Yet the dependencies
are identical — the same two arrows, pointing the same way. If the business logic imported a
database driver before the rename, it still does.

That is the whole point. **Coupling is an arrow existing**, and the only way to remove an arrow
is to delete the `import` line. Renaming relabels a box. A directory layout, on its own, cannot
decouple anything — which is why this skill decides arrow direction first and folder names
second.

(A related consequence: a directory's name and its package name are independent in Go. They
match by convention, not by rule.)

## Step 0 — pick the tier before drawing folders

Over-structuring a small service costs more than under-structuring, because every later
change pays the ceremony tax.

| Tier | When | Layout |
|------|------|--------|
| **1** | One binary, <10 endpoints, no swappable dependencies | `main.go` plus 2–4 packages under `internal/`. No `cmd/`. |
| **2** (default) | A real service: a DB, some clients, one or two binaries | The canonical layout below. |
| **3** | Several bounded contexts or many binaries | Tier 2 with one domain package per context, one `cmd/` per binary. Separate modules only when release cadences diverge. |

State the tier before proposing a layout. Unsure between 1 and 2? Pick 1 — promoting is a
`git mv`, demoting never happens.

## Dependency direction (non-negotiable)

```
   cmd/api ────────────────────────────────────┐  wires everything, imports everyone
      │                                        │
      ▼                                        ▼
internal/httpapi ────▶ internal/accounts ◀──── internal/sqlstore
   (transport)          (domain: types, rules,   (adapter: satisfies the
                         and the interfaces       interfaces the domain
                         it needs)                declared)
```

Arrows point inward. The domain imports neither transport nor adapters — it declares the
interfaces both are shaped around. So `internal/accounts` must not import `net/http`,
`database/sql`, a driver, or a vendor SDK. If it does, the boundary has already leaked.

## The default layout (tier 2)

```
user-service/
├── cmd/
│   ├── api/main.go            # load config, wire, serve, shut down
│   └── worker/main.go         # shares internal/, different entry point
├── internal/
│   ├── config/                # env/flags → a typed Config
│   ├── accounts/              # DOMAIN: entities, rules, the interfaces it needs
│   │   ├── accounts.go        #   User, errors, Store + Hasher interfaces
│   │   ├── service.go         #   Register, Authenticate, ChangeEmail …
│   │   └── service_test.go    #   in-memory fakes, no DB
│   ├── billing/               # second bounded context, same shape
│   ├── sqlstore/              # ADAPTER: implements accounts.Store, billing.Store
│   ├── payments/              # ADAPTER: implements billing.PaymentGateway (wraps stripe-go)
│   ├── jobs/                  # CAPABILITY: bounded worker pool (see concurrency.md)
│   └── httpapi/               # TRANSPORT: router, middleware, handlers, wire DTOs
│       ├── server.go
│       ├── accounts.go        #   handlers + their request/response types
│       └── middleware.go
├── migrations/
├── openapi.yaml
├── Makefile
└── go.mod
```

Two things commonly copied layouts get wrong:

- **Infrastructure goes inside `internal/`.** An `infrastructure/` directory at the repo
  root makes DB repos and clients importable by anything depending on your module, which
  defeats the reason `internal/` was chosen.
- **Adapters group by technology, not feature.** You swap one database for another, not "the
  user half of the database." Domain code groups by feature; adapters group by external
  system. Confine the driver-specific parts (error codes, placeholder syntax) to one place so
  the swap stays cheap.

## Interfaces: the default posture

**Every boundary that leaves the process gets an interface, declared by the caller** — DB,
HTTP client, queue, cache, object store, mailer, shell.

**Nothing else does.** In-process logic is tested by calling it. An interface with one
implementation and no test fake is indirection charging rent.

**Non-determinism is the exception that is not an interface.** Clock, randomness and ID
generation never leave the process, so inject them as plain function values —
`now func() time.Time`, `newID func() string` — not a `Clock` interface, which would be a
one-method interface with exactly one production implementation.

1. **Declare the interface in the package that consumes it.** `accounts` declares `Store`;
   `sqlstore` imports `accounts` and satisfies it. This is what inverts the dependency — the
   folder move does not.
2. **Only the methods that caller calls.** A 3-method interface is an interface; a 20-method
   `Repository` is the concrete type with extra steps.
3. **Assert satisfaction at compile time** where it is implemented:
   `var _ accounts.Store = (*AccountStore)(nil)`.

Accept interfaces, return structs — constructors return `*accounts.Service`.

Worked examples, fakes, and cycle-breaking: `references/interfaces.md`.

## Where does this file go?

| What is being written | Directory | Package |
|---|---|---|
| Entity or value type with business rules | `internal/<domain>/` | `accounts` |
| Interface for something the domain needs | same package as its **consumer** | `accounts` |
| A business workflow / use case | `internal/<domain>/service.go`, as a method | `accounts` |
| SQL, queries, row scanning, DB structs | `internal/sqlstore/` | `sqlstore` |
| HTTP handler, decode, encode, status mapping | `internal/httpapi/` | `httpapi` |
| Request/response DTO | `internal/httpapi/`, beside its handler | `httpapi` |
| Third-party API client (an adapter — it leaves the process) | `internal/<capability>/` | `payments` |
| Queue producer/consumer | `internal/eventbus/` | `eventbus` |
| Background work, bounded concurrency, a job queue | `internal/jobs/` | `jobs` |
| Password hashing, crypto, other in-process capabilities | `internal/<capability>/` | `pwhash` |
| Env parsing, flags, defaults | `internal/config/` | `config` |
| Wiring, lifecycle, graceful shutdown | `cmd/<binary>/main.go` | `main` |
| An atomic write across two stores | domain declares `Atomic`, adapter owns the tx | `accounts` + `sqlstore` |
| Something two domains both need | first try moving the interface to the consumer; a shared domain package is the last resort, and never `shared/` | — |
| Genuinely reusable outside this repo | a separate published module | — |

A use case is a **method on a service, not a package**. One package per endpoint turns a
20-endpoint service into 20+ packages, each exporting a DTO the next one imports.

## Package naming

- Lowercase, one word, no underscores, **no camelCase**. `useCases/`, `registerUser/`,
  `getProfile/` are legal identifiers but violate the naming convention every Go reader and
  linter expects. The package name comes from the `package` clause, not the directory — but
  they are conventionally identical, so a `useCases/` directory produces `package useCases`
  in practice. State this as convention, not as a language rule.
- **Name the package for the bounded context, not the entity**, so call sites don't stutter:
  `accounts.User`, not `user.User`.
- Never `util`, `common`, `helpers`, `shared`, `base`, `misc`, or an app-wide `models`. They
  have no boundary, so everything drifts into them.
- Name an adapter for **the external system it wraps** — that is the thing you swap. Use the
  **capability** instead whenever the system's name would collide with its own client library's
  package name, which in practice is most of them: `cache` not `redis`, `objectstore` not `s3`,
  `payments` not `stripe`, `eventbus` not `kafka`, `pwhash` not `bcrypt`, `sqlstore` not
  `sqlite`. Collision is the common case, so capability names are the common answer. See
  `references/layout.md`.
- Avoid `cmd/http/`; name binaries for what they are (`cmd/api/`), not their transport.

## Red flags

- A domain package importing `database/sql`, `net/http`, a driver, or a vendor SDK
- An interface declared in the same package as its only implementation
- An interface with one implementation and no test fake
- `pkg/` at the repo root, or a package named `utils`/`common`/`shared`
- An app-wide `models/` every other package imports
- A `dependencies.go` past ~150 lines, or returning a struct of 30 fields
- A **request** `context.Context` stored in a struct, or a ctx that is not the first parameter
  (a long-lived component holding its own lifecycle context, cancelled by its `Shutdown`, is the
  documented exception — see `references/concurrency.md`)
- A `Query`/`Exec` where a `QueryContext`/`ExecContext` exists — cancellation silently dropped
- `context.WithTimeout` in a leaf function, overriding a budget the edge already set — the
  exception is deliberately bounding one outbound call so it cannot eat the whole budget
  (see `references/layout.md`)
- `context.WithValue` with a bare `string` key, or used to pass a dependency
- A goroutine outliving its request while still holding the request's `ctx` — see
  `references/concurrency.md` for the bounded alternative
- An import cycle "fixed" by inventing a third package for the shared types — the cycle means
  the interface is declared on the wrong side
- Domain types carrying `json:` or `db:` tags — wire and table leaking inward

## References

| File | When to read |
|------|-------------|
| `references/interfaces.md` | Anything crossing a process boundary; deciding whether something deserves an interface; writing fakes; breaking an import cycle |
| `references/concurrency.md` | Running work in the background or in parallel: worker pools, job queues, capping how many run at once, backpressure, draining on shutdown, goroutine leaks, `errgroup` |
| `references/layout.md` | Standing up or restructuring a service: per-directory contracts, tier growth, `main.go` wiring, config, graceful shutdown, context deadlines/cancellation/values, test placement |
