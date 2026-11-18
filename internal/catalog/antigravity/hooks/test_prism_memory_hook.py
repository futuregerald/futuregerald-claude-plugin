#!/usr/bin/env python3
import importlib.util
import json
import os
import stat
import subprocess
import sys
from pathlib import Path
from unittest import mock

import pytest

HOOK_PATH = Path(__file__).resolve().parent / "prism_memory_hook.py"


def load_hook_module():
    spec = importlib.util.spec_from_file_location("prism_memory_hook", HOOK_PATH)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def run_hook(args, payload, env=None):
    merged_env = os.environ.copy()
    if env:
        merged_env.update(env)
    stdin_data = payload if isinstance(payload, str) else json.dumps(payload)
    proc = subprocess.run(
        [sys.executable, str(HOOK_PATH), *args],
        input=stdin_data,
        text=True,
        capture_output=True,
        env=merged_env,
        timeout=10,
    )
    return proc


def test_session_start_agy_emits_ephemeral_message(tmp_path):
    proc = run_hook(
        ["--dialect", "agy", "--event", "SessionStart"],
        {"conversationId": "conv-start-1", "workspacePaths": [str(tmp_path)]},
    )
    assert proc.returncode == 0
    out = json.loads(proc.stdout)
    assert "injectSteps" in out
    msg = out["injectSteps"][0]["ephemeralMessage"]
    assert "knowledge_search" in msg
    assert "session_search_memory" in msg
    assert "session_load_context" in msg
    assert '"project": "dev"' in msg


def test_session_start_claude_emits_additional_context(tmp_path):
    proc = run_hook(
        ["--dialect", "claude", "--event", "SessionStart"],
        {"session_id": "sess-start-1", "cwd": str(tmp_path)},
    )
    assert proc.returncode == 0
    out = json.loads(proc.stdout)
    hso = out["hookSpecificOutput"]
    assert hso["hookEventName"] == "SessionStart"
    ctx = hso["additionalContext"]
    assert "mcp__prism__knowledge_search" in ctx
    assert "mcp__prism__session_load_context" in ctx
    assert 'project="dev"' in ctx


def test_derive_project_key_from_git_repo_and_sanitizes(tmp_path):
    repo = tmp_path / "My-Cool-Repo"
    repo.mkdir()
    subprocess.run(["git", "init", str(repo)], check=True, capture_output=True)
    mod = load_hook_module()
    assert mod.derive_project_key(str(repo)) == "my-cool-repo"

    weird = tmp_path / "Weird Repo!! Name"
    weird.mkdir()
    subprocess.run(["git", "init", str(weird)], check=True, capture_output=True)
    assert mod.derive_project_key(str(weird)) == "weird-repo-name"

    punct = tmp_path / "!!!"
    punct.mkdir()
    subprocess.run(["git", "init", str(punct)], check=True, capture_output=True)
    assert mod.derive_project_key(str(punct)) == "dev"


def test_post_tool_use_sets_dirty_only_on_file_mutating_tools_and_scrubs_path(tmp_path):
    mod = load_hook_module()
    conv_id = f"conv-mut-{tmp_path.name}"
    state_file = mod.state_path(conv_id)
    state_file.unlink(missing_ok=True)

    for non_mut in ("call_mcp_tool", "run_command", "Bash", "read_file", "Read", "Grep"):
        proc = run_hook(
            ["--dialect", "agy", "--event", "PostToolUse"],
            {
                "conversationId": conv_id,
                "workspacePaths": [str(tmp_path)],
                "toolCall": {"name": non_mut, "args": {"CommandLine": "ls"}},
            },
        )
        assert proc.returncode == 0
        assert json.loads(proc.stdout) == {}
        if state_file.exists():
            assert json.loads(state_file.read_text()).get("dirty") is False

    secret_path = str(tmp_path / "token=supersecret12345.py")
    proc = run_hook(
        ["--dialect", "agy", "--event", "PostToolUse"],
        {
            "conversationId": conv_id,
            "workspacePaths": [str(tmp_path)],
            "toolCall": {"name": "edit_file", "args": {"Path": secret_path}},
        },
    )
    assert proc.returncode == 0
    assert json.loads(proc.stdout) == {}
    state = json.loads(state_file.read_text())
    assert state["dirty"] is True
    assert all("supersecret12345" not in f for f in state["files_changed"])
    assert any("[REDACTED]" in f for f in state["files_changed"])
    state_file.unlink(missing_ok=True)


