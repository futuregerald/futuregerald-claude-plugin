#!/usr/bin/env python3
"""
Extract every widget's query from a Datadog dashboard JSON so each one can be
run against live data (Gate 2). Static review cannot tell you a query returns
data — you must execute it. This prints a checklist to work through.

Usage:
    extract_queries.py path/to/dashboard.json
"""
import json
import sys


def qstrings(de):
    """Yield (kind, text) for every query AND formula in a widget definition.
    kind is a data_source (for queries) or 'formula' (for the expression the
    queries feed — pitfalls 3/5/7 say verify these too)."""
    for r in _as_list(de.get("requests")):
        # list_stream
        q = r.get("query")
        if isinstance(q, dict) and q.get("query_string") is not None:
            yield q.get("data_source", "logs_stream"), q["query_string"]
        # metrics legacy single string
        if isinstance(r.get("q"), str):
            yield "metrics", r["q"]
        for qq in r.get("queries", []) or []:
            ds = qq.get("data_source", "?")
            if isinstance(qq.get("query"), str):          # metrics-style
                yield ds, qq["query"]
            elif isinstance(qq.get("search"), dict):       # logs/spans/rum/events
                comp = qq.get("compute", {})
                agg = comp.get("aggregation")
                metric = comp.get("metric")
                gb = ",".join(g.get("facet", "?") for g in qq.get("group_by", []) or [])
                suffix = f"   [compute: {agg}{'('+metric+')' if metric else ''}"
                suffix += f"; group_by: {gb}]" if gb else "]"
                yield ds, qq["search"].get("query", "") + suffix
        for fm in r.get("formulas", []) or []:
            if fm.get("formula"):
                alias = f" as \"{fm['alias']}\"" if fm.get("alias") else ""
                yield "formula", fm["formula"] + alias


def _as_list(x):
    if x is None:
        return []
    return x if isinstance(x, list) else [x]


def walk(widgets, out):
    for w in widgets:
        de = w.get("definition", {})
        t = de.get("type")
        if t == "group":
            out.append((f"§ {de.get('title','')}", None, None))
            walk(de.get("widgets", []), out)
        elif t == "note":
            continue
        else:
            title = de.get("title", "(untitled)")
            found = list(qstrings(de))
            if not found:
                out.append((f"  • {title}", "!", "no query extracted — inspect manually"))
            for ds, q in found:
                out.append((f"  • {title}", ds, q))


def main():
    if len(sys.argv) != 2:
        print("usage: extract_queries.py path/to/dashboard.json", file=sys.stderr)
        sys.exit(2)
    d = json.load(open(sys.argv[1]))
    out = []
    walk(d.get("widgets", []), out)
    print("QUERY VERIFICATION CHECKLIST — run each against Datadog, confirm it "
          "returns the expected non-empty, correctly-shaped result.\n")
    for title, ds, q in out:
        if ds is None:
            print(f"\n{title}")
        else:
            print(f"{title}\n      [{ds}] {q}")
    print("\nFor each: does it return data? Is the count/shape what the title "
          "claims? Is an empty result meaningful (healthy zero) or a bug "
          "(wrong field / missing facet / broken telemetry)?")


if __name__ == "__main__":
    main()
