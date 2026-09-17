import json
import re

import pytest

import build_graph


def epic(nid, weight_opt=None, weight_pess=None, label=None):
    node = {"id": nid, "label": label or nid.upper(), "kind": "epic"}
    if weight_opt is not None:
        node["weight_opt"] = weight_opt
    if weight_pess is not None:
        node["weight_pess"] = weight_pess
    return node


def gate(nid, kind, label=None):
    return {"id": nid, "label": label or nid.upper(), "kind": kind}


def edge(src, dst, type_="hard", source="agent", note=None):
    e = {"from": src, "to": dst, "type": type_, "source": source}
    if note:
        e["note"] = note
    return e


# --------------------------------------------------------------------------- #
# Fixture 1 — acyclic, with a known longest path
# --------------------------------------------------------------------------- #

ACYCLIC = {
    "nodes": [
        epic("a", 1.0, 2.0),
        epic("b", 4.0, 8.0),
        epic("c", 2.0, 3.0),
        epic("d", 0.5, 1.0),
        epic("lonely", 3.0, 5.0),
    ],
    "edges": [
        edge("a", "b"),
        edge("b", "c"),
        edge("a", "d"),
        edge("d", "c"),
    ],
}


def test_acyclic_graph_reports_no_cycles():
    analysis = build_graph.analyse(ACYCLIC)
    assert analysis["cycles"] == []
    assert analysis["excluded_from_chain"] == []


def test_longest_path_picks_the_heavy_branch_not_the_long_one():
    # a->b->c weighs 1+4+2 = 7; a->d->c weighs 1+0.5+2 = 3.5. Same node count.
    analysis = build_graph.analyse(ACYCLIC)
    assert analysis["chains"]["opt"]["path"] == ["a", "b", "c"]
    assert analysis["chains"]["opt"]["weeks"] == 7.0


def test_pessimistic_chain_uses_pessimistic_weights():
    analysis = build_graph.analyse(ACYCLIC)
    assert analysis["chains"]["pess"]["weeks"] == 13.0


def test_orphan_is_listed_and_kept_out_of_the_chain():
    analysis = build_graph.analyse(ACYCLIC)
    assert analysis["orphans"] == ["lonely"]
    assert "lonely" not in analysis["chains"]["opt"]["path"]


def test_transitive_closure_exposes_indirect_blocking():
    analysis = build_graph.analyse(ACYCLIC)
    # c is blocked by a only through b and d — no dependency row says so.
    assert "c" in analysis["closure"]["a"]
    assert "c" not in analysis["adj"]["a"]


def test_edge_coverage_counts_epics_with_any_edge():
    analysis = build_graph.analyse(ACYCLIC)
    assert analysis["edge_coverage"] == (4, 5)


# --------------------------------------------------------------------------- #
# Fixture 2 — a separate cyclic graph. A cycle has no defined longest path,
# so it cannot share a fixture with one that has an expected answer.
# --------------------------------------------------------------------------- #

CYCLIC = {
    "nodes": [
        epic("x", 1.0, 2.0),
        epic("y", 1.0, 2.0),
        epic("z", 1.0, 2.0),
        epic("downstream", 5.0, 9.0),
        epic("elsewhere", 7.0, 11.0),
        epic("elsewhere2", 2.0, 3.0),
    ],
    "edges": [
        edge("x", "y"),
        edge("y", "z"),
        edge("z", "x"),
        edge("z", "downstream"),
        edge("elsewhere", "elsewhere2"),
    ],
}


def test_cycle_is_detected():
    analysis = build_graph.analyse(CYCLIC)
    assert analysis["cycles"] == [["x", "y", "z"]]


def test_cycle_members_and_their_descendants_leave_the_chain():
    analysis = build_graph.analyse(CYCLIC)
    assert analysis["excluded_from_chain"] == ["downstream", "x", "y", "z"]


def test_component_untouched_by_the_cycle_still_gets_a_chain():
    analysis = build_graph.analyse(CYCLIC)
    assert analysis["chains"]["opt"]["path"] == ["elsewhere", "elsewhere2"]
    assert analysis["chains"]["opt"]["weeks"] == 9.0


def test_self_loop_counts_as_a_cycle():
    graph = {"nodes": [epic("a", 1.0, 1.0)], "edges": [edge("a", "a")]}
    analysis = build_graph.analyse(graph)
    assert analysis["cycles"] == [["a"]]
    assert analysis["chains"] == {}


