import inspect
import json
import re
from datetime import datetime, timedelta, timezone
from pathlib import Path
from types import SimpleNamespace

import pytest

import jira_scan

FIXTURES = Path(__file__).resolve().parent / "fixtures" / "jira"


def _fixture(name):
    return (FIXTURES / name).read_text()


def _json_fixture(name):
    return json.loads(_fixture(name))


def _roster():
    return jira_scan.parse_roster(_fixture("roster.md"))


def test_parse_roster_reads_the_roster_table_with_account_ids():
    assert _roster() == [
        {"name": "Alice Example", "github": "alice", "role": "Engineering Manager",
         "jira_account_id": "acct-0001"},
        {"name": "Bob Example", "github": "bob", "role": "Software Engineer",
         "jira_account_id": None},
        {"name": "Carol Example", "github": "carol", "role": "Software Engineer",
         "jira_account_id": "acct-0003"},
    ]


def test_parse_roster_without_account_id_column_gives_none():
    markdown = (
        "## Roster\n\n"
        "| Name | GitHub Handle | Role | Notes |\n"
        "|------|--------------|------|-------|\n"
        "| Alice Example | alice | Software Engineer | |\n"
    )
    assert jira_scan.parse_roster(markdown) == [
        {"name": "Alice Example", "github": "alice", "role": "Software Engineer",
         "jira_account_id": None},
    ]


def test_parse_roster_ignores_other_tables():
    names = [person["name"] for person in _roster()]
    assert "Dana Example" not in names


def test_parse_roster_rejects_bracketed_placeholders():
    markdown = (
        "## Roster\n\n"
        "| Name | GitHub Handle | Role | Notes |\n"
        "|------|--------------|------|-------|\n"
        "| [Your Name] | [your-handle] | [Engineering Manager] | [Report author] |\n"
    )
    with pytest.raises(ValueError, match="team config not filled in"):
        jira_scan.parse_roster(markdown)


def test_parse_roster_without_a_roster_table_is_empty():
    assert jira_scan.parse_roster("# Team\n\nNo table here.\n") == []


def test_match_assignee_prefers_account_id():
    roster = _roster()
    assignee = {"accountId": "acct-0003", "displayName": "Alice Example", "active": True}
    assert jira_scan.match_assignee(assignee, roster)["name"] == "Carol Example"


def test_match_assignee_falls_back_to_case_insensitive_display_name():
    assignee = {"accountId": "acct-other", "displayName": "bob EXAMPLE", "active": True}
    assert jira_scan.match_assignee(assignee, _roster())["name"] == "Bob Example"


def test_match_assignee_unknown_returns_none():
    assignee = {"accountId": "acct-9999", "displayName": "Erin Outsider", "active": True}
    assert jira_scan.match_assignee(assignee, _roster()) is None


def test_match_assignee_none_returns_none():
    assert jira_scan.match_assignee(None, _roster()) is None


def _epic_description():
    return _json_fixture("epic_view.json")["fields"]["description"]


def test_adf_text_joins_text_nodes_across_blocks():
    assert jira_scan.adf_text(_epic_description(), 500) == (
        "Build the synthetic widget. Spec and the sheet here Design again"
    )


def test_adf_text_cuts_to_limit_with_ellipsis():
    text = jira_scan.adf_text(_epic_description(), 20)
    assert len(text) == 20
    assert text.endswith("…")
    assert text.startswith("Build the synthetic")


def test_adf_text_at_exact_limit_is_not_cut():
    node = {"type": "doc", "content": [{"type": "paragraph", "content": [
        {"type": "text", "text": "exactly"}]}]}
    assert jira_scan.adf_text(node, 7) == "exactly"


def test_adf_text_none_is_empty():
    assert jira_scan.adf_text(None, 200) == ""


def test_adf_links_collects_link_marks_and_inline_cards_in_order_without_duplicates():
    assert jira_scan.adf_links(_epic_description()) == [
        "https://example.atlassian.net/wiki/spaces/X/pages/1",
        "https://docs.google.com/document/d/abc",
        "https://docs.google.com/spreadsheets/d/def",
        "https://www.figma.com/file/ghi",
    ]


def test_adf_links_none_is_empty():
    assert jira_scan.adf_links(None) == []


@pytest.mark.parametrize(
    "url,expected",
    [
        ("https://x.atlassian.net/wiki/spaces/ENG/pages/123", "wiki"),
        ("https://docs.google.com/document/d/abc/edit", "doc"),
        ("https://docs.google.com/spreadsheets/d/abc/edit", "sheet"),
        ("https://www.figma.com/file/abc/Design", "design"),
        ("https://figma.com/design/abc", "design"),
        ("https://github.com/acme/repo/pull/1", None),
        ("https://x.atlassian.net/browse/E-1", None),
        ("not a url", None),
    ],
)
def test_classify_doc_link(url, expected):
    assert jira_scan.classify_doc_link(url) == expected


def _issue(key, category, status_name=None, assignee=None, summary=None, **extra):
    names = {"new": "To Do", "indeterminate": "In Progress 🛠️", "done": "Done ✅"}
    fields = {
        "summary": summary or f"Synthetic {key}",
        "status": {"name": status_name or names[category], "statusCategory": {"key": category}},
        "assignee": assignee,
    }
    fields.update(extra.pop("fields", {}))
    return {"key": key, "fields": fields, **extra}


def test_epic_progress_counts_by_status_category():
    children = ([_issue(f"C-{n}", "new") for n in range(3)]
                + [_issue(f"C-{n}", "indeterminate") for n in range(3, 5)]
                + [_issue(f"C-{n}", "done") for n in range(5, 10)])
    assert jira_scan.epic_progress(children) == {
        "total": 10, "done": 5, "in_progress": 2, "pct": 50,
    }


def test_epic_progress_empty_has_no_percentage():
    assert jira_scan.epic_progress([]) == {"total": 0, "done": 0, "in_progress": 0, "pct": None}


def test_epic_progress_rounds_the_percentage():
    children = [_issue("C-1", "done"), _issue("C-2", "new"), _issue("C-3", "new")]
    assert jira_scan.epic_progress(children)["pct"] == 33


def test_epic_progress_never_reads_resolution():
    children = [_issue("C-1", "done", fields={"resolution": {"name": "Won't Do"}})]
    assert jira_scan.epic_progress(children)["done"] == 1
    assert "resolution" not in inspect.getsource(jira_scan.epic_progress)


def test_child_to_epic_maps_each_child_to_its_epic():
    assert jira_scan.child_to_epic({"E-1": [{"key": "C-1"}], "E-2": [{"key": "C-2"}]}) == {
        "C-1": "E-1", "C-2": "E-2",
    }


def test_child_to_epic_empty():
    assert jira_scan.child_to_epic({}) == {}


def _grouped():
    return jira_scan.group_by_person(_json_fixture("window_search.json"), _roster())


def _keys(entries):
    return [entry["key"] for entry in entries]


def test_group_by_person_sorts_issues_into_completed_in_progress_and_stuck():
    people, _ = _grouped()
    assert _keys(people["Alice Example"]["completed"]) == ["C-1"]
    assert _keys(people["Alice Example"]["to_do"]) == ["C-5"]
    assert _keys(people["Bob Example"]["in_progress"]) == ["C-2"]
    assert _keys(people["Carol Example"]["stuck"]) == ["C-3", "C-4"]
    assert people["Carol Example"]["in_progress"] == []


