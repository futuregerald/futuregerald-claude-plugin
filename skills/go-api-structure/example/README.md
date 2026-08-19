# Runnable example

The service this skill describes, as a working Go module you can build and test:

```bash
cd skills/go-api-structure/example
go test ./... -race
```

No server, no Docker, no setup — the database is SQLite via `modernc.org/sqlite` (pure Go).
It is a **nested module**, so it is excluded from the parent repository's `go build ./...`.

## Layout

```
internal/accounts/   domain     — entities, rules, the interfaces it needs
internal/sqlstore/   adapter    — implements accounts.Store against SQL
internal/httpapi/    transport  — routing, handlers, wire types, timeouts
internal/jobs/       capability — bounded worker pool
functional/          the whole stack, driven through the front door
```

These packages are the source of truth, and the documentation quotes them.
`references/interfaces.md` and `references/concurrency.md` reproduce several of these files in
full — each such block is labelled **complete file**. `references/layout.md` and
`references/transport.md` quote **fragments** only, with imports and surrounding code elided. If
a quote and the code it names disagree, the code is right and the documentation is wrong.

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

One test covers the happy path. The rest cover the ways a real system fails:

| Test | The failure it pins down |
|---|---|
| `TestDuplicateEmailIsRejected…` | A second registration must not create a second row |
| `TestDuplicateLosingTheRaceIsA409NotA500` | Losing the insert race must be a conflict, not a server error |
| `TestConcurrentDuplicateRegistrations…` | Twelve simultaneous signups, exactly one user |
| `TestMalformedBodyIsRejectedAndWritesNothing` | A bad request writes no rows |
| `TestDatabaseFailureDoesNotLeakInternals` | A dead database must not spill schema or driver detail |
| `TestPasswordNeverStoredOrReturnedInPlaintext` | The secret never reaches the response or the table |
| `TestPartialWriteIsRolledBackEntirely` | A failed second step undoes the first |
| `TestCancelledRequestIsNotAServerError` | A client hanging up is not a 500 |

## A caution worth more than the tests

`TestDuplicateLosingTheRaceIsA409NotA500` exists because the *concurrent* version of it —
twelve goroutines registering the same address — **passed against a deliberately broken
adapter**. SQLite serialised the writes enough that the service's pre-check caught every
duplicate, so the constraint path was never reached and the test proved nothing.

The fix was to stop hoping for a race and force one: a `Hasher` that inserts the conflicting
row as a side effect, because it runs at exactly the moment between the lookup and the write.

The lesson generalises. **A test that has never failed has not been shown to test anything.**
Break the code on purpose and confirm the test goes red before trusting it.
