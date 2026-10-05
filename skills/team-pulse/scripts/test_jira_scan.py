import inspect
import json
import re
from datetime import datetime, timedelta, timezone
from pathlib import Path

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


def _child(key, summary, status="In Progress 🛠️", assignee=None):
    return {"key": key, "summary": summary, "status": status, "assignee": assignee}


def _data():
    people, unmatched = _grouped()
    return {
        "window": {"since": "2026-09-28", "until": "2026-10-05", "keys": ["DL"]},
        "count_jql": COUNT_JQL,
        "epics": [
            _epic("E-1", "Synthetic epic one",
                  children=[_child("C-1", "Synthetic done task", "Done ✅", "Alice Example"),
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


def test_render_digest_carries_collector_omissions_and_caps_comments_at_three():
    data = _data()
    data["omitted"] = {"issues": 13, "comments": 2}
    data["flagged"][0]["comments"] = [
        {"author": "Alice Example", "created": f"2026-09-2{n}T09:00:00.000+0000",
         "excerpt": f"note {n}"} for n in range(5)
    ]
    text = _render(data)
    assert "note 3" not in text and "note 4" not in text
    assert text.rstrip("\n").splitlines()[-1] == (
        "Omitted: 13 issues, 0 not-started epics, 4 comments (word limit 5000)"
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


NEVER_CUT = ("## Active epics", "## Flagged", "## Question candidates",
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