def test_group_by_person_entries_carry_key_summary_and_status():
    people, _ = _grouped()
    assert people["Alice Example"]["completed"] == [
        {"key": "C-1", "summary": "Synthetic done task", "status": "Done ✅"},
    ]


def test_group_by_person_blocked_match_is_case_insensitive():
    issue = _issue("C-1", "indeterminate", status_name="BLOCKED by vendor",
                   assignee={"accountId": "acct-0001", "displayName": "Alice Example"})
    people, _ = jira_scan.group_by_person([issue], _roster())
    assert _keys(people["Alice Example"]["stuck"]) == ["C-1"]


def test_group_by_person_lists_every_roster_member():
    people, _ = _grouped()
    assert list(people) == ["Alice Example", "Bob Example", "Carol Example"]


def test_group_by_person_returns_unmatched_assignees_once_each():
    _, unmatched = _grouped()
    assert unmatched == ["Erin Outsider"]


def test_group_by_person_skips_unassigned_issues():
    people, unmatched = _grouped()
    every_key = [entry["key"] for person in people.values()
                 for group in person.values() for entry in group]
    assert "C-7" not in every_key
    assert unmatched == ["Erin Outsider"]


def _comment(author, created, text):
    return {"author": {"accountId": f"acct-{author.lower()}", "displayName": author},
            "created": created,
            "body": {"type": "doc", "content": [{"type": "paragraph", "content": [
                {"type": "text", "text": text}]}]}}


def _days_ago(days):
    moment = datetime.now(timezone.utc) - timedelta(days=days, minutes=5)
    return moment.strftime("%Y-%m-%dT%H:%M:%S.000+0000")


def test_question_candidates_flags_a_newest_comment_with_a_question():
    comments = [
        _comment("Alice Example", _days_ago(5), "Started."),
        _comment("Carol Example", _days_ago(3), "Who owns the rollout plan? It blocks the launch."),
    ]
    assert jira_scan.question_candidates(comments, 20) == [
        {"author": "Carol Example", "created": comments[1]["created"], "age_days": 3,
         "excerpt": "Who owns the rollou…"},
    ]


def test_question_candidates_uses_the_newest_comment_whatever_the_list_order():
    comments = [
        _comment("Carol Example", _days_ago(1), "Done, no question here."),
        _comment("Alice Example", _days_ago(4), "Is this ready?"),
    ]
    assert jira_scan.question_candidates(comments, 300) == []


def test_question_candidates_answered_question_is_not_a_candidate():
    comments = [
        _comment("Alice Example", _days_ago(4), "Is this ready?"),
        _comment("Bob Example", _days_ago(2), "Yes, merged."),
    ]
    assert jira_scan.question_candidates(comments, 300) == []


def test_question_candidates_reads_the_fixture_comment_shape():
    comments = _json_fixture("epic_view.json")["fields"]["comment"]["comments"]
    candidates = jira_scan.question_candidates(comments, 300)
    assert [(c["author"], c["excerpt"]) for c in candidates] == [
        ("Carol Example", "Who owns the rollout plan?"),
    ]
    assert candidates[0]["age_days"] >= 0


def test_question_candidates_without_comments_is_empty():
    assert jira_scan.question_candidates([], 300) == []


COUNT_JQL = "parent = {EPIC} AND (resolution is EMPTY OR resolution not in (\"Won't Do\", Declined, Duplicate))"


def _epic(key, summary, children=(), assignee=None, description="Synthetic description.",
          done=1, total=2):
    return {
        "key": key,
        "summary": summary,
        "status": "In Progress 🛠️",
        "priority": "High",
        "parent": {"key": "I-1", "summary": "Synthetic initiative"},
        "assignee": assignee,
        "description": description,
        "progress": {"total": total, "done": done, "in_progress": total - done,
                     "pct": round(done * 100 / total) if total else None},
        "jql": {"total": COUNT_JQL.replace("{EPIC}", key),
                "done": COUNT_JQL.replace("{EPIC}", key) + " AND statusCategory = Done",
                "in_progress": COUNT_JQL.replace("{EPIC}", key) + ' AND statusCategory = "In Progress"'},
        "children": list(children),
    }


def _child(key, summary, status="In Progress 🛠️", assignee=None, category="indeterminate"):
    return {"key": key, "summary": summary, "status": status, "assignee": assignee,
            "category": category}


def _data():
    people, unmatched = _grouped()
    return {
        "window": {"since": "2026-09-28", "until": "2026-10-05", "keys": ["DL"]},
        "count_jql": COUNT_JQL,
        "epics": [
            _epic("E-1", "Synthetic epic one",
                  children=[_child("C-1", "Synthetic done task", "Done ✅", "Alice Example", "done"),
                            _child("C-3", "Synthetic blocked task", "Blocked", "Carol Example")],
                  assignee={"name": "Carol Example", "active": True}),
            _epic("E-2", "Synthetic epic two",
                  children=[_child("C-2", "Synthetic active task", assignee="Bob Example")],
                  assignee={"name": "Former Person", "active": False}, description=""),
        ],
        "child_to_epic": {"C-1": "E-1", "C-3": "E-1", "C-2": "E-2"},
        "people": people,
        "unmatched": unmatched,
        "flagged": [
            {"key": "C-3", "summary": "Synthetic blocked task", "status": "Blocked",
             "assignee": "Carol Example", "reason": "blocked",
             "comments": [{"author": "Alice Example", "created": "2026-09-30T09:00:00.000+0000",
                           "excerpt": "Waiting on the vendor."}]},
            {"key": "C-2", "summary": "Synthetic active task", "status": "In Progress 🛠️",
             "assignee": "Bob Example", "reason": "stalled", "comments": []},
        ],
        "questions": [
            {"key": "C-3", "author": "Carol Example", "created": "2026-10-01T09:00:00.000+0000",
             "age_days": 4, "excerpt": "Who owns the rollout plan?"},
        ],
        "doc_links": [
            {"url": "https://example.atlassian.net/wiki/spaces/X/pages/1", "epic": "E-1", "kind": "wiki"},
            {"url": "https://docs.google.com/document/d/abc", "epic": "E-2", "kind": "doc"},
        ],
        "not_started": [
            {"key": "E-7", "summary": "Synthetic idle epic", "created": "2026-05-01T00:00:00.000+0000",
             "age_days": 157, "assignee": "Bob Example"},
        ],
        "unassigned": [
            {"key": "C-5", "type": "Bug", "status": "To Do", "priority": "High",
             "summary": "Synthetic unassigned defect"},
        ],
        "failures": [],
        "omitted": {"issues": 0, "comments": 0},
    }


def _render(data=None, scope="team", word_limit=5000, person=None):
    return jira_scan.render_digest(data or _data(), scope, word_limit, person)


def _headings(text):
    return [line for line in text.splitlines() if line.startswith("## ")]


def _section_text(text, heading):
    start = text.index(heading + "\n")
    rest = text[start + len(heading) + 1:]
    end = rest.find("\n## ")
    return rest if end == -1 else rest[:end]


def test_render_digest_doc_links_heading():
    text = _render()
    lines = text.splitlines()
    start = lines.index("## Doc links")
    links = [line for line in lines[start + 1:] if line.startswith("- ")][:2]
    assert links == [
        "- https://example.atlassian.net/wiki/spaces/X/pages/1 (E-1, wiki)",
        "- https://docs.google.com/document/d/abc (E-2, doc)",
    ]


