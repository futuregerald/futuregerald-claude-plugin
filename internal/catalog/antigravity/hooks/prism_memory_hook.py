#!/usr/bin/env python3
"""Dual-dialect Prism Memory lifecycle hook for Antigravity and Claude Code."""

import argparse
import hashlib
import json
import os
import re
import select
import stat
import subprocess
import sys
import tempfile
import time
from pathlib import Path

MUTATING_TOOLS = frozenset({
    "edit_file",
    "replace_file_content",
    "write_file",
    "write_to_file",
    "Edit",
    "Write",
    "MultiEdit",
})

SECRET_SUBSTITUTIONS = [
    (re.compile(r"(?i)(Bearer\s+)[A-Za-z0-9._~+/=-]{8,}"), r"\1[REDACTED]"),
    (re.compile(r"\bAKIA[0-9A-Z]{16}\b"), "[REDACTED]"),
    (re.compile(r"\b(?:gh[pousr]|github_pat)_[A-Za-z0-9_]{20,}\b"), "[REDACTED]"),
    (re.compile(r"\bsk-(?:ant-)?[A-Za-z0-9_-]{16,}\b"), "[REDACTED]"),
    (re.compile(r"\bxox[baprs]-[A-Za-z0-9-]{10,}\b"), "[REDACTED]"),
    (re.compile(r"\bsbp_[A-Za-z0-9]{20,}\b"), "[REDACTED]"),
    (re.compile(r"\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b"), "[REDACTED]"),
    (re.compile(r"\bAIza[0-9A-Za-z_-]{35}\b"), "[REDACTED]"),
    (re.compile(r"\bBSA[A-Za-z0-9_-]{20,}\b"), "[REDACTED]"),
    (re.compile(r"-----BEGIN [A-Z ]*PRIVATE KEY-----"), "[REDACTED]"),
    (re.compile(r"(https?://)[^/\s:@]+:[^/\s@]+@"), r"\1[REDACTED]@"),
    (re.compile(r"(?i)((?:api[_-]?key|token|secret|password|passwd|pwd)\s*[:=]\s*['\"]?)[^\s'\"\\,;]{6,}(['\"]?)"), r"\1[REDACTED]\2"),
]

DEFAULT_SHIM_PATH = os.path.expanduser("~/Documents/dev/prism-mcp/dist/shim.js")
FALLBACK_SHIM_PATH = "/opt/prism-mcp/dist/shim.js"


def scrub_secrets(text: str) -> str:
    if not text:
        return text
    result = text
    for pattern, replacement in SECRET_SUBSTITUTIONS:
        result = pattern.sub(replacement, result)
    return result


def ensure_state_dir() -> Path:
    uid = os.getuid()
    state_dir = Path(tempfile.gettempdir()) / f"prism-hooks-{uid}"
    if state_dir.exists() or state_dir.is_symlink():
        st = os.lstat(state_dir)
        if stat.S_ISLNK(st.st_mode) or not stat.S_ISDIR(st.st_mode) or st.st_uid != uid:
            raise PermissionError(f"Unsafe state directory: {state_dir}")
        os.chmod(state_dir, 0o700)
    else:
        os.makedirs(state_dir, mode=0o700, exist_ok=True)
        st = os.lstat(state_dir)
        if stat.S_ISLNK(st.st_mode) or not stat.S_ISDIR(st.st_mode) or st.st_uid != uid:
            raise PermissionError(f"Unsafe state directory: {state_dir}")
        os.chmod(state_dir, 0o700)
    return state_dir


def sanitize_conversation_id(conversation_id: str) -> str:
    raw = conversation_id or "default"
    safe_id = "".join(c for c in raw if c.isalnum() or c in "-_")
    if not safe_id:
        safe_id = "default"
    if len(safe_id) > 64:
        digest = hashlib.sha256(raw.encode("utf-8", errors="replace")).hexdigest()[:16]
        safe_id = f"{safe_id[:47]}-{digest}"
    return safe_id


def state_path(conversation_id: str) -> Path:
    return ensure_state_dir() / f"prism-session-state-{sanitize_conversation_id(conversation_id)}.json"


def load_state(conversation_id: str) -> dict:
    default = {
        "dirty": False,
        "saved_ledger": False,
        "saved_handoff": False,
        "prompted_once": False,
        "files_changed": [],
        "project": "dev",
    }
    try:
        path = state_path(conversation_id)
    except Exception:
        return default
    if not path.exists():
        return default
    try:
        data = json.loads(path.read_text())
        if isinstance(data, dict):
            default.update(data)
            if not isinstance(default.get("files_changed"), list):
                default["files_changed"] = []
            return default
    except Exception:
        pass
    return default


