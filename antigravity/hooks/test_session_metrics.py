import json
import os
import shutil
import tempfile
import unittest

from breaker_rules import (
    is_authorized,
    read_ranges_overlap,
    normalize_command,
    is_test_runner,
    is_code_file,
    is_complaint,
    compaction_step_indices,
)
from session_metrics import (
    analyze_session,
    process_transcripts,
)

class TestBreakerRules(unittest.TestCase):
    def test_authorized_phrases_match(self):
        phrases = [
            "/goal do something big",
            "Please run this in autonomous mode",
            "You are the Lead Orchestrator for this task",
            "Please continue from pending steps now",
            "ingest ~/.gemini/handoffs/dev/2026-10-21-active_context.md and proceed",
            "implement phase 2 per plan",
            "implement stream B",
            "keep going until finished",
        ]
        for p in phrases:
            self.assertTrue(is_authorized(p, None), f"Failed to authorize: {p}")

    def test_unrelated_text_not_authorized(self):
        unrelated = [
            "hello world",
            "what is this code doing?",
            "can you explain this function?",
            "status update please",
            "why did the test fail?",
        ]
        for p in unrelated:
            self.assertFalse(is_authorized(p, None), f"Incorrectly authorized: {p}")

    def test_flag_file_authorizes(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            conv_id = "test-conv-1234"
            self.assertFalse(is_authorized("just check", conv_id, state_dir=tmpdir))
            flag_path = os.path.join(tmpdir, f"autonomous-{conv_id}")
            with open(flag_path, "w") as f:
                f.write("1")
            self.assertTrue(is_authorized("just check", conv_id, state_dir=tmpdir))

    def test_read_ranges_overlap(self):
        self.assertTrue(read_ranges_overlap((1, 50), (40, 90)))
        self.assertFalse(read_ranges_overlap((1, 50), (51, 90)))
        self.assertTrue(read_ranges_overlap((None, None), (None, None)))
        self.assertTrue(read_ranges_overlap((1, 50), (None, None)))
        self.assertTrue(read_ranges_overlap((None, None), (10, 20)))
        self.assertTrue(read_ranges_overlap((1, 50), (50, 100)))
        self.assertTrue(read_ranges_overlap((1, 50), (1, 50)))

    def test_normalize_command(self):
        self.assertEqual(normalize_command("   python3   -m   unittest  -v  \n"), "python3 -m unittest -v")
        self.assertEqual(normalize_command(""), "")
        self.assertEqual(normalize_command(None), "")

    def test_is_test_runner(self):
        test_cmds = [
            "pytest tests/",
            "python3 -m unittest test_foo -v",
            "python -m unittest",
            "vitest run",
            "jest --coverage",
            "go test ./pkg/...",
            "bundle exec rspec spec/",
            "npm test",
            "npx vitest run",
        ]
        for cmd in test_cmds:
            self.assertTrue(is_test_runner(cmd), f"Failed to identify test runner: {cmd}")

        non_test_cmds = [
            "cat pytest.txt",
            "git status",
            "ls -la",
            "python3 session_metrics.py",
            "echo 'testing'",
        ]
        for cmd in non_test_cmds:
            self.assertFalse(is_test_runner(cmd), f"Incorrectly identified as test runner: {cmd}")

    def test_is_code_file(self):
        self.assertFalse(is_code_file("README.md"))
        self.assertFalse(is_code_file("notes.txt"))
        self.assertFalse(is_code_file("package.json"))
        self.assertFalse(is_code_file("config.yaml"))
        self.assertFalse(is_code_file("deploy.yml"))
        self.assertFalse(is_code_file("/Users/dev/.gemini/antigravity/brain/session/file.py"))
        self.assertFalse(is_code_file("/tmp/scratch.py"))
        self.assertFalse(is_code_file("/repo/docs/plans/2026-10-21-plan.md"))
        self.assertTrue(is_code_file("app/models/user.rb"))
        self.assertTrue(is_code_file("pkg/server/handler.go"))
        self.assertTrue(is_code_file("src/index.ts"))
        self.assertTrue(is_code_file("hooks/session_metrics.py"))

    def test_is_complaint(self):
        complaints = [
            "why is this taking so long?",
            "why are you so slow",
            "taking so long to respond",
            "please stop right now",
            "stop",
        ]
        for c in complaints:
            self.assertTrue(is_complaint(c), f"Failed to identify complaint: {c}")

        non_complaints = [
            "good job",
            "continue",
            "what does stop_service do?",
            "slow down is not matched",
        ]
        for nc in non_complaints:
            self.assertFalse(is_complaint(nc), f"Incorrectly identified complaint: {nc}")

    def test_compaction_step_indices(self):
        steps = [
            {"step_index": 0, "type": "USER_INPUT", "content": "hello CONTEXT_SUMMARY"},
            {"step_index": 1, "type": "CHECKPOINT", "content": "{{ CHECKPOINT 0 }}"},
            {"step_index": 2, "type": "RUN_COMMAND", "content": "# Resuming from a compaction in output"},
            {"step_index": 3, "type": "PLANNER_RESPONSE", "content": "# Resuming from a compaction in thought"},
            {"step_index": 4, "type": "CHECKPOINT", "content": "# Resuming from a compaction\n\nPrevious context summary..."},
            {"step_index": 5, "type": "USER_INPUT", "content": "next task"},
            {"step_index": 6, "type": "CHECKPOINT", "content": "   # Resuming from a compaction with leading spaces"},
        ]
        indices = compaction_step_indices(steps)
        self.assertEqual(indices, [4, 6])

        steps_offset = [
            {"step_index": 10, "type": "USER_INPUT", "content": "hello"},
            {"step_index": 25, "type": "CHECKPOINT", "content": "# Resuming from a compaction\n\n1"},
            {"step_index": 30, "type": "USER_INPUT", "content": "next"},
            {"step_index": 99, "type": "CHECKPOINT", "content": "# Resuming from a compaction\n\n2"},
        ]
        self.assertEqual(compaction_step_indices(steps_offset), [1, 3])

class TestSessionMetrics(unittest.TestCase):
    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        shutil.rmtree(self.tmpdir)

    def _create_conv_dir(self, conv_id):
        logs_dir = os.path.join(self.tmpdir, conv_id, ".system_generated", "logs")
        os.makedirs(logs_dir, exist_ok=True)
        return logs_dir

    def test_main_vs_helper_split(self):
        logs_main = self._create_conv_dir("main-conv")
        main_steps = [
            {"step_index": 0, "type": "USER_INPUT", "content": "do work", "created_at": "2026-10-21T12:00:00Z"},
            {"step_index": 1, "type": "PLANNER_RESPONSE", "tool_calls": [{"name": "run_command", "args": {"CommandLine": "ls"}}], "created_at": "2026-10-21T12:00:05Z"},
        ]
        with open(os.path.join(logs_main, "transcript.jsonl"), "w") as f:
            for s in main_steps:
                f.write(json.dumps(s) + "\n")

        logs_helper = self._create_conv_dir("helper-conv")
        helper_steps = [
            {"step_index": 0, "type": "SYSTEM_MESSAGE", "content": "task instructions", "created_at": "2026-10-21T12:00:00Z"},
            {"step_index": 1, "type": "PLANNER_RESPONSE", "tool_calls": [{"name": "view_file", "args": {"AbsolutePath": "/repo/file.py"}}], "created_at": "2026-10-21T12:00:05Z"},
        ]
        with open(os.path.join(logs_helper, "transcript.jsonl"), "w") as f:
            for s in helper_steps:
                f.write(json.dumps(s) + "\n")

        data = process_transcripts(root=self.tmpdir, last_n=10)
        self.assertEqual(len(data["main_sessions"]), 1)
        self.assertEqual(data["main_sessions"][0]["conversation_id"], "main-conv")
        self.assertEqual(len(data["helper_agents"]), 1)
        self.assertEqual(data["helper_agents"][0]["conversation_id"], "helper-conv")
        self.assertEqual(data["helper_agents"][0]["n_planner_steps"], 1)

    def test_edit_resets_loop_count(self):
        logs_dir = self._create_conv_dir("loop-reset-conv")
        steps = [
            {"step_index": 0, "type": "USER_INPUT", "content": "investigate", "created_at": "2026-10-21T12:00:00Z"},
            {"step_index": 1, "type": "PLANNER_RESPONSE", "tool_calls": [
                {"name": "view_file", "args": {"AbsolutePath": "/repo/app.py", "StartLine": 1, "EndLine": 50}},
                {"name": "view_file", "args": {"AbsolutePath": "/repo/app.py", "StartLine": 20, "EndLine": 60}},
                {"name": "edit_file", "args": {"TargetFile": "/repo/app.py"}},
                {"name": "view_file", "args": {"AbsolutePath": "/repo/app.py", "StartLine": 1, "EndLine": 50}},
            ], "created_at": "2026-10-21T12:00:10Z"},
        ]
        with open(os.path.join(logs_dir, "transcript.jsonl"), "w") as f:
            for s in steps:
                f.write(json.dumps(s) + "\n")

        res = analyze_session(os.path.join(logs_dir, "transcript.jsonl"))
        self.assertEqual(res["turns"][0]["true_loop_hits"], 0)

        steps_loop = [
            {"step_index": 0, "type": "USER_INPUT", "content": "investigate loop", "created_at": "2026-10-21T12:00:00Z"},
            {"step_index": 1, "type": "PLANNER_RESPONSE", "tool_calls": [
                {"name": "view_file", "args": {"AbsolutePath": "/repo/app.py", "StartLine": 1, "EndLine": 50}},
                {"name": "view_file", "args": {"AbsolutePath": "/repo/app.py", "StartLine": 20, "EndLine": 60}},
                {"name": "view_file", "args": {"AbsolutePath": "/repo/app.py", "StartLine": 30, "EndLine": 70}},
            ], "created_at": "2026-10-21T12:00:10Z"},
        ]
        with open(os.path.join(logs_dir, "transcript.jsonl"), "w") as f:
            for s in steps_loop:
                f.write(json.dumps(s) + "\n")

        res_loop = analyze_session(os.path.join(logs_dir, "transcript.jsonl"))
        self.assertEqual(res_loop["turns"][0]["true_loop_hits"], 1)

    def test_system_message_gap_excluded_from_active_time(self):
        logs_dir = self._create_conv_dir("active-time-conv")
        steps = [
            {"step_index": 0, "type": "USER_INPUT", "content": "run task", "created_at": "2026-10-21T12:00:00Z"},
            {"step_index": 1, "type": "PLANNER_RESPONSE", "tool_calls": [{"name": "run_command", "args": {"CommandLine": "ls"}}], "created_at": "2026-10-21T12:00:10Z"},
            {"step_index": 2, "type": "SYSTEM_MESSAGE", "content": "subagent finished", "created_at": "2026-10-21T12:10:10Z"},
            {"step_index": 3, "type": "PLANNER_RESPONSE", "tool_calls": [{"name": "run_command", "args": {"CommandLine": "pwd"}}], "created_at": "2026-10-21T12:10:15Z"},
        ]
        with open(os.path.join(logs_dir, "transcript.jsonl"), "w") as f:
            for s in steps:
                f.write(json.dumps(s) + "\n")

        res = analyze_session(os.path.join(logs_dir, "transcript.jsonl"))
        self.assertEqual(res["turns"][0]["active_seconds"], 15.0)

    def test_estimate_resets_after_compaction(self):
        logs_dir = self._create_conv_dir("compaction-conv")
        compact_file = os.path.join(logs_dir, "transcript.jsonl")
        full_file = os.path.join(logs_dir, "transcript_full.jsonl")

        step0 = {"step_index": 0, "type": "USER_INPUT", "content": "x" * 400, "created_at": "2026-10-21T12:00:00Z"}
        step1 = {"step_index": 1, "type": "CHECKPOINT", "content": "# Resuming from a compaction\n\nState...", "created_at": "2026-10-21T12:00:01Z"}
        step2 = {"step_index": 2, "type": "USER_INPUT", "content": "y" * 100, "created_at": "2026-10-21T12:00:02Z"}

        for path in (compact_file, full_file):
            with open(path, "w") as f:
                f.write(json.dumps(step0) + "\n")
                f.write(json.dumps(step1) + "\n")
                f.write(json.dumps(step2) + "\n")

        res = analyze_session(compact_file)
        self.assertEqual(res["compaction_step_indices"], [1])
        self.assertEqual(len(res["segment_estimates_at_compaction"]), 1)
        self.assertLess(res["context_estimate_series"][-1], res["segment_estimates_at_compaction"][0])

    def test_compaction_step_indices_with_offset_step_index(self):
        logs_dir = self._create_conv_dir("compaction-offset-conv")
        compact_file = os.path.join(logs_dir, "transcript.jsonl")
        full_file = os.path.join(logs_dir, "transcript_full.jsonl")

        step0 = {"step_index": 5, "type": "USER_INPUT", "content": "a" * 800, "created_at": "2026-10-21T12:00:00Z"}
        step1 = {"step_index": 12, "type": "CHECKPOINT", "content": "# Resuming from a compaction\n\nState...", "created_at": "2026-10-21T12:00:01Z"}
        step2 = {"step_index": 18, "type": "USER_INPUT", "content": "b" * 200, "created_at": "2026-10-21T12:00:02Z"}

        for path in (compact_file, full_file):
            with open(path, "w") as f:
                f.write(json.dumps(step0) + "\n")
                f.write(json.dumps(step1) + "\n")
                f.write(json.dumps(step2) + "\n")

        res = analyze_session(compact_file)
        self.assertEqual(res["compaction_step_indices"], [1])
        self.assertEqual(len(res["segment_estimates_at_compaction"]), 1)
        self.assertGreater(res["segment_estimates_at_compaction"][0], 0)
        self.assertEqual(res["context_estimate_series"][1], 0)
        self.assertLess(res["context_estimate_series"][-1], res["segment_estimates_at_compaction"][0])

    def test_code_edits_reset_on_subagent_and_skip_md(self):
        logs_dir = self._create_conv_dir("code-edits-conv")
        steps = [
            {"step_index": 0, "type": "USER_INPUT", "content": "/goal build feature", "created_at": "2026-10-21T12:00:00Z"},
            {"step_index": 1, "type": "PLANNER_RESPONSE", "tool_calls": [
                {"name": "write_file", "args": {"Path": "/repo/docs/plan.md"}},
                {"name": "write_file", "args": {"Path": "/repo/notes.txt"}},
                {"name": "edit_file", "args": {"TargetFile": "/repo/src/a.py"}},
                {"name": "edit_file", "args": {"TargetFile": "/repo/src/b.py"}},
                {"name": "invoke_subagent", "args": {"TypeName": "test"}},
                {"name": "edit_file", "args": {"TargetFile": "/repo/src/c.py"}},
            ], "created_at": "2026-10-21T12:00:10Z"},
        ]
        with open(os.path.join(logs_dir, "transcript.jsonl"), "w") as f:
            for s in steps:
                f.write(json.dumps(s) + "\n")

        res = analyze_session(os.path.join(logs_dir, "transcript.jsonl"))
        self.assertEqual(res["turns"][0]["code_edits_by_main"], 2)

if __name__ == "__main__":
    unittest.main()
