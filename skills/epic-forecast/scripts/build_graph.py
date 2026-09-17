#!/usr/bin/env python3
"""Turn the manifest's `graph` block into a Mermaid diagram and an edge table.

Usage:
    python3 build_graph.py manifest.json [out.md]

Writes Markdown to `out.md`, or to stdout if no output path is given.
Standard library only — no dependencies, no venv.

Manifest schema (a top-level `graph` block, beside `sheets`)
------------------------------------------------------------
{
  "graph": {
    "nodes": [
      {"id": "e1", "label": "Search reindex", "key": "ABC-101", "kind": "epic",
       "weight_opt": 3.0, "weight_pess": 6.0},
      {"id": "q1", "label": "Code review on 3 children", "kind": "queue"},
      {"id": "d1", "label": "Retention policy call", "kind": "decision",
       "note": "owner: Legal"},
      {"id": "x1", "label": "Billing service (another team)", "kind": "external"},
      {"id": "e2", "label": "Consumer rollout", "key": "ABC-102", "kind": "epic",
       "weight_opt": 2.0, "weight_pess": 4.0}
    ],
    "edges": [
      {"from": "q1", "to": "e1", "type": "hard", "source": "agent",
       "note": "THE actual constraint"},
      {"from": "e1", "to": "e2", "type": "soft", "source": "tracker"}
    ]
  }
}

`id` is the join key. The Dependencies sheet rows carry the same ids, which is what
lets a dependency row reach the estimate a weighted chain needs — the human-readable
labels differ between sheets and cannot be joined on.

Edge direction is "blocks": `from` must complete before `to` can finish.

Node kinds
    epic      in-scope work with an estimate; the ONLY kind that carries weights
    external  another team, vendor or service; usually no ticket and no estimate
    decision  an undecided product/legal/management call; carry the owner in `note`
    queue     review or approval latency, e.g. a named PR awaiting review

Edge types
    hard  drives ordering and the critical chain
    soft  drawn dashed, reported, and EXCLUDED from ordering and the chain

Edge sources
    agent    from the research agents' dependency lists — authoritative
    tracker  from tracker link types — a supplement, flagged in the output so a
             reader can see the agents missed it
"""

import argparse
import json
import re
import sys
from collections import deque

MAX_NODES = 500
MAX_INDIRECT_PAIRS = 200
IMPLAUSIBLE_WEIGHT = 1000.0

NODE_KINDS = ("epic", "external", "decision", "queue")
EDGE_TYPES = ("hard", "soft")
EDGE_SOURCES = ("agent", "tracker")

KIND_SHAPES = {
    "epic": '{id}["{label}"]',
    "external": '{id}[["{label}"]]',
    "decision": '{id}{{"{label}"}}',
    "queue": '{id}(["{label}"])',
}

KIND_LEGEND = [
    ("epic", "rectangle", "in-scope work, carries an estimate"),
    ("external", "subroutine", "another team, vendor or service"),
    ("decision", "rhombus", "an undecided call, with an owner"),
    ("queue", "stadium", "review or approval latency"),
]


class ManifestError(ValueError):
    """The graph block is malformed in a way that makes the output meaningless."""


def _is_number(value):
    return not isinstance(value, bool) and isinstance(value, (int, float))


CONTROL_CHARS = re.compile(r"[\x00-\x1f\x7f\u2028\u2029]")

# Mermaid renders labels as HTML, and its flowchart lexer reads the shape
# delimiters even inside a quoted label. Numeric entities render as the literal
# character without ever reaching either parser as syntax.
#
# ORDER IS LOAD-BEARING. '#' opens a mermaid entity, so it is the escape
# character and must be escaped before anything that introduces one. A label
# reading "PR #1234; 6 days waiting" otherwise reaches mermaid as the entity
# '#1234;' and the ticket number disappears from the diagram.
MERMAID_ENTITIES = (
    ("#", "#35;"),
    ("%%", "#37;#37;"),
    ('"', "#quot;"), ("<", "#lt;"), (">", "#gt;"),
    ("[", "#91;"), ("]", "#93;"), ("(", "#40;"), (")", "#41;"),
    ("{", "#123;"), ("}", "#125;"),
)
MARKDOWN_PUNCTUATION = re.compile(r"([<>\[\]`*_])")