def test_markdown_names_the_cycle_and_refuses_to_publish_a_complete_chain():
    markdown = build_graph.render_markdown(build_graph.analyse(CYCLIC))
    assert "Cycles" in markdown
    assert "x, y, z" in markdown
    assert "do not read the partial chain below as complete" in markdown


def test_a_deep_chain_does_not_blow_the_stack():
    depth = build_graph.MAX_NODES
    nodes = [epic(f"n{i}", 1.0, 1.0) for i in range(depth)]
    edges = [edge(f"n{i}", f"n{i + 1}") for i in range(depth - 1)]
    analysis = build_graph.analyse({"nodes": nodes, "edges": edges})
    assert analysis["cycles"] == []
    assert analysis["chains"]["opt"]["weeks"] == float(depth)


def test_an_oversized_graph_is_refused_rather_than_rendered():
    nodes = [epic(f"n{i}", 1.0, 1.0) for i in range(build_graph.MAX_NODES + 1)]
    errors, _ = build_graph.validate({"nodes": nodes, "edges": []})
    assert any("the cap is" in e for e in errors)
    with pytest.raises(build_graph.ManifestError):
        build_graph.analyse({"nodes": nodes, "edges": []})


def test_the_transitive_block_list_is_capped():
    depth = 120  # a chain of N yields N*(N-1)/2 - (N-1) indirect pairs
    nodes = [epic(f"n{i}", 1.0, 1.0) for i in range(depth)]
    edges = [edge(f"n{i}", f"n{i + 1}") for i in range(depth - 1)]
    markdown = build_graph.render_markdown(
        build_graph.analyse({"nodes": nodes, "edges": edges})
    )
    bullets = [ln for ln in markdown.splitlines() if " is blocked by " in ln]
    assert len(bullets) == build_graph.MAX_INDIRECT_PAIRS
    assert "too dense to enumerate" in markdown


# --------------------------------------------------------------------------- #
# Untrusted label and note text
# --------------------------------------------------------------------------- #

HOSTILE = 'Ship | it ]) <img src=x onerror=alert(1)> [click](javascript:alert(1)) *bold* `code`'


def test_a_pipe_in_a_label_cannot_shift_the_edge_table():
    graph = {
        "nodes": [epic("a", 1.0, 1.0, HOSTILE), epic("b", 1.0, 1.0)],
        "edges": [edge("a", "b")],
    }
    markdown = build_graph.render_markdown(build_graph.analyse(graph))
    row = next(ln for ln in markdown.splitlines() if ln.startswith("| ") and "Ship" in ln)
    assert row.count("|") - row.count("\\|") == 6  # 5 cells => 6 delimiters


def test_html_and_javascript_links_do_not_survive_into_markdown():
    graph = {
        "nodes": [epic("a", 1.0, 1.0, HOSTILE), epic("b", 1.0, 1.0)],
        "edges": [edge("a", "b", note=HOSTILE)],
    }
    markdown = build_graph.render_markdown(build_graph.analyse(graph))
    # Escaped is the win: the characters survive as literal text, inert to a renderer.
    assert re.search(r"(?<!\\)<img", markdown) is None
    assert re.search(r"(?<!\\)\[click\]", markdown) is None
    assert markdown.count("\\<img") == markdown.count("<img")


def test_control_characters_are_stripped_from_markdown_and_mermaid():
    label = "gate\rclick e1 \"https://evil.example/\" _blank\u2028second"
    graph = {"nodes": [epic("a", 1.0, 1.0, label)], "edges": []}
    analysis = build_graph.analyse(graph)
    assert "\r" not in build_graph.render_markdown(analysis)
    mermaid = build_graph.render_mermaid(analysis)
    assert "\r" not in mermaid and "\u2028" not in mermaid


def test_mermaid_init_directive_in_a_label_is_neutralised():
    graph = {
        "nodes": [epic("a", 1.0, 1.0, '%%{init: {"themeCSS": "x"}}%% <b>hi</b>')],
        "edges": [],
    }
    mermaid = build_graph.render_mermaid(build_graph.analyse(graph))
    assert "%%{init" not in mermaid
    assert "<b>" not in mermaid


# --------------------------------------------------------------------------- #
# Weights
# --------------------------------------------------------------------------- #