def test_render_digest_team_scope_has_every_section_in_order():
    assert _headings(_render()) == [
        "## Active epics",
        "## Flagged",
        "## Unassigned",
        "## Question candidates",
        "## Unmatched assignees",
        "## Doc links",
        "## By person",
        "## Not started",
    ]


def test_render_digest_epic_block_is_compact():
    block = _section_text(_render(), "## Active epics")
    assert "### E-1 Synthetic epic one" in block
    assert "Priority: High" in block
    assert "Parent: I-1 Synthetic initiative" in block
    assert "Assignee: Carol Example" in block
    assert "1/2 done, 1 in progress, 50%" in block
    assert "Synthetic description." in block


def test_render_digest_flags_inactive_assignee_and_missing_description():
    block = _section_text(_render(), "## Active epics")
    e2 = block[block.index("### E-2"):]
    assert "Former Person (inactive)" in e2
    assert "no description on the ticket" in e2


def test_render_digest_lists_flagged_questions_and_unmatched():
    text = _render()
    flagged = _section_text(text, "## Flagged")
    assert "C-3" in flagged and "blocked" in flagged and "Waiting on the vendor." in flagged
    assert "C-3 Synthetic blocked task — Blocked · Carol Example · blocked · epic E-1" in flagged
    assert "C-2" in flagged and "stalled" in flagged
    questions = _section_text(text, "## Question candidates")
    assert "C-3" in questions and "Carol Example" in questions and "4 days" in questions
    assert "Who owns the rollout plan?" in questions
    assert "- Erin Outsider" in _section_text(text, "## Unmatched assignees")


def test_render_digest_by_person_and_not_started_detail():
    text = _render()
    by_person = _section_text(text, "## By person")
    assert "### Alice Example" in by_person
    assert "C-1 Synthetic done task" in by_person
    assert "C-4 Synthetic stalled task" in by_person
    assert "E-7 Synthetic idle epic" in _section_text(text, "## Not started")


def test_render_digest_states_count_jql_template_once():
    text = _render()
    assert text.count("{EPIC}") == 1
    assert text.count(COUNT_JQL) == 1
    assert "parent = E-1" not in text
    assert "parent = E-2" not in text


def test_render_digest_has_no_raw_json():
    data = _data()
    data["flagged"][0]["comments"][0]["excerpt"] = 'payload was {"id": 1}'
    data["questions"][0]["excerpt"] = 'see {"a":2}?'
    assert '{"' not in _render(data)


def test_render_digest_ends_with_omitted_line():
    assert _render().rstrip("\n").splitlines()[-1] == (
        "Omitted: 0 issues, 0 not-started epics, 0 comments (word limit 5000)"
    )


def test_render_digest_carries_collector_omissions_and_renders_only_the_newest_comment():
    data = _data()
    data["omitted"] = {"issues": 13, "comments": 2}
    data["flagged"][0]["comments"] = [
        {"author": "Alice Example", "created": f"2026-09-2{n}T09:00:00.000+0000",
         "excerpt": f"note {n}"} for n in range(5)
    ]
    text = _render(data)
    assert "note 4" in text
    assert not any(f"note {n}" in text for n in range(4))
    assert text.rstrip("\n").splitlines()[-1] == (
        "Omitted: 13 issues, 0 not-started epics, 6 comments (word limit 5000)"
    )


def test_render_digest_reports_failures_first():
    data = _data()
    data["failures"] = ["E-2"]
    text = _render(data)
    assert _headings(text)[0] == "## Not measured"
    assert "- E-2" in _section_text(text, "## Not measured")


def test_render_digest_person_scope_shows_only_that_person():
    text = _render(scope="person", person="Carol Example")
    headings = _headings(text)
    assert "## By person" not in headings
    assert headings == [
        "## Active epics",
        "## Flagged",
        "## Unassigned",
        "## Question candidates",
        "## Unmatched assignees",
        "## Doc links",
        "## Issues",
        "## Not started",
    ]
    issues = _section_text(text, "## Issues")
    assert "C-3" in issues and "C-4" in issues
    assert "C-1" not in issues and "C-2" not in issues
    assert "### E-1" in text and "### E-2" not in text
    assert "C-2" not in _section_text(text, "## Flagged")
    assert "(E-2, doc)" not in text
    assert "E-7" not in text


def test_render_digest_person_scope_is_case_insensitive():
    text = _render(scope="person", person="carol example")
    assert "C-3" in _section_text(text, "## Issues")


def test_render_digest_person_scope_requires_a_person():
    with pytest.raises(ValueError):
        _render(scope="person")


def test_render_digest_epic_scope_shows_one_epic_and_its_children():
    data = _data()
    data["epic"] = "E-1"
    text = _render(data, scope="epic")
    assert "## By person" not in _headings(text)
    assert "## Not started" not in _headings(text)
    assert "### E-1" in text and "### E-2" not in text
    children = _section_text(text, "## Children")
    assert "C-1 Synthetic done task" in children
    assert "C-3 Synthetic blocked task" in children
    assert "C-2" not in text
    assert "(E-1, wiki)" in text and "(E-2, doc)" not in text


def test_render_digest_rejects_unknown_scope():
    with pytest.raises(ValueError):
        _render(scope="galaxy")


def _big_data():
    roster = [{"name": f"Person {n}", "github": f"p{n}", "role": "Software Engineer",
               "jira_account_id": f"acct-{n:04d}"} for n in range(8)]
    issues = []
    for n in range(160):
        category = ("done", "indeterminate", "new")[n % 3]
        issues.append(_issue(f"C-{n}", category,
                             summary=f"Synthetic issue number {n} with a fairly long summary text",
                             assignee={"accountId": f"acct-{n % 8:04d}", "displayName": f"Person {n % 8}"}))
    people, unmatched = jira_scan.group_by_person(issues, roster)
    epics = [_epic(f"E-{n}", f"Synthetic epic {n}", description="Short synthetic description.")
             for n in range(30)]
    return {
        "window": {"since": "2026-09-28", "until": "2026-10-05", "keys": ["DL"]},
        "count_jql": COUNT_JQL,
        "epics": epics,
        "child_to_epic": {f"C-{n}": f"E-{n % 30}" for n in range(160)},
        "people": people,
        "unmatched": unmatched + ["Outside Person"],
        "flagged": [{"key": f"C-{n}", "summary": f"Synthetic issue number {n}", "status": "Blocked",
                     "assignee": f"Person {n % 8}", "reason": "blocked",
                     "comments": [{"author": "Person 1", "created": "2026-10-01T00:00:00.000+0000",
                                   "excerpt": "Still waiting."}]} for n in range(5)],
        "questions": [{"key": "C-1", "author": "Person 1", "created": "2026-10-01T00:00:00.000+0000",
                       "age_days": 4, "excerpt": "Any update?"}],
        "doc_links": [{"url": f"https://example.atlassian.net/wiki/spaces/X/pages/{n}",
                       "epic": f"E-{n}", "kind": "wiki"} for n in range(6)],
        "not_started": [{"key": f"N-{n}", "summary": f"Synthetic idle epic {n}",
                         "created": "2026-01-01T00:00:00.000+0000", "age_days": 300 - n,
                         "assignee": None} for n in range(25)],
        "failures": [],
        "omitted": {"issues": 0, "comments": 0},
    }


NEVER_CUT = ("## Active epics", "## Flagged", "## Unassigned", "## Question candidates",
             "## Unmatched assignees", "## Doc links")


def _words(text):
    return len(text.split())