def md(text):
    """Neutralise node labels and notes before they reach Markdown.

    Labels and notes come from ticket summaries and dependency notes, which are
    written by anyone who can comment on an item. Unescaped, a `|` shifts every
    later cell in the row so the reader sees the wrong source against the wrong
    edge, and raw HTML or a `javascript:` link survives into the HTML artefact
    the report format offers to generate.
    """
    out = CONTROL_CHARS.sub(" ", str(text)).replace("\\", "\\\\")
    return MARKDOWN_PUNCTUATION.sub(r"\\\1", out).replace("|", "\\|")


# --------------------------------------------------------------------------- #
# Validation
# --------------------------------------------------------------------------- #

def validate(graph):
    """Return (errors, warnings). Errors make the graph unusable; warnings do not."""
    errors, warnings = [], []

    if not isinstance(graph, dict):
        errors.append("graph is not an object")
        return errors, warnings

    nodes = graph.get("nodes")
    edges = graph.get("edges")
    if not isinstance(nodes, list):
        errors.append("graph.nodes is missing or is not a list")
        return errors, warnings
    if edges is None:
        edges = []
    if not isinstance(edges, list):
        errors.append("graph.edges is not a list")
        return errors, warnings

    if len(nodes) > MAX_NODES:
        errors.append(
            f"graph has {len(nodes)} nodes; the cap is {MAX_NODES}. Transitive closure and the "
            "indirect-blocking list are both quadratic, so a graph this size produces a report "
            "hundreds of megabytes long. Narrow the scope — a forecast nobody can read is not a "
            "forecast"
        )
        return errors, warnings

    seen = set()
    for i, node in enumerate(nodes):
        if not isinstance(node, dict):
            errors.append(f"node {i} is {type(node).__name__}, not an object")
            continue
        nid = node.get("id")
        if not nid:
            errors.append(f"node {i} has no id")
            continue
        if not isinstance(nid, str):
            errors.append(f"node {i} has id {nid!r}; an id must be a string")
            continue
        if nid in seen:
            errors.append(f"duplicate node id {nid!r}")
        seen.add(nid)

        kind = node.get("kind")
        if kind not in NODE_KINDS:
            errors.append(f"node {nid!r} has kind {kind!r}; expected one of {NODE_KINDS}")

        for key in ("weight_opt", "weight_pess"):
            weight = node.get(key)
            if weight is None:
                continue
            if not _is_number(weight):
                errors.append(
                    f"node {nid!r} has {key}={weight!r}; a weight must be a number"
                )
            elif weight < 0:
                errors.append(
                    f"node {nid!r} has {key}={weight!r}; a negative weight silently drops the "
                    "node from the chain and publishes a date that is too early"
                )
            elif weight > IMPLAUSIBLE_WEIGHT:
                warnings.append(
                    f"node {nid!r} has {key}={weight!r} engineer-weeks, which is almost "
                    "certainly a unit error rather than an estimate"
                )

        has_weight = node.get("weight_opt") is not None or node.get("weight_pess") is not None
        if kind == "epic":
            # Checked per bound, not "either". A node with weight_opt and no
            # weight_pess passes an either-check clean and is then priced at 0
            # weeks in the pessimistic chain, so the report publishes a narrower
            # band than the data supports with no warning attached.
            for key in ("weight_opt", "weight_pess"):
                if node.get(key) is None:
                    warnings.append(
                        f"epic node {nid!r} carries no {key}; it contributes 0 to that bound's "
                        "chain, which understates it"
                    )
            opt, pess = node.get("weight_opt"), node.get("weight_pess")
            if _is_number(opt) and _is_number(pess) and pess < opt:
                warnings.append(
                    f"epic node {nid!r} has weight_pess {pess} below weight_opt {opt}; the two "
                    "columns are probably transposed"
                )
        elif has_weight:
            errors.append(
                f"node {nid!r} is kind {kind!r} but carries a weight. Weights exist only on "
                "epic nodes — a decision or a review queue is not estimable work, and giving "
                "it a number hides a gate inside a total"
            )

    for i, edge in enumerate(edges):
        if not isinstance(edge, dict):
            errors.append(f"edge {i} is {type(edge).__name__}, not an object")
            continue
        src, dst = edge.get("from"), edge.get("to")
        if src not in seen:
            errors.append(f"edge {i} references unknown node {src!r} in `from`")
        if dst not in seen:
            errors.append(f"edge {i} references unknown node {dst!r} in `to`")
        if edge.get("type", "hard") not in EDGE_TYPES:
            errors.append(f"edge {i} has type {edge.get('type')!r}; expected one of {EDGE_TYPES}")
        if edge.get("source", "agent") not in EDGE_SOURCES:
            errors.append(
                f"edge {i} has source {edge.get('source')!r}; expected one of {EDGE_SOURCES}"
            )

    return errors, warnings


