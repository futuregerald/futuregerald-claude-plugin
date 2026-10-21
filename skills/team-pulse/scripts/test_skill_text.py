import ast
import re
from pathlib import Path

import pytest

SKILL = Path(__file__).resolve().parent.parent
SKILL_MD = SKILL / "SKILL.md"
AGENT_PROMPTS = SKILL / "references" / "agent-prompts.md"

TEXT_FILES = ["SKILL.md", "references/agent-prompts.md"]
BANNED = [
    r"\bone at a time\b",
    r"one (?:PR|ticket|item|meeting)[^.\n]*at a time",
    r"(?i)summari[sz]e[sd]? incrementally",
    r"(?i)process(?:es)? results incrementally",
    r"\bwrite_to_file\b",
    r"\brun_command\b",
    r"(?i)as you process it",
]


def read(path):
    return path.read_text(encoding="utf-8")


@pytest.mark.parametrize(
    "text_file,pattern",
    [(f, p) for f in TEXT_FILES for p in BANNED],
    ids=[f"{f}-{p}" for f in TEXT_FILES for p in BANNED],
)
def test_no_banned_phrases(text_file, pattern):
    text = read(SKILL / text_file)
    match = re.search(pattern, text)
    if match:
        line = text.count("\n", 0, match.start()) + 1
        pytest.fail(f"{text_file}:{line} matches {pattern!r}: {match.group(0)!r}")


def test_resolution_list_comes_from_config():
    text = read(AGENT_PROMPTS)
    assert '"Cancelled"' not in text
    assert "{EXCLUDED_RESOLUTIONS}" in text


def test_orchestrator_exception_names_both_scripts():
    text = read(SKILL_MD)
    start = text.index("## Architecture")
    end = text.index("### Why", start)
    architecture = text[start:end]
    assert "jira_scan.py" in architecture
    assert "pr_scan.py" in architecture


def test_no_github_digest():
    for text_file in TEXT_FILES:
        assert ".updates/github.md" not in read(SKILL / text_file), text_file


SCRIPTS = sorted(path.name for path in (SKILL / "scripts").glob("*.py")
                 if not path.name.startswith("test_"))


def _union_annotations(tree):
    annotations = []
    for node in ast.walk(tree):
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
            arguments = node.args.posonlyargs + node.args.args + node.args.kwonlyargs
            annotations += [argument.annotation for argument in arguments] + [node.returns]
        elif isinstance(node, ast.AnnAssign):
            annotations.append(node.annotation)
    return [annotation for annotation in annotations if annotation is not None
            and any(isinstance(part, ast.BinOp) and isinstance(part.op, ast.BitOr)
                    for part in ast.walk(annotation))]


def _has_future_annotations(tree):
    return any(isinstance(node, ast.ImportFrom) and node.module == "__future__"
               and any(alias.name == "annotations" for alias in node.names)
               for node in tree.body)


@pytest.mark.parametrize("script", SCRIPTS)
def test_scripts_import_on_python_3_9(script):
    source = read(SKILL / "scripts" / script)
    tree = ast.parse(source, feature_version=(3, 9))
    if _union_annotations(tree):
        assert _has_future_annotations(tree), f"{script} uses X | Y annotations without the future import"


def _between(text, start, end):
    begin = text.index(start)
    return text[begin:text.index(end, begin)]


def test_jira_scan_incomplete_exit_continues_the_run():
    codes = _between(read(SKILL_MD), "**Exit codes:**", "## Step 2b")
    incomplete = _between(codes, "**2 with `INCOMPLETE", "**2 with a `usage:`")
    assert "continue with step 2" in incomplete
    assert not re.search(r"\bstop\b", incomplete)


def test_step_3_names_where_question_candidates_come_from():
    step3 = _between(read(SKILL_MD), "## Step 3", "## Step 4")
    assert "lists every item" not in step3
    assert "flagged items and active epics" in step3


def test_step_3_covers_the_fallback_digest():
    step3 = _between(read(SKILL_MD), "## Step 3", "## Step 4")
    assert "Agent A's own sections" in step3


