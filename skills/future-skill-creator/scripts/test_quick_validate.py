import subprocess
import sys
from pathlib import Path

import pytest

import quick_validate

SCRIPTS = Path(__file__).resolve().parent
SKILLS_DIR = SCRIPTS.parent.parent


def make_skill(tmp_path, frontmatter, directory="demo-skill"):
    skill = tmp_path / directory
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
        (
            "name: demo-skill\ndescription: Does a thing.\nmodel: opus\neffort: high\nlanguages: [go]\n"
            "argument-hint: <path>\ntrigger: /demo\nversion: 1.0.0\nuser-invocable: true",
            True,
            "",
        ),
        ("name: demo-skill\ndescription: Does a thing.\nfoo: x", False, "Unexpected key"),
        ("name: demo-skill", False, "description"),
        ("name: demo-skill\ndescription: Uses <html> tags.", False, "angle brackets"),
        ("name: demo-skill\ndescription: Does a thing.\ntags:\n  - review\n  - quality", False, "inline list"),
        ("name: demo-skill\ndescription: \"TODO: say what it does\"", False, "TODO"),
        ("name: demo-skill\ndescription: Does a thing.\nauthor: \"TODO: your name\"", False, "TODO"),
        ("name: demo-skill\ndescription: Does a thing.\ntags: [review, todo]", False, "TODO"),
    ],
)
def test_validate_skill(tmp_path, frontmatter, valid, message_part):
    ok, message = quick_validate.validate_skill(make_skill(tmp_path, frontmatter))
    assert ok is valid, message
    assert message_part in message


def test_name_must_match_directory(tmp_path):
    skill = make_skill(tmp_path, "name: other-name\ndescription: Does a thing.", directory="demo-skill")
    ok, message = quick_validate.validate_skill(skill)
    assert not ok
    assert "directory" in message


def run_init(tmp_path):
    subprocess.run(
        [sys.executable, str(SCRIPTS / "init_skill.py"), "demo-skill", "--path", str(tmp_path)],
        check=True,
        capture_output=True,
    )
    return tmp_path / "demo-skill"


def test_init_skill_template_carries_author_tags_and_point_of_view(tmp_path):
    text = (run_init(tmp_path) / "SKILL.md").read_text()
    frontmatter = text.split("---")[1]
    assert "\nauthor:" in frontmatter
    assert "\ntags:" in frontmatter
    assert "## What good output looks like" in text


def test_unfilled_template_fails_validation(tmp_path):
    ok, message = quick_validate.validate_skill(run_init(tmp_path))
    assert not ok
    assert "TODO" in message


def test_filled_template_passes_validation(tmp_path):
    skill = run_init(tmp_path)
    skill_md = skill / "SKILL.md"
    body = skill_md.read_text().split("---", 2)[2]
    skill_md.write_text(
        "---\nname: demo-skill\ndescription: Does a thing. Use when X.\n"
        "author: Gerald Onyango\ntags: [demo]\n---" + body
    )
    ok, message = quick_validate.validate_skill(skill)
    assert ok, message


@pytest.mark.parametrize(
    "skill_dir",
    sorted(p.parent for p in SKILLS_DIR.glob("*/SKILL.md")),
    ids=lambda p: p.name,
)
def test_every_repo_skill_uses_only_known_frontmatter_keys(skill_dir):
    ok, message = quick_validate.validate_skill(skill_dir)
    assert "Unexpected key" not in message, message