# --------------------------------------------------------------------------- #
# Graph primitives
# --------------------------------------------------------------------------- #

def hard_adjacency(node_ids, edges):
    """Successor map over hard edges only. Soft edges never drive ordering."""
    adj = {nid: [] for nid in node_ids}
    for edge in edges:
        if edge.get("type", "hard") != "hard":
            continue
        src, dst = edge["from"], edge["to"]
        if src in adj and dst in adj and dst not in adj[src]:
            adj[src].append(dst)
    return adj


def find_cycles(node_ids, adj):
    """Cyclic strongly-connected components, as sorted id lists.

    Runs BEFORE any chain computation: a longest path over a cyclic graph has no
    defined answer and a naive DFS recurses until the stack blows. Iterative
    Tarjan, so a deep chain cannot blow the stack either.
    """
    index = {}
    low = {}
    on_stack = {}
    stack = []
    cycles = []
    counter = [0]

    for root in node_ids:
        if root in index:
            continue
        work = [(root, iter(adj[root]))]
        index[root] = low[root] = counter[0]
        counter[0] += 1
        stack.append(root)
        on_stack[root] = True

        while work:
            node, children = work[-1]
            advanced = False
            for child in children:
                if child not in index:
                    index[child] = low[child] = counter[0]
                    counter[0] += 1
                    stack.append(child)
                    on_stack[child] = True
                    work.append((child, iter(adj[child])))
                    advanced = True
                    break
                if on_stack.get(child):
                    low[node] = min(low[node], index[child])
            if advanced:
                continue

            work.pop()
            if work:
                parent = work[-1][0]
                low[parent] = min(low[parent], low[node])
            if low[node] == index[node]:
                component = []
                while True:
                    popped = stack.pop()
                    on_stack[popped] = False
                    component.append(popped)
                    if popped == node:
                        break
                if len(component) > 1 or node in adj[node]:
                    cycles.append(sorted(component))

    return sorted(cycles)


def reachable_from(starts, direction):
    """Everything reachable from `starts` along `direction`, excluding the starts."""
    seen = set()
    frontier = list(starts)
    while frontier:
        for neighbour in direction[frontier.pop()]:
            if neighbour not in seen:
                seen.add(neighbour)
                frontier.append(neighbour)
    return seen


def transitive_closure(node_ids, adj):
    """Downstream reachability per node, over hard edges. Assumes an acyclic input."""
    order = topological_order(node_ids, adj)
    reach = {nid: set() for nid in node_ids}
    for node in reversed(order):
        acc = set()
        for succ in adj[node]:
            acc.add(succ)
            acc |= reach[succ]
        reach[node] = acc
    return reach


def topological_order(node_ids, adj):
    """Kahn's algorithm. Raises if the subgraph still holds a cycle."""
    indegree = {nid: 0 for nid in node_ids}
    for node in node_ids:
        for succ in adj[node]:
            indegree[succ] += 1
    queue = deque(nid for nid in node_ids if indegree[nid] == 0)
    order = []
    while queue:
        node = queue.popleft()
        order.append(node)
        for succ in adj[node]:
            indegree[succ] -= 1
            if indegree[succ] == 0:
                queue.append(succ)
    if len(order) != len(node_ids):
        raise ManifestError("topological_order called on a graph that still contains a cycle")
    return order


def longest_path(node_ids, adj, weight_of):
    """Heaviest path through the DAG. Returns (path, total).

    Node-weighted: the total is the sum of the weights of the nodes ON the path.
    Unweighted nodes (external, decision, queue) contribute 0 weeks, which is why
    the caller reports them by name as gates rather than folding them into a number.
    Ties break on the longer node count, so a chain through named gates is preferred
    to an equal-weight chain that hides them.
    """
    if not node_ids:
        return [], 0.0
    order = topological_order(node_ids, adj)
    best = {}
    nxt = {}
    for node in reversed(order):
        weight = weight_of(node)
        choice, best_score, best_len = None, 0.0, 0
        for succ in adj[node]:
            score, length = best[succ]
            if score > best_score or (score == best_score and length > best_len):
                choice, best_score, best_len = succ, score, length
        best[node] = (weight + best_score, 1 + best_len)
        nxt[node] = choice

    head = max(order, key=lambda n: (best[n][0], best[n][1]))
    path = []
    node = head
    while node is not None:
        path.append(node)
        node = nxt[node]
    return path, round(best[head][0], 4)