def test_epic_start_names_its_source():
    resolve = _between(read(SKILL_MD), "**Epic start, for Agent F**", "**Word budget for Agent F**")
    assert "Created" in resolve and "jira.md" in resolve


def test_agent_e_reads_reviews_in_bulk():
    agent_e = _between(read(AGENT_PROMPTS), "## Agent E", "## Combining")
    assert "--json number,title,author,reviews" in agent_e
    assert "For each candidate PR" not in agent_e
    assert "pulls/{N}/reviews" not in agent_e


def test_skill_lists_high_priority_section():
    assert "## High priority outside epics" in read(SKILL_MD)


def test_agent_a_lists_high_priority_outside_epics():
    agent_a = _between(read(AGENT_PROMPTS), "## Agent A", "## Agent C")
    assert "## High priority outside epics" in agent_a


def test_step_3_says_who_a_question_waits_on():
    step3 = _between(read(SKILL_MD), "## Step 3", "## Step 4")
    assert "@mentions" in step3 and "→" in step3


def test_agent_a_comments_follow_the_mention_rule():
    agent_a = _between(read(AGENT_PROMPTS), "## Agent A", "## Agent C")
    assert "@mentions someone other than its author" in agent_a


def test_agent_e_repo_placeholder_matches_the_rest_of_the_file():
    agent_e = _between(read(AGENT_PROMPTS), "## Agent E", "## Combining")
    assert "--repo {ORG}/{REPO}" in agent_e
    assert "nameWithOwner" not in agent_e


def test_skill_lists_closed_or_ongoing_section():
    assert "## Closed or ongoing epics" in read(SKILL_MD)


def test_step_3_puts_long_stalled_on_person_cards():
    step3 = _between(read(SKILL_MD), "## Step 3", "## Step 4")
    assert "Long-stalled" in step3 and "person's card" in step3


def test_epic_start_reads_active_or_not_started():
    resolve = _between(read(SKILL_MD), "**Epic start, for Agent F**", "**Word budget for Agent F**")
    assert "## Active epics" in resolve and "## Not started" in resolve


def test_skill_says_how_to_build_done_and_in_progress_count_queries():
    step2a = _between(read(SKILL_MD), "## Step 2a", "## Step 2b")
    assert "AND statusCategory = Done" in step2a


SHIPPED_TEXT_FILES = ["SKILL.md", "references/agent-prompts.md", "references/team.md", "references/report-format.md", "references/payload.md"]


@pytest.mark.parametrize("name", SHIPPED_TEXT_FILES)
def test_no_meeting_vendor_is_named(name):
    text = read(SKILL / name).casefold()
    for vendor in ("krisp", "search_meetings"):
        assert vendor not in text, f"{name} names {vendor}"


def _agent_c():
    return _between(read(AGENT_PROMPTS), "## Agent C", "## Agent F")


def test_agent_c_follows_the_source_order():
    agent_c = _agent_c()
    positions = [agent_c.index(marker) for marker in
                 ("1. User-named sources", "2. Configured meeting tool", "3. No tool configured")]
    assert positions == sorted(positions)


def test_agent_c_reads_are_bounded_paged_and_read_only():
    agent_c = _agent_c()
    assert "at most 12" in agent_c
    assert "never create, update or delete" in agent_c
    assert "oldest" in agent_c and "dropped" in agent_c
    assert "one search per person" not in agent_c


def test_agent_c_does_not_fall_back_when_a_configured_source_fails():
    assert "no fallback" in _agent_c()


def test_skill_resolves_meeting_sources():
    resolve = _between(read(SKILL_MD), "## Step 1", "## Step 2")
    assert "{MEETING_SOURCES}" in resolve and "{USER_MEETING_SOURCES}" in resolve


def test_team_template_shows_meeting_sources_as_an_example_not_a_value():
    team = read(SKILL / "references/team.md")
    assert "`- **Meeting sources:** " in team
    assert not any(line.startswith("- **Meeting sources:**") for line in team.splitlines())


def test_agent_c_runs_discovery_when_no_tool_is_configured():
    agent_c = _agent_c()
    assert "`none`" in agent_c and "go to 3" in agent_c