def test_render_digest_word_cap_collapses_by_person_and_keeps_never_cut_sections():
    data = _big_data()
    limit = 900
    uncapped = _render(data, word_limit=10 ** 6)
    text = _render(data, word_limit=limit)
    assert _words(uncapped) > limit
    for n in range(30):
        assert f"### E-{n} " in text
    for heading in NEVER_CUT:
        assert _section_text(text, heading) == _section_text(uncapped, heading)
    assert _words(_section_text(text, "## By person")) < _words(_section_text(uncapped, "## By person"))
    by_person = _section_text(text, "## By person")
    assert "fairly long summary" not in by_person
    assert set(re.findall(r"\bC-\d+\b", by_person)) == {f"C-{n}" for n in range(160)}
    never_words = sum(_words(heading) + _words(_section_text(text, heading)) for heading in NEVER_CUT)
    assert _words(text) - never_words <= limit
    assert text.rstrip("\n").splitlines()[-1] == (
        "Omitted: 160 issues, 15 not-started epics, 0 comments (word limit 900)"
    )


def test_render_digest_not_started_keeps_the_ten_oldest_when_collapsed():
    data = _big_data()
    text = _render(data, word_limit=900)
    not_started = _section_text(text, "## Not started")
    for n in range(10):
        assert f"N-{n} " in not_started
    assert "N-10 " not in not_started
    assert "15 more" in not_started


def test_render_digest_cuts_by_person_before_not_started():
    data = _big_data()
    uncapped = _render(data, word_limit=10 ** 6)
    by_person_words = _words(_section_text(uncapped, "## By person"))
    limit = _words(uncapped) - by_person_words // 4
    text = _render(data, word_limit=limit)
    assert "fairly long summary" not in _section_text(text, "## By person")
    assert "N-24 " in _section_text(text, "## Not started")
    assert text.rstrip("\n").splitlines()[-1].startswith("Omitted: 160 issues, 0 not-started epics")


def _flagged_items(number, comments_each):
    return [{"key": f"F-{n}", "summary": f"Synthetic flagged {n}", "status": "Blocked",
             "assignee": "Alice Example", "reason": "blocked",
             "comments": [{"author": "Bob Example", "created": f"2026-09-2{c}T09:00:00.000+0000",
                           "excerpt": f"excerpt {n}-{c}"} for c in range(comments_each)]}
            for n in range(number)]


def test_render_digest_flagged_keeps_every_line_but_excerpts_only_the_first_fifteen():
    data = _data()
    data["flagged"] = _flagged_items(20, 3)
    flagged = _section_text(_render(data), "## Flagged")
    item_lines = [line for line in flagged.splitlines() if line.startswith("- F-")]
    excerpt_lines = [line for line in flagged.splitlines() if line.startswith("  - ")]
    assert len(item_lines) == 20
    assert len(excerpt_lines) == 15
    assert all("excerpt" in line for line in excerpt_lines)
    assert _render(data).rstrip("\n").splitlines()[-1] == (
        "Omitted: 0 issues, 0 not-started epics, 45 comments (word limit 5000)"
    )


def test_render_digest_flagged_excerpts_follow_the_given_order_and_use_the_newest_comment():
    data = _data()
    data["flagged"] = _flagged_items(20, 3)
    flagged = _section_text(_render(data), "## Flagged")
    excerpts = [line for line in flagged.splitlines() if line.startswith("  - ")]
    assert [line.split(": ", 1)[1] for line in excerpts] == [f"excerpt {n}-2" for n in range(15)]


def test_render_digest_flagged_excerpt_limit_is_a_named_constant():
    assert jira_scan.FLAGGED_WITH_EXCERPTS == 15


def test_render_digest_flagged_line_shows_the_age():
    data = _data()
    data["flagged"][0]["age_days"] = 12
    flagged = _section_text(_render(data), "## Flagged")
    assert ("- C-3 Synthetic blocked task — Blocked · Carol Example · blocked · "
            "12 days in status category · epic E-1") in flagged


def test_render_digest_unassigned_section_sits_right_after_flagged():
    headings = _headings(_render())
    assert headings.index("## Unassigned") == headings.index("## Flagged") + 1


def test_render_digest_unassigned_line_shows_key_type_status_priority_and_trimmed_summary():
    data = _data()
    data["unassigned"] = [{"key": "C-6", "type": "Story", "status": "In Progress 🛠️",
                           "priority": "Medium", "summary": "word " * 40}]
    lines = _section_text(_render(data), "## Unassigned").strip().splitlines()
    assert len(lines) == 1
    prefix = "- C-6 · Story · In Progress 🛠️ · Medium · "
    assert lines[0].startswith(prefix)
    summary = lines[0][len(prefix):]
    assert len(summary) == 80 and summary.endswith("…")


def test_render_digest_unassigned_is_never_cut():
    data = _big_data()
    data["unassigned"] = [{"key": f"U-{n}", "type": "Bug", "status": "To Do", "priority": "High",
                           "summary": f"Synthetic defect {n}"} for n in range(12)]
    uncapped = _render(data, word_limit=10 ** 6)
    text = _render(data, word_limit=900)
    assert _section_text(text, "## Unassigned") == _section_text(uncapped, "## Unassigned")
    assert "U-11" in _section_text(text, "## Unassigned")


def test_render_digest_without_unassigned_says_none():
    data = _data()
    data["unassigned"] = []
    assert _section_text(_render(data), "## Unassigned").strip() == "None."


def _mixed_children():
    return [
        _child("C-30", "Synthetic queued later", "To Do", None, "new"),
        _child("C-21", "Synthetic building", "In Progress 🛠️", "Bob Example"),
        _child("C-40", "Synthetic finished", "Done ✅", "Alice Example", "done"),
        _child("C-12", "Synthetic in review", "Code Review 🔍", "Carol Example"),
        _child("C-9", "Synthetic building early", "In Progress 🛠️", None),
        _child("C-3", "Synthetic queued", "To Do", "Alice Example", "new"),
        _child("C-41", "Synthetic also finished", "Done ✅", "Bob Example", "done"),
        _child("C-15", "Synthetic peer REVIEW", "Peer Review", "Bob Example"),
    ]


def test_open_children_orders_review_then_in_progress_then_to_do_and_drops_done():
    assert [child["key"] for child in jira_scan.open_children(_mixed_children())] == [
        "C-12", "C-15", "C-9", "C-21", "C-3", "C-30"]


def test_render_digest_team_epic_block_lists_four_open_children_and_the_rest_as_a_count():
    data = _data()
    data["epics"][0]["children"] = _mixed_children()
    block = _section_text(_render(data), "## Active epics")
    e1 = block[block.index("### E-1"):block.index("### E-2")]
    lines = e1.splitlines()
    start = lines.index("- Open children:")
    assert lines[start + 1:start + 6] == [
        "  - C-12 Code Review 🔍 · Carol Example · Synthetic in review",
        "  - C-15 Peer Review · Bob Example · Synthetic peer REVIEW",
        "  - C-9 In Progress 🛠️ · unassigned · Synthetic building early",
        "  - C-21 In Progress 🛠️ · Bob Example · Synthetic building",
        "  - +2 more open",
    ]
    assert "C-40" not in e1 and "C-41" not in e1


def test_render_digest_open_child_summary_is_trimmed_to_80_characters():
    data = _data()
    data["epics"][0]["children"] = [_child("C-2", "long " * 40, "In Progress 🛠️", "Bob Example")]
    block = _section_text(_render(data), "## Active epics")
    line = next(line for line in block.splitlines() if line.startswith("  - C-2 "))
    summary = line.split(" · ")[-1]
    assert len(summary) == 80 and summary.endswith("…")