@pytest.mark.parametrize("weight,fragment", [
    ("3 weeks", "must be a number"),
    (-99.0, "negative weight"),
    (True, "must be a number"),
])
def test_a_weight_that_is_not_a_non_negative_number_is_refused(weight, fragment):
    node = {"id": "a", "label": "A", "kind": "epic", "weight_opt": weight, "weight_pess": 1.0}
    errors, _ = build_graph.validate({"nodes": [node], "edges": []})
    assert any(fragment in e for e in errors)


def test_an_implausible_weight_warns():
    graph = {"nodes": [epic("a", 5000.0, 5000.0)], "edges": []}
    _, warnings = build_graph.validate(graph)
    assert any("unit error" in w for w in warnings)


# --------------------------------------------------------------------------- #
# Fixture 3 — the chain runs through unweighted gates
# --------------------------------------------------------------------------- #

GATED = {
    "nodes": [
        gate("q1", "queue", "Code review on 3 children"),
        epic("e1", 3.0, 6.0, "Search reindex"),
        gate("d1", "decision", "Retention policy call"),
        epic("e2", 2.0, 4.0, "Consumer rollout"),
        gate("x1", "external", "Billing service (another team)"),
    ],
    "edges": [
        edge("q1", "e1", note="THE actual constraint"),
        edge("e1", "d1"),
        edge("d1", "e2"),
        edge("x1", "e2"),
    ],
}


def test_gates_contribute_no_weeks_but_are_named():
    analysis = build_graph.analyse(GATED)
    chain = analysis["chains"]["opt"]
    assert chain["path"] == ["q1", "e1", "d1", "e2"]
    assert chain["weeks"] == 5.0  # 3 + 2; the queue and the decision add nothing
    assert chain["gates"] == ["q1", "d1"]


def test_markdown_reports_epic_weeks_plus_named_gates():
    markdown = build_graph.render_markdown(build_graph.analyse(GATED))
    assert "5 engineer-weeks" in markdown
    assert "carry no estimate" in markdown
    assert "Code review on 3 children (queue)" in markdown
    assert "Retention policy call (decision)" in markdown


def test_a_gate_may_not_carry_a_weight():
    graph = {
        "nodes": [dict(gate("d1", "decision"), weight_opt=2.0), epic("e1", 1.0, 1.0)],
        "edges": [edge("d1", "e1")],
    }
    errors, _ = build_graph.validate(graph)
    assert any("Weights exist only on epic nodes" in e for e in errors)
    with pytest.raises(build_graph.ManifestError):
        build_graph.analyse(graph)


# --------------------------------------------------------------------------- #
# Soft edges, sourcing, and validation
# --------------------------------------------------------------------------- #

def test_soft_edges_are_drawn_but_never_drive_ordering():
    graph = {
        "nodes": [epic("a", 1.0, 1.0), epic("b", 9.0, 9.0)],
        "edges": [edge("a", "b", type_="soft")],
    }
    analysis = build_graph.analyse(graph)
    assert analysis["adj"]["a"] == []
    assert analysis["chains"]["opt"]["weeks"] == 9.0
    assert "-.->" in build_graph.render_mermaid(analysis)


def test_tracker_only_edges_are_flagged_as_agent_misses():
    graph = {
        "nodes": [epic("a", 1.0, 1.0), epic("b", 1.0, 1.0)],
        "edges": [edge("a", "b", source="tracker")],
    }
    markdown = build_graph.render_markdown(build_graph.analyse(graph))
    assert "found only in the tracker; the agents missed it" in markdown


@pytest.mark.parametrize(
    "graph,fragment",
    [
        ({"nodes": [epic("a", 1.0, 1.0), epic("a", 1.0, 1.0)], "edges": []}, "duplicate node id"),
        ({"nodes": [epic("a", 1.0, 1.0)], "edges": [edge("a", "ghost")]}, "unknown node"),
        ({"nodes": [{"id": "a", "kind": "saga"}], "edges": []}, "expected one of"),
        ({"nodes": [epic("a", 1.0, 1.0)], "edges": [edge("a", "a", type_="maybe")]}, "expected one of"),
    ],
)
def test_validation_rejects_malformed_graphs(graph, fragment):
    errors, _ = build_graph.validate(graph)
    assert any(fragment in e for e in errors)


def test_epic_with_no_weight_warns_rather_than_failing():
    graph = {"nodes": [epic("a")], "edges": []}
    errors, warnings = build_graph.validate(graph)
    assert errors == []
    assert sum("contributes 0 to that bound" in w for w in warnings) == 2


