# Datadog dashboard JSON essentials

Just enough schema to build/edit dashboards correctly. For query correctness
see query-pitfalls.md.

Table of contents:
1. Top-level shape
2. Widget types
3. Log/spans query block (search + compute + group_by)
4. Formulas (default_zero, cardinality, rates)
5. conditional_formats
6. Per-widget time window
7. Template variables & env dimensions
8. Log-based metrics (long-term trends)
9. Importing
10. Editing large JSON safely

---

## 1. Top-level shape
```json
{
  "title": "...", "description": "...",
  "layout_type": "ordered", "reflow_type": "auto",
  "template_variables": [{"name":"env","prefix":"env","available_values":["production","staging"],"default":"production"}],
  "widgets": [ /* notes, groups, data widgets */ ]
}
```
There is **no top-level default time range** honored on import — the picker range
lives in the URL/session. To force a default window, set per-widget `time`
(section 6). Sections are `group` widgets; number them in the title (`"1 · …"`)
and keep numbering contiguous.

## 2. Widget types
- `note` — markdown. No query, no time. Good for section intros and honesty labels.
- `group` — `layout_type:"ordered"`, holds `widgets:[...]`. A section.
- `query_value` — single scalar (a tile). `requests[0].response_format:"scalar"`.
- `timeseries` — lines/bars over time. `response_format:"timeseries"`, `display_type`.
- `toplist` — ranked bars by a group_by facet. `response_format:"scalar"`.
- `query_table` — multi-column table; multiple queries joined on a shared group_by. `response_format:"scalar"`.
- `list_stream` — raw log/event stream; uses `query.query_string`, not `queries[]`.

**`response_format` must match the widget type** or import fails / the widget is
blank: `"scalar"` for query_value / toplist / query_table; `"timeseries"` for
timeseries. Mismatch is a common import-breaker.

## 3. Log/spans query block
```json
{
  "name": "fail", "data_source": "logs",
  "search": { "query": "service:x env:$env.value \"event.name\" @metadata.status:failed" },
  "indexes": ["*"],
  "group_by": [{ "facet": "@metadata.tool", "limit": 25,
                 "sort": { "order":"desc", "aggregation":"cardinality", "metric":"@metadata.execution_id" } }],
  "compute": { "aggregation": "cardinality", "metric": "@metadata.execution_id" }
}
```
- `compute.aggregation`: `count` (no metric), or `cardinality`/`sum`/`avg`/`min`/`max`/percentiles (**require** `metric`).
- `count` = event count. `cardinality(@id)` = distinct entities. See pitfall 4.
- metrics data_source uses a single string: `"query": "sum:trace.sidekiq.job.hits{env:$env.value AND service:x} by {resource_name}.as_count()"`.

## 4. Formulas
`formulas:[{ "formula":"<expr>", "alias":"label" }]`. Expressions reference query
`name`s. Useful functions:
- `default_zero(q)` — treat a null (a query matching zero events, or a missing
  group) as 0. Wrap **every operand** of any multi-query arithmetic — scalar rates
  and grouped tables alike (pitfall 3). Per-operand: `default_zero(a)/(default_zero(a)
  +default_zero(b))`, not `default_zero(a)/(a+b)`.
- Rate: `default_zero(fail) / (default_zero(succ) + default_zero(fail)) * 100`.
  An empty denominator still renders "No data" (x/0 is undefined), not a fake 0%.
- A formula can carry its own `conditional_formats` and (table only)
  `cell_display_mode` — but `cell_display_mode:"bar"` suppresses the colors
  (pitfall 8).

## 5. conditional_formats
```json
"conditional_formats": [
  { "comparator": ">", "value": 20, "palette": "white_on_red" },
  { "comparator": ">", "value": 5,  "palette": "white_on_yellow" },
  { "comparator": "<=","value": 5,  "palette": "white_on_green" }
]
```
First match wins — order accordingly. Lives on a `query_value`/`toplist` request,
or on a `query_table` formula. Palettes: `white_on_red|yellow|green`, `*_on_white`.
Omit thresholds deliberately for "informational" tiles (and title them so).

## 6. Per-widget time window
```json
"time": { "type": "live", "unit": "day", "value": 14 }
```
Goes inside a data widget's `definition`. Legacy form `{"live_span":"1w"}` has a
fixed enum (no 14-day option); the `type:"live"` form takes arbitrary
unit+value. Set on every data widget to pin a board to a fixed window (e.g.
match log retention); omit to follow the global picker. Notes/groups get no time.

## 7. Template variables & env dimensions
`$env.value` interpolates the selected value. The dimension differs per source
(pitfall 9): logs `env:$env.value`; metrics `{env:$env.value}`; k8s events
`kube_namespace:$env.value`. Verify every widget responds to the selector.

## 8. Log-based metrics (long-term trends)
Raw logs expire (index tiers ~3–60 days, often 15). For trends beyond your
retention, define a **log-based metric**
(Logs → Generate Metrics, ~15-month retention): a filter query, a
count-or-measure, and low-cardinality group-by tags. Never tag with
scan_token/execution_id/user_id (pitfall 10). Then chart the metric with a
`metrics` query.

## 9. Importing
The Datadog MCP server is read-only — it cannot create/update dashboards. Import
by pasting JSON in the UI (New Dashboard → Configuration → Import Dashboard JSON)
or via the Dashboards API (`POST /api/v1/dashboard`). After import, open the board
and confirm every widget renders (Gate 2/3) — import success is not render
success. Field/facet references, `@metadata.*` paths, service names, and tool
values in these docs are **examples from one setup**; substitute your own,
verified against live events.

## 10. Editing large JSON safely
Exported dashboards are large. Prefer a small Python script (load JSON, mutate
the parsed structure, `json.dump(indent=2)`, re-`json.load` to validate) over
string edits — it survives reformatting and avoids whitespace-match failures.
After any change: re-parse, run `scripts/validate_dashboard.py`, then re-verify
affected queries against live data.
