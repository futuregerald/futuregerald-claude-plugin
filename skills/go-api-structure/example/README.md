# Runnable example

The service this skill describes, as a working Go module you can build and test:

```bash
cd skills/go-api-structure/example
go test ./... -race -shuffle=on
golangci-lint run
```

Both are gates, not one gate and one suggestion. The module ships its own
[`.golangci.yml`](.golangci.yml), and every linter enabled in it maps to a red flag `SKILL.md`
states in prose — `containedctx` for a request context on a struct, `errorlint` for a `==`
comparison that stops matching the moment someone adds `%w`, `noctx` for a call that drops
cancellation. A passing test suite says nothing about any of them.

No server, no Docker, no setup — the database is SQLite via `modernc.org/sqlite` (pure Go).
It is a **nested module**, so it is excluded from the parent repository's `go build ./...`.

## Layout

```
internal/accounts/   domain     — entities, rules, the interfaces it needs
internal/sqlstore/   adapter    — implements accounts.Store against SQL
internal/httpapi/    transport  — route table, handlers, wire types, JSON encoding,
                                  middleware, health probes, timeouts
internal/jobs/       capability — bounded worker pool
functional/          the whole stack, driven through the front door
```

`internal/httpapi/` is six files, and the split is the package's own argument: `routes.go` is the
whole API surface on one screen, `server.go` the composition root and lifecycle, `accounts.go`
the handler and its wire types, `encoding.go` the generic decode/encode/status mapping,
`middleware.go` the chain, and `health.go` the two probes.

These packages are the source of truth, and the documentation quotes them.
`references/interfaces.md` reproduces six of these files in full and `references/concurrency.md`
one; each such block is labelled **complete file** and is byte-identical to its source, so a
diff against the file it names is empty. Both of those references also carry **fragments**, and
`references/layout.md` and `references/transport.md` carry fragments only — imports and
surrounding code elided. If a quote and the code it names disagree, the code is right and the
documentation is wrong.

## The tests are the point

Read `functional/flow_test.go` first. It wires the real transport, real domain service, real
SQL adapter and a real database, then drives them through HTTP — no mocks below the handler.

That is deliberate. Every package here has unit tests and they all pass. The two defects that
actually shipped during this skill's development were:

- an adapter that queried the connection pool instead of the active transaction, so writes
  **survived a rollback**; and
- a worker pool that returned success for jobs it then **silently dropped**.

Both were invisible to unit tests, because every individual function was correct. Only
exercising the flow across components found them.

## Mostly failure paths

`TestRegisterPersistsUserAndReturnsIt` covers the happy path. The other nine cover the ways a
real system fails. That is every test function in `functional/flow_test.go`, in source order:

| Test | The failure it pins down |
|---|---|
| `TestDuplicateEmailIsRejectedAndDoesNotCreateASecondRow` | A second registration must not create a second row |
| `TestConcurrentDuplicateRegistrationsYieldExactlyOneUser` | Twelve simultaneous signups, exactly one user |
| `TestMalformedBodyIsRejectedAndWritesNothing` | A bad request writes no rows |
| `TestDatabaseFailureDoesNotLeakInternals` | A dead database must not spill schema or driver detail |
| `TestReadyzTurnsRedWhenTheDatabaseIsGone` | A dead dependency turns `/readyz` 503 and names it, while `/healthz` stays 200 — failing readiness must not trigger a restart |
| `TestPasswordNeverStoredOrReturnedInPlaintext` | The secret never reaches the response or the table |
| `TestPartialWriteIsRolledBackEntirely` | A failed second step undoes the first |
| `TestCancelledRequestIsNotAServerError` | A client that is already gone gets `TimeoutHandler`'s bare 503 with an empty body, never a 500 — and a control request proves the fixture still reaches the handler |
| `TestDuplicateLosingTheRaceIsA409NotA500` | Losing the insert race must be a conflict, not a server error |

## A caution worth more than the tests

`TestDuplicateLosingTheRaceIsA409NotA500` exists because the *concurrent* version of it —
twelve goroutines registering the same address — **passed against a deliberately broken
adapter**. SQLite serialised the writes enough that the service's pre-check caught every
duplicate, so the constraint path was never reached and the test proved nothing.

The fix was to stop hoping for a race and force one: a `Hasher` that inserts the conflicting
row as a side effect, because it runs at exactly the moment between the lookup and the write.

The lesson generalises. **A test that has never failed has not been shown to test anything.**
Break the code on purpose and confirm the test goes red before trusting it.
