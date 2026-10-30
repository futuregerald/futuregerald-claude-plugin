#!/usr/bin/env python3
import json
import os
import re
import shutil
import sys
import tempfile
from datetime import datetime, timezone
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from breaker_rules import (
    is_authorized,
    read_ranges_overlap,
    read_coverage,
    is_re_read,
    normalize_command,
    is_test_runner,
    is_code_file,
    decode_json_strings,
    rule_mode,
    probe_target_path,
    append_breaker_log,
    extract_conversation_id,
    compaction_step_indices,
    context_estimate_tokens,
    load_baton_state,
    save_baton_state,
)

STRIP_KEYS = {"toolAction", "toolSummary", "WaitMsBeforeAsync", "Description", "Instruction", "IsDaemon"}

def deny(name, s_idx, r):
    sys.stdout.write(json.dumps({"decision": "deny", "reason": r}))
    sys.exit(0)

def check_and_handle_trigger(rule, target, reason, tool_name, step_idx, conv_id, is_subagent, collector=None, stop_on_enforce=True):
    mode = rule_mode(rule)
    if mode == "off":
        return False
    entry = {
        "ts": datetime.now(timezone.utc).isoformat(),
        "conversation_id": conv_id,
        "is_subagent": is_subagent,
        "rule": rule,
        "target": target,
        "mode": mode,
    }
    append_breaker_log(entry)
    if collector is not None:
        collector.append({
            "rule": rule,
            "target": target,
            "reason": reason,
            "mode": mode,
        })
    if stop_on_enforce and mode == "enforce":
        deny(tool_name, step_idx, reason)
    return True

def canonicalize_call(name, args):
    if not isinstance(args, dict):
        args = {}
    cleaned_args = {k: v for k, v in args.items() if k not in STRIP_KEYS}
    decoded = decode_json_strings(cleaned_args)
    canon_name = name
    canon_args = {}
    if canon_name in ("read_file", "view_file"):
        canon_name = "READ_FILE"
        if "AbsolutePath" in decoded:
            canon_args["Path"] = decoded["AbsolutePath"]
        elif "Path" in decoded:
            canon_args["Path"] = decoded["Path"]
        if "StartLine" in decoded and decoded["StartLine"] is not None:
            try:
                canon_args["StartLine"] = int(decoded["StartLine"])
            except Exception:
                pass
        if "EndLine" in decoded and decoded["EndLine"] is not None:
            try:
                canon_args["EndLine"] = int(decoded["EndLine"])
            except Exception:
                pass
        if "ByteOffset" in decoded and decoded["ByteOffset"] is not None:
            try:
                canon_args["ByteOffset"] = int(decoded["ByteOffset"])
            except Exception:
                pass
    elif canon_name in ("edit_file", "replace_file_content"):
        canon_name = "EDIT_FILE"
        if "TargetFile" in decoded:
            canon_args["Path"] = decoded["TargetFile"]
        elif "Path" in decoded:
            canon_args["Path"] = decoded["Path"]
        elif "AbsolutePath" in decoded:
            canon_args["Path"] = decoded["AbsolutePath"]
        for k, v in sorted(decoded.items()):
            if k not in ("TargetFile", "Path", "AbsolutePath"):
                canon_args[k] = v
    elif canon_name in ("write_file", "write_to_file"):
        canon_name = "WRITE_FILE"
        if "TargetFile" in decoded:
            canon_args["Path"] = decoded["TargetFile"]
        elif "Path" in decoded:
            canon_args["Path"] = decoded["Path"]
        elif "AbsolutePath" in decoded:
            canon_args["Path"] = decoded["AbsolutePath"]
        for k, v in sorted(decoded.items()):
            if k not in ("TargetFile", "Path", "AbsolutePath"):
                canon_args[k] = v
    elif canon_name == "run_command":
        canon_name = "run_command"
        if "CommandLine" in decoded:
            cmd = decoded["CommandLine"]
            if isinstance(cmd, str):
                canon_args["CommandLine"] = cmd.strip()
            else:
                canon_args["CommandLine"] = cmd
        for k, v in sorted(decoded.items()):
            if k != "CommandLine":
                canon_args[k] = v
    elif canon_name == "call_mcp_tool":
        canon_name = "call_mcp_tool"
        canon_args = decoded
    else:
        canon_args = decoded
    return canon_name, canon_args

