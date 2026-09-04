# Query Pitfalls — why a widget returns no / wrong data

Read this before writing or reviewing any Datadog dashboard query. These are the
data-dependent failure modes that pass JSON validation and static review but
render empty tiles, blank cells, or numbers that mean something other than the
title claims. Each is caught only by running the query against live data.

Table of contents:
1. Field-name reality gap
2. No-data vs healthy-zero ambiguity
3. Sparse-join null propagation (blank cells in tables/formulas)
4. Count vs cardinality (over-counting lifecycle events)
5. Denominator semantics (rate means what?)
6. Facet / measure does not exist yet
7. Division-by-zero rendering
8. conditional_formats ordering and the cell_display_mode:"bar" conflict
9. Env / template-variable dimension mismatch
10. Retention wall and metric tag cardinality
11. Phrase-search tokenization

---

## 1. Field-name reality gap
**Symptom:** tile empty or group-by shows nothing, even though "the field is right there."
**Cause:** the query uses a field name from a PR description, doc, ticket, or memory — not the name the code actually emits. Names drift (a PR titled "rename to `scan_state`" merged code that emits `scan_status`).
**Check:** pull a real event — `search_datadog_logs` with `extra_fields: ["*"]` — and read the actual `@metadata.*` (or attribute) paths. The payload key you see is the query path (strip the `custom.`/`attributes.` display prefix → `@key`).
**Fix:** query the verified name. Never trust a name you have not seen in a live event. This is the single most common cause of an empty widget.

## 2. No-data vs healthy-zero ambiguity
**Symptom:** an empty failure tile — is the system healthy, or is the sensor dark?
**Cause:** a raw failure count renders identically for "zero failures," "telemetry broken," and "nothing ran." A viewer cannot tell which.
**Check:** for every failure/error count, ask "what does 0 mean here?"
**Fix:** put a **denominator/total next to every failure count** (executions, scans, attempts). If the total is > 0 the sensor is alive; if the total is 0 the sensor is dark — ignore the rate. Consider a telemetry-liveness monitor (absence of the event) as the real backstop.

## 3. Null propagation in multi-query formulas (blank cells / blank tiles)
**Symptom:** a derived value is blank while its inputs are populated. In a table:
a row shows `succeeded: 13` but `runs`/`success rate` blank. In a scalar tile: a
failure-rate tile shows "No data" in a window that had zero failures but real
successes (honest answer is 0%).
**Cause:** a query that matches **zero events returns null**, and `null` propagates
through arithmetic (`null + x = null`, `null / x = null`). This affects BOTH:
- **Grouped** formulas (query_table joined on a facet): a group present in one
  query and absent in another → null for that group → blank cell. Example: a
  per-tool table with separate `succeeded` and `failed` queries — a tool that
  never failed has no row in the `failed` query, so `runs = succ + fail` is null.
- **Scalar** rate tiles (query_value with two queries, no group_by): if one query
  matches zero events in the window, the whole tile goes blank instead of showing
  the real rate.
**Check:** run the widget against a window where one input query matches nothing
(a tool with zero failures; a period with zero failures). Look for blanks/No-data
where a real number is expected.
**Fix:** wrap **every operand** in `default_zero()` so a genuine zero renders as 0:
`default_zero(succ) / (default_zero(succ) + default_zero(fail)) * 100`.
A partially-wrapped formula (`default_zero(succ)/(succ+fail)`) still breaks — wrap
each name. Note this does NOT create a misleading 0% on a truly-empty denominator:
`default_zero(a)/(default_zero(a)+default_zero(b))` with everything zero is `0/0`,
which still renders **"No data"** (see pitfall 7) — the honest result.
**Caveat (pitfall 2):** defaulting a *numerator* to 0 when its telemetry is dark
can read as a healthy 0 — pair every rate with a total/liveness tile.

## 4. Count vs cardinality (over-counting)
**Symptom:** a count looks too high; two widgets that should agree disagree slightly.
**Cause:** lifecycle events fire multiple times per entity (a tool execution emits queued→starting→running→succeeded). Plain `count` of events ≠ number of executions. Retries and duplicate emissions inflate further.
**Check:** compare `count(*)` to `cardinality(entity_id)` for the same filter.
**Fix:** to count "how many X," use `cardinality` of the entity id (e.g. `@metadata.execution_id`), not event count. **Use the same method across every widget that feeds the same rate** — otherwise the numerator and denominator drift and the rate is wrong.