def test_post_tool_use_marks_saved_ledger_clears_dirty_and_files_changed(tmp_path):
    mod = load_hook_module()
    conv_id = f"conv-save-{tmp_path.name}"
    state_file = mod.state_path(conv_id)
    state_file.unlink(missing_ok=True)

    run_hook(
        ["--dialect", "agy", "--event", "PostToolUse"],
        {
            "conversationId": conv_id,
            "workspacePaths": [str(tmp_path)],
            "toolCall": {"name": "write_file", "args": {"Path": "/tmp/x.py"}},
        },
    )
    state_before = json.loads(state_file.read_text())
    assert state_before["dirty"] is True
    assert "/tmp/x.py" in state_before["files_changed"]

    run_hook(
        ["--dialect", "agy", "--event", "PostToolUse"],
        {
            "conversationId": conv_id,
            "workspacePaths": [str(tmp_path)],
            "toolCall": {
                "name": "call_mcp_tool",
                "args": {"ServerName": "prism", "ToolName": "session_save_ledger", "Arguments": {}},
            },
        },
    )
    state = json.loads(state_file.read_text())
    assert state["saved_ledger"] is True
    assert state["dirty"] is False
    assert state["files_changed"] == []

    run_hook(
        ["--dialect", "claude", "--event", "PostToolUse"],
        {
            "session_id": conv_id,
            "cwd": str(tmp_path),
            "tool_name": "Edit",
            "tool_input": {"file_path": "/tmp/y.py"},
        },
    )
    assert json.loads(state_file.read_text())["dirty"] is True

    run_hook(
        ["--dialect", "claude", "--event", "PostToolUse"],
        {
            "session_id": conv_id,
            "cwd": str(tmp_path),
            "tool_name": "mcp__prism__session_save_ledger",
            "tool_input": {},
        },
    )
    state = json.loads(state_file.read_text())
    assert state["saved_ledger"] is True
    assert state["dirty"] is False
    assert state["files_changed"] == []
    state_file.unlink(missing_ok=True)


def test_state_dir_and_file_permissions_and_id_capping(tmp_path):
    mod = load_hook_module()
    long_id = "conv-" + ("a" * 120)
    sanitized = mod.sanitize_conversation_id(long_id)
    assert len(sanitized) <= 64

    path = mod.state_path(long_id)
    path.unlink(missing_ok=True)
    mod.save_state(long_id, {"dirty": True, "files_changed": []})

    dir_mode = stat.S_IMODE(os.lstat(path.parent).st_mode)
    file_mode = stat.S_IMODE(os.lstat(path).st_mode)
    assert dir_mode == 0o700
    assert file_mode == 0o600
    assert path.parent.name == f"prism-hooks-{os.getuid()}"
    path.unlink(missing_ok=True)


def test_stop_clean_session_returns_empty(tmp_path):
    mod = load_hook_module()
    conv_id = f"conv-clean-{tmp_path.name}"
    mod.state_path(conv_id).unlink(missing_ok=True)

    proc = run_hook(
        ["--dialect", "agy", "--event", "Stop"],
        {"conversationId": conv_id, "workspacePaths": [str(tmp_path)], "terminationReason": "NO_TOOL_CALL"},
    )
    assert proc.returncode == 0
    assert json.loads(proc.stdout) == {}


def test_stop_dirty_prompts_once_then_guards_second_stop_and_clears_dirty_on_fallback(tmp_path):
    mod = load_hook_module()
    conv_id = f"conv-stop-{tmp_path.name}"
    state_file = mod.state_path(conv_id)
    state_file.unlink(missing_ok=True)

    run_hook(
        ["--dialect", "agy", "--event", "PostToolUse"],
        {
            "conversationId": conv_id,
            "workspacePaths": [str(tmp_path)],
            "toolCall": {"name": "edit_file", "args": {"Path": "/tmp/z.py"}},
        },
    )

    first = run_hook(
        ["--dialect", "agy", "--event", "Stop"],
        {"conversationId": conv_id, "workspacePaths": [str(tmp_path)], "terminationReason": "NO_TOOL_CALL"},
    )
    assert first.returncode == 0
    first_out = json.loads(first.stdout)
    assert first_out["decision"] == "continue"
    assert "session_save_ledger" in first_out["reason"]

    with mock.patch.object(mod, "fallback_save_to_prism", return_value=True) as mock_save:
        rc = mod.handle_event(
            "Stop",
            "agy",
            {"conversationId": conv_id, "workspacePaths": [str(tmp_path)], "terminationReason": "NO_TOOL_CALL"},
        )
        assert rc == 0
        assert mock_save.call_count == 1
    after = json.loads(state_file.read_text())
    assert after["saved_handoff"] is True
    assert after["dirty"] is False
    state_file.unlink(missing_ok=True)