def test_agent_c_still_reads_user_sources_when_the_tool_is_unavailable():
    assert "still read the sources from 1" in _agent_c()


def test_agent_c_ranks_by_the_epics_in_scope():
    assert "{EPIC_KEYS}" in _agent_c()
    assert "{EPIC_KEYS}" in _between(read(SKILL_MD), "## Step 1", "## Step 2")


def test_agent_c_reads_summaries_not_whole_transcripts():
    agent_c = _agent_c()
    assert "plus the speaker names" not in agent_c
    assert "only when there is no summary" in agent_c


def test_agent_c_treats_only_dropped_items_as_missing():
    agent_c = _agent_c()
    assert "dropped" in agent_c and "smaller page" in agent_c


def test_report_format_names_meeting_sources_generically():
    assert "never by tool name" in read(SKILL / "references/report-format.md")


def test_meetings_grant_includes_read_for_user_files():
    row = next(l for l in read(SKILL_MD).splitlines() if l.startswith("| Meetings | Agent C"))
    assert "`Read`" in row


PAYLOAD_MD = SKILL / "references" / "payload.md"
PAYLOAD_EXAMPLES = ["team", "person", "initiative"]


@pytest.mark.parametrize("stem", PAYLOAD_EXAMPLES)
def test_example_payloads_are_v2_and_match_their_scope(stem):
    import json

    payload = json.loads(read(SKILL / "assets" / "payloads" / f"{stem}.json"))
    assert payload["schema_version"] == 2
    assert payload["scope"] == stem
    assert "generated_at" not in payload
    assert "dashboard_path" not in payload


def test_payload_reference_covers_plain_text_fields():
    text = read(PAYLOAD_MD)
    assert "&amp;" in text and "plain" in text


def test_payload_reference_requires_backticking_angle_brackets():
    text = read(PAYLOAD_MD)
    assert "backtick" in text and "<" in text


def test_payload_reference_names_the_links_check():
    assert "pulsed config show --json" in read(PAYLOAD_MD)


def test_payload_reference_states_section_notes_difference():
    text = read(PAYLOAD_MD)
    assert "replaces" in text and "appended" in text


def test_examples_carry_no_real_identifiers():
    for path in (SKILL / "assets" / "payloads").iterdir():
        assert not re.search(r"(?i)cobalt|atlassian\.net", read(path)), path.name


def _step_4():
    text = read(SKILL_MD)
    return text[text.index("## Step 4: Deliver"):text.index("## Scoping Variations")]


def test_step_4_submits_and_renders_through_pulsed():
    step4 = _step_4()
    assert "pulsed report submit" in step4 and "pulsed report render" in step4


def test_step_4_writes_markdown_only_on_request():
    step4 = _step_4()
    assert "--format md" in step4
    assert "Save a publication-grade markdown document" not in step4


def test_step_4_has_a_fallback_when_pulsed_is_missing():
    assert "command -v pulsed" in _step_4()


def test_step_4_reads_links_config():
    assert "pulsed config show --json" in _step_4()


def test_step_4_renders_with_force():
    lines = [line for line in _step_4().splitlines() if "pulsed report render" in line]
    assert lines and all("--force" in line for line in lines)


def test_step_4_keeps_confluence_and_slack_adaptation():
    assert "Confluence or Slack" in _step_4()


def test_report_format_says_html_only_by_default():
    text = read(SKILL / "references" / "report-format.md")
    assert "markdown only" in text.casefold()
    assert "Every team pulse run generates" not in text


def test_forecast_has_one_deliverable_by_default():
    method = read(SKILL / "references" / "forecast" / "method.md")
    report_format = read(SKILL / "references" / "forecast" / "report-format.md")
    for text in (method, report_format):
        for phrase in ("exactly two files", "two files", "Two deliverables"):
            assert phrase.casefold() not in text.casefold(), phrase
    phase_7 = method[method.index("## Phase 7"):]
    assert "only when the user asks" in phase_7 and "markdown" in phase_7.casefold()
