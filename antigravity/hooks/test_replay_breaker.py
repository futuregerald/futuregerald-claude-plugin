#!/usr/bin/env python3
import json
import os
import shutil
import tempfile
import unittest
from pathlib import Path

import autonomous_breaker
import replay_breaker

class TestReplayBreaker(unittest.TestCase):

    def setUp(self):
        self.test_tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        shutil.rmtree(self.test_tmpdir, ignore_errors=True)

    def write_jsonl(self, path: Path, records: list[dict]):
        path.parent.mkdir(parents=True, exist_ok=True)
        with open(path, "w", encoding="utf-8") as f:
            for r in records:
                f.write(json.dumps(r) + "\n")

    def test_trueloop_overlapping_reads_3rd_triggers(self):
        root = Path(self.test_tmpdir) / "brain"
        conv_id = "test-trueloop-conv"
        logs_dir = root / conv_id / ".system_generated" / "logs"
        t_path = logs_dir / "transcript.jsonl"
        steps = [
            {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "analyze code"},
            {
                "step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "tool_calls": [{"name": "read_file", "args": {"Path": "/repo/foo.py", "StartLine": 1, "EndLine": 100}}]
            },
            {"step_index": 2, "source": "TOOL", "type": "GENERIC", "content": "file contents"},
            {
                "step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "tool_calls": [{"name": "read_file", "args": {"Path": "/repo/foo.py", "StartLine": 50, "EndLine": 150}}]
            },
            {"step_index": 4, "source": "TOOL", "type": "GENERIC", "content": "file contents"},
            {
                "step_index": 5, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "tool_calls": [{"name": "read_file", "args": {"Path": "/repo/foo.py", "StartLine": 80, "EndLine": 120}}]
            },
        ]
        self.write_jsonl(t_path, steps)
        out_path = Path(self.test_tmpdir) / "out.json"
        res = replay_breaker.run_replay(root=str(root), last_n=10, out_path=str(out_path))
        self.assertEqual(res["rules"]["TRUELOOP"]["total_triggers"], 1)
        trueloop_triggers = res["trueloop_probe_triggers"]["TRUELOOP"]
        self.assertEqual(len(trueloop_triggers), 1)
        self.assertEqual(trueloop_triggers[0]["session"], conv_id)
        self.assertEqual(trueloop_triggers[0]["step"], 5)
        self.assertEqual(trueloop_triggers[0]["target"], "/repo/foo.py")

    def test_multiple_rules_on_one_call_recorded(self):
        root = Path(self.test_tmpdir) / "brain"
        conv_id = "test-multi-rule-conv"
        logs_dir = root / conv_id / ".system_generated" / "logs"
        t_path = logs_dir / "transcript.jsonl"
        steps = [
            {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "fix bug"},
        ]
        curr = 1
        for i in range(44):
            steps.append({
                "step_index": curr, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "tool_calls": [{"name": "read_file", "args": {"Path": f"/repo/unique_{i}.py"}}]
            })
            curr += 1
            steps.append({"step_index": curr, "source": "TOOL", "type": "GENERIC", "content": "ok"})
            curr += 1
        steps.append({
            "step_index": curr, "source": "MODEL", "type": "PLANNER_RESPONSE",
            "tool_calls": [{"name": "read_file", "args": {"Path": "/repo/unique_43.py"}}]
        })
        self.write_jsonl(t_path, steps)
        out_path = Path(self.test_tmpdir) / "out.json"
        res = replay_breaker.run_replay(root=str(root), last_n=10, out_path=str(out_path))
        multi_triggers = [
            call for s in res["sessions"] if s["conversation_id"] == conv_id
            for call in s["calls"] if len(call["triggered"]) > 1
        ]
        self.assertTrue(len(multi_triggers) >= 1)
        rules_triggered = [r["rule"] for r in multi_triggers[0]["triggered"]]
        self.assertIn("CHECKPOINT", rules_triggered)
        self.assertIn("DUP", rules_triggered)

    def test_inflight_semantics_parallel_reads_no_dup(self):
        root = Path(self.test_tmpdir) / "brain"
        conv_id = "test-inflight-conv"
        logs_dir = root / conv_id / ".system_generated" / "logs"
        t_path = logs_dir / "transcript.jsonl"
        steps = [
            {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "read files"},
            {
                "step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/repo/a.py"}},
                    {"name": "read_file", "args": {"Path": "/repo/b.py"}},
                    {"name": "read_file", "args": {"Path": "/repo/c.py"}},
                ]
            },
        ]
        self.write_jsonl(t_path, steps)
        out_path = Path(self.test_tmpdir) / "out.json"
        res = replay_breaker.run_replay(root=str(root), last_n=10, out_path=str(out_path))
        self.assertEqual(res["rules"]["DUP"]["total_triggers"], 0)

    def test_replay_never_touches_real_state_or_metrics(self):
        real_state_dir = Path.home() / ".gemini" / "state"
        real_metrics_file = Path.home() / ".gemini" / "metrics" / "breaker.jsonl"
        state_files_before = set(os.listdir(real_state_dir)) if real_state_dir.exists() else set()
        metrics_size_before = os.path.getsize(real_metrics_file) if real_metrics_file.exists() else 0

        root = Path(self.test_tmpdir) / "brain"
        conv_id = "test-isolation-conv"
        logs_dir = root / conv_id / ".system_generated" / "logs"
        t_path = logs_dir / "transcript.jsonl"
        steps = [
            {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "inspect"},
            {
                "step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "tool_calls": [{"name": "read_file", "args": {"Path": "/repo/test.py"}}]
            },
        ]
        self.write_jsonl(t_path, steps)
        out_path = Path(self.test_tmpdir) / "out.json"
        replay_breaker.run_replay(root=str(root), last_n=10, out_path=str(out_path))

        state_files_after = set(os.listdir(real_state_dir)) if real_state_dir.exists() else set()
        metrics_size_after = os.path.getsize(real_metrics_file) if real_metrics_file.exists() else 0
        self.assertEqual(state_files_before, state_files_after)
        self.assertEqual(metrics_size_before, metrics_size_after)

    def test_baton_state_persists_across_calls_within_transcript(self):
        root = Path(self.test_tmpdir) / "brain"
        conv_id = "test-baton-persist-conv"
        logs_dir = root / conv_id / ".system_generated" / "logs"
        t_path = logs_dir / "transcript.jsonl"
        t_full_path = logs_dir / "transcript_full.jsonl"
        padding = "x" * 450000
        steps = [
            {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "analyze system"},
            {
                "step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "content": padding,
                "tool_calls": [{"name": "read_file", "args": {"Path": "/repo/1.py"}}]
            },
            {"step_index": 2, "source": "TOOL", "type": "GENERIC", "content": "ok"},
            {
                "step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "content": "second",
                "tool_calls": [{"name": "read_file", "args": {"Path": "/repo/2.py"}}]
            },
        ]
        self.write_jsonl(t_path, steps)
        self.write_jsonl(t_full_path, steps)
        out_path = Path(self.test_tmpdir) / "out.json"
        res = replay_breaker.run_replay(root=str(root), last_n=10, out_path=str(out_path))
        baton_warns = [
            r for s in res["sessions"] if s["conversation_id"] == conv_id
            for c in s["calls"] for r in c["triggered"]
            if r["rule"] == "BATON" and "Warn" in r.get("reason", "")
        ]
        self.assertEqual(len(baton_warns), 1)

    def test_hooks_json_shasum_unchanged(self):
        import hashlib
        hooks_json_path = Path.home() / ".gemini" / "config" / "hooks.json"
        with open(hooks_json_path, "rb") as f:
            digest = hashlib.sha1(f.read()).hexdigest()
        self.assertEqual(digest, "51d1f3b1c5ca7a8484fc9e32e5ea94152907c771")

    def test_unauthorized_then_authorized_turn_1_checkpoint(self):
        root = Path(self.test_tmpdir) / "brain"
        conv_id = "test-unauth-then-auth-conv"
        logs_dir = root / conv_id / ".system_generated" / "logs"
        t_path = logs_dir / "transcript.jsonl"
        steps = [
            {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "please check this"},
        ]
        curr = 1
        for i in range(45):
            steps.append({"step_index": curr, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [{"name": "read_file", "args": {"Path": f"/repo/unique_{i}.py"}}]})
            curr += 1
            steps.append({"step_index": curr, "source": "TOOL", "type": "GENERIC", "content": "ok"})
            curr += 1
        steps.append({"step_index": curr, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "implement Phase 1"})
        curr += 1
        steps.append({"step_index": curr, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [{"name": "read_file", "args": {"Path": "/repo/another.py"}}]})

        self.write_jsonl(t_path, steps)
        out_path = Path(self.test_tmpdir) / "out.json"
        res = replay_breaker.run_replay(root=str(root), last_n=10, out_path=str(out_path))
        cp_triggers = [
            c for s in res["sessions"] if s["conversation_id"] == conv_id
            for c in s["calls"] for tr in c["triggered"] if tr["rule"] == "CHECKPOINT"
        ]
        self.assertEqual(len(cp_triggers), 1)



if __name__ == "__main__":
    unittest.main()
