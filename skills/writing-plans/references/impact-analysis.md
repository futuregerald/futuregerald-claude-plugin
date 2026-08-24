# Impact Analysis

The dominant failure mode is a locally-correct change with unconsidered downstream effects.
The code compiles, the new test passes, and something three call sites away breaks.

Before modifying **any** function, method, class, endpoint or schema, reconstruct its call
chain in both directions. This section is mandatory; a plan without one is auto-rejected.

The reviewer will build its own version and diff it against yours. **Any caller it found that
you did not name is a finding, and its severity is at least IMPORTANT.** So the bar is not
"I read the function and it looks fine" — it is "I can name every caller and say what each
one expects".

## Upward

Who depends on this? Every direct caller, then transitively out to real entry points:
controller action, job, rake task, CLI command, webhook, scheduler, public API, `main`,
CI workflow, release config.

**Test files are callers.** They are the most commonly missed category in the measured
corpus, and in a compiled language they are the ones that break the build. A helper used by
six test files is a six-way dependency.

For each caller, record what it does with the return value and which part of the contract it
relies on — shape, nil case, zero value, error identity, ordering, exception, side effect.

Include callers outside this repo: other services, front ends, API consumers, anything
reading the same table, and any file that is shipped as an artifact.

## Downward

What does this depend on? Every method called and its side effects: DB writes, external HTTP,
queue enqueues, cache mutations, emails, file IO, goroutines, locks, channels, context
propagation, logging.

Which are inside a transaction and which are not. What raises, and who is expected to catch
it. What happens on partial failure — if step 4 of 7 throws, what is left behind?

## Invisible edges

A call graph cannot see these. Grep for them **by name, deliberately, every time** — this is
where the breakage graphs miss actually lives.

- `send`, `public_send`, `constantize`, `method_missing`, delegation
- interface satisfaction — a type silently stops satisfying an interface
- serializers, callbacks, observers
- struct tags, including `json:"-"`; embedded types
- job class names stored as strings, config-driven dispatch, handler registries
- reflection, generated code, `//go:embed`, build tags, ldflags `-X`
- shell scripts and CI YAML that grep source
- docs shipped inside a release archive
- test helpers shared across files

Record what you checked and what each turned up, including the empty ones. "Checked
reflection: none in this repo" is a useful line; silence is not.

## Contract

State current behaviour precisely — return shapes, nil and empty cases, exceptions raised,
default values, ordering, side effects. Then which of those the change alters, and for each,
the specific callers affected.

Vague contracts are how a plan passes review and still breaks production: "returns the user"
hides whether it returns `nil`, raises, or returns an empty relation.

## Coverage

Which callers have tests and which do not. What test would fail if the change were wrong.

**A caller with no test is a risk to record, not to ignore.** Say so explicitly — the honest
line is "`OrganizationsController#settings` calls this and has no spec", not silence.

Also state what your test suite does *not* prove. If the suite builds a synthetic fixture and
never exercises the real tree, then a green run is not evidence for this change, and the plan
must say so — otherwise someone will cite it as if it were.

## How

Graph tools first — `graphify path`, `trace_call_path`, `search_graph`, `query_graph` — then
grep for what graphs cannot see. See [[indexing]] for using an index safely, including when
not to trust one.

Conclusions rest on files you actually read. Record claims in the [[claim-ledger]]; when a
plan has several tasks that each change the tree, carry the analysis forward with
[[multiphase]].