def test_a_weight_missing_on_only_one_bound_still_warns():
    # An "either" check passes this clean and then prices the node at 0 weeks in
    # the pessimistic chain, publishing a narrower band than the data supports.
    graph = {"nodes": [epic("a", weight_opt=4.0)], "edges": []}
    errors, warnings = build_graph.validate(graph)
    assert errors == []
    assert any("no weight_pess" in w for w in warnings)
    assert not any("no weight_opt" in w for w in warnings)


def test_transposed_bounds_are_flagged():
    graph = {"nodes": [epic("a", 9.0, 3.0)], "edges": []}
    _, warnings = build_graph.validate(graph)
    assert any("probably transposed" in w for w in warnings)


@pytest.mark.parametrize("graph,fragment", [
    ({"nodes": ["a"], "edges": []}, "not an object"),
    ({"nodes": [epic("a", 1.0, 1.0)], "edges": ["a->a"]}, "not an object"),
    ({"nodes": [{"id": 101, "label": "A", "kind": "epic", "weight_opt": 1.0}], "edges": []},
     "must be a string"),
])
def test_shapes_that_would_crash_later_are_refused_up_front(graph, fragment):
    errors, _ = build_graph.validate(graph)
    assert any(fragment in e for e in errors)


def test_a_graph_that_is_not_an_object_is_refused():
    errors, _ = build_graph.validate("hello")
    assert any("not an object" in e for e in errors)


def test_main_reports_a_malformed_graph_instead_of_a_traceback(tmp_path, capsys):
    path = tmp_path / "manifest.json"
    path.write_text(json.dumps(
        {"graph": {"nodes": [{"id": "a", "kind": "epic", "weight_opt": "3 weeks"}], "edges": []}}
    ))
    assert build_graph.main([str(path)]) == 1
    assert "must be a number" in capsys.readouterr().err


# --------------------------------------------------------------------------- #
# The behaviours a mutation survived before these existed
# --------------------------------------------------------------------------- #

def test_zero_weight_gates_stay_on_the_chain_as_a_trailing_tail():
    # Weight alone cannot distinguish e1 from e1->q1->d1; the node-count
    # tie-break is what keeps the two gates in the report.
    graph = {
        "nodes": [epic("e1", 5.0, 5.0), gate("q1", "queue"), gate("d1", "decision")],
        "edges": [edge("e1", "q1"), edge("q1", "d1")],
    }
    chain = build_graph.analyse(graph)["chains"]["opt"]
    assert chain["path"] == ["e1", "q1", "d1"]
    assert chain["gates"] == ["q1", "d1"]


def test_an_ancestor_of_a_second_cycle_is_excluded_too():
    # With one shared visited set the forward pass claims "y" and the reverse
    # pass then never reaches "x", which gets published as the whole chain.
    graph = {
        "nodes": [
            epic("a", 1.0, 1.0), epic("b", 1.0, 1.0),
            epic("c", 1.0, 1.0), epic("d", 1.0, 1.0),
            epic("y", 1.0, 1.0), epic("x", 40.0, 80.0, "Big epic X"),
        ],
        "edges": [
            edge("a", "b"), edge("b", "a"),
            edge("c", "d"), edge("d", "c"),
            edge("a", "y"), edge("y", "c"), edge("x", "y"),
        ],
    }
    analysis = build_graph.analyse(graph)
    assert analysis["cycles"] == [["a", "b"], ["c", "d"]]
    assert analysis["excluded_from_chain"] == ["a", "b", "c", "d", "x", "y"]
    assert analysis["chains"] == {}


def test_bounds_that_run_along_different_paths_are_reported_as_such():
    graph = {
        "nodes": [epic("a", 5.0, 5.0), epic("b", 1.0, 20.0), epic("c", 4.0, 1.0)],
        "edges": [edge("a", "b"), edge("a", "c")],
    }
    analysis = build_graph.analyse(graph)
    assert analysis["chains"]["opt"]["path"] == ["a", "c"]
    assert analysis["chains"]["pess"]["path"] == ["a", "b"]
    markdown = build_graph.render_markdown(analysis)
    assert "run along different paths" in markdown


def test_a_soft_edge_still_counts_as_coverage_and_removes_an_orphan():
    graph = {
        "nodes": [epic("a", 1.0, 1.0), epic("b", 1.0, 1.0)],
        "edges": [edge("a", "b", type_="soft")],
    }
    analysis = build_graph.analyse(graph)
    assert analysis["orphans"] == []
    assert analysis["edge_coverage"] == (2, 2)


