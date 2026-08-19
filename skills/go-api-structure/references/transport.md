# The HTTP Edge: Handlers, Middleware, and Hardening

Everything that knows about the wire lives in one package. This file is what goes in it and
why. The import-direction contract for that package — what it may and may not import — belongs
to [`layout.md`](layout.md); shutdown, timeouts and context propagation are there too.

Every code block below is a **fragment** copied from
[`example/internal/httpapi/`](../example/internal/httpapi/). Those files are the source of
truth. Read them rather than trusting a quote here to have stayed current.

- [The transport composition root](#the-transport-composition-root)
- [Handlers are closures](#handlers-are-closures)
- [decode, encode, and why there is no Validator interface](#decode-encode-and-why-there-is-no-validator-interface)
- [Middleware](#middleware)
- [Logging at the edge](#logging-at-the-edge)
- [Tracing](#tracing)
- [Health versus readiness](#health-versus-readiness)
- [Input hardening](#input-hardening)

## The transport composition root

`NewServer` is to the transport package what `main` is to the binary: the one function that
assembles the parts. It defaults the config, builds the middleware stack, registers the routes
and constructs the `*http.Server` — and it does not listen. Listening happens in `Run`, which
is a separate concern with a separate test.

```go
// fragment — example/internal/httpapi/server.go
func NewServer(cfg ServerConfig, log *slog.Logger, svc registrar, ready ...ReadinessCheck) *Server
```

Four things about that signature are deliberate.

**`ServerConfig` is declared in this package, not imported from `internal/config`.** Same
argument as a consumer-declared interface: the component states what it needs, and `main` maps
the loaded application config onto it. That is what lets `internal/config` stay a leaf that
nothing below `cmd/` imports — see [`layout.md`](layout.md#config).

**A nil logger is tolerated; a nil `ReadinessCheck.Check` is a panic.** The asymmetry is not an
oversight. A missing logger has a safe default — `slog.New(slog.DiscardHandler)` — that costs
observability and nothing else, and the alternative is a nil dereference on the first request
rather than at construction. A missing check func has no safe default: skipping it reports
ready for a dependency nobody verified, and failing it forever makes the instance permanently
unroutable. Both surface as a misleading `/readyz` instead of the wiring bug they are, so the
constructor refuses to build.

**`ready` is cloned.** A variadic parameter is not automatically a private copy —
`NewServer(cfg, log, svc, checks...)` passes the caller's own backing array, so retaining the
slice would let the caller mutate what `/readyz` reports after construction. It is cloned
**once** and shared between the route table and the struct field; two clones would be two
answers to the same question, and only one of them is the one `/readyz` serves.

**`Handler()` exposes the fully wrapped handler.** That is what lets a functional test drive
the real middleware stack, the real routes and the real service through `httptest` without
binding a port. A transport package whose only entry point is `Run` can only be tested over a
socket.

### `routes.go` is the whole API surface, in one file

```go
// fragment — example/internal/httpapi/routes.go
func addRoutes(mux *http.ServeMux, log *slog.Logger, svc registrar, ready []ReadinessCheck) {
	mux.Handle("POST /users", handleRegister(svc))
	mux.Handle("GET /healthz", handleLive())
	mux.Handle("GET /readyz", handleReady(log, ready))
}
```

Registering routes inline in `NewServer` works right up to the third endpoint, at which point
the list is buried in a constructor that also defaults config, builds middleware and assembles
an `*http.Server`. Two things are worth the separate function: a reader asking "what does this
service expose?" gets the answer from one screen, and a reviewer sees a new public endpoint
appear in a diff without reading any plumbing.

`addRoutes` takes the dependencies the handlers need rather than the `*Server`, so the route
table cannot reach into the server's lifecycle, and a test can build a mux without building a
server.

Method patterns (`"POST /users"`) require **Go 1.22+**. On 1.21 and earlier that string parses
as a host pattern and silently never matches — no error, no panic, just 404s.

## Handlers are closures

A handler is a function returning `http.HandlerFunc`, and its dependencies are its parameters:

```go
// fragment — example/internal/httpapi/accounts.go
func handleRegister(svc registrar) http.HandlerFunc {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	…
	return func(w http.ResponseWriter, r *http.Request) { … }
}
```

The alternative — a `Handlers` struct with a method per endpoint — hands every handler every
dependency the service has. The signature stops telling you what an endpoint touches, and a
test for one endpoint has to construct all of them. Whereas `handleRegister(svc registrar)`
declares its entire world in its parameter list. The full handler, with its error mapping, is
in [`interfaces.md`](interfaces.md#internalhttpapiaccountsgo--transport-owns-the-wire-format-complete-file).

**Request and response types are declared inside the handler.** They are the shape one endpoint
has promised to keep, so no other handler should be able to couple to them. A package-scope
`type registerRequest struct` invites the second endpoint to reuse it, and then a field added
for endpoint A silently changes the contract of endpoint B — a wire-format change with no diff
at the site that broke.

The closure also gives you a place for per-handler setup that should happen once rather than
per request: the `valid` closure below, a compiled regexp, a parsed template.

Status codes are decided here and nowhere else. The domain returns errors; transport decides
that `accounts.ErrEmailTaken` means 409.

## decode, encode, and why there is no Validator interface

Three helpers in [`encoding.go`](../example/internal/httpapi/encoding.go) carry every request
and response in the package.

```go
// fragment — example/internal/httpapi/encoding.go
func decode[T any](w http.ResponseWriter, r *http.Request) (T, error)
func encode(w http.ResponseWriter, status int, v any) error
func decodeStatus(err error) (int, string)
```

`decode`'s type parameter is load-bearing: `T` appears in no argument, so it cannot be inferred
and the caller must name it — `decode[request](w, r)`. That is what lets `decode` allocate and
return the value. A plain `any` would push allocation and a type assertion onto every caller.

`encode`'s `v` is deliberately *not* a type parameter. It would be inferred from the argument
and used exactly once, compiling to the same thing while implying a constraint that is not
there.

`encode` writes the status before the body, because `WriteHeader` commits it. A failure partway
through encoding therefore cannot be upgraded to a 500 — the client has already been told 201.
The error is returned rather than swallowed so a caller with a logger in scope can record it.

`decodeStatus` is the single place that decides what a rejected body is told. Keeping the
mapping in one function is what stops the fourth handler from inventing its own status for the
same failure.

### The Validator interface is not available here

The familiar shape is an interface with a `Valid(ctx) map[string]string` method that `decode`
calls before returning. It cannot coexist with handler-local request types, and the constraint
is the language, not taste: **Go forbids declaring methods on function-local types.**
`func (r request) Valid() …` written inside a function is a syntax error. A `Validator`
interface therefore forces `request` back to package scope and gives up exactly the
encapsulation that declaring it in the handler bought.

So the rule is:

- **Default: local type, local closure.** The request type lives in the handler and validation
  is a closure beside it, as `valid` is in `handleRegister`.
- **Package-scope type with a `Valid()` method only when the type is genuinely shared** — the
  same body accepted by several endpoints, or a schema versioned as part of a public API. Then
  the type has a reason to be visible, and a method on it is the natural home for the rule.

What is validated here is **shape and presence only**. A password-strength rule is a business
rule and belongs in the domain; putting it in the handler makes transport own domain policy,
and makes the rule unenforceable from any other entry point — a CLI, a seed script, a second
transport.

Presence checking is also the only thing standing between a body of `null` and a 201. `null`
decoded into a struct is a documented no-op: it succeeds and leaves the zero value, arriving at
the handler indistinguishable from `{}`. Neither is malformed, so no decoder guard can catch
it; the presence check is what turns both into a 422.

## Middleware

Middleware is `func(http.Handler) http.Handler`, and anything with dependencies is written as a
factory returning that: `requestLog(log *slog.Logger) func(http.Handler) http.Handler`. The
dependency is captured once at construction instead of being looked up per request.

```go
// fragment — example/internal/httpapi/middleware.go
func chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
```

Applied back-to-front so that the **first middleware listed is the outermost** — the order
written is the order a request travels. That is the whole middleware framework this package
needs. Named stacks, per-route groups and registration-order resolution buy indirection, not
capability.

The order in `NewServer` is `recoverPanic → requestID → requestLog → TimeoutHandler → mux`, and
each position is load-bearing:

- **`recoverPanic` is outermost** so it also catches a panic raised by the middleware inside it,
  not only by a handler. The cost of that placement is that the request ID does not exist yet,
  so the panic log cannot carry it — an accepted trade, since a middleware panic with no
  recovery kills the process.
- **`requestID` precedes `requestLog`** because the log line carries the ID.
- **`requestLog` wraps `TimeoutHandler` rather than sitting under it.** Outside, it records the
  status the client actually received, including the 503 the timeout writes, and its
  `recordingWriter` is touched by one goroutine only. Inside, it would run on the handler
  goroutine — which may still be alive after the timeout has already responded. That is a data
  race and a wrong status.
- **`TimeoutHandler` is what puts a deadline on every request context**, so handlers inherit
  cancellation without each one remembering to set it. See
  [`layout.md`](layout.md#context-deadlines-cancellation-values) for what downstream code owes
  that deadline.

Two traps live in this file.

**Wrapping a `ResponseWriter` silently drops optional interfaces.** `recordingWriter` exists
because the `ResponseWriter` interface will not tell you afterwards whether anything was
written or with what status. But a wrapper satisfies only the interfaces it declares, so the
original's `http.Flusher`, `http.Hijacker` and `http.Pusher` disappear — and streaming
responses and WebSocket upgrades break. The example has no such handler and says so; production
code needs explicit pass-through or a library such as `felixge/httpsnoop`.

**`http.ErrAbortHandler` must be re-panicked.** It is net/http's documented way for a handler to
abandon a response deliberately; the server expects to catch it and drop the connection
silently. Swallowing it converts an intentional abort into a bogus 500.

```go
// fragment — example/internal/httpapi/middleware.go, inside recoverPanic's deferred func
if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
	panic(rec)
}
```

`recover()` returns `any`, so `rec == http.ErrAbortHandler` compares interface values and misses
a panic value that *wraps* the sentinel. Assert to `error` first, then `errors.Is`. Re-panic
with `rec`, not `err`, so the value the server catches and the stack it unwinds are the
original ones.

Once the status is on the wire it cannot be recalled, so a recovered panic that finds a response
already written logs and returns rather than appending an error object to a body the client is
mid-way through parsing.

## Logging at the edge

**The domain returns errors and logs nothing. The transport logs once, at the boundary, with
the request ID.**

This is the consumer-declared-interface argument applied to a cross-cutting dependency. A
`*slog.Logger` field on a domain struct is a dependency the domain did not need in order to
compute anything, and it makes every constructor and every test carry one. A package-level
logger is worse: it is global mutable state that cannot be substituted per test, so log
assertions become order-dependent and parallel tests interleave.

The mechanics follow from the placement:

- One line per request, written by `requestLog`, where every request necessarily passes.
  Volume is predictable and fields are uniform.
- Handlers below stay silent on the success path. A handler that also logs turns one request
  into three lines that have to be correlated.
- Errors that reach the edge are logged there, once, with the request ID from the context.
  Errors that are *handled* — `accounts.ErrEmailTaken` becoming a 409 — are not errors of the
  server and do not get an error log.
- A cancelled request (`context.Canceled`) means the client hung up. That is not a server fault
  and should not be logged as one.
- Raw dependency errors go to the log, never to the response body. See
  [Health versus readiness](#health-versus-readiness).

`main` owns the logger for the same reason it owns signal handling: one process, one place
deciding. It constructs the `*slog.Logger` and passes it to `NewServer`, which passes it to the
middleware and to `handleReady` — see [`layout.md`](layout.md#wiring-in-main).

## Tracing

The example takes no tracing dependency; this is what it would take, and why the edge is where
it goes.

- **Inbound:** wrap the routed handler in `otelhttp.NewHandler`. It extracts the W3C
  `traceparent` header into the request context and starts a server span — one wrapper, at the
  same place the rest of the stack is composed, rather than a span started in every handler.
- **Outbound:** set `otelhttp.NewTransport` as the `Transport` of every `*http.Client` the
  service owns. It injects `traceparent` into requests it sends, which is what joins your spans
  to the next service's.

Both halves ride `context.Context` and nothing else. **That is the payoff for the ctx-first
rule** the skill already enforces: because every function takes `ctx` as its first parameter
and passes the request's context down, adding tracing is two lines of wiring rather than a
signature change through every layer. A codebase that dropped `ctx` somewhere in the middle
gets a trace that stops there, and no compiler error telling it where.

Same reasoning as the logger: the tracer is configured in `main`, and no domain package imports
OpenTelemetry.

## Health versus readiness

Two endpoints, because they answer different questions with different consequences.
`/healthz` decides whether to **restart the process**. `/readyz` decides whether to **route
traffic** to it.

**`handleLive` consults nothing**, and that is the whole design. An orchestrator kills a process
that fails liveness, and killing a healthy process because its database is unreachable fixes
nothing: it drops the in-flight requests, loses whatever was cached, and adds a reconnect storm
to an outage already in progress. Because the dependency is shared, every replica fails at the
same instant, so the entire deployment enters a restart loop while the processes themselves were
fine. This is the single most common way a health endpoint makes an incident worse. The only
thing the endpoint can honestly report is that the process is running and still scheduling
handlers — which answering at all proves.

**`handleReady` runs the registered checks**, which is exactly why it is a second endpoint
rather than a second opinion from the first. Failing readiness pulls the instance out of the
load-balancer pool and leaves it running, so it rejoins the moment its dependency returns — no
restart, no lost warm state, no thundering herd.

Four details in [`health.go`](../example/internal/httpapi/health.go) are each a real failure:

- **Every check runs; the loop does not stop at the first failure.** A body naming one broken
  dependency while staying silent about the other two reads as "only this one is broken" and
  sends the responder down a single trail.
- **`ctx.Err()` is re-tested before each check.** If the client hung up while an earlier check
  was dialling, probing the rest only piles load onto dependencies that are already struggling,
  to produce a body nobody is left to read.
- **The results slice is allocated non-nil.** A nil slice marshals to `null`, and every
  dashboard parsing the response then has to special-case it — or, more often, does not, and
  breaks on the one instance that has no dependencies.
- **The raw error goes to the log, never to the body.** `/readyz` is almost always
  unauthenticated, and a dependency error is a free description of the internal topology:
  hostnames, ports, driver names, schema names. The check's `Name` is what a responder needs
  from the body; the log line carries the rest.

Checks run **sequentially**. Fanning out would save a few milliseconds on a probe nobody is
waiting on, in exchange for several goroutines writing one result set and a synchronisation bug
that only appears when a dependency is already failing.

`main` supplies the checks, because `main` is the only place holding the `*sql.DB` — which is
how `httpapi` gets a database check without importing a driver.

## Input hardening

`decode` applies four guards before a handler sees anything. Each is a real production failure,
and each maps to a status that is true.

| Guard | Failure it prevents | Status |
|---|---|---|
| Media type | A form post, or a client that forgot the header, decoded as if it were JSON | **415** |
| Body cap (1 MiB) | One client making the server allocate until it dies | **413** |
| Unknown fields | A client sending `admin:true` and believing it worked | **400** |
| Trailing data | Two concatenated objects, of which only the first is honoured | **400** |

**Media type — checked first, before the body is so much as wrapped.** Refusing a format the
server does not speak should not require reading megabytes to find out; the cap below bounds
that cost but does not avoid it.

```go
// fragment — example/internal/httpapi/encoding.go
mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
if err != nil || !strings.EqualFold(mediaType, "application/json") {
	return v, errUnsupportedMediaType
}
```

`mime.ParseMediaType`, not a string compare: `application/json; charset=utf-8` is legal and
extremely common, and comparing the raw header would 415 every client that sends a charset.
Parsing also rejects an empty header for free — there is no media type to parse — which is the
missing-header case. `strings.EqualFold` keeps the RFC 9110 rule that media types are
case-insensitive visible at the comparison rather than resting on a normalisation happening out
of sight.

The header is **required, not merely checked when present**. A guard that rejects `text/plain`
while accepting a request with no `Content-Type` at all is bypassed by dropping the header,
which makes it decorative.

**415, not 400.** The request is syntactically fine; the server simply does not speak the format
it declares. Telling the client the body is malformed sends them hunting for a bug in a body
that is correct. Same reasoning family as 413 rather than 400 for an oversized body: it was
well-formed, just too big.

**The cap and the unknown-field guard are two lines**, and both must come before the decode:

```go
// fragment — example/internal/httpapi/encoding.go
r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

dec := json.NewDecoder(r.Body)
dec.DisallowUnknownFields()
```

`http.MaxBytesReader` takes `w` as well as the body so it can also stop the server reading the
rest of the connection. `DisallowUnknownFields` turns a typo'd or malicious field into a 400
rather than a silently ignored one.

### The trailing-value guard has three outcomes, not two

Decoding again into a throwaway value asks "was there anything after the first one?". Its three
answers are genuinely different and must stay apart.

```go
// fragment — example/internal/httpapi/encoding.go
switch err := dec.Decode(&struct{}{}); {
case err == nil:
	return v, errors.New("body must contain a single JSON object")
case errors.Is(err, io.EOF):
	return v, nil
default:
	return v, fmt.Errorf("read body after first JSON value: %w", err)
}
```

`nil` means a second value decoded cleanly, so the client sent extras. `io.EOF` means nothing
followed — the success case, not an error. Anything else means the *read* failed: the body
overran the cap only after the first value, or the connection broke.

Collapsing this to `if err != nil` drops the first case entirely. Wrapping unconditionally with
`fmt.Errorf("…: %w", err)` is worse, because `%w` with a nil `err` does not wrap anything — it
produces an error whose text is the literal `%!w(<nil>)`. Either way the `*http.MaxBytesError`
raised by the second read is destroyed, `decodeStatus` can no longer see it, and a 413 silently
becomes a 400. That was a real defect caught in review of this example.

### A new rejection rule's blast radius is the tests that send a body

Adding the media-type guard turned all 17 existing fixtures into 415s, because
`httptest.NewRequest` sets no `Content-Type` — so none of them reached the code they were
written to exercise. Most failed loudly, which is the good case. One asserted only "not a 500",
so it **kept passing while covering nothing**: the cancelled-request path it existed for was
never entered again.

The general lesson: when a change adds a way to reject input, the tests at risk are the ones
that **send a body**, not the ones that name the changed function. Grep for the payload, not the
symbol. And an assertion loose enough to pass on an unrelated status is an assertion that cannot
report its own irrelevance — see
[`layout.md`](layout.md#start-with-functional-tests-not-unit-tests) on confirming a test can
fail.

The example's fixtures now default to the honest thing — `post` sends the header, `postAs`
exists for the tests that deliberately do not — in
[`accounts_test.go`](../example/internal/httpapi/accounts_test.go).

### A fifth guard arrives with `encoding/json` v2

**As of Go 1.26 the v2 implementation is opt-in**, behind `GOEXPERIMENT=jsonv2`; the announced
direction is that it becomes what `encoding/json` is built on, with `GOEXPERIMENT=nojsonv2` as
the opt-out. Check the release notes for the toolchain you are on rather than a version number
written here. v1 semantics are intended to be preserved, so this is not a rewrite you have to
plan for — but **error text differs**, which is what breaks code that asserts on a decode
error's message rather than on its type.

Importing `encoding/json/v2` explicitly buys two rejections v1 does not make: **duplicate object
names and invalid UTF-8**. Both are worth opting into on a public API. Duplicate keys are a
classic parser differential — a gateway that honours the first `"role"` in
`{"role":"user","role":"admin"}` and a backend that honours the last disagree about who is an
admin, and neither is malformed under v1's rules, so neither rejects it. `DisallowUnknownFields`
does not help: both keys are known.

This module targets `go 1.25.0` and stays on v1.

### What this does not cover

Authentication and authorization, rate limiting, CORS, and request-size limits at the proxy are
all edge concerns too, and all out of scope here. The placement rule is the same as everything
else in this file: they are middleware or handler wrappers in `internal/httpapi`, configured in
`main`, and no domain package learns they exist.
