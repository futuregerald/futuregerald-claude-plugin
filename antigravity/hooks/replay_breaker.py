#!/usr/bin/env python3
import argparse
import glob
import json
import os
import shutil
import sys
import tempfile
import time
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import autonomous_breaker
from breaker_rules import (
    is_authorized,
    is_complaint,
    compaction_step_indices,
)

def run_replay(
    root: str = "~/.gemini/antigravity/brain",
    last_n: int = 150,
    out_path: str = "~/.gemini/metrics/replay-2026-10-21.json",
    sample_every: int = 1,
) -> dict:
    t0 = time.time()
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

    rule_names = ["DUP", "EXPLORE", "STRIKES", "PROBE", "TRUELOOP", "CHECKPOINT", "H1", "H2", "BATON"]
    rule_stats = {r: {"total_triggers": 0, "sessions_affected": set(), "main_triggers": 0, "helper_triggers": 0} for r in rule_names}

    session_results = []
    all_trueloop_triggers = []
    all_probe_triggers = []
    h1_sessions = set()
    h2_helper_sessions = set()
    total_helpers = 0

    all_compactions = []
    total_baton_blocks = 0
    total_baton_warns = 0

    all_normal_unauth_turns = 0
    hit_normal_unauth_turns = 0

    authorized_main_sessions = []

    replay_tmpdir = tempfile.mkdtemp(prefix="replay_breaker_")
    try:
        for p_idx, transcript_path in enumerate(selected_paths):
            if sample_every > 1 and (p_idx % sample_every != 0):
                continue

            conv_id = Path(transcript_path).parent.parent.parent.name
            full_path = Path(transcript_path).parent / "transcript_full.jsonl"

            steps = []
            try:
                with open(transcript_path, "r", encoding="utf-8") as f:
                    for line in f:
                        l_str = line.strip()
                        if l_str:
                            steps.append(json.loads(l_str))
            except Exception:
                continue

            if not steps:
                continue

            full_steps = []
            if full_path.is_file():
                try:
                    with open(full_path, "r", encoding="utf-8") as ff:
                        for line in ff:
                            fl_str = line.strip()
                            if fl_str:
                                full_steps.append(json.loads(fl_str))
                except Exception:
                    full_steps = []

            is_subagent = not any(s.get("type") == "USER_INPUT" for s in steps)
            if is_subagent:
                total_helpers += 1

            conv_tmpdir = os.path.join(replay_tmpdir, f"conv_{conv_id}")
            state_dir = os.path.join(conv_tmpdir, "state")
            log_dir = os.path.join(conv_tmpdir, "metrics")
            os.makedirs(state_dir, exist_ok=True)
            os.makedirs(log_dir, exist_ok=True)
            log_path = os.path.join(log_dir, "breaker.jsonl")

            real_state_dir = os.path.expanduser("~/.gemini/state")
            real_flag = os.path.join(real_state_dir, f"autonomous-{conv_id}")
            if os.path.exists(real_flag):
                shutil.copy2(real_flag, os.path.join(state_dir, f"autonomous-{conv_id}"))

            env = {
                "BREAKER_STATE_DIR": state_dir,
                "BREAKER_LOG_PATH": log_path,
                "BREAKER_DUP_MODE": "enforce",
                "BREAKER_EXPLORE_MODE": "enforce",
                "BREAKER_STRIKES_MODE": "enforce",
                "BREAKER_PROBE_MODE": "enforce",
                "BREAKER_TRUELOOP_MODE": "enforce",
                "BREAKER_CHECKPOINT_MODE": "enforce",
                "BREAKER_H1_MODE": "enforce",
                "BREAKER_H2_MODE": "enforce",
                "BREAKER_BATON_MODE": "enforce",
            }

            trunc_compact_path = os.path.join(conv_tmpdir, "transcript.jsonl")
            trunc_full_path = os.path.join(conv_tmpdir, "transcript_full.jsonl")

            session_calls = []
            session_warn_steps = []
            session_checkpoint_hits = 0

            for k, step in enumerate(steps):
                if step.get("type") == "USER_INPUT":
                    if is_authorized(step.get("content") or ""):
                        flag_path = os.path.join(state_dir, f"autonomous-{conv_id}")
                        if not os.path.exists(flag_path):
                            with open(flag_path, "w") as ff:
                                ff.write("1\n")
                    continue
                if step.get("type") != "PLANNER_RESPONSE":
                    continue

                tool_calls = step.get("tool_calls") or []
                trunc_fields = step.get("truncated_fields") or []
                if "tool_calls" in trunc_fields and k < len(full_steps) and full_steps[k]:
                    tool_calls = full_steps[k].get("tool_calls") or tool_calls

                if not tool_calls:
                    continue

                for j, call in enumerate(tool_calls):
                    prior_steps = steps[:k]
                    step_k = dict(step)
                    step_k["tool_calls"] = tool_calls[:j + 1]
                    s_idx = step_k.get("step_index")
                    if s_idx is None:
                        s_idx = k
                        step_k["step_index"] = k

                    sim_steps = prior_steps + [step_k]
                    with open(trunc_compact_path, "w", encoding="utf-8") as cf:
                        for s_rec in sim_steps:
                            cf.write(json.dumps(s_rec) + "\n")

                    if full_steps:
                        full_prior = full_steps[:k]
                        full_step_k = dict(full_steps[k] if k < len(full_steps) else step)
                        full_step_k["tool_calls"] = tool_calls[:j + 1]
                        if full_step_k.get("step_index") is None:
                            full_step_k["step_index"] = s_idx
                        sim_full = full_prior + [full_step_k]
                        with open(trunc_full_path, "w", encoding="utf-8") as ff:
                            for s_rec in sim_full:
                                ff.write(json.dumps(s_rec) + "\n")

                    payload = {
                        "stepIdx": s_idx + j + 1,
                        "transcriptPath": trunc_compact_path,
                        "toolCall": call,
                        "conversationId": conv_id,
                    }

                    triggered = autonomous_breaker.evaluate(payload, env=env)
                    session_calls.append({
                        "step": k,
                        "call_idx": j,
                        "tool": call.get("name"),
                        "triggered": triggered,
                    })

                    for tr in triggered:
                        r_name = tr["rule"]
                        if r_name in rule_stats:
                            rule_stats[r_name]["total_triggers"] += 1
                            rule_stats[r_name]["sessions_affected"].add(conv_id)
                            if is_subagent:
                                rule_stats[r_name]["helper_triggers"] += 1
                            else:
                                rule_stats[r_name]["main_triggers"] += 1

                        if r_name == "TRUELOOP":
                            all_trueloop_triggers.append({
                                "session": conv_id,
                                "step": k,
                                "target": tr.get("target"),
                            })
                        elif r_name == "PROBE":
                            all_probe_triggers.append({
                                "session": conv_id,
                                "step": k,
                                "target": tr.get("target"),
                            })
                        elif r_name == "H1":
                            h1_sessions.add(conv_id)
                        elif r_name == "H2":
                            h2_helper_sessions.add(conv_id)
                        elif r_name == "CHECKPOINT":
                            session_checkpoint_hits += 1
                        elif r_name == "BATON":
                            if "Warn" in tr.get("reason", ""):
                                total_baton_warns += 1
                                session_warn_steps.append(k)
                            else:
                                total_baton_blocks += 1

            if not is_subagent:
                u_indices = [idx for idx, s in enumerate(steps) if s.get("type") == "USER_INPUT"]
                has_any_auth = False
                for t_idx, u_idx in enumerate(u_indices):
                    nxt_idx = u_indices[t_idx + 1] if t_idx + 1 < len(u_indices) else len(steps)
                    u_text = steps[u_idx].get("content") or ""
                    auth = is_authorized(u_text, conv_id, state_dir=state_dir)
                    if auth:
                        has_any_auth = True
                    complaint = False
                    if t_idx + 1 < len(u_indices):
                        nxt_text = steps[u_indices[t_idx + 1]].get("content") or ""
                        complaint = is_complaint(nxt_text)
                    if not auth and not complaint:
                        all_normal_unauth_turns += 1
                        turn_calls_hit = any(
                            any(r["rule"] == "CHECKPOINT" for r in c["triggered"])
                            for c in session_calls if u_idx < c["step"] < nxt_idx
                        )
                        if turn_calls_hit:
                            hit_normal_unauth_turns += 1

                if has_any_auth:
                    authorized_main_sessions.append({
                        "conversation_id": conv_id,
                        "step_count": len(steps),
                        "checkpoint_hits": session_checkpoint_hits,
                    })

                comp_indices = compaction_step_indices(steps)
                prev_comp = -1
                for comp_idx in comp_indices:
                    warns_in_seg = [w for w in session_warn_steps if prev_comp < w < comp_idx]
                    if warns_in_seg:
                        earliest_warn = warns_in_seg[0]
                        steps_before = comp_idx - earliest_warn
                        all_compactions.append({
                            "session": conv_id,
                            "compaction_step": comp_idx,
                            "warn_fired": True,
                            "steps_before_compaction": steps_before,
                            "ge_15_steps": steps_before >= 15,
                        })
                    else:
                        all_compactions.append({
                            "session": conv_id,
                            "compaction_step": comp_idx,
                            "warn_fired": False,
                            "steps_before_compaction": None,
                            "ge_15_steps": False,
                        })
                    prev_comp = comp_idx

            session_results.append({
                "conversation_id": conv_id,
                "is_subagent": is_subagent,
                "step_count": len(steps),
                "calls": session_calls,
            })
    finally:
        shutil.rmtree(replay_tmpdir, ignore_errors=True)

    authorized_main_sessions.sort(key=lambda s: s["step_count"], reverse=True)
    top_4_auth = authorized_main_sessions[:4]
    top_4_checkpoint_triggers = sum(s["checkpoint_hits"] for s in top_4_auth)

    rules_out = {}
    for r, st in rule_stats.items():
        rules_out[r] = {
            "total_triggers": st["total_triggers"],
            "sessions_affected": len(st["sessions_affected"]),
            "main_triggers": st["main_triggers"],
            "helper_triggers": st["helper_triggers"],
        }

    checkpoint_normal_share = round(hit_normal_unauth_turns / all_normal_unauth_turns, 4) if all_normal_unauth_turns > 0 else 0.0
    h2_share = round(len(h2_helper_sessions) / total_helpers, 4) if total_helpers > 0 else 0.0

    compaction_count = len(all_compactions)
    warn_fired_count = sum(1 for c in all_compactions if c["warn_fired"])
    warn_ge_15_count = sum(1 for c in all_compactions if c["ge_15_steps"])
    baton_warn_share = round(warn_fired_count / compaction_count, 4) if compaction_count > 0 else 0.0
    baton_ge_15_share = round(warn_ge_15_count / compaction_count, 4) if compaction_count > 0 else 0.0

    duration = round(time.time() - t0, 2)

    result = {
        "metadata": {
            "root": exp_root,
            "transcripts_evaluated": len(selected_paths),
            "duration_seconds": duration,
        },
        "rules": rules_out,
        "checkpoint": {
            "normal_unauthorized_turns_total": all_normal_unauth_turns,
            "normal_unauthorized_turns_hit": hit_normal_unauth_turns,
            "share_normal_unauthorized_turns_hit": checkpoint_normal_share,
            "longest_4_authorized_sessions_checkpoint_triggers": top_4_checkpoint_triggers,
            "longest_4_authorized_sessions": top_4_auth,
        },
        "trueloop_probe_triggers": {
            "TRUELOOP": all_trueloop_triggers,
            "PROBE": all_probe_triggers,
        },
        "h1": {
            "sessions_affected": sorted(list(h1_sessions)),
            "total_triggers": rules_out["H1"]["total_triggers"],
        },
        "h2": {
            "total_helpers": total_helpers,
            "helpers_hit": len(h2_helper_sessions),
            "share_helpers_hit": h2_share,
            "total_triggers": rules_out["H2"]["total_triggers"],
        },
        "baton": {
            "total_compactions": compaction_count,
            "warn_fired_count": warn_fired_count,
            "warn_fired_share": baton_warn_share,
            "warn_ge_15_steps_ahead_count": warn_ge_15_count,
            "warn_ge_15_steps_ahead_share": baton_ge_15_share,
            "total_warns": total_baton_warns,
            "total_blocks": total_baton_blocks,
            "compactions": all_compactions,
        },
        "dup_explore_strikes": {
            "DUP": rules_out["DUP"]["total_triggers"],
            "EXPLORE": rules_out["EXPLORE"]["total_triggers"],
            "STRIKES": rules_out["STRIKES"]["total_triggers"],
        },
        "sessions": session_results,
    }

    if out_path:
        exp_out = os.path.expanduser(out_path)
        os.makedirs(os.path.dirname(exp_out), exist_ok=True)
        with open(exp_out, "w", encoding="utf-8") as f:
            json.dump(result, f, indent=2)

    print_summary(result)
    return result