def orphans(node_ids, edges):
    """Nodes with no edge in either direction, hard or soft — visibly parallelisable."""
    touched = set()
    for edge in edges:
        touched.add(edge["from"])
        touched.add(edge["to"])
    return [nid for nid in node_ids if nid not in touched]


# --------------------------------------------------------------------------- #
# Analysis
# --------------------------------------------------------------------------- #

def analyse(graph):
    """Run the whole pipeline. Cycle detection first, always."""
    errors, warnings = validate(graph)
    if errors:
        raise ManifestError("; ".join(errors))

    nodes = graph["nodes"]
    edges = graph.get("edges") or []
    by_id = {n["id"]: n for n in nodes}
    node_ids = [n["id"] for n in nodes]

    adj = hard_adjacency(node_ids, edges)
    cycles = find_cycles(node_ids, adj)
    cyclic_ids = {nid for cycle in cycles for nid in cycle}

    # A node whose paths can run into a cycle has no defined longest path either,
    # so excluding only the cycle members would silently understate every chain
    # that enters one. Drop the cycle plus its directed ancestors and descendants.
    # A node off to the side that neither reaches a cycle nor is reached from one
    # keeps a well-defined chain and stays in.
    excluded = set(cyclic_ids)
    if cyclic_ids:
        reverse = {nid: [] for nid in node_ids}
        for node in node_ids:
            for succ in adj[node]:
                reverse[succ].append(node)
        # Each direction gets its own visited set. Sharing one lets the forward
        # pass claim a node and the reverse pass then refuse to expand through
        # it, so with two or more cycles an ancestor of the second survives and
        # gets published as the whole critical chain.
        excluded |= reachable_from(cyclic_ids, adj)
        excluded |= reachable_from(cyclic_ids, reverse)

    acyclic_ids = [nid for nid in node_ids if nid not in excluded]
    acyclic_adj = {nid: [s for s in adj[nid] if s not in excluded] for nid in acyclic_ids}

    chains = {}
    closure = {}
    if acyclic_ids:
        closure = transitive_closure(acyclic_ids, acyclic_adj)
        for bound in ("opt", "pess"):
            key = f"weight_{bound}"

            def weight_of(nid, key=key):
                return float(by_id[nid].get(key) or 0.0)

            path, total = longest_path(acyclic_ids, acyclic_adj, weight_of)
            gates = [nid for nid in path if by_id[nid]["kind"] != "epic"]
            chains[bound] = {"path": path, "weeks": total, "gates": gates}

    epic_ids = [nid for nid in node_ids if by_id[nid]["kind"] == "epic"]
    linked_epics = {e["from"] for e in edges} | {e["to"] for e in edges}
    coverage = (len([e for e in epic_ids if e in linked_epics]), len(epic_ids))

    return {
        "by_id": by_id,
        "node_ids": node_ids,
        "edges": edges,
        "adj": adj,
        "warnings": warnings,
        "cycles": cycles,
        "excluded_from_chain": sorted(excluded),
        "closure": closure,
        "chains": chains,
        "orphans": orphans(node_ids, edges),
        "edge_coverage": coverage,
    }


# --------------------------------------------------------------------------- #
# Rendering
# --------------------------------------------------------------------------- #

def mermaid_id(node_id):
    safe = re.sub(r"[^0-9A-Za-z_]", "_", node_id)
    return safe if safe and not safe[0].isdigit() else f"n_{safe}"


def mermaid_id_map(node_ids):
    """Unique mermaid ids. `ABC-101` and `ABC.101` both sanitise to `ABC_101`.

    Mermaid keeps the last declaration under a shared id, so a collision silently
    deletes a node from the picture while the edge table below still lists both.
    """
    used = {}
    mapping = {}
    for nid in node_ids:
        candidate = mermaid_id(nid)
        if candidate in used:
            used[candidate] += 1
            candidate = f"{candidate}__{used[candidate]}"
        else:
            used[candidate] = 1
        mapping[nid] = candidate
    return mapping


def mermaid_label(text):
    """Mermaid takes HTML entities inside a quoted label; a raw quote closes it.

    Mermaid renders labels as HTML by default, and reads `%%{init: ...}%%`
    directives out of its own input, so a ticket summary reaching this function
    unaltered is both an HTML injection and a config injection.
    """
    out = CONTROL_CHARS.sub(" ", str(text))
    for char, entity in MERMAID_ENTITIES:
        out = out.replace(char, entity)
    return out


