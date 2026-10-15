#!/usr/bin/env python3
import json
import os
import re
import subprocess
import sys
from datetime import datetime
from pathlib import Path

EDIT_TOOLS = {"Edit", "Write", "MultiEdit", "NotebookEdit"}


def git(cwd, *args):
    result = subprocess.run(["git", "-C", str(cwd), *args], capture_output=True, text=True)
    return result.stdout.strip() if result.returncode == 0 else ""


def project_key(cwd):
    common_dir = git(cwd, "rev-parse", "--path-format=absolute", "--git-common-dir")
    if not common_dir:
        return "dev"
    name = Path(re.sub(r"/\.git/?$", "", common_dir)).name.lower()
    return re.sub(r"[^a-z0-9]+", "-", name).strip("-") or "dev"


def baton_dir():
    return Path(os.environ.get("BATON_PASS_DIR") or Path.home() / ".claude" / "baton-pass")


def safe_session(session_id):
    return re.sub(r"[^A-Za-z0-9-]+", "-", session_id)[:64] or "unknown"


def edited_paths_in(transcript):
    paths = []
    try:
        lines = transcript.read_text(errors="replace").splitlines()
    except OSError:
        return paths
    for line in lines:
        try:
            record = json.loads(line)
        except ValueError:
            continue
        content = record.get("message", {}).get("content") if isinstance(record, dict) else None
        if not isinstance(content, list):
            continue
        for block in content:
            if isinstance(block, dict) and block.get("type") == "tool_use" and block.get("name") in EDIT_TOOLS:
                tool_input = block.get("input") or {}
                path = tool_input.get("file_path") or tool_input.get("notebook_path")
                if path:
                    paths.append(path)
    return paths


def session_edited_paths(transcript, session_id):
    transcripts = [transcript] + sorted((transcript.parent / session_id / "subagents").glob("*.jsonl"))
    seen = {}
    for path in transcripts:
        for edited in edited_paths_in(path):
            seen.setdefault(edited, None)
    return list(seen)


def write_checkpoint(payload):
    session_id = str(payload.get("session_id") or "")
    cwd = Path(payload.get("cwd") or os.getcwd())
    transcript = Path(payload.get("transcript_path") or "")
    out_dir = baton_dir() / project_key(cwd)
    out_dir.mkdir(parents=True, exist_ok=True)
    stamp = datetime.now().strftime("%Y-%m-%d-%H%M%S")
    edited = session_edited_paths(transcript, session_id) if transcript.name else []
    lines = [
        f"# Pre-compaction checkpoint — {stamp}",
        "",
        f"- **Session:** {session_id}",
        f"- **Trigger:** {payload.get('trigger') or 'unknown'}",
        f"- **Transcript (JSONL, full pre-compaction history):** {transcript}",
        f"- **Directory:** {cwd}",
        f"- **Branch:** {git(cwd, 'branch', '--show-current') or 'N/A'}",
        f"- **HEAD:** {git(cwd, 'rev-parse', 'HEAD') or 'N/A'}",
        "",
        "## Files edited this session (including sub-agents)",
        "",
        *([f"- {p}" for p in edited] or ["(none found)"]),
        "",
        "## Uncommitted changes",
        "",
        "```",
        git(cwd, "status", "--short") or "(clean or not a repo)",
        "```",
        "",
    ]
    (out_dir / f"{stamp}-{safe_session(session_id)}-precompact.md").write_text("\n".join(lines))


def latest_checkpoint(cwd, session_id):
    matches = sorted((baton_dir() / project_key(cwd)).glob(f"*-{safe_session(session_id)}-precompact.md"))
    return matches[-1] if matches else None


def compact_context(payload):
    cwd = Path(payload.get("cwd") or os.getcwd())
    transcript = payload.get("transcript_path") or "unknown"
    checkpoint = latest_checkpoint(cwd, str(payload.get("session_id") or ""))
    parts = [
        "Context was just compacted, and the summary is lossy.",
        f"The full pre-compaction transcript is at {transcript}: grep it for any detail the summary dropped before re-reading files.",
    ]
    if checkpoint:
        parts.append(f"Pre-compaction checkpoint (branch, HEAD, files edited, uncommitted changes): {checkpoint}.")
    return " ".join(parts)


def main():
    try:
        payload = json.loads(sys.stdin.read() or "null")
        if not isinstance(payload, dict):
            return
        event = payload.get("hook_event_name")
        if event == "PreCompact":
            write_checkpoint(payload)
        elif event == "SessionStart" and payload.get("source") == "compact":
            print(json.dumps({"hookSpecificOutput": {"hookEventName": "SessionStart", "additionalContext": compact_context(payload)}}))
    except Exception:
        return


if __name__ == "__main__":
    main()
    sys.exit(0)