def save_state(conversation_id: str, state: dict) -> None:
    tmp_path = None
    try:
        target = state_path(conversation_id)
        state_dir = target.parent
        with tempfile.NamedTemporaryFile(
            mode="w",
            encoding="utf-8",
            dir=str(state_dir),
            prefix=".prism-state-",
            suffix=".tmp",
            delete=False,
        ) as tf:
            tmp_path = tf.name
            os.chmod(tmp_path, 0o600)
            tf.write(json.dumps(state))
            tf.flush()
        os.replace(tmp_path, target)
        tmp_path = None
    except Exception:
        if tmp_path:
            try:
                os.unlink(tmp_path)
            except Exception:
                pass


def derive_project_key(cwd: str) -> str:
    if not cwd or not os.path.isdir(cwd):
        return "dev"
    try:
        proc = subprocess.run(
            ["git", "-C", cwd, "rev-parse", "--path-format=absolute", "--git-common-dir"],
            capture_output=True,
            text=True,
            timeout=2,
        )
        if proc.returncode != 0:
            return "dev"
        common_dir = proc.stdout.strip()
        if not common_dir:
            return "dev"
        resolved = Path(common_dir)
        if not resolved.is_absolute():
            resolved = (Path(cwd) / resolved).resolve()
        raw_name = resolved.parent.name.strip().lower()
        sanitized = re.sub(r"[^a-z0-9._-]+", "-", raw_name).strip(".-")
        return sanitized if sanitized else "dev"
    except Exception:
        return "dev"


def extract_cwd(payload: dict) -> str:
    workspace_paths = payload.get("workspacePaths")
    if isinstance(workspace_paths, list) and workspace_paths:
        first = workspace_paths[0]
        if isinstance(first, str) and first.strip():
            return first.strip()
    cwd = payload.get("cwd")
    if isinstance(cwd, str) and cwd.strip():
        return cwd.strip()
    return os.getcwd()


def extract_conversation_id(payload: dict) -> str:
    for key in ("conversationId", "session_id", "sessionId", "conversation_id"):
        val = payload.get(key)
        if isinstance(val, str) and val.strip():
            return val.strip()
    return "default"


def extract_tool_info(payload: dict) -> tuple[str, dict]:
    tool_call = payload.get("toolCall")
    if isinstance(tool_call, dict):
        name = str(tool_call.get("name") or "")
        args = tool_call.get("args")
        return name, args if isinstance(args, dict) else {}
    name = str(payload.get("tool_name") or "")
    args = payload.get("tool_input")
    return name, args if isinstance(args, dict) else {}


def resolve_shim_path() -> str | None:
    env_path = os.environ.get("PRISM_SHIM_PATH")
    if env_path:
        return env_path if os.path.isfile(env_path) else None
    for candidate in (DEFAULT_SHIM_PATH, FALLBACK_SHIM_PATH):
        if candidate and os.path.isfile(candidate):
            return candidate
    return None