def render_mermaid(analysis):
    by_id = analysis["by_id"]
    chain = analysis["chains"].get("opt", {}).get("path", [])
    mid = mermaid_id_map(analysis["node_ids"])

    lines = ["```mermaid", "graph LR"]
    for nid in analysis["node_ids"]:
        node = by_id[nid]
        label = node.get("label", nid)
        if node.get("key"):
            label = f"{label} ({node['key']})"
        lines.append(
            "    "
            + KIND_SHAPES[node["kind"]].format(id=mid[nid], label=mermaid_label(label))
        )

    for edge in analysis["edges"]:
        arrow = "-.->" if edge.get("type", "hard") == "soft" else "-->"
        tag = "|tracker only|" if edge.get("source") == "tracker" else ""
        lines.append(f"    {mid[edge['from']]} {arrow}{tag} {mid[edge['to']]}")

    if chain:
        lines.append("    classDef chain stroke:#b34700,stroke-width:3px;")
        lines.append("    class " + ",".join(mid[n] for n in chain) + " chain;")
    lines.append("```")
    return "\n".join(lines)


def _describe_gate(node):
    """Gate label, kind, and its `note` — which is where the decision owner lives."""
    described = f"{md(node.get('label', node['id']))} ({node['kind']}"
    if node.get("note"):
        described += f" — {md(node['note'])}"
    return described + ")"


def _fmt_weeks(value):
    return f"{value:g}"


