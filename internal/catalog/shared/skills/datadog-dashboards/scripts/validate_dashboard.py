#!/usr/bin/env python3
"""
Structural linter for a Datadog dashboard JSON export (Gate 1).

Catches STATIC failure modes only: invalid JSON, malformed widgets, formula
references to a query that does not exist, multi-query arithmetic missing
default_zero() (null propagation -> blank/No-data), cell_display_mode vs
conditional_formats conflicts, cardinality/sum computes missing a metric,
section-numbering drift, and note cross-references to a section that does not
exist.

It does NOT and CANNOT confirm a query returns data — that is Gate 2 (run the
query against Datadog). Passing this linter means the JSON is well-formed, not
that the dashboard is correct.

Usage:
    validate_dashboard.py path/to/dashboard.json
Exit code 0 = no errors (warnings allowed), 1 = errors found or bad JSON.
"""
import json
import re
import sys

DATA_TYPES = {"timeseries", "query_value", "toplist", "query_table",
              "list_stream", "scatterplot", "heatmap", "sunburst",
              "table", "geomap", "treemap", "distribution"}
# request keys that indicate a non-queries[] request shape (scatterplot/hostmap/etc.)
DICT_REQUEST_KEYS = {"x", "y", "fill", "size", "table"}
errors, warnings = [], []
missing_time = []


def err(m): errors.append(m)
def warn(m): warnings.append(m)


def walk(widgets, path="widgets"):
    for i, w in enumerate(widgets):
        de = (w or {}).get("definition")
        if not isinstance(de, dict):
            err(f"{path}[{i}]: widget has no 'definition' object")
            continue
        t = de.get("type")
        p = f"{path}[{i}]({t}:{de.get('title','') or 'note'})"
        if t == "group":
            walk(de.get("widgets", []), p + ".widgets")
        elif t == "note":
            continue
        elif t in DATA_TYPES:
            check_data_widget(de, p)
            if "time" not in de:
                missing_time.append(de.get("title", "?"))
        # unknown types: leave alone; Datadog adds new ones


def check_data_widget(de, p):
    reqs = de.get("requests")
    if reqs in (None, [], {}):
        err(f"{p}: data widget has no 'requests'")
        return
    if de["type"] == "list_stream":
        for r in _as_list(reqs):
            q = r.get("query")
            if not q or q.get("query_string") is None:
                warn(f"{p}: list_stream request missing query.query_string")
        return
    for r in _as_list(reqs):
        # scatterplot/hostmap etc. use x/y/fill/table request shapes — skip cleanly
        if any(k in r for k in DICT_REQUEST_KEYS):
            continue
        queries = r.get("queries")
        formulas = r.get("formulas")
        if queries is None:
            if "q" not in r and "query" not in r:
                warn(f"{p}: request has neither 'queries' nor legacy 'q'")
            continue
        names = [q.get("name") for q in queries if q.get("name")]
        for q in queries:
            check_query(q, p)
        check_formulas(formulas, names, r, p)


def check_query(q, p):
    comp = q.get("compute") or {}
    agg = comp.get("aggregation")
    ds = q.get("data_source")
    if ds in ("logs", "spans", "rum", "events", "profiles") and \
            agg in ("cardinality", "sum", "avg", "min", "max", "median",
                    "pc75", "pc90", "pc95", "pc99", "percentile") and not comp.get("metric"):
        err(f"{p} query '{q.get('name')}': compute '{agg}' needs a 'metric' "
            f"(the @attribute to aggregate); only 'count' takes no metric")
    for g in q.get("group_by") or []:
        if not g.get("facet"):
            err(f"{p} query '{q.get('name')}': group_by entry missing 'facet'")


IDENT = re.compile(r"[A-Za-z_]\w*")


def _bare_idents(expr):
    """Identifiers in a formula that are NOT function calls (not followed by '(')
    and NOT part of a number like 1e3. These should be query names."""
    out = []
    for m in IDENT.finditer(expr):
        start, end = m.start(), m.end()
        if start > 0 and (expr[start - 1].isdigit() or expr[start - 1] == "."):
            continue  # e.g. the 'e3' in 1e3
        after = expr[end:].lstrip()
        if after.startswith("("):
            continue  # function call, e.g. default_zero(...)
        out.append(m.group())
    return out


