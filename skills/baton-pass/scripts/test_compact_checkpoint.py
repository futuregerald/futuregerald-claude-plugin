import json
import subprocess
import sys
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parent / "compact_checkpoint.py"


def tool_use(name, **tool_input):
    return {
        "type": "assistant",
        "message": {"role": "assistant", "content": [{"type": "tool_use", "name": name, "input": tool_input}]},
    }


def write_transcript(path, records):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("\n".join(json.dumps(r) for r in records) + "\n")


def run_hook(payload, baton_dir, cwd):
    stdin = payload if isinstance(payload, str) else json.dumps(payload)
    return subprocess.run(
        [sys.executable, str(SCRIPT)],
        input=stdin,
        capture_output=True,
        text=True,
        cwd=cwd,
        env={"BATON_PASS_DIR": str(baton_dir), "PATH": "/usr/bin:/bin:/usr/local/bin:/opt/homebrew/bin"},
    )


def make_repo(parent, name):
    repo = parent / name
    repo.mkdir()
    subprocess.run(["git", "init", "-q", "-b", "work"], cwd=repo, check=True)
    return repo


def precompact(session_id, transcript, cwd):
    return {
        "session_id": session_id,
        "transcript_path": str(transcript),
        "cwd": str(cwd),
        "hook_event_name": "PreCompact",
        "trigger": "auto",
        "custom_instructions": None,
    }


def compact_start(session_id, transcript, cwd, source="compact"):
    return {
        "session_id": session_id,
        "transcript_path": str(transcript),
        "cwd": str(cwd),
        "hook_event_name": "SessionStart",
        "source": source,
    }


@pytest.fixture
def setup(tmp_path):
    repo = make_repo(tmp_path, "My_Repo")
    transcript = tmp_path / "projects" / "s1.jsonl"
    write_transcript(
        transcript,
        [
            tool_use("Edit", file_path="/x/edited.py"),
            tool_use("Write", file_path="/x/written.py"),
            tool_use("Edit", file_path="/x/edited.py"),
            tool_use("Read", file_path="/x/only_read.py"),
            tool_use("NotebookEdit", notebook_path="/x/nb.ipynb"),
        ],
    )
    return repo, transcript, tmp_path / "baton"


def checkpoints(baton_dir, key):
    return sorted((baton_dir / key).glob("*-precompact.md"))


def test_precompact_writes_checkpoint_with_edited_files(setup):
    repo, transcript, baton = setup
    result = run_hook(precompact("s1", transcript, repo), baton, repo)
    assert result.returncode == 0
    assert result.stdout == ""
    [checkpoint] = checkpoints(baton, "my-repo")
    text = checkpoint.read_text()
    assert text.count("/x/edited.py") == 1
    assert "/x/written.py" in text
    assert "/x/nb.ipynb" in text
    assert "/x/only_read.py" not in text
    assert "**Branch:** work" in text
    assert f"**Transcript (JSONL, full pre-compaction history):** {transcript}" in text
    assert "**Session:** s1" in text


def test_precompact_lists_subagent_edits(setup):
    repo, transcript, baton = setup
    write_transcript(
        transcript.parent / "s1" / "subagents" / "agent-a.jsonl",
        [tool_use("Edit", file_path="/x/by_subagent.py")],
    )
    run_hook(precompact("s1", transcript, repo), baton, repo)
    [checkpoint] = checkpoints(baton, "my-repo")
    assert "/x/by_subagent.py" in checkpoint.read_text()


