import argparse
from datetime import datetime
import glob
import json
import math
import os
import sys

from breaker_rules import (
    is_authorized,
    read_ranges_overlap,
    normalize_command,
    is_test_runner,
    is_code_file,
    is_complaint,
    compaction_step_indices,
    decode_json_strings,
    context_estimate_tokens,
)

def parse_iso_time(ts_str):
    if not ts_str or not isinstance(ts_str, str):
        return None
    try:
        clean = ts_str.replace("Z", "+00:00")
        return datetime.fromisoformat(clean)
    except Exception:
        return None

def compute_percentiles(vals):
    if not vals:
        return {"count": 0, "min": 0, "p10": 0, "p50": 0, "p90": 0, "max": 0}
    s = sorted(vals)
    n = len(s)
    def p_val(p):
        k = (n - 1) * (p / 100.0)
        f = math.floor(k)
        c = math.ceil(k)
        if f == c:
            return s[int(k)]
        return s[f] * (c - k) + s[c] * (k - f)
    return {
        "count": n,
        "min": s[0],
        "p10": round(p_val(10), 2),
        "p50": round(p_val(50), 2),
        "p90": round(p_val(90), 2),
        "max": s[-1],
    }

def round_down_5k(val):
    return int(val // 5000) * 5000

def analyze_session(compact_path: str, full_path: str | None = None, state_dir: str = "~/.gemini/state") -> dict:
    conv_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(compact_path))))
    conv_id = os.path.basename(conv_dir)

    compact_steps = []
    try:
        with open(compact_path, "r", encoding="utf-8", errors="replace") as f:
            for line in f:
                line_str = line.strip()
                if not line_str:
                    continue
                try:
                    compact_steps.append(json.loads(line_str))
                except Exception:
                    continue
    except Exception:
        return {"conversation_id": conv_id, "error": "failed_to_read_compact"}

    if full_path is None:
        cand = os.path.join(os.path.dirname(compact_path), "transcript_full.jsonl")
        if os.path.exists(cand):
            full_path = cand

    full_raw_bytes = []
    full_steps = []
    if full_path and os.path.exists(full_path):
        try:
            with open(full_path, "rb") as f:
                for bline in f:
                    full_raw_bytes.append(len(bline))
                    try:
                        full_steps.append(json.loads(bline.decode("utf-8", errors="replace")))
                    except Exception:
                        full_steps.append(None)
        except Exception:
            pass

    if not full_raw_bytes:
        try:
            with open(compact_path, "rb") as f:
                for bline in f:
                    full_raw_bytes.append(len(bline))
        except Exception:
            pass

    for idx, step in enumerate(compact_steps):
        trunc = step.get("truncated_fields") or []
        if ("tool_calls" in trunc or "content" in trunc) and idx < len(full_steps) and full_steps[idx]:
            if "tool_calls" in trunc:
                step["tool_calls"] = full_steps[idx].get("tool_calls", step.get("tool_calls", []))
            if "content" in trunc:
                step["content"] = full_steps[idx].get("content", step.get("content", ""))

    is_subagent = (len(compact_steps) > 0) and not any(s.get("type") == "USER_INPUT" for s in compact_steps)

    if is_subagent:
        n_planner = sum(1 for s in compact_steps if s.get("type") == "PLANNER_RESPONSE")
        n_calls = sum(len(s.get("tool_calls", [])) for s in compact_steps)
        explore_count = 0
        for s in compact_steps:
            for c in s.get("tool_calls", []):
                name = c.get("name", "")
                if name in ("read_file", "view_file", "call_mcp_tool", "search_web", "grep_search", "find_by_name", "list_dir"):
                    explore_count += 1
        return {
            "conversation_id": conv_id,
            "is_subagent": True,
            "n_planner_steps": n_planner,
            "n_calls": n_calls,
            "explore_count": explore_count,
            "explore_budget_hits": explore_count > 20,
            "total_steps": len(compact_steps),
        }

    u_indices = [idx for idx, s in enumerate(compact_steps) if s.get("type") == "USER_INPUT"]
    turns = []
    for turn_idx, start_idx in enumerate(u_indices):
        end_idx = u_indices[turn_idx + 1] if turn_idx + 1 < len(u_indices) else len(compact_steps)
        turn_steps = compact_steps[start_idx:end_idx]
        user_text = turn_steps[0].get("content", "")
        authorized = is_authorized(user_text, conv_id, state_dir=state_dir)
        complaint_followed = False
        if turn_idx + 1 < len(u_indices):
            next_user_text = compact_steps[u_indices[turn_idx + 1]].get("content", "")
            complaint_followed = is_complaint(next_user_text)
        normal = (not authorized) and (not complaint_followed)

        active_seconds = 0.0
        for k in range(len(turn_steps) - 1):
            s_curr = turn_steps[k]
            s_next = turn_steps[k + 1]
            if s_next.get("type") == "SYSTEM_MESSAGE":
                continue
            t1 = parse_iso_time(s_curr.get("created_at"))
            t2 = parse_iso_time(s_next.get("created_at"))
            if t1 and t2 and t2 > t1:
                active_seconds += (t2 - t1).total_seconds()

        turn_tool_calls = 0
        path_read_ranges = {}
        cmd_counts = {}
        current_code_edit_run = 0
        max_code_edit_run = 0
        turn_true_loop_hits = 0

        for s in turn_steps:
            for c in s.get("tool_calls", []):
                turn_tool_calls += 1
                name = c.get("name", "")
                args = decode_json_strings(c.get("args") or {})
                if not isinstance(args, dict):
                    args = {}

                if name in ("read_file", "view_file"):
                    path = args.get("AbsolutePath") or args.get("Path") or args.get("path") or args.get("TargetFile") or ""
                    s_line = args.get("StartLine")
                    e_line = args.get("EndLine")
                    try:
                        s_line = int(s_line) if s_line is not None else None
                    except Exception:
                        s_line = None
                    try:
                        e_line = int(e_line) if e_line is not None else None
                    except Exception:
                        e_line = None
                    rng = (s_line, e_line)
                    prior = path_read_ranges.get(path, [])
                    overlapping = [item for item in prior if read_ranges_overlap(item[0], rng)]
                    if not overlapping:
                        path_read_ranges.setdefault(path, []).append((rng, 1))
                    else:
                        max_c = max(item[1] for item in overlapping)
                        new_c = max_c + 1
                        path_read_ranges.setdefault(path, []).append((rng, new_c))
                        if new_c >= 3:
                            turn_true_loop_hits += 1

                elif name == "run_command":
                    raw_cmd = args.get("CommandLine") or args.get("cmd") or ""
                    norm_cmd = normalize_command(raw_cmd)
                    if norm_cmd and not is_test_runner(norm_cmd):
                        cnt = cmd_counts.get(norm_cmd, 0) + 1
                        cmd_counts[norm_cmd] = cnt
                        if cnt >= 3:
                            turn_true_loop_hits += 1

                elif name in ("edit_file", "replace_file_content", "write_file", "write_to_file"):
                    path = args.get("TargetFile") or args.get("Path") or args.get("path") or args.get("AbsolutePath") or ""
                    if path in path_read_ranges:
                        path_read_ranges[path] = []
                    cmd_counts.clear()
                    if is_code_file(path):
                        current_code_edit_run += 1
                        if current_code_edit_run > max_code_edit_run:
                            max_code_edit_run = current_code_edit_run

                elif name == "invoke_subagent":
                    current_code_edit_run = 0

        turns.append({
            "turn_idx": turn_idx,
            "n_calls": turn_tool_calls,
            "authorized": authorized,
            "complaint_followed": complaint_followed,
            "normal": normal,
            "active_seconds": round(active_seconds, 2),
            "true_loop_hits": turn_true_loop_hits,
            "code_edits_by_main": max_code_edit_run,
        })

    comp_indices = compaction_step_indices(compact_steps)
    estimates = []
    segment_estimates_at_compaction = []
    last_comp_pos = -1
    for idx, s in enumerate(compact_steps):
        if idx in comp_indices:
            seg_bytes = full_raw_bytes[last_comp_pos + 1:idx] if last_comp_pos >= 0 else full_raw_bytes[:idx]
            segment_estimates_at_compaction.append(sum(seg_bytes) // 4)
            last_comp_pos = idx
        estimates.append(context_estimate_tokens(full_raw_bytes[:idx + 1], [c for c in comp_indices if c <= idx]))

    max_estimate = max(estimates) if estimates else 0

    return {
        "conversation_id": conv_id,
        "is_subagent": False,
        "turns": turns,
        "compaction_step_indices": comp_indices,
        "segment_estimates_at_compaction": segment_estimates_at_compaction,
        "context_estimate_series": estimates,
        "max_context_estimate": max_estimate,
        "total_steps": len(compact_steps),
    }

def process_transcripts(
    root: str = "~/.gemini/antigravity/brain",
    last_n: int = 150,
    out_path: str | None = None,
    replay: bool = False,
    state_dir: str = "~/.gemini/state",
) -> dict:
    exp_root = os.path.expanduser(root)
    pattern = os.path.join(exp_root, "*", ".system_generated", "logs", "transcript.jsonl")
    matched_paths = glob.glob(pattern)

    file_info = []
    for p in matched_paths:
        try:
            mtime = os.path.getmtime(p)
            file_info.append((mtime, p))
        except Exception:
            continue

    file_info.sort(key=lambda x: x[0], reverse=True)
    selected_paths = [p for _, p in file_info[:last_n]]

    main_sessions = []
    helper_agents = []

    for path in selected_paths:
        res = analyze_session(path, state_dir=state_dir)
        if res.get("is_subagent"):
            helper_agents.append(res)
        else:
            main_sessions.append(res)

    normal_unauth_calls = []
    for s in main_sessions:
        for t in s.get("turns", []):
            if t.get("normal"):
                normal_unauth_calls.append(t["n_calls"])

    normal_calls_stats = compute_percentiles(normal_unauth_calls)
    proposed_breaker_checkpoint_calls = int(round(normal_calls_stats["p90"])) if normal_calls_stats["p90"] > 0 else 45

    helper_planner_steps = [h["n_planner_steps"] for h in helper_agents]
    helper_steps_stats = compute_percentiles(helper_planner_steps)
    proposed_breaker_subagent_step_max = int(round(helper_steps_stats["p90"])) if helper_steps_stats["p90"] > 0 else 60

    authorized_edits = []
    edit_distribution = {}
    for s in main_sessions:
        for t in s.get("turns", []):
            if t.get("authorized"):
                edits = t["code_edits_by_main"]
                authorized_edits.append(edits)
                edit_distribution[str(edits)] = edit_distribution.get(str(edits), 0) + 1

    auth_edits_stats = compute_percentiles(authorized_edits)
    auth_edits_stats["distribution"] = edit_distribution
    proposed_k = int(round(auth_edits_stats["p90"])) if auth_edits_stats["p90"] > 0 else 6

    all_segment_estimates_at_compaction = []
    for s in main_sessions:
        all_segment_estimates_at_compaction.extend(s.get("segment_estimates_at_compaction", []))

    compaction_stats = compute_percentiles(all_segment_estimates_at_compaction)
    p10_compaction = compaction_stats["p10"]

    if p10_compaction > 0:
        proposed_baton_warn = round_down_5k(0.75 * p10_compaction)
        proposed_baton_block = round_down_5k(0.92 * p10_compaction)
    else:
        proposed_baton_warn = 200000
        proposed_baton_block = 300000

    warn_crossing_count = 0
    block_crossing_count = 0
    earliest_warn_steps = []

    for s in main_sessions:
        warn_step = None
        block_step = None
        for step_idx, est in enumerate(s.get("context_estimate_series", [])):
            if warn_step is None and est >= proposed_baton_warn:
                warn_step = step_idx
            if block_step is None and est >= proposed_baton_block:
                block_step = step_idx
        s["earliest_warn_step"] = warn_step
        s["earliest_block_step"] = block_step
        if warn_step is not None:
            warn_crossing_count += 1
            earliest_warn_steps.append(warn_step)
        if block_step is not None:
            block_crossing_count += 1

    earliest_warn_stats = compute_percentiles(earliest_warn_steps)

    total_loop_hits = 0
    sessions_with_loops = 0
    for s in main_sessions:
        s_hits = sum(t.get("true_loop_hits", 0) for t in s.get("turns", []))
        total_loop_hits += s_hits
        if s_hits > 0:
            sessions_with_loops += 1

    summary = {
        "normal_unauthorized_n_calls": {
            **normal_calls_stats,
            "proposed_breaker_checkpoint_calls": proposed_breaker_checkpoint_calls,
        },
        "helper_n_planner_steps": {
            **helper_steps_stats,
            "proposed_breaker_subagent_step_max": proposed_breaker_subagent_step_max,
        },
        "authorized_code_edits_by_main": {
            **auth_edits_stats,
            "proposed_k": proposed_k,
        },
        "context_estimates": {
            "segment_estimate_at_compaction": compaction_stats,
            "proposed_baton_warn_tokens": proposed_baton_warn,
            "proposed_baton_block_tokens": proposed_baton_block,
            "sessions_crossing_warn_count": warn_crossing_count,
            "sessions_crossing_block_count": block_crossing_count,
            "earliest_warn_step": earliest_warn_stats,
        },
        "true_loops": {
            "total_hits": total_loop_hits,
            "sessions_affected": sessions_with_loops,
        },
    }

    result = {
        "summary": summary,
        "main_sessions_count": len(main_sessions),
        "helper_agents_count": len(helper_agents),
        "main_sessions": main_sessions,
        "helper_agents": helper_agents,
    }

    if replay:
        for s in main_sessions:
            cid = s["conversation_id"]
            if s.get("earliest_warn_step") is not None:
                print(f"[REPLAY] {cid}: crossed warn ({proposed_baton_warn}) at step {s['earliest_warn_step']} (max={s['max_context_estimate']}) -> BATON WARN")
            if s.get("earliest_block_step") is not None:
                print(f"[REPLAY] {cid}: crossed block ({proposed_baton_block}) at step {s['earliest_block_step']} -> BATON BLOCK")
            for t in s.get("turns", []):
                t_idx = t["turn_idx"]
                if not t["authorized"] and proposed_breaker_checkpoint_calls > 0:
                    hits = t["n_calls"] // proposed_breaker_checkpoint_calls
                    if hits > 0:
                        print(f"[REPLAY] {cid} Turn {t_idx}: CHECKPOINT would fire {hits} time(s) ({t['n_calls']} calls)")
                if t["true_loop_hits"] > 0:
                    print(f"[REPLAY] {cid} Turn {t_idx}: TRUE-LOOP would fire {t['true_loop_hits']} time(s)")
                if t["authorized"] and t["code_edits_by_main"] > proposed_k:
                    print(f"[REPLAY] {cid} Turn {t_idx}: H1 CODE-EDIT would fire ({t['code_edits_by_main']} edits > {proposed_k})")
        for h in helper_agents:
            cid = h["conversation_id"]
            if h["n_planner_steps"] >= proposed_breaker_subagent_step_max:
                print(f"[REPLAY] {cid} (Helper): H2 STEP-MAX would fire ({h['n_planner_steps']} >= {proposed_breaker_subagent_step_max})")

    if out_path:
        exp_out = os.path.expanduser(out_path)
        os.makedirs(os.path.dirname(exp_out), exist_ok=True)
        with open(exp_out, "w", encoding="utf-8") as f:
            json.dump(result, f, indent=2)

    return result

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--last", type=int, default=150, help="Most recent N transcripts")
    parser.add_argument("--out", type=str, default="~/.gemini/metrics/baseline-2026-10-21.json", help="Output JSON path")
    parser.add_argument("--replay", action="store_true", help="Print per-turn tripwire hits")
    parser.add_argument("--root", type=str, default="~/.gemini/antigravity/brain", help="Brain dir root")
    args = parser.parse_args()

    res = process_transcripts(root=args.root, last_n=args.last, out_path=args.out, replay=args.replay)
    summary_json = json.dumps(res["summary"], indent=2)
    print(summary_json)

if __name__ == "__main__":
    main()
