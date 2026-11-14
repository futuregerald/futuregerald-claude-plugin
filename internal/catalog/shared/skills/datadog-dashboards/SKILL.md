---
name: datadog-dashboards
description: Create, edit, and review Datadog dashboard JSON so every widget actually renders the data the user wants — not merely valid JSON. Enforces three gates: (1) valid JSON structure, (2) every query verified against LIVE data, (3) each section verified to answer its intended question. Use when building a Datadog dashboard, reviewing or importing/exporting dashboard JSON, writing dashboard queries (query_value, timeseries, toplist, query_table, list_stream), adding rates or denominators, or debugging why a tile shows "No data", a blank cell, or a wrong number. Also use when planning what a dashboard should track over time: it puts trend widgets on metrics and events, because logs age out in days. Triggers include "datadog dashboard", "dashboard JSON", "review this dashboard", "build/create a dashboard", "widget shows no data", "why is this tile empty", "does this query return data", "trend", "over time", "week over week", "log-based metric", "retention".
---

# Datadog Dashboards

## Core principle

A dashboard is correct only when it **renders what the user wants against real
data**. Valid JSON and correct-looking queries are necessary but not sufficient.
The failure modes that matter — empty tiles, blank cells, numbers that mean
something other than the title claims — are **data-dependent and invisible in
static review**. Never declare a dashboard done from inspection alone. Verify
against live data.

This skill exists because static/LLM review reliably catches schema, field-name,
and consistency bugs and reliably **misses** runtime data behavior (sparse joins,
empty facets, count-vs-cardinality drift, no-data-vs-zero). Only running the
queries closes that gap.

## Trends need metrics and events, not logs

**Whenever a dashboard is planned, built or reviewed, propose the metrics and
events it needs for trends. Do not wait for a tile to come up empty.** Logs are
the shortest-lived data in Datadog: index retention is usually days (often 15),
and indexed spans a few weeks. Metrics and events are typically kept for about
15 months. A widget built on logs can never show more than the log window,
however wide its time picker.

Check the real retention before designing. Search logs oldest-first over a wide
window and read the oldest timestamp, then do the same for spans, events and one
metric. Do not assume.

Choose each widget's source by the question it answers:

| Question | Source |
|---|---|
| What happened just now, which requests failed, show me the lines | Logs or spans (`list_stream`, recent toplists) |
| Is it getting better or worse, week over week, since the release, SLO | **Metric** |
| Rate, error rate or latency of an instrumented service | **APM trace metrics** (`trace.<operation>.hits`, `.errors`, `.duration`), which already exist with long retention |
| A business or domain count that only appears in logs | **Log-based metric** (a count, or a distribution over a numeric attribute) |
| Something not in logs or traces | **Custom metric** emitted by the app (DogStatsD or OpenTelemetry) |
| Why did the line move | **Events** overlaid on the timeseries: deploys (change tracking), feature-flag flips, backfills, incidents, config changes |

When proposing a metric:

- **It has no history before the day it is created.** Say so, and create it early.
- **Give exact definitions:** name, query, aggregation and group-by tags. Creating
  a metric is usually a console or API step the tools here cannot do, so the user
  needs something to paste.
- **Group only by low-cardinality tags** (status, reason, tool, env, service),
  never ids, tokens or users. See pitfall 10 in `references/query-pitfalls.md`.
- **Pair every rate with its denominator metric**, as for log tiles.

When proposing events, name where each one comes from (an existing integration,
CI calling the events API on deploy, the app emitting one on a flag change or
backfill) and which trend widget it will explain.

## The three gates (all required, in order)

**Gate 1 — Structure.** JSON parses; every widget is well-formed.
Run `scripts/validate_dashboard.py <file>`. Fix all errors; weigh warnings.

**Gate 2 — Data.** Every query returns what you expect, confirmed by **running it
against Datadog** (MCP `search_datadog_logs` / `analyze_datadog_logs` / metrics,
or the UI). Field names, facets, and measures must exist in the *actual emitted
events* — not in a PR description, doc, or memory.
Use `scripts/extract_queries.py <file>` to get the per-widget checklist.

**Gate 3 — Intent.** For each section and each widget, ask *"what question does
this answer, and will it render that?"* A widget can pass Gates 1–2 and still
answer the wrong question or mislead.