def test_render_digest_epic_without_open_children_says_none():
    data = _data()
    data["epics"][0]["children"] = [_child("C-1", "Synthetic done task", "Done ✅", "Alice Example",
                                           "done")]
    block = _section_text(_render(data), "## Active epics")
    e1 = block[block.index("### E-1"):block.index("### E-2")]
    assert "- Open children: none" in e1
    assert "C-1" not in e1


def test_render_digest_open_children_appear_only_in_team_scope():
    data = _data()
    data["epic"] = "E-1"
    epic_text = _render(data, scope="epic")
    person_text = _render(scope="person", person="Carol Example")
    assert "Open children" not in epic_text
    assert "Open children" not in person_text


def _raw(key, category, issuetype, assignee=None, status_name=None):
    return _issue(key, category, status_name=status_name, assignee=assignee,
                  fields={"issuetype": {"name": issuetype}, "priority": {"name": "High"}})


def test_unassigned_issues_keeps_unassigned_bugs_and_unassigned_in_progress_work():
    issues = [
        _raw("U-1", "new", "Bug"),
        _raw("U-2", "indeterminate", "Story"),
        _raw("U-3", "new", "Story"),
        _raw("U-4", "new", "Bug", assignee={"accountId": "acct-0001", "displayName": "Alice Example"}),
        _raw("U-5", "new", "BUG"),
    ]
    assert jira_scan.unassigned_issues(issues) == [
        {"key": "U-1", "type": "Bug", "status": "To Do", "priority": "High",
         "summary": "Synthetic U-1"},
        {"key": "U-2", "type": "Story", "status": "In Progress 🛠️", "priority": "High",
         "summary": "Synthetic U-2"},
        {"key": "U-5", "type": "BUG", "status": "To Do", "priority": "High",
         "summary": "Synthetic U-5"},
    ]


def test_unassigned_issues_dedupes_by_key_and_skips_done_items_and_epics():
    issues = [
        _raw("U-1", "new", "Bug"),
        _raw("U-1", "new", "Bug"),
        _raw("U-6", "done", "Bug"),
        _raw("U-7", "indeterminate", "Epic"),
    ]
    assert [item["key"] for item in jira_scan.unassigned_issues(issues)] == ["U-1"]


BANNER ="A newer version of acli is available. You are running an outdated version (1.3.14)."


def _completed(stdout="", stderr="", returncode=0):
    return SimpleNamespace(stdout=stdout, stderr=stderr, returncode=returncode)


def _runner(handler, calls=None):
    def run(cmd, **kwargs):
        if calls is not None:
            calls.append(list(cmd))
        result = handler(list(cmd))
        if isinstance(result, SimpleNamespace):
            return result
        return _completed(json.dumps(result))
    return run


def test_run_acli_strips_the_outdated_version_banner():
    runner = _runner(lambda cmd: _completed(BANNER + "\n" + json.dumps([{"key": "C-1"}])))
    assert jira_scan.run_acli(["jira", "workitem", "search"], runner=runner) == [{"key": "C-1"}]


def test_run_acli_prefixes_the_acli_binary():
    calls = []
    jira_scan.run_acli(["jira", "workitem", "view", "C-1"], runner=_runner(lambda cmd: {}, calls))
    assert calls == [["acli", "jira", "workitem", "view", "C-1"]]


def test_run_acli_raises_with_stderr_on_non_zero_exit():
    runner = _runner(lambda cmd: _completed(stderr="✗ Error: field 'created' is not allowed",
                                            returncode=1))
    with pytest.raises(jira_scan.AcliError, match="field 'created' is not allowed"):
        jira_scan.run_acli(["jira", "workitem", "search"], runner=runner)


def test_run_acli_turns_a_missing_binary_into_acli_error():
    def runner(cmd, **kwargs):
        raise FileNotFoundError("acli")
    with pytest.raises(jira_scan.AcliError, match="not found"):
        jira_scan.run_acli(["jira", "auth", "status"], runner=runner)


def test_run_acli_names_the_rate_limit_on_http_429():
    runner = _runner(lambda cmd: _completed(stderr="request failed: 429 Too Many Requests",
                                            returncode=1))
    with pytest.raises(jira_scan.AcliError, match="rate limit"):
        jira_scan.run_acli(["jira", "workitem", "search"], runner=runner)


def test_run_acli_treats_429_as_an_error_even_on_exit_zero():
    runner = _runner(lambda cmd: _completed(stdout="[]", stderr="HTTP 429 retry later"))
    with pytest.raises(jira_scan.AcliError, match="rate limit"):
        jira_scan.run_acli(["jira", "workitem", "search"], runner=runner)


def test_search_command_line_paginates_as_json_with_the_exact_fields():
    calls = []
    result = jira_scan.search("project in (DL)", "key,summary,status,assignee,issuetype,priority",
                              runner=_runner(lambda cmd: [{"key": "C-1"}], calls))
    assert result == [{"key": "C-1"}]
    line = " ".join(calls[0])
    assert line.startswith("acli jira workitem search ")
    assert "--paginate --json" in line
    assert calls[0][calls[0].index("--jql") + 1] == "project in (DL)"
    assert calls[0][calls[0].index("--fields") + 1] == "key,summary,status,assignee,issuetype,priority"


def test_search_with_no_results_is_an_empty_list():
    assert jira_scan.search("x", "key,summary", runner=_runner(lambda cmd: None)) == []
    assert jira_scan.search("x", "key,summary", runner=_runner(lambda cmd: [None])) == []


def test_view_command_line_asks_for_json_with_the_exact_fields():
    calls = []
    result = jira_scan.view("E-1", "parent", runner=_runner(lambda cmd: {"key": "E-1"}, calls))
    assert result == {"key": "E-1"}
    assert calls == [["acli", "jira", "workitem", "view", "E-1", "--fields", "parent", "--json"]]


def test_count_reads_the_number_from_the_count_line():
    calls = []
    runner = _runner(lambda cmd: _completed(BANNER + "\n✓ Number of work items in the search: 43\n"),
                     calls)
    assert jira_scan.count("parent = E-1 AND statusCategory = Done", runner=runner) == 43
    assert calls == [["acli", "jira", "workitem", "search", "--jql",
                      "parent = E-1 AND statusCategory = Done", "--count"]]


def test_count_without_a_number_raises():
    with pytest.raises(jira_scan.AcliError, match="count"):
        jira_scan.count("x", runner=_runner(lambda cmd: _completed("nothing here")))


ITEM_FIELDS = "key,summary,status,assignee,issuetype,priority"
EPIC_FIELDS = "key,summary,status,assignee,description,priority"
EPIC_VIEW_FIELDS = "summary,description,priority,parent,assignee,status,created,updated,comment"
NOT_STARTED_VIEW_FIELDS = "created,summary,description,assignee,priority"
FLAGGED_VIEW_FIELDS = "comment,statuscategorychangedate"
REJECTED_SEARCH_FIELDS = ("created", "parent", "resolution", "updated", "comment")
ALICE = {"accountId": "acct-0001", "displayName": "Alice Example", "active": True}
BOB = {"accountId": "acct-other", "displayName": "Bob Example", "active": True}
CAROL = {"accountId": "acct-0003", "displayName": "Carol Example", "active": True}
ERIN = {"accountId": "acct-9999", "displayName": "Erin Outsider", "active": True}
EXCLUDED_JQL = '("Won\'t Do", "Declined", "Duplicate")'


