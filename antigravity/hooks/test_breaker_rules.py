import json
import os
import shutil
import tempfile
import unittest

from breaker_rules import (
    rule_mode,
    probe_target_path,
    append_breaker_log,
    extract_conversation_id,
    context_estimate_tokens,
    load_baton_state,
    save_baton_state,
)

class TestBreakerRulesNew(unittest.TestCase):
    def test_rule_mode_defaults(self):
        self.assertEqual(rule_mode("PROBE"), "log")
        self.assertEqual(rule_mode("TRUELOOP"), "log")
        self.assertEqual(rule_mode("CHECKPOINT"), "log")
        self.assertEqual(rule_mode("DUP"), "enforce")
        self.assertEqual(rule_mode("EXPLORE"), "enforce")
        self.assertEqual(rule_mode("STRIKES"), "enforce")
        self.assertEqual(rule_mode("BATON"), "enforce")
        self.assertEqual(rule_mode("H1"), "log")
        self.assertEqual(rule_mode("H2"), "log")

    def test_rule_mode_env_override(self):
        try:
            os.environ["BREAKER_PROBE_MODE"] = "enforce"
            os.environ["BREAKER_TRUELOOP_MODE"] = "off"
            os.environ["BREAKER_DUP_MODE"] = "log"
            self.assertEqual(rule_mode("probe"), "enforce")
            self.assertEqual(rule_mode("TRUELOOP"), "off")
            self.assertEqual(rule_mode("dup"), "log")
        finally:
            os.environ.pop("BREAKER_PROBE_MODE", None)
            os.environ.pop("BREAKER_TRUELOOP_MODE", None)
            os.environ.pop("BREAKER_DUP_MODE", None)

    def test_probe_target_path(self):
        cmd1 = 'python3 -c "import json; f = open(\'/tmp/output.txt\')"'
        self.assertEqual(probe_target_path(cmd1), "/tmp/output.txt")

        cmd2 = 'python3 -c \'import re; print(re.search(r"foo|bar", open("/tmp/output.txt").read()))\''
        self.assertEqual(probe_target_path(cmd2), "/tmp/output.txt")

        cmd3 = 'python3 - <<EOF\nprint(open("/tmp/data.csv").read())\nEOF'
        self.assertEqual(probe_target_path(cmd3), "/tmp/data.csv")

        cmd4 = 'python3 /tmp/parse.py /tmp/output.txt'
        self.assertIsNone(probe_target_path(cmd4))

        cmd5 = 'python3 -c "print(1)"'
        self.assertIsNone(probe_target_path(cmd5))

        cmd6 = 'cat /tmp/output.txt'
        self.assertIsNone(probe_target_path(cmd6))

        self.assertIsNone(probe_target_path('python3 -c "print(\'/- \')"'))
        self.assertIsNone(probe_target_path('python3 -c "print(\'/dev\')"'))
        self.assertIsNone(probe_target_path('python3 -c "print(\'/x\')"'))
        self.assertIsNone(probe_target_path('python3 -c "print(\'/dev/null\')"'))
        self.assertIsNone(probe_target_path('python3 -c "print(\'/dev/stdin\')"'))

        bfa3cfd7_cmd = 'gh pr list --repo work-org/example-repo --state all --limit 100 | python3 -c "\nimport sys, json\nfor p in team_prs:\n    print(f\\\"(+{p[\'additions\']}/-{p[\'deletions\']})\\\")\n"'
        self.assertIsNone(probe_target_path(bfa3cfd7_cmd))

        cmd_rel = "python3 - <<'EOF'\nopen('docs/plans/a.md')\nEOF"
        self.assertIsNone(probe_target_path(cmd_rel))

        cmd_tilde = 'python3 -c "open(\'~/.gemini/x/y.json\')"'
        self.assertEqual(probe_target_path(cmd_tilde), os.path.expanduser("~/.gemini/x/y.json"))

        cmd_glob = 'python3 -c "open(\'~/.gemini/brain/*/logs/t.jsonl\')"'
        cmd_specific = 'python3 -c "open(\'~/.gemini/brain/abc/logs/t.jsonl\')"'
        self.assertIsNone(probe_target_path(cmd_glob))
        self.assertEqual(probe_target_path(cmd_specific), os.path.expanduser("~/.gemini/brain/abc/logs/t.jsonl"))

        url_cmd = 'python3 -c "import urllib.request; urllib.request.urlopen(\'https://api.github.com/repos/foo/bar\')"'
        self.assertIsNone(probe_target_path(url_cmd))
        self.assertIsNone(probe_target_path('python3 -c "open(\'//foo/bar\')"'))
        self.assertIsNone(probe_target_path('python3 -c "open(\'http:/foo/bar\')"'))

    def test_append_breaker_log(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            log_path = os.path.join(tmpdir, "nested", "breaker.jsonl")
            entry = {
                "ts": "2026-10-21T12:00:00Z",
                "conversation_id": "conv-1",
                "is_subagent": False,
                "rule": "PROBE",
                "target": "/tmp/output.txt",
                "mode": "log"
            }
            append_breaker_log(entry, log_path=log_path)
            self.assertTrue(os.path.exists(log_path))
            with open(log_path, "r", encoding="utf-8") as f:
                lines = f.readlines()
            self.assertEqual(len(lines), 1)
            self.assertEqual(json.loads(lines[0]), entry)

    def test_append_breaker_log_unwritable_swallowed(self):
        entry = {"rule": "PROBE"}
        append_breaker_log(entry, log_path="/nonexistent-root/x/y/breaker.jsonl")

    def test_extract_conversation_id(self):
        payload1 = {
            "transcriptPath": "/Users/you/.gemini/antigravity/brain/5c16b26c-b0a8-43f4-85cc-cd31c35c059c/.system_generated/logs/transcript.jsonl"
        }
        self.assertEqual(extract_conversation_id(payload1), "5c16b26c-b0a8-43f4-85cc-cd31c35c059c")

        payload2 = {
            "transcriptPath": "/tmp/some/test/path/transcript.jsonl",
            "conversationId": "fallback-id"
        }
        self.assertEqual(extract_conversation_id(payload2), "fallback-id")

    def test_context_estimate_tokens(self):
        bytes_list = [400, 400, 800]
        self.assertEqual(context_estimate_tokens(bytes_list, []), 400)
        self.assertEqual(context_estimate_tokens(bytes_list, [0]), 300)
        self.assertEqual(context_estimate_tokens(bytes_list, [1]), 200)
        self.assertEqual(context_estimate_tokens(bytes_list, [2]), 0)
        self.assertEqual(context_estimate_tokens([], []), 0)

    def test_load_and_save_baton_state(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            initial = load_baton_state("conv-xyz", 0, state_dir=tmpdir)
            self.assertEqual(initial, {"warned_at_estimate": None, "handoff_at_estimate": None, "segment": 0})

            saved_data = {"warned_at_estimate": 105000, "handoff_at_estimate": None, "segment": 0}
            save_baton_state("conv-xyz", saved_data, state_dir=tmpdir)

            loaded = load_baton_state("conv-xyz", 0, state_dir=tmpdir)
            self.assertEqual(loaded["warned_at_estimate"], 105000)
            self.assertIsNone(loaded["handoff_at_estimate"])
            self.assertEqual(loaded["segment"], 0)

            new_segment_loaded = load_baton_state("conv-xyz", 1, state_dir=tmpdir)
            self.assertIsNone(new_segment_loaded["warned_at_estimate"])
            self.assertIsNone(new_segment_loaded["handoff_at_estimate"])
            self.assertEqual(new_segment_loaded["segment"], 1)

            other_conv = load_baton_state("conv-abc", 0, state_dir=tmpdir)
            self.assertIsNone(other_conv["warned_at_estimate"])
            self.assertIsNone(other_conv["handoff_at_estimate"])

    def test_save_baton_state_unwritable_fails_open(self):
        save_baton_state("conv-fail", {"warned_at_estimate": 100}, state_dir="/nonexistent/root/dir")

    def test_verify_live_hooks_json(self):
        hooks_path = os.path.expanduser("~/.gemini/config/hooks.json")
        with open(hooks_path, "r", encoding="utf-8") as f:
            data = json.load(f)

        all_required = [
            "call_mcp_tool", "edit_file", "find_by_name", "grep_search", "invoke_subagent",
            "list_dir", "manage_task", "read_file", "read_url_content", "replace_file_content",
            "run_command", "schedule", "search_web", "send_message", "view_file",
            "write_file", "write_to_file"
        ]
        current = sorted(m["matcher"] for m in data["autonomous-circuit-breaker"]["PreToolUse"])
        self.assertEqual(current, sorted(all_required))
        self.assertNotIn("baton-pass-subagent-gate", data)

    def test_py_compile_all_modules(self):
        import py_compile
        hooks_dir = os.path.dirname(os.path.abspath(__file__))
        for mod in ("autonomous_breaker.py", "breaker_rules.py", "session_metrics.py"):
            py_compile.compile(os.path.join(hooks_dir, mod), doraise=True)

    def test_shasum_hooks_json_and_state_dir(self):
        import hashlib
        hooks_path = os.path.expanduser("~/.gemini/config/hooks.json")
        with open(hooks_path, "rb") as f:
            h = hashlib.sha1(f.read()).hexdigest()
        print(f"\n[SHASUM] hooks.json: {h}")

        state_dir = os.path.expanduser("~/.gemini/state")
        baton_files = []
        if os.path.exists(state_dir):
            baton_files = [f for f in os.listdir(state_dir) if f.startswith("baton-")]
        print(f"[STATE_DIR] baton files in ~/.gemini/state/: {baton_files}")
        self.assertEqual(baton_files, [])

    def test_r1_authorized_patterns(self):
        from breaker_rules import is_authorized
        self.assertTrue(is_authorized("Start Phase 3 (Ingest) for pulseD"))
        self.assertTrue(is_authorized("Start Phase 4 (Query Surface)"))
        self.assertTrue(is_authorized("Read /x/docs/plans/BRIEF-phase-2.md in full before doing anything. It is your complete instructions for Phase 2"))
        self.assertTrue(is_authorized("your complete instructions for Phase 2"))
        self.assertFalse(is_authorized("provide complete instructions for setup"))
        self.assertFalse(is_authorized("did we run phase tests yet?"))
        self.assertFalse(is_authorized("how do I execute phase queries?"))
        self.assertFalse(is_authorized("what's the next phase"))
        self.assertTrue(is_authorized("see brief-phase-1.md"))
        self.assertTrue(is_authorized("check handoff-phase-2.txt"))

    def test_r2_read_coverage_helpers(self):
        from breaker_rules import read_coverage, is_re_read
        self.assertLess(read_coverage((300, 400), [(400, 500)]), 0.5)
        self.assertFalse(is_re_read((300, 400), [(400, 500)]))

        self.assertLess(read_coverage((98, 153), [(1, 100)]), 0.5)
        self.assertFalse(is_re_read((98, 153), [(1, 100)]))

        self.assertAlmostEqual(read_coverage((10, 60), [(10, 60)]), 1.0)
        self.assertTrue(is_re_read((10, 60), [(10, 60)]))

        self.assertAlmostEqual(read_coverage((10, 60), [(None, None)]), 1.0)
        self.assertTrue(is_re_read((10, 60), [(None, None)]))

if __name__ == "__main__":
    unittest.main()