def test_checkpoint_records_head_and_unstaged_changes(setup):
    repo, transcript, baton = setup
    (repo / "f.txt").write_text("one")
    subprocess.run(["git", "add", "f.txt"], cwd=repo, check=True)
    subprocess.run(
        ["git", "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "commit", "-qm", "c"],
        cwd=repo,
        check=True,
    )
    (repo / "f.txt").write_text("two")
    head = subprocess.run(["git", "rev-parse", "HEAD"], cwd=repo, capture_output=True, text=True).stdout.strip()
    run_hook(precompact("s1", transcript, repo), baton, repo)
    text = checkpoints(baton, "my-repo")[0].read_text()
    assert f"**HEAD:** {head}" in text
    assert "\n M f.txt\n" in text


def test_malformed_transcript_records_still_write_a_checkpoint(setup):
    repo, transcript, baton = setup
    write_transcript(
        transcript,
        [
            {"type": "assistant", "message": None},
            {"type": "assistant", "message": "text"},
            {"type": "assistant", "message": {"content": [{"type": "tool_use", "name": "Edit", "input": "x"}]}},
            tool_use("Edit", file_path="/x/still_found.py"),
        ],
    )
    run_hook(precompact("s1", transcript, repo), baton, repo)
    text = checkpoints(baton, "my-repo")[0].read_text()
    assert "**Branch:** work" in text
    assert "/x/still_found.py" in text


def test_missing_transcript_path_is_reported_as_unknown(setup):
    repo, _, baton = setup
    payload = precompact("s1", "", repo)
    payload["transcript_path"] = None
    run_hook(payload, baton, repo)
    assert "full pre-compaction history):** unknown" in checkpoints(baton, "my-repo")[0].read_text()


def test_checkpoint_is_private_to_the_user(setup):
    repo, transcript, baton = setup
    run_hook(precompact("s1", transcript, repo), baton, repo)
    [checkpoint] = checkpoints(baton, "my-repo")
    assert baton.stat().st_mode & 0o077 == 0
    assert checkpoint.parent.stat().st_mode & 0o077 == 0
    assert checkpoint.stat().st_mode & 0o077 == 0


def test_session_id_cannot_steer_the_subagent_scan(setup):
    repo, transcript, baton = setup
    write_transcript(
        transcript.parent / "elsewhere" / "subagents" / "agent-x.jsonl",
        [tool_use("Edit", file_path="/x/outside.py")],
    )
    run_hook(precompact("../projects/elsewhere", transcript, repo), baton, repo)
    [checkpoint] = checkpoints(baton, "my-repo")
    assert "/x/outside.py" not in checkpoint.read_text()


def test_key_collapses_runs_of_special_characters(tmp_path):
    repo = make_repo(tmp_path, "a__b")
    result = run_hook(precompact("s1", tmp_path / "none.jsonl", repo), tmp_path / "baton", repo)
    assert result.returncode == 0
    assert checkpoints(tmp_path / "baton", "a-b")


def test_key_strips_leading_separator(tmp_path):
    repo = make_repo(tmp_path, "_lead")
    run_hook(precompact("s1", tmp_path / "none.jsonl", repo), tmp_path / "baton", repo)
    assert checkpoints(tmp_path / "baton", "lead")


def test_precompact_outside_git_uses_dev_key(tmp_path):
    plain = tmp_path / "plain"
    plain.mkdir()
    result = run_hook(precompact("s1", tmp_path / "none.jsonl", plain), tmp_path / "baton", plain)
    assert result.returncode == 0
    assert checkpoints(tmp_path / "baton", "dev")


@pytest.mark.parametrize("stdin", ["", "not json", "[]", '{"hook_event_name": "PreCompact"}'])
def test_bad_input_never_fails(tmp_path, stdin):
    result = run_hook(stdin, tmp_path / "baton", tmp_path)
    assert result.returncode == 0
    assert result.stdout == ""


def test_compact_session_start_points_at_this_sessions_checkpoint(setup):
    repo, transcript, baton = setup
    (baton / "my-repo").mkdir(parents=True)
    (baton / "my-repo" / "2099-01-01-000000-old-ACTIVE_CONTEXT.md").write_text("stale")
    run_hook(precompact("s1", transcript, repo), baton, repo)
    [mine] = checkpoints(baton, "my-repo")
    run_hook(precompact("other-session", transcript, repo), baton, repo)

    result = run_hook(compact_start("s1", transcript, repo), baton, repo)

    assert result.returncode == 0
    output = json.loads(result.stdout)["hookSpecificOutput"]
    assert output["hookEventName"] == "SessionStart"
    assert str(mine) in output["additionalContext"]
    assert str(transcript) in output["additionalContext"]
    assert "ACTIVE_CONTEXT" not in output["additionalContext"]


def test_compact_session_start_without_checkpoint_still_names_transcript(setup):
    repo, transcript, baton = setup
    result = run_hook(compact_start("s1", transcript, repo), baton, repo)
    assert result.returncode == 0
    assert str(transcript) in json.loads(result.stdout)["hookSpecificOutput"]["additionalContext"]


@pytest.mark.parametrize("source", ["startup", "resume", "clear"])
def test_other_session_start_sources_print_nothing(setup, source):
    repo, transcript, baton = setup
    result = run_hook(compact_start("s1", transcript, repo, source=source), baton, repo)
    assert result.returncode == 0
    assert result.stdout == ""