def _cfg(**overrides):
    cfg = {"keys": ["DL"], "since": "2026-09-28", "until": "2026-10-05",
           "excluded": ["Won't Do", "Declined", "Duplicate"], "roster": _roster(),
           "max_flagged": 60, "scope": "team", "person": None, "epic": None}
    cfg.update(overrides)
    return cfg


def _task(key, category, assignee=None, status_name=None, issuetype="Task"):
    return _issue(key, category, status_name=status_name, assignee=assignee,
                  fields={"issuetype": {"name": issuetype}, "priority": {"name": "Medium"}})


def _parent_view(key, parent_key=None, parent_type="Epic"):
    fields = {}
    if parent_key:
        fields["parent"] = {"key": parent_key, "fields": {
            "summary": f"Synthetic {parent_key}", "issuetype": {"name": parent_type}}}
    return {"key": key, "fields": fields}


def _epic_view(key, summary, category="indeterminate", assignee=None, description=None,
               comments=(), created="2026-08-01T10:00:00.000+0000"):
    names = {"new": "To Do", "indeterminate": "In Progress 🛠️", "done": "Done ✅"}
    return {"key": key, "fields": {
        "summary": summary,
        "status": {"name": names[category], "statusCategory": {"key": category}},
        "priority": {"name": "Medium"},
        "assignee": assignee,
        "created": created,
        "description": description,
        "comment": {"comments": list(comments)},
    }}


def _count_out(number):
    return _completed(f"✓ Number of work items in the search: {number}\n")


def _jql(cmd):
    return cmd[cmd.index("--jql") + 1]


def _fields(cmd):
    return cmd[cmd.index("--fields") + 1]


def _world(fail=(), unassigned=False):
    window = [
        _task("C-1", "done", ALICE),
        _task("C-2", "indeterminate", CAROL),
        _task("C-3", "indeterminate", CAROL, status_name="Blocked"),
        _task("C-8", "indeterminate", ALICE),
        _task("C-9", "indeterminate", BOB),
        _task("C-10", "new", ERIN),
        _task("E-1", "indeterminate", CAROL, issuetype="Epic"),
    ]
    epics = [_task("E-1", "indeterminate", CAROL, issuetype="Epic"),
             _task("E-2", "indeterminate", None, issuetype="Epic")]
    children = {
        "E-1": [_task("C-1", "done", ALICE), _task("C-2", "indeterminate", CAROL)],
        "E-2": [_task("C-3", "indeterminate", CAROL, status_name="Blocked"),
                _task("C-4", "indeterminate", CAROL, status_name="Code Review 🔍")],
        "E-9": [_task("C-9", "indeterminate", BOB), _task("C-11", "done", BOB)],
    }
    if unassigned:
        loose = [_task("U-1", "new", None, issuetype="Bug"),
                 _task("U-2", "indeterminate", None, issuetype="Story"),
                 _task("U-3", "new", None, issuetype="Story"),
                 _task("U-4", "new", ALICE, issuetype="Bug")]
        window = window + [loose[0]]
        children["E-2"] = children["E-2"] + loose
    stalled = [_task("C-4", "indeterminate", CAROL, status_name="Code Review 🔍"),
               _task("S-1", "indeterminate", ERIN)]
    not_started = [_task("N-1", "new", ALICE, issuetype="Epic"),
                   _task("N-2", "new", None, issuetype="Epic"),
                   _task("E-9", "new", BOB, issuetype="Epic")]
    done_counts = {"N-1": 0, "N-2": 2, "E-9": 0}
    epic_views = {
        "E-1": _json_fixture("epic_view.json"),
        "E-2": _epic_view("E-2", "Synthetic epic two"),
        "E-9": _epic_view("E-9", "Synthetic epic nine", category="new", assignee=BOB),
    }
    parent_views = {"C-8": _parent_view("C-8", "X-1", parent_type="Story"),
                    "C-9": _parent_view("C-9", "E-9"),
                    "C-10": _parent_view("C-10"),
                    "C-3": _parent_view("C-3", "E-2")}
    not_started_views = {"N-1": _epic_view("N-1", "Synthetic queued epic", category="new",
                                           assignee=ALICE, created=_days_ago(30))}
    comment_views = {
        "C-3": {"key": "C-3", "fields": {"statuscategorychangedate": _days_ago(12),
                                         "comment": {"comments": [
            _comment("Alice Example", _days_ago(2), "Can we unblock this today?")]}}},
        "C-4": {"key": "C-4", "fields": {"comment": {"comments": [
            _comment("Carol Example", _days_ago(day), f"Update {day}.") for day in (9, 8, 7, 6, 5)]}}},
    }

    def handler(cmd):
        if cmd[1:4] == ["jira", "auth", "status"]:
            return _completed("✓ Authenticated")
        if cmd[1:4] == ["jira", "workitem", "view"]:
            key, fields = cmd[4], _fields(cmd)
            if key in fail:
                return _completed(stderr=f"✗ Error: failed to fetch {key}", returncode=1)
            table = {EPIC_VIEW_FIELDS: epic_views, "parent": parent_views,
                     NOT_STARTED_VIEW_FIELDS: not_started_views,
                     FLAGGED_VIEW_FIELDS: comment_views}[fields]
            return table[key]
        jql = _jql(cmd)
        if "--count" in cmd:
            return _count_out(done_counts[jql.split()[2]])
        if jql.startswith("parent = "):
            key = jql.split()[2]
            if key in fail:
                return _completed(stderr="✗ Error: failed to parse JQL query", returncode=1)
            return children[key]
        if 'statusCategory = "To Do"' in jql:
            return not_started
        if "issuetype = Epic" in jql:
            return epics
        if "NOT status CHANGED" in jql:
            return stalled
        if "ORDER BY updated DESC" in jql:
            return window
        raise AssertionError(f"unexpected acli call: {cmd}")

    return handler


def _collect(handler=None, calls=None, **overrides):
    return jira_scan.collect(_cfg(**overrides), runner=_runner(handler or _world(), calls), workers=4)


def _views(calls, fields):
    return sorted(cmd[4] for cmd in calls
                  if cmd[1:4] == ["jira", "workitem", "view"] and _fields(cmd) == fields)


def _children_searches(calls):
    return sorted(_jql(cmd).split()[2] for cmd in calls
                  if "--jql" in cmd and _jql(cmd).startswith("parent = ") and "--count" not in cmd)


def test_collect_runs_the_planned_jql():
    calls = []
    _collect(calls=calls)
    jqls = {_jql(cmd) for cmd in calls if "--jql" in cmd}
    assert {
        'project in (DL) AND updated >= "2026-09-28" AND updated <= "2026-10-05 23:59" ORDER BY updated DESC',
        'project in (DL) AND issuetype = Epic AND (statusCategory = "In Progress" OR updated >= "2026-09-28")',
        f"parent = E-1 AND (resolution is EMPTY OR resolution not in {EXCLUDED_JQL})",
        'project in (DL) AND statusCategory = "In Progress" AND NOT status CHANGED AFTER -5d '
        'ORDER BY statusCategoryChangedDate ASC',
        'project in (DL) AND issuetype = Epic AND statusCategory = "To Do"',
        "parent = N-1 AND statusCategory = Done",
    } <= jqls


