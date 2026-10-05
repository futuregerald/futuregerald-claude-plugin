import subprocess
import sys
from pathlib import Path

import pytest

import quick_validate

SCRIPTS = Path(__file__).resolve().parent


def make_skill(tmp_path, frontmatter):
    skill = tmp_path / "demo-skill"
    skill.mkdir()
    (skill / "SKILL.md").write_text(f"---\n{frontmatter}\n---\n\n# Demo\n")
    return skill


@pytest.mark.parametrize(
    "frontmatter,valid,message_part",
    [
        ("name: demo-skill\ndescription: Does a thing. Use when X.", True, ""),
        (
            "name: demo-skill\ndescription: Does a thing.\nauthor: Gerald Onyango\ntags: [review, quality]",
            True,
            "",
        ),
        ("name: demo-skill\ndescription: Does a thing.\nfoo: x", False, "Unexpected key"),
        ("name: demo-skill", False, "description"),
        ("name: demo-skill\ndescription: Uses <html> tags.", False, "angle brackets"),
    ],
)
def test_validate_skill(tmp_path, frontmatter, valid, message_part):
    ok, message = quick_validate.validate_skill(make_skill(tmp_path, frontmatter))
    assert ok is valid, message
    assert message_part in message


def test_init_skill_template_carries_author_tags_and_point_of_view(tmp_path):
    result = subprocess.run(
        [sys.executable, str(SCRIPTS / "init_skill.py"), "demo-skill", "--path", str(tmp_path)],
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0, result.stderr
    text = (tmp_path / "demo-skill" / "SKILL.md").read_text()
    frontmatter = text.split("---")[1]
    assert "\nauthor:" in frontmatter
    assert "\ntags:" in frontmatter
    assert "## What good output looks like" in text


def test_init_skill_output_passes_validation(tmp_path):
    subprocess.run(
        [sys.executable, str(SCRIPTS / "init_skill.py"), "demo-skill", "--path", str(tmp_path)],
        check=True,
        capture_output=True,
    )
    ok, message = quick_validate.validate_skill(tmp_path / "demo-skill")
    assert ok, message