def fallback_save_to_prism(
    project: str,
    conversation_id: str,
    state: dict,
    cwd: str,
    tool_name: str = "session_save_handoff",
) -> bool:
    shim_path = resolve_shim_path()
    if not shim_path:
        return False

    files = [scrub_secrets(str(f)) for f in (state.get("files_changed") or [])[:50]]
    summary = scrub_secrets(
        f"Auto-saved fallback handoff for project '{project}' (conversation {conversation_id}). "
        f"Files touched: {', '.join(files) if files else 'none recorded'}."
    )
    if tool_name == "session_save_ledger":
        tool_args = {
            "project": project,
            "conversation_id": conversation_id,
            "summary": summary,
            "files_changed": files,
        }
    else:
        tool_args = {
            "project": project,
            "last_summary": summary,
            "open_todos": ["Resume from uncommitted changes and verify pending tasks."],
            "key_context": scrub_secrets(f"conversation={conversation_id}; files_changed={', '.join(files)}"),
        }

    messages = [
        {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {"name": "prism-memory-hook", "version": "1.0.0"},
            },
        },
        {
            "jsonrpc": "2.0",
            "method": "notifications/initialized",
            "params": {},
        },
        {
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/call",
            "params": {
                "name": tool_name,
                "arguments": tool_args,
            },
        },
    ]
    wire = "\n".join(json.dumps(m) for m in messages) + "\n"
    env = os.environ.copy()
    env.setdefault("PRISM_STORAGE", "local")
    env.setdefault("PRISM_ENABLE_DASHBOARD", "false")
    env.setdefault("PRISM_HDC_ENABLED", "true")
    env.setdefault("PRISM_ENABLE_HIVEMIND", "true")

    proc = None
    try:
        proc = subprocess.Popen(
            ["node", shim_path],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            cwd=cwd if cwd and os.path.isdir(cwd) else None,
            env=env,
        )
        if proc.stdin is None or proc.stdout is None:
            return False
        proc.stdin.write(wire.encode("utf-8"))
        proc.stdin.flush()

        deadline = time.monotonic() + 2.0
        fd = proc.stdout.fileno()
        buf = b""
        while True:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                return False
            ready, _, _ = select.select([fd], [], [], remaining)
            if not ready:
                return False
            chunk = os.read(fd, 4096)
            if not chunk:
                return False
            buf += chunk
            while b"\n" in buf:
                raw_line, buf = buf.split(b"\n", 1)
                line = raw_line.decode("utf-8", errors="replace").strip()
                if not line:
                    continue
                try:
                    msg = json.loads(line)
                except Exception:
                    continue
                if isinstance(msg, dict) and msg.get("id") == 2:
                    if "error" in msg and msg["error"] is not None:
                        return False
                    res = msg.get("result")
                    if isinstance(res, dict) and res.get("isError") is True:
                        return False
                    return True
    except Exception:
        return False
    finally:
        if proc is not None:
            try:
                if proc.stdin:
                    proc.stdin.close()
            except Exception:
                pass
            try:
                if proc.stdout:
                    proc.stdout.close()
            except Exception:
                pass
            try:
                proc.terminate()
                proc.wait(timeout=0.5)
            except Exception:
                try:
                    proc.kill()
                    proc.wait(timeout=0.5)
                except Exception:
                    pass


def build_session_start_output(dialect: str, project: str) -> dict:
    proj_json = json.dumps(project)
    if dialect == "claude":
        msg = (
            "MANDATORY PRISM-FIRST & CONTEXT LOAD: "
            "(1) Before grep/index/web search for any prior info, memories, decisions, or history, "
            "search Prism FIRST via mcp__prism__knowledge_search / mcp__prism__session_search_memory "
            "(omit 'project' filter on initial searches so cross-repo memories surface). "
            f"(2) On engineering/debugging/resuming tasks, call mcp__prism__session_load_context(project={proj_json}, level=\"standard\") on Turn 1."
        )
        return {
            "hookSpecificOutput": {
                "hookEventName": "SessionStart",
                "additionalContext": msg,
            }
        }
    msg = (
        "MANDATORY PRISM-FIRST & CONTEXT LOAD: "
        "(1) Before grep/index/web search for any prior info, memories, decisions, or history, "
        "search Prism FIRST via call_mcp_tool (ServerName: 'prism', knowledge_search / session_search_memory "
        "— omit 'project' filter on initial searches so cross-repo memories surface). "
        f"(2) On engineering/debugging/resuming tasks, call session_load_context({{\"project\": {proj_json}, \"level\": \"standard\"}}) on Turn 1."
    )
    return {"injectSteps": [{"ephemeralMessage": msg}]}