def test_collect_search_fields_are_pinned_and_never_ask_for_rejected_fields():
    calls = []
    _collect(calls=calls)
    searches = [cmd for cmd in calls if "--jql" in cmd and "--count" not in cmd]
    by_kind = {}
    for cmd in searches:
        jql = _jql(cmd)
        kind = ("children" if jql.startswith("parent = ") else
                "not_started" if 'statusCategory = "To Do"' in jql else
                "epics" if "issuetype = Epic" in jql else
                "stalled" if "NOT status CHANGED" in jql else "window")
        by_kind.setdefault(kind, set()).add(_fields(cmd))
    assert by_kind == {
        "window": {ITEM_FIELDS},
        "children": {ITEM_FIELDS},
        "stalled": {ITEM_FIELDS},
        "epics": {EPIC_FIELDS},
        "not_started": {EPIC_FIELDS},
    }
    for cmd in searches:
        names = _fields(cmd).split(",")
        assert not set(names) & set(REJECTED_SEARCH_FIELDS), cmd


def test_collect_call_list_children_epic_views_and_parent_views():
    calls = []
    _collect(calls=calls)
    assert _children_searches(calls) == ["E-1", "E-2", "E-9"]
    assert _views(calls, EPIC_VIEW_FIELDS) == ["E-1", "E-2", "E-9"]
    assert _views(calls, "parent") == ["C-10", "C-8", "C-9"]


def test_collect_second_round_for_a_to_do_parent_epic():
    data = _collect()
    assert data["child_to_epic"] == {"C-1": "E-1", "C-2": "E-1", "C-3": "E-2", "C-4": "E-2",
                                     "C-9": "E-9", "C-11": "E-9"}
    assert [epic["key"] for epic in data["epics"]] == ["E-1", "E-2", "E-9"]
    nine = data["epics"][2]
    assert nine["status"] == "To Do"
    assert nine["progress"] == {"total": 2, "done": 1, "in_progress": 1, "pct": 50}
    assert nine["assignee"] == {"name": "Bob Example", "active": True}


def test_collect_no_second_round_for_a_non_epic_parent():
    calls = []
    data = _collect(calls=calls)
    assert "X-1" not in _children_searches(calls)
    assert "X-1" not in _views(calls, EPIC_VIEW_FIELDS)
    assert "C-8" not in data["child_to_epic"]


def test_collect_epic_block_fields():
    one = _collect()["epics"][0]
    assert one["key"] == "E-1"
    assert one["summary"] == "Synthetic epic one"
    assert one["status"] == "In Progress 🛠️"
    assert one["priority"] == "High"
    assert one["parent"] == {"key": "I-1", "summary": "Synthetic initiative"}
    assert one["assignee"] == {"name": "Carol Example", "active": True}
    assert one["description"].startswith("Build the synthetic widget.")
    assert len(one["description"]) <= 200
    assert one["progress"] == {"total": 2, "done": 1, "in_progress": 1, "pct": 50}
    total = f"parent = E-1 AND (resolution is EMPTY OR resolution not in {EXCLUDED_JQL})"
    assert one["jql"] == {"total": total, "done": total + " AND statusCategory = Done",
                          "in_progress": total + ' AND statusCategory = "In Progress"'}
    assert one["children"] == [
        {"key": "C-1", "summary": "Synthetic C-1", "status": "Done ✅", "assignee": "Alice Example",
         "category": "done"},
        {"key": "C-2", "summary": "Synthetic C-2", "status": "In Progress 🛠️",
         "assignee": "Carol Example", "category": "indeterminate"},
    ]


def test_collect_count_jql_template_and_window():
    data = _collect()
    assert data["count_jql"] == ("parent = {EPIC} AND (resolution is EMPTY OR resolution not in "
                                 + EXCLUDED_JQL + ")")
    assert data["window"] == {"since": "2026-09-28", "until": "2026-10-05", "keys": ["DL"]}


def test_collect_not_started_counts_each_once_and_lists_only_zero_done_epics_with_age():
    calls = []
    data = _collect(calls=calls)
    not_start_searches = [cmd for cmd in calls if "--jql" in cmd
                          and 'statusCategory = "To Do"' in _jql(cmd)]
    assert len(not_start_searches) == 1
    counted = sorted(_jql(cmd).split()[2] for cmd in calls if "--count" in cmd)
    assert counted == ["E-9", "N-1", "N-2"]
    assert _views(calls, NOT_STARTED_VIEW_FIELDS) == ["N-1"]
    assert [(epic["key"], epic["age_days"], epic["assignee"], epic["summary"])
            for epic in data["not_started"]] == [("N-1", 30, "Alice Example", "Synthetic queued epic")]
    assert data["not_started"][0]["created"].startswith(_days_ago(30)[:10])


def test_collect_flags_blocked_then_stalled_and_skips_out_of_scope_items():
    calls = []
    data = _collect(calls=calls)
    assert _views(calls, FLAGGED_VIEW_FIELDS) == ["C-3", "C-4"]
    assert [(item["key"], item["reason"], item["assignee"]) for item in data["flagged"]] == [
        ("C-3", "blocked", "Carol Example"), ("C-4", "stalled", "Carol Example")]


def test_collect_flagged_age_comes_from_the_status_category_change_date():
    flagged = {item["key"]: item for item in _collect()["flagged"]}
    assert flagged["C-3"]["age_days"] == 12
    assert flagged["C-4"]["age_days"] is None


def test_collect_lists_unassigned_bugs_and_in_progress_work_once_each():
    data = _collect(handler=_world(unassigned=True))
    assert data["unassigned"] == [
        {"key": "U-1", "type": "Bug", "status": "To Do", "priority": "Medium",
         "summary": "Synthetic U-1"},
        {"key": "U-2", "type": "Story", "status": "In Progress 🛠️", "priority": "Medium",
         "summary": "Synthetic U-2"},
    ]
    assert "U-1" not in [name for name in data["unmatched"]]


def test_collect_with_nothing_unassigned_gives_an_empty_list():
    assert _collect()["unassigned"] == []


def test_collect_keeps_the_newest_three_comments_and_counts_the_rest():
    data = _collect()
    c4 = data["flagged"][1]
    assert [comment["excerpt"] for comment in c4["comments"]] == ["Update 7.", "Update 6.", "Update 5."]
    assert c4["comments"][0]["author"] == "Carol Example"
    assert data["omitted"] == {"issues": 0, "comments": 2}


def test_collect_question_candidates_from_flagged_and_epic_comments():
    questions = {item["key"]: item["excerpt"] for item in _collect()["questions"]}
    assert questions == {"C-3": "Can we unblock this today?", "E-1": "Who owns the rollout plan?"}


def test_collect_people_unmatched_and_doc_links():
    data = _collect()
    assert [entry["key"] for entry in data["people"]["Carol Example"]["stuck"]] == ["C-3"]
    assert [entry["key"] for entry in data["people"]["Alice Example"]["completed"]] == ["C-1"]
    assert [entry["key"] for entry in data["people"]["Bob Example"]["in_progress"]] == ["C-9"]
    assert data["unmatched"] == ["Erin Outsider"]
    assert data["doc_links"] == [
        {"url": "https://example.atlassian.net/wiki/spaces/X/pages/1", "epic": "E-1", "kind": "wiki"},
        {"url": "https://docs.google.com/document/d/abc", "epic": "E-1", "kind": "doc"},
        {"url": "https://docs.google.com/spreadsheets/d/def", "epic": "E-1", "kind": "sheet"},
        {"url": "https://www.figma.com/file/ghi", "epic": "E-1", "kind": "design"},
    ]
    assert data["failures"] == []