## 5. Denominator semantics (rate means what?)
**Symptom:** a rate is defensible but subtly misleading.
**Cause:** the denominator does not mean what the reader assumes. "distinct `scan_token` across all logs" is "scans mentioned," not "scans started" — a long-running or late-triaged scan inflates it. Per-cycle events (`process_cycle.*` fire every cycle) inflate a per-scan denominator.
**Check:** state in words what the denominator counts and whether it matches the title.
**Fix:** count distinct entities (`cardinality(scan_token)`) not events; exclude expected-noise groups; and **label approximations `(approx)`** with a note explaining the proxy and what would make it exact.

## 6. Facet / measure does not exist yet
**Symptom:** a group_by or a `sum`/`cardinality` compute renders empty even though the field is correct.
**Cause:** in logs, grouping by a field generally needs a **facet**, and
`sum`/`avg` on a numeric field needs a **measure**. Datadog auto-creates some
facets and supports facet-less search, but a field that has never appeared in the
indexed window (e.g. `failed_reason` when there have been zero failures) may have
no facet/measure yet, so the group_by or measure renders empty.
**Check:** confirm the facet/measure exists in Logs → Configuration, or that live events carry it.
**Fix:** create the facet/measure in the Datadog UI, or note that the widget lights up only after the first qualifying event.

## 7. Division-by-zero rendering
**Behavior:** `a / b * 100` with `b = 0` renders **"No data" / "—"**, NOT `0%`.
Datadog treats division by zero as undefined, not zero. This is the honest
result for a rate with no denominator, and it is why `default_zero` (pitfall 3)
does not manufacture a fake 0%: `default_zero(a)/default_zero(b)` with `b`'s
underlying count zero is still `x/0` → No data.
**Do not** try to force a rate to show 0% when nothing ran — "No data" is correct.
**Check:** run the rate in an empty window and confirm it reads "No data," and in
a zero-numerator-but-nonzero-denominator window (e.g. zero failures, real
successes) confirm it reads the real number (0%), not blank — if blank, an operand
needs `default_zero` (pitfall 3).

## 8. conditional_formats ordering and the cell_display_mode:"bar" conflict
**Symptom:** color thresholds do not show, or the wrong color shows.
**Cause A:** `cell_display_mode: "bar"` on a table formula **silently disables** `conditional_formats` on that formula.
**Cause B:** conditional_formats are first-match-wins; misordered comparators color the wrong band.
**Fix A:** drop `cell_display_mode` (or set `"number"`) when you want colors.
**Fix B:** order so the first matching rule is correct. Higher-is-worse (failure rate): `>20 red, >5 yellow, <=5 green`. Higher-is-better (success/enrichment): `<50 red, <90 yellow, >=90 green`.

## 9. Env / template-variable dimension mismatch
**Symptom:** the env selector works for some widgets, not others.
**Cause:** the env dimension differs by data source. Logs use the `env:` tag;
metrics use `env:` inside `{}`. Kubernetes events may not carry an `env:` tag at
all — in clusters where the namespace encodes the environment, filter on
`kube_namespace:` instead (infra-specific; confirm how your cluster tags events).
**Check:** switch the template variable and confirm every widget responds.
**Fix:** use the right dimension per source: `env:$env.value` (logs),
`{env:$env.value AND ...}` (metrics), and for k8s events either `env:$env.value`
(if tagged) or `kube_namespace:$env.value` (if namespace == env).

## 10. Retention wall and metric tag cardinality
**Symptom:** "trend over months" is empty beyond ~2 weeks; or a log-based metric explodes cost / is rejected.
**Cause:** log retention is short (index tiers are typically 3/7/15/30/45/60 days
— often 15; check your index config); APM similar. Long-range trend needs
**log-based metrics** (~15-month retention). But metric tags must be low-cardinality.
**Fix:** promote the handful of KPIs worth long-term trending to log-based metrics. **Never tag a metric with high-cardinality fields** (scan_token, execution_id, session_token, user_id). Safe group-by tags: tool, status, reason, jobs_status, env.

## 11. Phrase-search tokenization
**Symptom:** an event-name filter matches too much or too little.
**Cause:** full-text `"continuous_triage.process_cycle.processed"` is tokenized; dots split, and the phrase matches anywhere in the message.
**Fix:** prefer filtering on the exact message/event-name attribute when one exists; if using phrase search, verify the match count against a known-good window.