def print_summary(res: dict) -> None:
    meta = res["metadata"]
    rules = res["rules"]
    cp = res["checkpoint"]
    h1 = res["h1"]
    h2 = res["h2"]
    baton = res["baton"]
    print("================ REPLAY BREAKER SUMMARY ================")
    print(f"Evaluated {meta['transcripts_evaluated']} transcripts in {meta['duration_seconds']}s")
    print(f"{'Rule':<12} {'Total':<8} {'Sessions':<10} {'Main':<8} {'Helper':<8}")
    print("-" * 48)
    for r_name in ["DUP", "EXPLORE", "STRIKES", "PROBE", "TRUELOOP", "CHECKPOINT", "H1", "H2", "BATON"]:
        st = rules.get(r_name, {})
        print(f"{r_name:<12} {st.get('total_triggers', 0):<8} {st.get('sessions_affected', 0):<10} {st.get('main_triggers', 0):<8} {st.get('helper_triggers', 0):<8}")
    print("-" * 48)
    print(f"CHECKPOINT: {cp['normal_unauthorized_turns_hit']}/{cp['normal_unauthorized_turns_total']} normal unauth turns ({cp['share_normal_unauthorized_turns_hit']*100:.1f}%), {cp['longest_4_authorized_sessions_checkpoint_triggers']} in top-4 auth")
    print(f"H1 SESSIONS: {len(h1['sessions_affected'])} {h1['sessions_affected']}")
    print(f"H2 HELPERS:  {h2['helpers_hit']}/{h2['total_helpers']} ({h2['share_helpers_hit']*100:.1f}%)")
    explore_cap = int(os.environ.get("BREAKER_SUBAGENT_EXPLORE_MAX", 100))
    print(f"EXPLORE:     {rules.get('EXPLORE', {}).get('total_triggers', 0)} triggers across {rules.get('EXPLORE', {}).get('sessions_affected', 0)} helper sessions (cap={explore_cap})")
    print(f"BATON WARN:  {baton['warn_fired_count']}/{baton['total_compactions']} compactions ({baton['warn_fired_share']*100:.1f}%), {baton['warn_ge_15_steps_ahead_count']} >=15 steps ahead ({baton['warn_ge_15_steps_ahead_share']*100:.1f}%)")
    print(f"BATON BLOCK: {baton['total_blocks']} total blocks")
    print("=========================================================")

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=str, default="~/.gemini/antigravity/brain", help="Brain root dir")
    parser.add_argument("--last", type=int, default=150, help="Evaluate most recent N transcripts")
    parser.add_argument("--out", type=str, default="~/.gemini/metrics/replay-2026-10-21.json", help="Output JSON path")
    parser.add_argument("--sample-every", type=int, default=1, help="Sample every M transcripts")
    args = parser.parse_args()

    run_replay(root=args.root, last_n=args.last, out_path=args.out, sample_every=args.sample_every)

if __name__ == "__main__":
    main()