def test_collect_records_a_failed_children_search_and_drops_that_epic_block():
    data = _collect(handler=_world(fail=("E-2",)))
    assert data["failures"] == ["E-2"]
    assert "E-2" in data["errors"]
    assert [epic["key"] for epic in data["epics"]] == ["E-1", "E-9"]


def test_collect_epic_scope_adds_the_requested_epic_and_records_it():
    calls = []
    data = _collect(calls=calls, scope="epic", epic="E-9")
    assert data["epic"] == "E-9"
    assert _children_searches(calls).count("E-9") == 1


def _flag_world():
    blocked = [_task(f"B-{n}", "indeterminate", ALICE, status_name="Blocked") for n in range(1, 4)]
    stalled = [_task(f"S-{n}", "indeterminate", ALICE) for n in range(1, 71)]

    def handler(cmd):
        if cmd[1:4] == ["jira", "workitem", "view"]:
            fields = _fields(cmd)
            if fields == "parent":
                return _parent_view(cmd[4])
            if fields == FLAGGED_VIEW_FIELDS:
                return {"key": cmd[4], "fields": {"comment": {"comments": []}}}
            raise AssertionError(cmd)
        jql = _jql(cmd)
        if 'statusCategory = "To Do"' in jql or "issuetype = Epic" in jql:
            return []
        if "NOT status CHANGED" in jql:
            return stalled
        if "ORDER BY updated DESC" in jql:
            return blocked
        raise AssertionError(cmd)

    return handler


def test_collect_caps_flagged_at_max_flagged_keeping_blocked_and_the_oldest_stalled():
    calls = []
    data = _collect(handler=_flag_world(), calls=calls, max_flagged=60)
    fetched = _views(calls, FLAGGED_VIEW_FIELDS)
    assert len(fetched) == 60
    assert {"B-1", "B-2", "B-3"} <= set(fetched)
    assert set(fetched) - {"B-1", "B-2", "B-3"} == {f"S-{n}" for n in range(1, 58)}
    assert [item["key"] for item in data["flagged"]][:3] == ["B-1", "B-2", "B-3"]
    assert data["omitted"]["issues"] == 13
    text = jira_scan.render_digest(data, "team", 10 ** 6)
    assert text.rstrip("\n").splitlines()[-1].startswith("Omitted: 13 issues")


def _argv(out, *extra):
    return ["--keys", "DL", "--since", "2026-09-28", "--until", "2026-10-05",
            "--config", str(FIXTURES / "roster.md"),
            "--excluded-resolutions", "Won't Do, Declined, Duplicate",
            "--word-limit", "5000", "--workers", "4", "--out", str(out), *list(extra)]


def test_main_returns_1_when_acli_auth_fails(tmp_path, capsys):
    def handler(cmd):
        return _completed(stderr="✗ Error: not logged in", returncode=1)
    assert jira_scan.main(_argv(tmp_path / "out"), runner=_runner(handler)) == 1
    assert "acli unavailable" in capsys.readouterr().err
    assert not (tmp_path / "out").exists()


def test_main_returns_1_when_acli_is_missing(tmp_path, capsys):
    def runner(cmd, **kwargs):
        raise FileNotFoundError("acli")
    assert jira_scan.main(_argv(tmp_path), runner=runner) == 1
    assert "acli unavailable" in capsys.readouterr().err


def test_main_reports_incomplete_epics_and_still_writes_both_files(tmp_path, capsys):
    code = jira_scan.main(_argv(tmp_path), runner=_runner(_world(fail=("E-2",))))
    assert code == 2
    assert "INCOMPLETE: E-2" in capsys.readouterr().err
    data = json.loads((tmp_path / "jira.json").read_text())
    assert data["failures"] == ["E-2"]
    assert "## Not measured" in (tmp_path / "jira.md").read_text()


def test_main_happy_path_writes_jira_json_and_jira_md(tmp_path, capsys):
    assert jira_scan.main(_argv(tmp_path), runner=_runner(_world())) == 0
    data = json.loads((tmp_path / "jira.json").read_text())
    assert {"window", "epics", "child_to_epic", "people", "unmatched", "unassigned", "flagged",
            "doc_links", "failures"} <= set(data)
    assert data["child_to_epic"]["C-9"] == "E-9"
    digest = (tmp_path / "jira.md").read_text()
    assert digest.startswith("# Jira digest: DL, 2026-09-28 to 2026-10-05")
    assert "## Doc links" in digest
    assert digest.rstrip("\n").splitlines()[-1].endswith("(word limit 5000)")
    out = capsys.readouterr().out
    assert str(tmp_path / "jira.json") in out


def test_main_jira_json_feeds_pr_scan_jira_map(tmp_path):
    import pr_scan
    jira_scan.main(_argv(tmp_path), runner=_runner(_world()))
    assert pr_scan.load_jira_map(str(tmp_path / "jira.json"))["C-1"] == "E-1"


def test_main_person_scope_renders_only_that_person(tmp_path):
    argv = _argv(tmp_path, "--scope", "person", "--person", "Bob Example")
    assert jira_scan.main(argv, runner=_runner(_world())) == 0
    digest = (tmp_path / "jira.md").read_text()
    assert "Scope: person (Bob Example)" in digest
    assert "## Issues" in digest


def test_main_passes_max_flagged_to_collect(tmp_path):
    calls = []
    argv = _argv(tmp_path, "--max-flagged", "1")
    assert jira_scan.main(argv, runner=_runner(_world(), calls)) == 0
    assert _views(calls, FLAGGED_VIEW_FIELDS) == ["C-3"]
    assert json.loads((tmp_path / "jira.json").read_text())["omitted"]["issues"] == 1


def test_main_cannot_read_config_returns_1(tmp_path, capsys):
    argv = _argv(tmp_path)
    argv[argv.index("--config") + 1] = str(tmp_path / "absent.md")
    assert jira_scan.main(argv, runner=_runner(_world())) == 1
    assert "cannot read --config" in capsys.readouterr().err


@pytest.mark.parametrize("extra", [
    ["--scope", "person"],
    ["--scope", "epic"],
    ["--scope", "sprint"],
])
def test_main_rejects_incomplete_scope_arguments(tmp_path, extra):
    with pytest.raises(SystemExit) as raised:
        jira_scan.main(_argv(tmp_path, *extra), runner=_runner(_world()))
    assert raised.value.code == 2


def test_main_requires_excluded_resolutions(tmp_path):
    argv = _argv(tmp_path)
    position = argv.index("--excluded-resolutions")
    del argv[position:position + 2]
    with pytest.raises(SystemExit) as raised:
        jira_scan.main(argv, runner=_runner(_world()))
    assert raised.value.code == 2


def test_main_defaults_max_flagged_to_60_and_workers_to_8():
    parser = jira_scan.build_parser()
    args = parser.parse_args(["--keys", "DL", "--since", "2026-09-28", "--config", "x",
                              "--excluded-resolutions", "Won't Do", "--word-limit", "100"])
    assert args.max_flagged == 60
    assert args.workers == 8


def test_collect_reports_raw_counts_for_cross_checking():
    assert _collect()["counts"] == {"window": 7, "stalled": 2, "blocked": 1, "flagged_candidates": 2}


def test_collect_counts_flagged_candidates_before_the_cap():
    counts = _collect(handler=_flag_world(), max_flagged=60)["counts"]
    assert counts == {"window": 3, "stalled": 70, "blocked": 3, "flagged_candidates": 73}
