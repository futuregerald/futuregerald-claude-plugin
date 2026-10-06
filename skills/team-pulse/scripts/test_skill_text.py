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