def effective_tool_calls(step, is_last_planner_step, step_idx, in_c_name, in_c_args):
    t_calls = step.get("tool_calls") or []
    if not is_last_planner_step or not t_calls:
        return t_calls
    s_idx_val = step.get("step_index")
    if isinstance(s_idx_val, int) and s_idx_val < step_idx <= s_idx_val + len(t_calls):
        inflight_pos = step_idx - s_idx_val - 1
        cand = t_calls[inflight_pos]
        cand_name, cand_args = canonicalize_call(cand.get("name", ""), cand.get("args") or {})
        if cand_name == in_c_name and cand_args == in_c_args:
            return t_calls[:inflight_pos]
    if isinstance(s_idx_val, int) and step_idx == s_idx_val:
        if len(t_calls) > 0:
            cand = t_calls[-1]
            cand_name, cand_args = canonicalize_call(cand.get("name", ""), cand.get("args") or {})
            if cand_name == in_c_name and cand_args == in_c_args:
                return t_calls[:-1]
    return t_calls

def _run_breaker(payload, stop_on_enforce=True, collector=None):
    passthrough_out = os.environ.get("BREAKER_PASSTHROUGH_OUT", "")
    if not isinstance(payload, dict):
        if stop_on_enforce:
            if passthrough_out:
                sys.stdout.write(passthrough_out)
            sys.exit(0)
        return collector or []

    step_idx = payload.get("stepIdx", 0)
    tool_call = payload.get("toolCall")
    if not isinstance(tool_call, dict):
        tool_call = {}
    tool_name = tool_call.get("name", "")
    tool_args = tool_call.get("args")
    if not isinstance(tool_args, dict):
        tool_args = {}
    conv_id = extract_conversation_id(payload)
    state_dir = os.environ.get("BREAKER_STATE_DIR", "~/.gemini/state")

    t_path_str = payload.get("transcriptPath")
    if not t_path_str:
        if not conv_id:
            if stop_on_enforce:
                if passthrough_out:
                    sys.stdout.write(passthrough_out)
                sys.exit(0)
            return collector or []
        t_path = Path.home() / ".gemini" / "antigravity" / "brain" / conv_id / ".system_generated" / "logs" / "transcript.jsonl"
    else:
        t_path = Path(str(t_path_str))

    if t_path.name == "transcript_full.jsonl":
        compact_path = t_path.parent / "transcript.jsonl"
        full_path = t_path
        if compact_path.is_file():
            t_path = compact_path
    else:
        full_path = t_path.parent / "transcript_full.jsonl"

    if not t_path.is_file():
        if stop_on_enforce:
            if passthrough_out:
                sys.stdout.write(passthrough_out)
            sys.exit(0)
        return collector or []

    lines = []
    try:
        with open(t_path, "r", encoding="utf-8") as f:
            for line in f:
                line_str = line.strip()
                if line_str:
                    lines.append(line_str)
    except Exception:
        if stop_on_enforce:
            if passthrough_out:
                sys.stdout.write(passthrough_out)
            sys.exit(0)
        return collector or []

    full_lines = None
    full_lines_bytes = []
    if full_path.is_file():
        try:
            with open(full_path, "rb") as ff:
                full_lines_bytes = [len(bline) for bline in ff]
        except Exception:
            full_lines_bytes = []
    if not full_lines_bytes:
        try:
            with open(t_path, "rb") as tf:
                full_lines_bytes = [len(bline) for bline in tf]
        except Exception:
            full_lines_bytes = []

    steps = []
    for idx, l in enumerate(lines):
        try:
            step = json.loads(l)
        except Exception:
            continue
        trunc = step.get("truncated_fields") or []
        if "tool_calls" in trunc and full_path.is_file():
            if full_lines is None:
                try:
                    with open(full_path, "r", encoding="utf-8") as ff:
                        full_lines = [fl.strip() for fl in ff if fl.strip()]
                except Exception:
                    full_lines = []
            if full_lines and idx < len(full_lines):
                try:
                    full_step = json.loads(full_lines[idx])
                    step["tool_calls"] = full_step.get("tool_calls", step.get("tool_calls", []))
                except Exception:
                    pass
        steps.append(step)

    is_subagent = (len(steps) > 0) and not any(s.get("type") == "USER_INPUT" for s in steps)
    in_c_name, in_c_args = canonicalize_call(tool_name, tool_args)

    last_global_planner_idx = -1
    for idx, s in enumerate(steps):
        if s.get("type") == "PLANNER_RESPONSE":
            last_global_planner_idx = idx
        elif s.get("type") == "USER_INPUT":
            last_global_planner_idx = -1

    if is_subagent:
        subagent_explore_count = 0
        for idx, s in enumerate(steps):
            if s.get("type") == "PLANNER_RESPONSE":
                t_calls = effective_tool_calls(s, idx == last_global_planner_idx, step_idx, in_c_name, in_c_args)
                for c in t_calls:
                    c_name, c_args = canonicalize_call(c.get("name", ""), c.get("args") or {})
                    if c_name in ("READ_FILE", "call_mcp_tool"):
                        subagent_explore_count += 1
                    elif c_name == "run_command":
                        cmd = str(c_args.get("CommandLine", ""))
                        if not is_test_runner(cmd):
                            subagent_explore_count += 1

        in_is_exploratory = (
            in_c_name in ("READ_FILE", "call_mcp_tool") or
            (in_c_name == "run_command" and not is_test_runner(str(in_c_args.get("CommandLine", ""))))
        )
        explore_max = int(os.environ.get("BREAKER_SUBAGENT_EXPLORE_MAX", 100))
        if in_is_exploratory and subagent_explore_count >= explore_max:
            target = f"{tool_name}:{subagent_explore_count}/{explore_max}"
            reason = f"CIRCUIT BREAKER (Subagent Exploration Budget Reached): You have reached the maximum exploration budget ({explore_max}) in this subagent. Stop reading files or running exploratory commands. Immediately call write_file / write_to_file to save your digest or handoff with the evidence gathered so far, then finish your turn."
            check_and_handle_trigger("EXPLORE", target, reason, tool_name, step_idx, conv_id, is_subagent, collector=collector, stop_on_enforce=stop_on_enforce)

        consecutive_test_failures = 0
        for idx, s in enumerate(steps):
            if s.get("type") == "PLANNER_RESPONSE":
                t_calls = effective_tool_calls(s, idx == last_global_planner_idx, step_idx, in_c_name, in_c_args)
                for c in t_calls:
                    c_name, c_args = canonicalize_call(c.get("name", ""), c.get("args") or {})
                    if c_name == "run_command":
                        cmd = str(c_args.get("CommandLine", ""))
                        if is_test_runner(cmd):
                            for next_s in steps[idx + 1:]:
                                if next_s.get("type") == "PLANNER_RESPONSE":
                                    break
                                if next_s.get("type") == "GENERIC":
                                    content = next_s.get("content", "")
                                    if "The command exited with code 0." in content:
                                        consecutive_test_failures = 0
                                        break
                                    elif re.search(r"The command exited with code [1-9]\d*\.", content):
                                        consecutive_test_failures += 1
                                        break

        strikes_max = int(os.environ.get("BREAKER_TEST_STRIKES_MAX", 4))
        if consecutive_test_failures >= strikes_max and (in_c_name in ("run_command", "EDIT_FILE")):
            target = f"{tool_name}:{consecutive_test_failures}/{strikes_max}"
            reason = "CIRCUIT BREAKER (4-Strike Test-Fix Escalation): Tests have failed 4 consecutive times in this subagent. Stop editing code or re-running tests. Immediately write your 4-section handoff (including the failing test output and diff) using write_file / write_to_file and return control to the Lead Orchestrator for root-cause escalation."
            check_and_handle_trigger("STRIKES", target, reason, tool_name, step_idx, conv_id, is_subagent, collector=collector, stop_on_enforce=stop_on_enforce)

        step_max = int(os.environ.get("BREAKER_SUBAGENT_STEP_MAX", 130))
        if step_max > 0:
            planner_count = sum(1 for s in steps if s.get("type") == "PLANNER_RESPONSE")
            if planner_count >= step_max:
                is_handoff_write = False
                if tool_name in ("write_file", "write_to_file", "edit_file", "replace_file_content"):
                    p = str(tool_args.get("Path") or tool_args.get("TargetFile") or tool_args.get("AbsolutePath") or "")
                    p_low = p.lower()
                    if "handoff" in p_low or (p_low.endswith(".md") and ("/tmp/" in p_low or "/brain/" in p_low)):
                        is_handoff_write = True
                if tool_name != "send_message" and not is_handoff_write:
                    target = f"planner_turns:{planner_count}/{step_max}"
                    reason = f"CIRCUIT BREAKER (Subagent Step Cap): {planner_count}/{step_max} planner turns used: write your 4-section handoff and return via send_message"
                    check_and_handle_trigger("H2", target, reason, tool_name, step_idx, conv_id, is_subagent, collector=collector, stop_on_enforce=stop_on_enforce)

    last_user_idx = -1
    last_user_content = ""
    for i, s in enumerate(steps):
        if s.get("type") == "USER_INPUT":
            last_user_idx = i
            last_user_content = s.get("content") or ""

    if not is_subagent:
        comp_indices = compaction_step_indices(steps)
        current_segment = len(comp_indices)
        estimate = context_estimate_tokens(full_lines_bytes, comp_indices)
        baton_state = load_baton_state(conv_id, current_segment, state_dir=state_dir)

        in_p = str(tool_args.get("Path") or tool_args.get("TargetFile") or tool_args.get("AbsolutePath") or "")
        is_active_context_path = in_p.endswith("-ACTIVE_CONTEXT.md") or in_p.endswith("/ACTIVE_CONTEXT.md") or in_p == "ACTIVE_CONTEXT.md"

        if is_active_context_path and tool_name in ("write_file", "write_to_file", "edit_file", "replace_file_content"):
            baton_state["handoff_at_estimate"] = estimate
            save_baton_state(conv_id, baton_state, state_dir=state_dir)
            if stop_on_enforce:
                if passthrough_out:
                    sys.stdout.write(passthrough_out)
                sys.exit(0)
            return collector or []

        handoff_at_estimate = baton_state.get("handoff_at_estimate")
        is_unlocked = (handoff_at_estimate is not None and estimate <= handoff_at_estimate + 50000)

        baton_block_tokens = int(os.environ.get("BATON_BLOCK_TOKENS", 235000))
        baton_warn_tokens = int(os.environ.get("BATON_WARN_TOKENS", 205000))

        if baton_block_tokens > 0 and estimate >= baton_block_tokens and not is_unlocked:
            is_prism_call = (tool_name == "call_mcp_tool" and tool_args.get("ServerName") == "prism")
            is_active_context_read = (tool_name in ("read_file", "view_file") and is_active_context_path)
            if not (is_prism_call or is_active_context_read):
                target = f"estimate:{estimate}/{baton_block_tokens}"
                reason = f"CIRCUIT BREAKER (Baton Block): Context estimate {estimate} >= {baton_block_tokens}. Only ACTIVE_CONTEXT.md handoff and Prism saves are allowed. Hand off now."
                check_and_handle_trigger("BATON", target, reason, tool_name, step_idx, conv_id, is_subagent, collector=collector, stop_on_enforce=stop_on_enforce)

        warned_at_estimate = baton_state.get("warned_at_estimate")
        if baton_warn_tokens > 0 and baton_warn_tokens <= estimate < baton_block_tokens and not is_unlocked and warned_at_estimate is None:
            is_auth = is_authorized(last_user_content, conv_id, state_dir=state_dir)
            prev_is_sys = False
            if last_global_planner_idx > 0 and steps[last_global_planner_idx - 1].get("type") == "SYSTEM_MESSAGE":
                prev_is_sys = True
            elif steps and steps[-1].get("type") == "SYSTEM_MESSAGE":
                prev_is_sys = True

            trigger_warn = (not is_auth) or (tool_name == "invoke_subagent" or prev_is_sys or estimate >= baton_warn_tokens + 10000)
            if trigger_warn:
                target = f"estimate:{estimate}/{baton_warn_tokens}"
                reason = f"CIRCUIT BREAKER (Baton Warn): Context estimate {estimate} >= {baton_warn_tokens}. Hand off at the next natural break using baton-pass."
                mode = rule_mode("BATON")
                if mode in ("enforce", "log"):
                    baton_state["warned_at_estimate"] = estimate
                    save_baton_state(conv_id, baton_state, state_dir=state_dir)
                check_and_handle_trigger("BATON", target, reason, tool_name, step_idx, conv_id, is_subagent, collector=collector, stop_on_enforce=stop_on_enforce)

    if not is_subagent and last_user_idx >= 0 and is_authorized(last_user_content, conv_id, state_dir=state_dir):
        edit_max = int(os.environ.get("BREAKER_MAIN_EDIT_MAX", 6))
        if edit_max > 0:
            in_p = str(tool_args.get("Path") or tool_args.get("TargetFile") or tool_args.get("AbsolutePath") or "")
            if tool_name in ("edit_file", "replace_file_content", "write_file", "write_to_file") and is_code_file(in_p):
                post_user_steps = steps[last_user_idx + 1:]
                post_user_planner_indices = [i for i, s in enumerate(post_user_steps) if s.get("type") == "PLANNER_RESPONSE"]
                last_post_user_planner_idx = post_user_planner_indices[-1] if post_user_planner_indices else -1
                code_edits = 0
                for i, s in enumerate(post_user_steps):
                    if s.get("type") == "PLANNER_RESPONSE":
                        is_last = (i == last_post_user_planner_idx)
                        t_calls = effective_tool_calls(s, is_last, step_idx, in_c_name, in_c_args)
                        for c in t_calls:
                            c_name, c_args = canonicalize_call(c.get("name", ""), c.get("args") or {})
                            if c.get("name") == "invoke_subagent" or c_name == "invoke_subagent":
                                code_edits = 0
                            elif c_name in ("EDIT_FILE", "WRITE_FILE"):
                                p = str(c_args.get("Path") or "")
                                if is_code_file(p):
                                    code_edits += 1
                if code_edits >= edit_max:
                    target = f"{in_p}:{code_edits}/{edit_max}"
                    reason = f"CIRCUIT BREAKER (Main-Session Code-Edit Cap): {code_edits}/{edit_max} code edits in this stream. Send this to an implementation helper agent via invoke_subagent."
                    check_and_handle_trigger("H1", target, reason, tool_name, step_idx, conv_id, is_subagent, collector=collector, stop_on_enforce=stop_on_enforce)

    if not is_subagent and last_user_idx >= 0 and not is_authorized(last_user_content, conv_id, state_dir=state_dir):
        checkpoint_calls = int(os.environ.get("BREAKER_CHECKPOINT_CALLS", 45))
        if checkpoint_calls > 0:
            post_user_steps = steps[last_user_idx + 1:]
            post_user_planner_indices = [i for i, s in enumerate(post_user_steps) if s.get("type") == "PLANNER_RESPONSE"]
            last_post_user_planner_idx = post_user_planner_indices[-1] if post_user_planner_indices else -1
            prior_calls_count = 0
            for i, s in enumerate(post_user_steps):
                if s.get("type") == "PLANNER_RESPONSE":
                    is_last = (i == last_post_user_planner_idx)
                    t_calls = effective_tool_calls(s, is_last, step_idx, in_c_name, in_c_args)
                    prior_calls_count += len(t_calls)
            current_call_index = prior_calls_count + 1
            if current_call_index % checkpoint_calls == 0:
                target = f"turn_calls:{current_call_index}"
                reason = f"CIRCUIT BREAKER (Checkpoint): Checkpoint: {current_call_index} calls since the user's last message. Send found / unknown / options and wait."
                check_and_handle_trigger("CHECKPOINT", target, reason, tool_name, step_idx, conv_id, is_subagent, collector=collector, stop_on_enforce=stop_on_enforce)

    segment_steps = steps[last_user_idx + 1:] if last_user_idx >= 0 else steps
    segment_planner_indices = [i for i, s in enumerate(segment_steps) if s.get("type") == "PLANNER_RESPONSE"]
    last_segment_planner_idx = segment_planner_indices[-1] if segment_planner_indices else -1

    if tool_name == "run_command":
        raw_cmd = str(tool_args.get("CommandLine", ""))
        probe_target = probe_target_path(raw_cmd)
        if probe_target:
            probe_count = 0
            for i, s in enumerate(segment_steps):
                if s.get("type") == "PLANNER_RESPONSE":
                    is_last = (i == last_segment_planner_idx)
                    t_calls = effective_tool_calls(s, is_last, step_idx, in_c_name, in_c_args)
                    for c in t_calls:
                        c_name, c_args = canonicalize_call(c.get("name", ""), c.get("args") or {})
                        if c_name in ("EDIT_FILE", "WRITE_FILE"):
                            if os.path.expanduser(c_args.get("Path", "")) == os.path.expanduser(probe_target):
                                probe_count = 0
                        elif c_name == "run_command":
                            c_cmd = str(c_args.get("CommandLine", ""))
                            if probe_target_path(c_cmd) == probe_target:
                                probe_count += 1
            if probe_count + 1 >= 3:
                name = Path(probe_target).stem or "probe"
                reason = f"CIRCUIT BREAKER (Probe Tripwire): 3rd inline probe of {probe_target}: write /tmp/parse_{name}.py once and run it"
                check_and_handle_trigger("PROBE", probe_target, reason, tool_name, step_idx, conv_id, is_subagent, collector=collector, stop_on_enforce=stop_on_enforce)

    if in_c_name == "READ_FILE":
        read_path = in_c_args.get("Path", "")
        norm_read_path = os.path.abspath(os.path.expanduser(read_path)) if read_path else ""
        s_line = in_c_args.get("StartLine")
        e_line = in_c_args.get("EndLine")
        try:
            s_line = int(s_line) if s_line is not None else None
        except Exception:
            s_line = None
        try:
            e_line = int(e_line) if e_line is not None else None
        except Exception:
            e_line = None
        in_rng = (s_line, e_line)

        trueloop_window = int(os.environ.get("BREAKER_TRUELOOP_WINDOW", 20))
        planner_steps = [(i, s) for i, s in enumerate(segment_steps) if s.get("type") == "PLANNER_RESPONSE"]
        if len(planner_steps) > trueloop_window:
            window_steps = planner_steps[-trueloop_window:]
        else:
            window_steps = planner_steps

        prior_reads = []
        for i, s in window_steps:
            is_last = (i == last_segment_planner_idx)
            t_calls = effective_tool_calls(s, is_last, step_idx, in_c_name, in_c_args)
            for c in t_calls:
                c_name, c_args = canonicalize_call(c.get("name", ""), c.get("args") or {})
                if c_name in ("EDIT_FILE", "WRITE_FILE"):
                    c_p = c_args.get("Path", "")
                    if c_p and norm_read_path and os.path.abspath(os.path.expanduser(c_p)) == norm_read_path:
                        prior_reads = []
                elif c_name == "READ_FILE":
                    c_p = c_args.get("Path", "")
                    if c_p and norm_read_path and os.path.abspath(os.path.expanduser(c_p)) == norm_read_path:
                        c_s = c_args.get("StartLine")
                        c_e = c_args.get("EndLine")
                        try:
                            c_s = int(c_s) if c_s is not None else None
                        except Exception:
                            c_s = None
                        try:
                            c_e = int(c_e) if c_e is not None else None
                        except Exception:
                            c_e = None
                        prior_reads.append((c_s, c_e))

        qualifying_read_count = sum(1 for c_rng in prior_reads if is_re_read(in_rng, [c_rng]))
        if qualifying_read_count + 1 >= 3:
            reason = f"CIRCUIT BREAKER (True-Loop Tripwire): 3rd overlapping read of {read_path} (lines {in_rng[0] or 1}-{in_rng[1] or 'end'}). Proceed using data already in context, change approach, or ask the user."
            check_and_handle_trigger("TRUELOOP", read_path, reason, tool_name, step_idx, conv_id, is_subagent, collector=collector, stop_on_enforce=stop_on_enforce)

    elif in_c_name == "run_command":
        raw_cmd = str(in_c_args.get("CommandLine", ""))
        norm_cmd = normalize_command(raw_cmd)
        if norm_cmd and not is_test_runner(norm_cmd):
            trueloop_window = int(os.environ.get("BREAKER_TRUELOOP_WINDOW", 20))
            planner_steps = [(i, s) for i, s in enumerate(segment_steps) if s.get("type") == "PLANNER_RESPONSE"]
            if len(planner_steps) > trueloop_window:
                window_steps = planner_steps[-trueloop_window:]
            else:
                window_steps = planner_steps

            cmd_count = 0
            for i, s in window_steps:
                is_last = (i == last_segment_planner_idx)
                t_calls = effective_tool_calls(s, is_last, step_idx, in_c_name, in_c_args)
                for c in t_calls:
                    c_name, c_args = canonicalize_call(c.get("name", ""), c.get("args") or {})
                    if c_name in ("EDIT_FILE", "WRITE_FILE"):
                        cmd_count = 0
                    elif c_name == "run_command":
                        c_raw = str(c_args.get("CommandLine", ""))
                        c_norm = normalize_command(c_raw)
                        if c_norm == norm_cmd:
                            cmd_count += 1
            if cmd_count + 1 >= 3:
                reason = f"CIRCUIT BREAKER (True-Loop Tripwire): 3rd execution of '{norm_cmd}' with no file edit in between. Change approach, inspect prior output, or ask the user."
                check_and_handle_trigger("TRUELOOP", norm_cmd, reason, tool_name, step_idx, conv_id, is_subagent, collector=collector, stop_on_enforce=stop_on_enforce)

    post_user_steps = steps[last_user_idx + 1:] if last_user_idx >= 0 else steps
    planner_step_indices = [i for i, s in enumerate(post_user_steps) if s.get("type") == "PLANNER_RESPONSE"]
    dup_window = int(os.environ.get("BREAKER_DUP_WINDOW", 6))

    last_post_planner_idx = planner_step_indices[-1] if planner_step_indices else -1
    non_empty_planner_indices = []
    for i in planner_step_indices:
        s = post_user_steps[i]
        t_calls = effective_tool_calls(s, i == last_post_planner_idx, step_idx, in_c_name, in_c_args)
        if t_calls:
            non_empty_planner_indices.append((i, t_calls))

    window_start_idx = non_empty_planner_indices[-dup_window][0] if len(non_empty_planner_indices) >= dup_window else 0

    tracked_calls = []
    for i, t_calls in non_empty_planner_indices:
        for c in t_calls:
            c_name, c_args = canonicalize_call(c.get("name", ""), c.get("args") or {})
            if c_name in ("EDIT_FILE", "WRITE_FILE"):
                p = c_args.get("Path")
                tracked_calls = [
                    tc for tc in tracked_calls
                    if not (tc[1] == "READ_FILE" and tc[2].get("Path") == p) and tc[1] != "run_command"
                ]
            tracked_calls.append((i, c_name, c_args))

    active_window_calls = [tc for tc in tracked_calls if tc[0] >= window_start_idx]
    is_test_cmd = (in_c_name == "run_command" and is_test_runner(str(in_c_args.get("CommandLine", ""))))
    if not is_test_cmd and tool_name in ("read_file", "view_file", "run_command", "call_mcp_tool", "edit_file", "replace_file_content"):
        for _, c_name, c_args in active_window_calls:
            if c_name == in_c_name and c_args == in_c_args:
                target = str(in_c_args.get("Path") or in_c_args.get("CommandLine") or in_c_args)
                reason = "CIRCUIT BREAKER (Duplicate Call): You already executed this exact tool call with identical arguments recently without modifying the target file in between. Do not repeat identical calls. Proceed using the data already in your context or take a different action."
                check_and_handle_trigger("DUP", target, reason, tool_name, step_idx, conv_id, is_subagent, collector=collector, stop_on_enforce=stop_on_enforce)
                break

    return collector or []