def test_gates_are_not_reported_as_the_safest_work_to_start():
    graph = {
        "nodes": [epic("a", 1.0, 1.0), gate("d1", "decision", "Retention policy call")],
        "edges": [],
    }
    markdown = build_graph.render_markdown(build_graph.analyse(graph))
    safest = next(ln for ln in markdown.splitlines() if "safest work to start" in ln)
    assert "Retention policy call" not in safest
    assert "connected to nothing" in markdown


def test_a_decision_node_carries_its_owner_onto_the_chain_line():
    graph = {
        "nodes": [
            epic("e1", 3.0, 3.0),
            {"id": "d1", "label": "Retention policy call", "kind": "decision",
             "note": "owner: Legal"},
        ],
        "edges": [edge("e1", "d1")],
    }
    markdown = build_graph.render_markdown(build_graph.analyse(graph))
    assert "Retention policy call (decision — owner: Legal)" in markdown


def test_ids_that_sanitise_to_the_same_mermaid_id_stay_distinct():
    graph = {
        "nodes": [
            {"id": "ABC-101", "label": "First", "kind": "epic", "weight_opt": 1.0,
             "weight_pess": 1.0},
            {"id": "ABC.101", "label": "Second", "kind": "epic", "weight_opt": 1.0,
             "weight_pess": 1.0},
        ],
        "edges": [],
    }
    mermaid = build_graph.render_mermaid(build_graph.analyse(graph))
    declared = re.findall(r"^    (\w+)\[", mermaid, re.M)
    assert len(declared) == len(set(declared)) == 2


def test_a_hash_in_a_label_survives_into_the_diagram():
    # '#' opens a mermaid entity, so an unescaped "PR #1234; waiting" is parsed
    # as the entity '#1234;' and the ticket number vanishes from the picture.
    graph = {"nodes": [epic("a", 1.0, 1.0, "PR #1234; 6 days waiting")], "edges": []}
    mermaid = build_graph.render_mermaid(build_graph.analyse(graph))
    assert "#35;1234;" in mermaid
    assert "#1234;" not in mermaid.replace("#35;1234;", "")


def test_an_empty_graph_does_not_claim_everything_is_in_a_cycle():
    markdown = build_graph.render_markdown(build_graph.analyse({"nodes": [], "edges": []}))
    assert "The graph is empty" in markdown
    assert "inside a cycle" not in markdown


# --------------------------------------------------------------------------- #
# Degenerate inputs
# --------------------------------------------------------------------------- #

def test_single_node_says_so_rather_than_emitting_an_empty_section():
    markdown = build_graph.render_markdown(
        build_graph.analyse({"nodes": [epic("a", 2.0, 4.0)], "edges": []})
    )
    assert "no dependencies were found" in markdown
    assert "2 engineer-weeks" in markdown


def test_no_edges_across_many_nodes_does_not_read_as_good_news():
    graph = {"nodes": [epic("a", 1.0, 1.0), epic("b", 1.0, 1.0)], "edges": []}
    markdown = build_graph.render_markdown(build_graph.analyse(graph))
    assert "dependency research came back thin" in markdown


def test_mermaid_escapes_quotes_in_labels():
    graph = {
        "nodes": [{"id": "a", "label": 'the "big" one', "kind": "epic", "weight_opt": 1.0}],
        "edges": [],
    }
    mermaid = build_graph.render_mermaid(build_graph.analyse(graph))
    assert '#quot;big#quot;' in mermaid
    assert '"the "big" one"' not in mermaid


def test_mermaid_ids_survive_keys_with_punctuation():
    graph = {
        "nodes": [{"id": "ABC-101", "label": "x", "kind": "epic", "weight_opt": 1.0}],
        "edges": [],
    }
    mermaid = build_graph.render_mermaid(build_graph.analyse(graph))
    assert "ABC_101[" in mermaid


def test_main_reports_a_missing_graph_block_without_crashing(tmp_path, capsys):
    path = tmp_path / "manifest.json"
    path.write_text(json.dumps({"sheets": []}))
    assert build_graph.main([str(path)]) == 2
    assert "no top-level `graph` block" in capsys.readouterr().err


def test_main_writes_markdown(tmp_path):
    manifest = tmp_path / "manifest.json"
    manifest.write_text(json.dumps({"graph": GATED}))
    out = tmp_path / "graph.md"
    assert build_graph.main([str(manifest), str(out)]) == 0
    assert "Dependency graph" in out.read_text()
