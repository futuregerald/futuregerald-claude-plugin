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