def test_stop_claude_blocks_once_and_respects_stop_hook_active(tmp_path):
    mod = load_hook_module()
    conv_id = f"sess-stop-{tmp_path.name}"
    state_file = mod.state_path(conv_id)
    state_file.unlink(missing_ok=True)

    run_hook(
        ["--dialect", "claude", "--event", "PostToolUse"],
        {
            "session_id": conv_id,
            "cwd": str(tmp_path),
            "tool_name": "Write",
            "tool_input": {"file_path": "/tmp/c.py"},
        },
    )

    active = run_hook(
        ["--dialect", "claude", "--event", "Stop"],
        {"session_id": conv_id, "cwd": str(tmp_path), "stop_hook_active": True},
        env={"PRISM_SHIM_PATH": str(tmp_path / "nonexistent-shim.js")},
    )
    assert active.returncode == 0
    assert json.loads(active.stdout) == {}

    first = run_hook(
        ["--dialect", "claude", "--event", "Stop"],
        {"session_id": conv_id, "cwd": str(tmp_path), "stop_hook_active": False},
    )
    assert first.returncode == 0
    out = json.loads(first.stdout)
    assert out["decision"] == "block"
    assert "mcp__prism__session_save_ledger" in out["reason"]
    state_file.unlink(missing_ok=True)


def test_stop_skips_non_no_tool_call_termination_reason(tmp_path):
    mod = load_hook_module()
    conv_id = f"conv-term-{tmp_path.name}"
    state_file = mod.state_path(conv_id)
    state_file.unlink(missing_ok=True)

    run_hook(
        ["--dialect", "agy", "--event", "PostToolUse"],
        {
            "conversationId": conv_id,
            "workspacePaths": [str(tmp_path)],
            "toolCall": {"name": "edit_file", "args": {"Path": "/tmp/t.py"}},
        },
    )
    proc = run_hook(
        ["--dialect", "agy", "--event", "Stop"],
        {"conversationId": conv_id, "workspacePaths": [str(tmp_path)], "terminationReason": "USER_CANCELLED"},
    )
    assert proc.returncode == 0
    assert json.loads(proc.stdout) == {}
    state_file.unlink(missing_ok=True)


def test_secret_scrubbing_redacts_all_supported_patterns():
    mod = load_hook_module()
    google_key = "AIza" + ("0123456789abcdef0123456789abcdef012")
    brave_key = "BSA" + ("aBcDeFgHiJkLmNoPqRsTuVwX")
    sample = (
        "Authorization: bearer abcdef1234567890_secret "
        "AKIAIOSFODNN7EXAMPLE "
        "ghp_1234567890abcdefghij1234567890abcdef "
        "sk-ant-api03-abcdefghijklmnopqrstuvwx "
        f"{google_key} "
        f"{brave_key} "
        "-----BEGIN RSA PRIVATE KEY----- "
        "https://alice:s3cr3tp4ss@example.com/repo "
        "password=supersecretvalue123"
    )
    scrubbed = mod.scrub_secrets(sample)
    for secret in (
        "abcdef1234567890_secret",
        "AKIAIOSFODNN7EXAMPLE",
        "ghp_1234567890abcdefghij1234567890abcdef",
        "sk-ant-api03-abcdefghijklmnopqrstuvwx",
        google_key,
        brave_key,
        "BEGIN RSA PRIVATE KEY",
        "alice:s3cr3tp4ss",
        "supersecretvalue123",
    ):
        assert secret not in scrubbed
    assert "https://[REDACTED]@example.com/repo" in scrubbed