def check_formulas(formulas, names, request, p):
    if not formulas:
        return
    nameset = set(names)
    for f in formulas:
        expr = f.get("formula", "")
        refs = [t for t in _bare_idents(expr) if t in nameset]
        # (a) formula references a query name that does not exist (rename bug)
        for t in _bare_idents(expr):
            if t not in nameset:
                err(f"{p} formula '{expr}': references '{t}', which is not a "
                    f"query name in this request {sorted(nameset)}. Undefined "
                    f"reference (likely a renamed/removed query).")
        # (b) null-propagation: multi-query arithmetic must default_zero EVERY
        # operand. A query that matches zero events returns null; null + x = null,
        # so the cell/tile renders blank/No-data instead of the real number.
        # Applies to grouped (query_table, per-group) AND scalar (query_value)
        # rates alike. Note: an empty *denominator* still renders "No data" even
        # with default_zero (x/0 is undefined), which is the honest result.
        if len(set(refs)) >= 2 and re.search(r"[+\-*/]", expr):
            unwrapped = sorted({r for r in refs
                                if not re.search(rf"default_zero\(\s*{re.escape(r)}\b", expr)})
            if unwrapped:
                warn(f"{p} formula '{expr}': operand(s) {unwrapped} not wrapped in "
                     f"default_zero(). A query matching zero events returns null and "
                     f"propagates through the arithmetic -> blank/No-data instead of the "
                     f"real value. Wrap EACH operand: default_zero(a)/(default_zero(a)+"
                     f"default_zero(b))*100. (Div-by-zero on an empty denominator still "
                     f"correctly renders No data.)")
        # (c) bar display mode silently disables conditional_formats on same formula
        if f.get("cell_display_mode") == "bar" and f.get("conditional_formats"):
            err(f"{p} formula '{expr}': cell_display_mode:'bar' overrides the "
                f"conditional_formats on the same formula (threshold colors will not "
                f"render). Drop cell_display_mode or set 'number'.")
        check_conditional_formats(f.get("conditional_formats"), p, expr)
    check_conditional_formats(request.get("conditional_formats"), p, "<request>")


def check_conditional_formats(cfs, p, ctx):
    if not cfs:
        return
    for cf in cfs:
        if not all(k in cf for k in ("comparator", "value", "palette")):
            err(f"{p} conditional_format ({ctx}): each entry needs comparator, value, palette")


def check_sections(widgets):
    titles = [w["definition"]["title"] for w in widgets
              if w.get("definition", {}).get("type") == "group"
              and w["definition"].get("title")]
    nums = [int(m.group(1)) for tt in titles
            for m in [re.match(r"\s*(\d+)\s*[·.\-)]", tt)] if m]
    if nums and nums != list(range(1, len(nums) + 1)):
        err(f"Section numbers are {nums}, expected contiguous "
            f"{list(range(1, len(nums)+1))} (reorder/renumber drift).")
    return len(nums)


def note_contents(widgets, out):
    for w in widgets:
        de = w.get("definition", {})
        if de.get("type") == "note":
            out.append(de.get("content", ""))
        elif de.get("type") == "group":
            note_contents(de.get("widgets", []), out)


def check_note_cross_refs(widgets, n_sections):
    notes = []
    note_contents(widgets, notes)
    for content in notes:
        for m in re.finditer(r"section\s*\(?(\d+)\)?", content, re.IGNORECASE):
            n = int(m.group(1))
            if n_sections and n > n_sections:
                warn(f"A note references 'section {n}' but there are only "
                     f"{n_sections} sections — likely a stale cross-reference.")


def _as_list(x):
    return x if isinstance(x, list) else ([] if x is None else [x])


def main():
    if len(sys.argv) != 2:
        print("usage: validate_dashboard.py path/to/dashboard.json", file=sys.stderr)
        sys.exit(2)
    try:
        d = json.load(open(sys.argv[1]))
    except json.JSONDecodeError as e:
        print(f"❌ INVALID JSON: {e}")
        sys.exit(1)
    widgets = d.get("widgets", [])
    walk(widgets)
    n = check_sections(widgets)
    check_note_cross_refs(widgets, n)
    if missing_time:
        warn(f"{len(missing_time)} data widget(s) have no per-widget 'time' "
             f"({', '.join(missing_time[:5])}{'…' if len(missing_time) > 5 else ''}). "
             f"Fine if you rely on the global picker; set it only to pin a fixed window.")

    for w in warnings:
        print(f"⚠️  {w}\n")
    for e in errors:
        print(f"❌ {e}\n")
    print(f"── {len(errors)} error(s), {len(warnings)} warning(s) ──")
    print("Gate 1 (structure) only. Now run every query against live data (Gate 2) "
          "and confirm each section answers its question (Gate 3).")
    sys.exit(1 if errors else 0)


if __name__ == "__main__":
    main()