def evaluate(payload: dict, env: dict | None = None) -> list[dict]:
    orig_environ = os.environ.copy()
    temp_dir = None
    try:
        effective_env = dict(env) if env else {}
        if "BREAKER_STATE_DIR" not in effective_env:
            temp_dir = tempfile.mkdtemp()
            effective_env["BREAKER_STATE_DIR"] = temp_dir
        if "BREAKER_LOG_PATH" not in effective_env:
            effective_env["BREAKER_LOG_PATH"] = os.devnull
        for k, v in effective_env.items():
            os.environ[k] = str(v)
        collector = []
        _run_breaker(payload, stop_on_enforce=False, collector=collector)
        return collector
    finally:
        os.environ.clear()
        os.environ.update(orig_environ)
        if temp_dir and os.path.exists(temp_dir):
            try:
                shutil.rmtree(temp_dir)
            except Exception:
                pass

def main():
    passthrough_out = os.environ.get("BREAKER_PASSTHROUGH_OUT", "")
    try:
        raw_input = sys.stdin.read()
        if not raw_input.strip():
            sys.exit(0)
        payload = json.loads(raw_input)
    except Exception:
        if passthrough_out:
            sys.stdout.write(passthrough_out)
        sys.exit(0)

    _run_breaker(payload, stop_on_enforce=True)
    if passthrough_out:
        sys.stdout.write(passthrough_out)
    sys.exit(0)

if __name__ == "__main__":
    try:
        main()
    except Exception:
        passthrough_out = os.environ.get("BREAKER_PASSTHROUGH_OUT", "")
        if passthrough_out:
            sys.stdout.write(passthrough_out)
        sys.exit(0)