def test_fallback_save_to_prism_sends_handoff_schema_env_and_parses_jsonrpc(tmp_path, monkeypatch):
    mod = load_hook_module()
    capture_file = tmp_path / "captured.json"
    fake_shim = tmp_path / "fake_shim.js"
    fake_shim.write_text(
        "const fs = require('fs');\n"
        "const readline = require('readline');\n"
        "const msgs = [];\n"
        "const rl = readline.createInterface({ input: process.stdin });\n"
        "rl.on('line', (line) => {\n"
        "  if (!line.trim()) return;\n"
        "  const m = JSON.parse(line);\n"
        "  msgs.push(m);\n"
        "  if (m.id === 1) {\n"
        "    process.stdout.write(JSON.stringify({ jsonrpc: '2.0', id: 1, result: {} }) + '\\n');\n"
        "  } else if (m.id === 2) {\n"
        f"    fs.writeFileSync({json.dumps(str(capture_file))}, JSON.stringify({{ msgs, env: {{\n"
        "      PRISM_STORAGE: process.env.PRISM_STORAGE,\n"
        "      PRISM_ENABLE_DASHBOARD: process.env.PRISM_ENABLE_DASHBOARD,\n"
        "      PRISM_HDC_ENABLED: process.env.PRISM_HDC_ENABLED,\n"
        "      PRISM_ENABLE_HIVEMIND: process.env.PRISM_ENABLE_HIVEMIND,\n"
        "    }}));\n"
        "    process.stdout.write(JSON.stringify({ jsonrpc: '2.0', id: 2, result: { content: [] } }) + '\\n');\n"
        "  }\n"
        "});\n"
    )
    monkeypatch.setenv("PRISM_SHIM_PATH", str(fake_shim))
    for k in ("PRISM_STORAGE", "PRISM_ENABLE_DASHBOARD", "PRISM_HDC_ENABLED", "PRISM_ENABLE_HIVEMIND"):
        monkeypatch.delenv(k, raising=False)

    ok = mod.fallback_save_to_prism(
        "my-proj",
        "conv-42",
        {"files_changed": ["a.py", "b.py"]},
        str(tmp_path),
        tool_name="session_save_handoff",
    )
    assert ok is True
    captured = json.loads(capture_file.read_text())
    assert captured["env"] == {
        "PRISM_STORAGE": "local",
        "PRISM_ENABLE_DASHBOARD": "false",
        "PRISM_HDC_ENABLED": "true",
        "PRISM_ENABLE_HIVEMIND": "true",
    }
    call_msg = [m for m in captured["msgs"] if m.get("id") == 2][0]
    args = call_msg["params"]["arguments"]
    assert args["project"] == "my-proj"
    assert "last_summary" in args
    assert args["open_todos"] == ["Resume from uncommitted changes and verify pending tasks."]
    assert "conversation=conv-42" in args["key_context"]
    assert "a.py, b.py" in args["key_context"]
    assert "next_steps" not in args

    err_shim = tmp_path / "err_shim.js"
    err_shim.write_text(
        "const readline = require('readline');\n"
        "const rl = readline.createInterface({ input: process.stdin });\n"
        "rl.on('line', (line) => {\n"
        "  const m = JSON.parse(line);\n"
        "  if (m.id === 2) {\n"
        "    process.stdout.write(JSON.stringify({ jsonrpc: '2.0', id: 2, result: { isError: true } }) + '\\n');\n"
        "  }\n"
        "});\n"
    )
    monkeypatch.setenv("PRISM_SHIM_PATH", str(err_shim))
    assert mod.fallback_save_to_prism("my-proj", "conv-42", {"files_changed": []}, str(tmp_path)) is False


def test_session_end_and_precompact_trigger_fallback_when_dirty(tmp_path):
    mod = load_hook_module()
    conv_id = f"sess-end-{tmp_path.name}"
    state_file = mod.state_path(conv_id)
    mod.save_state(conv_id, {
        "dirty": True,
        "saved_ledger": False,
        "saved_handoff": False,
        "prompted_once": False,
        "files_changed": ["/tmp/foo.py"],
        "project": "demo",
    })
    with mock.patch.object(mod, "fallback_save_to_prism", return_value=True) as mock_save:
        rc = mod.handle_event("SessionEnd", "claude", {"session_id": conv_id, "cwd": str(tmp_path)})
        assert rc == 0
        assert mock_save.call_count == 1
        rc2 = mod.handle_event("PreCompact", "claude", {"session_id": conv_id, "cwd": str(tmp_path)})
        assert rc2 == 0
        assert mock_save.call_count == 2
    state_file.unlink(missing_ok=True)


def test_invalid_and_empty_stdin_fails_open():
    for bad in ("", "   ", "{not valid json", "[]"):
        proc = run_hook(["--dialect", "agy", "--event", "Stop"], bad)
        assert proc.returncode == 0
        assert json.loads(proc.stdout) == {}