def render_markdown(analysis):
    by_id = analysis["by_id"]
    out = ["## Dependency graph", ""]

    node_count = len(analysis["node_ids"])
    edge_count = len(analysis["edges"])
    if node_count == 1 and edge_count == 0:
        out += [
            "One item was in scope and no dependencies were found, so there is no ordering "
            "to recommend and no chain to compute. That is the finding, not an empty section.",
            "",
        ]
    elif edge_count == 0:
        out += [
            f"No dependencies were found across {node_count} items. Either they are genuinely "
            "independent — in which case they are fully parallelisable — or the dependency "
            "research came back thin. Say which before treating this as good news.",
            "",
        ]

    out += [render_mermaid(analysis), ""]
    out += [
        "Shapes: "
        + " · ".join(f"**{k}** {shape} — {why}" for k, shape, why in KIND_LEGEND)
        + ". Solid edges are hard dependencies; dashed edges are soft and never drive ordering. "
        "The highlighted run is the optimistic critical chain; where the two bounds diverge, "
        "the pessimistic path is named in the text below rather than drawn.",
        "",
    ]

    if analysis["cycles"]:
        out += ["### Cycles — a data-quality finding, not a schedule", ""]
        for cycle in analysis["cycles"]:
            names = " → ".join(md(by_id[n].get("label", n)) for n in cycle)
            out.append(f"- **{names} → …** ({', '.join(cycle)})")
        out += [
            "",
            "A cyclic dependency has no defined longest path, so no chain was computed for "
            f"the {len(analysis['excluded_from_chain'])} items connected to it. Somebody has "
            "recorded a dependency backwards, or two pieces of work genuinely need splitting "
            "apart. Resolve it and re-run — do not read the partial chain below as complete.",
            "",
        ]

    out += ["### Critical chain — this is the date-bearing number", ""]
    if not analysis["node_ids"]:
        out += ["The graph is empty — no items were supplied, so there is no chain.", ""]
    elif not analysis["chains"]:
        out += [
            "No chain could be computed: every item is inside a cycle, or upstream or "
            "downstream of one. See above.",
            "",
        ]
    else:
        for bound, label in (("opt", "Optimistic"), ("pess", "Pessimistic")):
            chain = analysis["chains"][bound]
            names = " → ".join(md(by_id[n].get("label", n)) for n in chain["path"])
            line = f"- **{label}: {_fmt_weeks(chain['weeks'])} engineer-weeks** of epic work"
            if chain["gates"]:
                gates = ", ".join(_describe_gate(by_id[g]) for g in chain["gates"])
                line += f", **plus these gates, which carry no estimate: {gates}**"
            out += [line, f"  - Path: {names}"]
        same = (
            analysis["chains"]["opt"]["path"] == analysis["chains"]["pess"]["path"]
        )
        out += [
            "",
            "Both bounds run along the same path."
            if same
            else "**The two bounds run along different paths** — which item decides the date "
            "depends on which bound holds. Name both in the report.",
            "",
            "The chain is reported as epic weeks **plus named gates**, never as a single "
            "number. A legal decision or a review queue has no estimate; folding it in as "
            "zero weeks would publish a date that silently assumes it clears instantly.",
            "",
        ]

    if analysis["excluded_from_chain"]:
        out += [
            f"Excluded from the chain computation (cycle-connected): "
            f"{', '.join(analysis['excluded_from_chain'])}.",
            "",
        ]

    out += ["### Edges", "", "| From | To | Type | Source | Note |", "|---|---|---|---|---|"]
    for edge in analysis["edges"]:
        note = md(edge.get("note", "")) if edge.get("note") else ""
        if edge.get("source") == "tracker":
            note = (note + " — " if note else "") + "**found only in the tracker; the agents missed it**"
        out.append(
            f"| {md(by_id[edge['from']].get('label', edge['from']))} "
            f"| {md(by_id[edge['to']].get('label', edge['to']))} "
            f"| {edge.get('type', 'hard')} | {edge.get('source', 'agent')} | {note} |"
        )
    out.append("")

    indirect = []
    for node, reachable in analysis["closure"].items():
        direct = set(analysis["adj"][node])
        for far in sorted(reachable - direct):
            indirect.append((node, far))
    if indirect:
        out += ["### Blocked transitively", "", "Not visible in any single dependency row:", ""]
        for src, dst in indirect[:MAX_INDIRECT_PAIRS]:
            out.append(
                f"- **{md(by_id[dst].get('label', dst))}** is blocked by "
                f"**{md(by_id[src].get('label', src))}** through the chain above"
            )
        if len(indirect) > MAX_INDIRECT_PAIRS:
            out.append(
                f"- … and {len(indirect) - MAX_INDIRECT_PAIRS} more transitive blocks. The graph "
                "is too dense to enumerate; read the chain and the edge table instead"
            )
        out.append("")

    epic_orphans = [n for n in analysis["orphans"] if by_id[n]["kind"] == "epic"]
    gate_orphans = [n for n in analysis["orphans"] if by_id[n]["kind"] != "epic"]
    if epic_orphans or gate_orphans:
        out += ["### Unblocked and unblocking", ""]
    if epic_orphans:
        names = ", ".join(md(by_id[n].get("label", n)) for n in epic_orphans)
        out += [
            f"No dependencies in either direction: {names}. These are parallelisable now and "
            "are the safest work to start.",
            "",
        ]
    if gate_orphans:
        names = ", ".join(
            f"{md(by_id[n].get('label', n))} ({by_id[n]['kind']})" for n in gate_orphans
        )
        out += [
            f"**Recorded but connected to nothing: {names}.** A decision, queue or external "
            "dependency with no edge either blocks something nobody recorded, or should not be "
            "in the graph. It is not work to start — resolve which before publishing.",
            "",
        ]

    linked, total = analysis["edge_coverage"]
    out += [
        "### Link coverage",
        "",
        f"{linked} of {total} epics carry at least one edge (gates are excluded from the "
        "denominator; they exist only to be edges). A sparse graph is ambiguous — it "
        "means either genuinely independent work or dependency research that came back thin, "
        "and the two have opposite consequences. Say which one this is.",
        "",
    ]

    if analysis["warnings"]:
        out += ["### Data-quality warnings", ""]
        out += [f"- {w}" for w in analysis["warnings"]]
        out.append("")

    return "\n".join(out)


# --------------------------------------------------------------------------- #

def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.split("\n", 1)[0])
    parser.add_argument("manifest", help="JSON manifest carrying a top-level `graph` block")
    parser.add_argument("out", nargs="?", help="Markdown output path; stdout if omitted")
    args = parser.parse_args(argv)

    with open(args.manifest) as fh:
        manifest = json.load(fh)

    graph = manifest.get("graph")
    if graph is None:
        print(
            "manifest has no top-level `graph` block; nothing to draw",
            file=sys.stderr,
        )
        return 2

    try:
        analysis = analyse(graph)
    except ManifestError as exc:
        print(f"graph is unusable: {exc}", file=sys.stderr)
        return 1

    markdown = render_markdown(analysis)
    if args.out:
        with open(args.out, "w") as fh:
            fh.write(markdown + "\n")
        print(f"wrote {args.out}: {len(analysis['node_ids'])} nodes, {len(analysis['edges'])} edges")
    else:
        print(markdown)
    return 0


if __name__ == "__main__":
    sys.exit(main())