def handle_event(event: str, dialect: str, payload: dict) -> int:
    if not isinstance(payload, dict):
        print("{}")
        return 0

    conversation_id = extract_conversation_id(payload)
    cwd = extract_cwd(payload)
    project = derive_project_key(cwd)

    if event == "SessionStart":
        print(json.dumps(build_session_start_output(dialect, project)))
        return 0

    if event == "PostToolUse":
        if payload.get("error"):
            print("{}")
            return 0
        tool_name, args = extract_tool_info(payload)
        if not tool_name:
            print("{}")
            return 0

        state = load_state(conversation_id)
        state["project"] = project
        changed = False

        if tool_name in MUTATING_TOOLS:
            state["dirty"] = True
            state["saved_ledger"] = False
            file_path = (
                args.get("Path")
                or args.get("TargetFile")
                or args.get("file_path")
                or args.get("path")
                or ""
            )
            if isinstance(file_path, str) and file_path.strip():
                fp = scrub_secrets(file_path.strip())
                files = state.get("files_changed") or []
                if fp not in files and len(files) < 50:
                    files.append(fp)
                    state["files_changed"] = files
            changed = True
        elif tool_name == "call_mcp_tool":
            server = str(args.get("ServerName") or args.get("server_name") or "").lower()
            mcp_tool = str(args.get("ToolName") or args.get("tool_name") or "")
            if server == "prism":
                if mcp_tool == "session_save_ledger":
                    state["saved_ledger"] = True
                    state["dirty"] = False
                    state["prompted_once"] = False
                    state["files_changed"] = []
                    changed = True
                elif mcp_tool == "session_save_handoff":
                    state["saved_handoff"] = True
                    changed = True
        elif tool_name == "mcp__prism__session_save_ledger":
            state["saved_ledger"] = True
            state["dirty"] = False
            state["prompted_once"] = False
            state["files_changed"] = []
            changed = True
        elif tool_name == "mcp__prism__session_save_handoff":
            state["saved_handoff"] = True
            changed = True

        if changed:
            save_state(conversation_id, state)
        print("{}")
        return 0

    if event == "Stop":
        term_reason = str(payload.get("terminationReason") or "")
        continuation_active = bool(payload.get("continuationActive") or payload.get("continuation_active"))
        stop_hook_active = bool(payload.get("stop_hook_active") or payload.get("stopHookActive"))

        state = load_state(conversation_id)
        dirty = bool(state.get("dirty"))
        saved_ledger = bool(state.get("saved_ledger"))
        prompted_once = bool(state.get("prompted_once"))

        if (
            (term_reason and term_reason != "NO_TOOL_CALL")
            or continuation_active
            or stop_hook_active
            or (not dirty)
            or saved_ledger
            or prompted_once
        ):
            if dirty and not saved_ledger and (continuation_active or stop_hook_active or prompted_once):
                if fallback_save_to_prism(project, conversation_id, state, cwd, tool_name="session_save_handoff"):
                    state["saved_handoff"] = True
                    state["dirty"] = False
                    save_state(conversation_id, state)
            print("{}")
            return 0

        state["prompted_once"] = True
        state["project"] = project
        save_state(conversation_id, state)

        proj_json = json.dumps(project)
        conv_json = json.dumps(conversation_id)
        if dialect == "claude":
            reason = (
                f"Auto-save required before finishing: call mcp__prism__session_save_ledger "
                f"(project={proj_json}) and mcp__prism__session_save_handoff if work remains."
            )
            print(json.dumps({"decision": "block", "reason": reason}))
        else:
            reason = (
                "Auto-save required before finishing: call call_mcp_tool(ServerName='prism', "
                f"ToolName='session_save_ledger', Arguments={{\"project\": {proj_json}, \"conversation_id\": {conv_json}, "
                "\"summary\": \"...\", \"files_changed\": [...]}) and, if work remains, 'session_save_handoff'."
            )
            print(json.dumps({"decision": "continue", "reason": reason}))
        return 0

    if event in ("SessionEnd", "PreCompact"):
        state = load_state(conversation_id)
        dirty = bool(state.get("dirty"))
        saved_ledger = bool(state.get("saved_ledger"))
        if dirty and not saved_ledger:
            if fallback_save_to_prism(project, conversation_id, state, cwd, tool_name="session_save_handoff"):
                state["saved_handoff"] = True
                save_state(conversation_id, state)
        print("{}")
        return 0

    print("{}")
    return 0


def parse_args(argv: list[str]) -> tuple[str, str]:
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument("--dialect", choices=("agy", "claude"), default="agy")
    parser.add_argument("--event", default="")
    parser.add_argument("positional_event", nargs="?", default="")
    ns, _ = parser.parse_known_args(argv)
    event = ns.event or ns.positional_event or ""
    return ns.dialect, event


def main() -> int:
    dialect, event = parse_args(sys.argv[1:])
    try:
        raw = sys.stdin.read()
    except Exception:
        print("{}")
        return 0

    if not raw or not raw.strip():
        print("{}")
        return 0

    try:
        payload = json.loads(raw)
    except Exception:
        print("{}")
        return 0

    if not isinstance(payload, dict):
        print("{}")
        return 0

    if not event:
        event = str(payload.get("hookEvent") or payload.get("hook_event_name") or "")

    return handle_event(event, dialect, payload)


if __name__ == "__main__":
    raise SystemExit(main())