Do not tell the user a dashboard is ready until all three gates pass. If Gate 2
is impossible (no data access), say so explicitly — do not substitute inspection
for verification.

## Systematic debugging discipline

Treat every widget as a **hypothesis** about what will render, then test it.

- **Evidence before assertions.** Before claiming a widget works (or a review is
  clean), run its query and read the actual output. Reviewing your own mental
  model with a copy of your own mental model will not catch an error in that model.
- **Reproduce, don't guess.** When a tile is empty or a number is off, run the
  widget's exact query in the explorer and observe. Do not theorize a cause.
- **One variable at a time.** Isolate: is the event landing? the field name? the
  facet? the compute? the formula? Narrow before fixing.
- **Verify names at the source.** Pull a real event with `extra_fields:["*"]` and
  read the true `@attribute` paths. Codebases and PRs lie about field names; live
  events do not.

## Workflow — creating a dashboard

1. **Capture intent per section.** Before writing JSON, write one sentence per
   section: "this section answers **X** for **audience Y**." Keep it — it is the
   Gate-3 checklist.
2. **Pick the source for each widget** (see *Trends need metrics and events*
   above). Every trend widget goes on a metric. List the log-based metrics,
   custom metrics and events the board needs, with definitions, and give that
   list to the user with the JSON.
3. **Discover the real schema.** Before writing any query, confirm actual event
   names, field paths, facet values, and measures by querying live data. Never
   trust a field name you have not seen in a live event.
4. **Draft the JSON.** See `references/datadog-json.md` for widget shapes,
   formulas (`default_zero`, `cardinality`), conditional_formats, per-widget time.
5. **Gate 1** — validate structure.
6. **Gate 2** — run every query against live data; confirm non-empty and shaped
   as intended. Watch `references/query-pitfalls.md` while doing this.
7. **Gate 3** — for each section, confirm it renders the sentence from step 1.
8. **Iterate.**

## Workflow — reviewing a dashboard

Apply the same three gates to existing JSON. Specifically:

- Run `validate_dashboard.py` (Gate 1).
- **Extract every query and run each against live data** (Gate 2) — do **not**
  approve from static reading. This is exactly the class of bug static review
  misses.
- For each section, state what a viewer would conclude and whether that matches
  the user's intent (Gate 3).
- Flag every trend widget built on logs or spans, and every time window longer
  than its source's retention. Recommend the metric or event that replaces it,
  with its definition.
- When you report a review, mark findings and say plainly which gates you could
  and could not verify (e.g. "Gate 2 verified against prod; ffuf tile returns 15
  runs" vs "Gate 2 not run — no data access").

## Gate 3 — "will it render what they want?" checklist

For every widget and section, ask:

- What exact question does this answer?
- Against current data, what will it show — a number, a trend, empty? Is empty
  **meaningful** (healthy zero) or a **bug** (wrong field / missing facet / dark
  telemetry)?
- Can "no data" be told apart from "healthy zero"? Every failure count needs a
  **denominator/total** beside it.
- Does the number **mean what the title claims**? Check denominator semantics,
  count-vs-cardinality, and that approximations are labeled `(approx)`.
- Does the widget's **time window fit the retention of its source**? Is every
  trend on a metric, with events overlaid where a change could explain a shift?
- Is the **most useful information at the top** of the section and the board?

## Pitfalls

Before writing or reviewing queries, read `references/query-pitfalls.md` — the
catalog of reasons a query returns no/wrong data: field-name reality gap,
no-data-vs-zero, sparse-join nulls, count-vs-cardinality, denominator semantics,
missing facets/measures, division-by-zero, conditional_formats /
`cell_display_mode` conflicts, env-dimension mismatch, retention & metric tag
cardinality, phrase-search tokenization.

## Bundled resources

- `scripts/validate_dashboard.py` — Gate 1 structural linter (JSON, widget shape,
  sparse-join risk, cell_display_mode/conditional_formats conflict, section
  numbering, stale note cross-references).
- `scripts/extract_queries.py` — dumps a per-widget query checklist for Gate 2.
- `references/query-pitfalls.md` — the data-returns failure-mode catalog.
- `references/datadog-json.md` — widget/query/formula schema essentials.
