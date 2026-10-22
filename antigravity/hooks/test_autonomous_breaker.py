#!/usr/bin/env python3
import json
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

BREAKER_SCRIPT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "autonomous-breaker.sh")

class TestAutonomousBreaker(unittest.TestCase):

    def setUp(self):
        self.test_tmpdir = tempfile.mkdtemp()
        real_state_dir = os.path.expanduser("~/.gemini/state")
        if os.path.exists(real_state_dir):
            for fname in os.listdir(real_state_dir):
                if fname.startswith("baton-"):
                    try:
                        os.remove(os.path.join(real_state_dir, fname))
                    except Exception:
                        pass

    def tearDown(self):
        shutil.rmtree(self.test_tmpdir, ignore_errors=True)

    def execute_breaker(self, payload, env=None):
        proc_env = os.environ.copy()
        proc_env["BREAKER_LOG_PATH"] = os.path.join(self.test_tmpdir, "breaker.jsonl")
        proc_env["BREAKER_STATE_DIR"] = self.test_tmpdir
        if env:
            proc_env.update(env)
        res = subprocess.run(
            [BREAKER_SCRIPT],
            input=json.dumps(payload),
            capture_output=True,
            text=True,
            env=proc_env
        )
        return res

    def write_jsonl(self, file_path, items):
        with open(file_path, "w", encoding="utf-8") as f:
            for item in items:
                f.write(json.dumps(item) + "\n")

    def test_passthrough_outputs_empty_stdout_never_allow(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "hello"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [{"name": "read_file", "args": {"Path": "/tmp/a.py"}}]}
            ])
            payload = {
                "conversationId": "test-passthrough",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {
                    "name": "view_file",
                    "args": {"AbsolutePath": "/tmp/b.py", "StartLine": 1, "EndLine": 800}
                }
            }
            res = self.execute_breaker(payload)
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")
            self.assertNotIn("allow", res.stdout)

    def test_tripwire1_blocks_exact_duplicate_call_in_window(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "review"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "view_file", "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "GENERIC", "content": "file contents"}
            ])
            payload = {
                "conversationId": "test-dup",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {
                    "name": "read_file",
                    "args": {"Path": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}
                }
            }
            res = self.execute_breaker(payload)
            self.assertEqual(res.returncode, 0)
            data = json.loads(res.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Duplicate Call)", data.get("reason", ""))

    def test_tripwire1_blocks_view_file_repeated_around_readonly_command(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "find caller"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "view_file", "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}}
                ]},
                {"step_index": 2, "source": "SYSTEM", "type": "GENERIC", "content": "file contents"},
                {"step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "grep foo /tmp/x.rb"}}
                ]},
                {"step_index": 4, "source": "SYSTEM", "type": "GENERIC", "content": "The command exited with code 0."}
            ])
            payload = {
                "conversationId": "test-readonly-dup",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {
                    "name": "view_file",
                    "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}
                }
            }
            res = self.execute_breaker(payload)
            self.assertEqual(res.returncode, 0)
            data = json.loads(res.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Duplicate Call)", data.get("reason", ""))

    def test_tripwire1_allows_different_line_ranges_or_mcp_ids(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "read more"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "view_file", "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "call_mcp_tool", "args": {"ServerName": "openwhispr", "ToolName": "get_note", "Arguments": {"id": "note-1"}}}
                ]}
            ])
            payload_view = {
                "conversationId": "test-diff-args",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {
                    "name": "view_file",
                    "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 801, "EndLine": 1600}
                }
            }
            res_view = self.execute_breaker(payload_view)
            self.assertEqual(res_view.returncode, 0)
            self.assertEqual(res_view.stdout, "")

            payload_mcp = {
                "conversationId": "test-diff-args",
                "transcriptPath": str(t_path),
                "stepIdx": 5,
                "toolCall": {
                    "name": "call_mcp_tool",
                    "args": {"ServerName": "openwhispr", "ToolName": "get_note", "Arguments": {"id": "note-2"}}
                }
            }
            res_mcp = self.execute_breaker(payload_mcp)
            self.assertEqual(res_mcp.returncode, 0)
            self.assertEqual(res_mcp.stdout, "")

    def test_tripwire1_allows_re_read_or_re_test_after_file_edit(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "edit and re-read"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "view_file", "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "replace_file_content", "args": {"TargetFile": "/tmp/x.rb", "TargetContent": "a", "ReplacementContent": "b"}}
                ]}
            ])
            payload_read = {
                "conversationId": "test-reset-edit",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {
                    "name": "view_file",
                    "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}
                }
            }
            res_read = self.execute_breaker(payload_read)
            self.assertEqual(res_read.returncode, 0)
            self.assertEqual(res_read.stdout, "")

            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "test and edit"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "pytest"}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "edit_file", "args": {"Path": "/tmp/x.rb", "Target": "a", "Replacement": "b"}}
                ]}
            ])
            payload_test = {
                "conversationId": "test-reset-edit",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "pytest"}
                }
            }
            res_test = self.execute_breaker(payload_test)
            self.assertEqual(res_test.returncode, 0)
            self.assertEqual(res_test.stdout, "")

    def test_tripwire1_decodes_json_encoded_arg_strings(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "list files"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "\"ls -la /tmp\""}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "GENERIC", "content": "output"}
            ])
            payload = {
                "conversationId": "test-json-decode",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "ls -la /tmp"}
                }
            }
            res = self.execute_breaker(payload)
            self.assertEqual(res.returncode, 0)
            data = json.loads(res.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Duplicate Call)", data.get("reason", ""))

    def test_tripwire1_resets_across_user_input(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "turn 1"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "view_file", "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}}
                ]},
                {"step_index": 2, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "turn 2"}
            ])
            payload = {
                "conversationId": "test-user-reset",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {
                    "name": "view_file",
                    "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}
                }
            }
            res = self.execute_breaker(payload)
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_tripwire2_counts_run_command_and_blocks_subagent_after_explore_budget(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            subagent_steps = [
                {"step_index": 0, "source": "SYSTEM", "type": "SYSTEM_MESSAGE", "content": "subagent start"}
            ]
            step_idx = 1
            for i in range(12):
                subagent_steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"find . -name file_{i}.rb"}}]
                })
                step_idx += 2
            for i in range(6):
                subagent_steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "call_mcp_tool", "args": {"ServerName": "openwhispr", "ToolName": "get_note", "Arguments": {"id": f"note_{i}"}}}]
                })
                step_idx += 2
            for i in range(2):
                subagent_steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "view_file", "args": {"AbsolutePath": f"/tmp/source_{i}.rb", "StartLine": 1, "EndLine": 800}}]
                })
                step_idx += 2

            self.write_jsonl(t_path, subagent_steps)
            payload = {
                "conversationId": "subagent-budget-test",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {
                    "name": "read_file",
                    "args": {"Path": "/tmp/new_source.rb", "StartLine": 1, "EndLine": 800}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_SUBAGENT_EXPLORE_MAX": "20"})
            self.assertEqual(res.returncode, 0)
            data = json.loads(res.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Subagent Exploration Budget Reached)", data.get("reason", ""))

    def test_tripwire2_never_blocks_main_session_exploration(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            main_steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "explore widely"}
            ]
            step_idx = 1
            for i in range(30):
                main_steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"echo exploratory_{i}"}}]
                })
                step_idx += 2
            self.write_jsonl(t_path, main_steps)
            payload = {
                "conversationId": "main-no-budget",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {
                    "name": "view_file",
                    "args": {"AbsolutePath": "/tmp/another.rb", "StartLine": 1, "EndLine": 800}
                }
            }
            res = self.execute_breaker(payload)
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_tripwire3_blocks_subagent_after_4_consecutive_test_failures(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            test_commands = [
                "npx vitest run test/a.spec.ts",
                "bundle exec rspec spec/models/a_spec.rb",
                "pytest tests/test_a.py",
                "go test ./pkg/..."
            ]
            steps = [
                {"step_index": 0, "source": "SYSTEM", "type": "SYSTEM_MESSAGE", "content": "subagent start"}
            ]
            step_idx = 1
            for cmd in test_commands:
                steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": cmd}}]
                })
                step_idx += 1
                steps.append({
                    "step_index": step_idx, "source": "SYSTEM", "type": "GENERIC",
                    "content": "The command exited with code 1."
                })
                step_idx += 1
            self.write_jsonl(t_path, steps)

            payload_cmd = {
                "conversationId": "subagent-strikes",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "go test ./pkg/..."}
                }
            }
            res_cmd = self.execute_breaker(payload_cmd)
            self.assertEqual(res_cmd.returncode, 0)
            data_cmd = json.loads(res_cmd.stdout)
            self.assertEqual(data_cmd.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (4-Strike Test-Fix Escalation)", data_cmd.get("reason", ""))

            payload_edit = {
                "conversationId": "subagent-strikes",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {
                    "name": "edit_file",
                    "args": {"Path": "/tmp/fix.go", "Target": "foo", "Replacement": "bar"}
                }
            }
            res_edit = self.execute_breaker(payload_edit)
            self.assertEqual(res_edit.returncode, 0)
            data_edit = json.loads(res_edit.stdout)
            self.assertEqual(data_edit.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (4-Strike Test-Fix Escalation)", data_edit.get("reason", ""))

    def test_tripwire3_resets_when_a_test_passes(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "SYSTEM", "type": "SYSTEM_MESSAGE", "content": "subagent start"}
            ]
            step_idx = 1
            for _ in range(3):
                steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": "pytest tests/test_foo.py"}}]
                })
                step_idx += 1
                steps.append({
                    "step_index": step_idx, "source": "SYSTEM", "type": "GENERIC",
                    "content": "The command exited with code 1."
                })
                step_idx += 1

            steps.append({
                "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "tool_calls": [{"name": "run_command", "args": {"CommandLine": "pytest tests/test_foo.py"}}]
            })
            step_idx += 1
            steps.append({
                "step_index": step_idx, "source": "SYSTEM", "type": "GENERIC",
                "content": "The command exited with code 0."
            })
            step_idx += 1

            steps.append({
                "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "tool_calls": [{"name": "run_command", "args": {"CommandLine": "pytest tests/test_foo.py"}}]
            })
            step_idx += 1
            steps.append({
                "step_index": step_idx, "source": "SYSTEM", "type": "GENERIC",
                "content": "The command exited with code 1."
            })
            step_idx += 1

            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "subagent-test-reset",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {
                    "name": "edit_file",
                    "args": {"Path": "/tmp/fix.py", "Target": "x", "Replacement": "y"}
                }
            }
            res = self.execute_breaker(payload)
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_truncated_tool_calls_and_fail_open(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "check truncated"},
                {
                    "step_index": 1,
                    "source": "MODEL",
                    "type": "PLANNER_RESPONSE",
                    "truncated_fields": ["tool_calls"],
                    "tool_calls": []
                }
            ])
            self.write_jsonl(t_full_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "check truncated"},
                {
                    "step_index": 1,
                    "source": "MODEL",
                    "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "read_file", "args": {"Path": "/tmp/full.txt", "StartLine": 1, "EndLine": 800}}]
                }
            ])
            payload_trunc = {
                "conversationId": "trunc-test",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {
                    "name": "view_file",
                    "args": {"AbsolutePath": "/tmp/full.txt", "StartLine": 1, "EndLine": 800}
                }
            }
            res_trunc = self.execute_breaker(payload_trunc)
            self.assertEqual(res_trunc.returncode, 0)
            data_trunc = json.loads(res_trunc.stdout)
            self.assertEqual(data_trunc.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Duplicate Call)", data_trunc.get("reason", ""))

            payload_missing = {
                "conversationId": "missing-file",
                "transcriptPath": "/tmp/does-not-exist-transcript.jsonl",
                "stepIdx": 1,
                "toolCall": {"name": "read_file", "args": {"Path": "/tmp/foo"}}
            }
            res_missing = self.execute_breaker(payload_missing)
            self.assertEqual(res_missing.returncode, 0)
            self.assertEqual(res_missing.stdout, "")

            malformed_path = Path(tmpdir) / "malformed.jsonl"
            with open(malformed_path, "w") as f:
                f.write("not json\n")
            payload_malformed = {
                "conversationId": "malformed-test",
                "transcriptPath": str(malformed_path),
                "stepIdx": 1,
                "toolCall": {"name": "read_file", "args": {"Path": "/tmp/foo"}}
            }
            res_malformed = self.execute_breaker(payload_malformed)
            self.assertEqual(res_malformed.returncode, 0)
            self.assertEqual(res_malformed.stdout, "")

            payload_exit1 = {
                "conversationId": "exit-tool-test",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {"name": "write_file", "args": {"Path": "/tmp/foo", "Content": "bar"}}
            }
            self.assertEqual(self.execute_breaker(payload_exit1).stdout, "")

            payload_exit2 = {
                "conversationId": "exit-tool-test",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {"name": "write_to_file", "args": {"TargetFile": "/tmp/foo", "CodeContent": "bar"}}
            }
            self.assertEqual(self.execute_breaker(payload_exit2).stdout, "")

            payload_exit3 = {
                "conversationId": "exit-tool-test",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {"name": "send_message", "args": {"Recipient": "parent", "Message": "done"}}
            }
            self.assertEqual(self.execute_breaker(payload_exit3).stdout, "")

            payload_null_conv = {
                "conversationId": None,
                "transcriptPath": None,
                "stepIdx": 1,
                "toolCall": {"name": "read_file", "args": {"Path": "/tmp/foo"}}
            }
            res_null = self.execute_breaker(payload_null_conv, env={"BREAKER_DUP_WINDOW": "not-an-int"})
            self.assertEqual(res_null.returncode, 0)
            self.assertEqual(res_null.stdout, "")

            payload_bad_env = {
                "conversationId": "exit-tool-test",
                "transcriptPath": str(t_path),
                "stepIdx": 1,
                "toolCall": {"name": "read_file", "args": {"Path": "/tmp/foo"}}
            }
            res_bad_env = self.execute_breaker(payload_bad_env, env={"BREAKER_DUP_WINDOW": "not-an-int"})
            self.assertEqual(res_bad_env.returncode, 0)
            self.assertEqual(res_bad_env.stdout, "")

    def test_live_inflight_planner_response_in_transcript_does_not_self_block(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            turn1_steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "read 3 files"},
                {
                    "step_index": 1,
                    "source": "MODEL",
                    "type": "PLANNER_RESPONSE",
                    "tool_calls": [
                        {"name": "read_file", "args": {"Path": "\"/tmp/f1.rb\""}},
                        {"name": "read_file", "args": {"Path": "\"/tmp/f2.rb\""}},
                        {"name": "read_file", "args": {"Path": "\"/tmp/f3.rb\""}}
                    ]
                }
            ]
            self.write_jsonl(t_path, turn1_steps)
            self.write_jsonl(t_full_path, turn1_steps)

            for idx, path_str in [(2, "/tmp/f1.rb"), (3, "/tmp/f2.rb"), (4, "/tmp/f3.rb")]:
                res = self.execute_breaker({
                    "conversationId": "live-inflight",
                    "transcriptPath": str(t_full_path),
                    "stepIdx": idx,
                    "toolCall": {"name": "read_file", "args": {"Path": path_str}}
                })
                self.assertEqual(res.returncode, 0)
                self.assertEqual(res.stdout, "")

            turn2_steps = turn1_steps + [
                {"step_index": 2, "source": "MODEL", "type": "GENERIC", "content": "f1"},
                {"step_index": 3, "source": "MODEL", "type": "GENERIC", "content": "f2"},
                {"step_index": 4, "source": "MODEL", "type": "GENERIC", "content": "f3"},
                {
                    "step_index": 5,
                    "source": "MODEL",
                    "type": "PLANNER_RESPONSE",
                    "tool_calls": [
                        {"name": "read_file", "args": {"Path": "\"/tmp/f1.rb\""}}
                    ]
                }
            ]
            self.write_jsonl(t_path, turn2_steps)
            self.write_jsonl(t_full_path, turn2_steps)

            res_dup = self.execute_breaker({
                "conversationId": "live-inflight",
                "transcriptPath": str(t_full_path),
                "stepIdx": 6,
                "toolCall": {"name": "read_file", "args": {"Path": "/tmp/f1.rb"}}
            })
            self.assertEqual(res_dup.returncode, 0)
            data_dup = json.loads(res_dup.stdout)
            self.assertEqual(data_dup.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Duplicate Call)", data_dup.get("reason", ""))

    def test_task9_deny_writes_log_line(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            log_path = Path(tmpdir) / "breaker.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "review"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "view_file", "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}}
                ]},
            ])
            payload = {
                "conversationId": "test-c1-log",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {
                    "name": "read_file",
                    "args": {"Path": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_LOG_PATH": str(log_path)})
            self.assertEqual(res.returncode, 0)
            data = json.loads(res.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertTrue(log_path.exists())
            with open(log_path, "r", encoding="utf-8") as f:
                log_lines = [json.loads(line) for line in f if line.strip()]
            self.assertEqual(len(log_lines), 1)
            entry = log_lines[0]
            self.assertEqual(entry["conversation_id"], "test-c1-log")
            self.assertEqual(entry["is_subagent"], False)
            self.assertEqual(entry["rule"], "DUP")
            self.assertEqual(entry["mode"], "enforce")
            self.assertEqual(entry["target"], "/tmp/x.rb")
            self.assertIn("ts", entry)

    def test_task9_log_mode_allows_and_logs(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            log_path = Path(tmpdir) / "breaker.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "review"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "view_file", "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}}
                ]},
            ])
            payload = {
                "conversationId": "test-c2-mode-log",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {
                    "name": "read_file",
                    "args": {"Path": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_LOG_PATH": str(log_path), "BREAKER_DUP_MODE": "log"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")
            self.assertTrue(log_path.exists())
            with open(log_path, "r", encoding="utf-8") as f:
                log_lines = [json.loads(line) for line in f if line.strip()]
            self.assertEqual(len(log_lines), 1)
            self.assertEqual(log_lines[0]["mode"], "log")
            self.assertEqual(log_lines[0]["rule"], "DUP")

    def test_task9_off_mode_does_nothing(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            log_path = Path(tmpdir) / "breaker.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "review"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "view_file", "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}}
                ]},
            ])
            payload = {
                "conversationId": "test-c2-mode-off",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {
                    "name": "read_file",
                    "args": {"Path": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_LOG_PATH": str(log_path), "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")
            self.assertFalse(log_path.exists())

    def test_task9_unwritable_log_path_preserves_decision(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "review"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "view_file", "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}}
                ]},
            ])
            payload = {
                "conversationId": "test-c1-unwritable",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {
                    "name": "read_file",
                    "args": {"Path": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_LOG_PATH": "/nonexistent-root/x/y.jsonl"})
            self.assertEqual(res.returncode, 0)
            data = json.loads(res.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Duplicate Call)", data.get("reason", ""))

    def test_probe_tripwire_2_probes_allow_3rd_denies(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "probe file"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "python3 -c \"open('/tmp/output.txt').read()\""}}
                ]},
                {"step_index": 2, "source": "SYSTEM", "type": "GENERIC", "content": "done"},
                {"step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "python3 -c \"import json; json.load(open('/tmp/output.txt'))\""}}
                ]},
                {"step_index": 4, "source": "SYSTEM", "type": "GENERIC", "content": "done"},
            ])
            payload_allow = {
                "conversationId": "test-probe-2",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "python3 -c \"import json; json.load(open('/tmp/output.txt'))\""}
                }
            }
            res_allow = self.execute_breaker(payload_allow, env={"BREAKER_PROBE_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res_allow.returncode, 0)

            payload_deny = {
                "conversationId": "test-probe-3",
                "transcriptPath": str(t_path),
                "stepIdx": 5,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "python3 -c \"len(open('/tmp/output.txt').readlines())\""}
                }
            }
            res_deny = self.execute_breaker(payload_deny, env={"BREAKER_PROBE_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res_deny.returncode, 0)
            data = json.loads(res_deny.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertIn("3rd inline probe of /tmp/output.txt", data.get("reason", ""))
            self.assertIn("write /tmp/parse_output.py once and run it", data.get("reason", ""))

    def test_probe_tripwire_probes_of_3_different_paths_allow(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "probe 3 files"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "python3 -c \"open('/tmp/a.txt')\""}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "python3 -c \"open('/tmp/b.txt')\""}}
                ]},
            ])
            payload = {
                "conversationId": "test-probe-diff",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "python3 -c \"open('/tmp/c.txt')\""}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_PROBE_MODE": "enforce"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_probe_tripwire_user_input_resets(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "probe 1"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "python3 -c \"open('/tmp/output.txt')\""}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "python3 -c \"len(open('/tmp/output.txt').read())\""}}
                ]},
                {"step_index": 3, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "probe 2"},
            ])
            payload = {
                "conversationId": "test-probe-reset",
                "transcriptPath": str(t_path),
                "stepIdx": 5,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "python3 -c \"open('/tmp/output.txt').close()\""}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_PROBE_MODE": "enforce"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_probe_tripwire_script_file_allows(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [{"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "run scripts"}]
            for i in range(4):
                steps.append({
                    "step_index": i + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"python3 /tmp/parse.py /tmp/output.txt --part {i}"}}]
                })
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-probe-script",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "python3 /tmp/parse.py /tmp/output.txt --part 4"}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_PROBE_MODE": "enforce"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_probe_tripwire_no_path_allows(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [{"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "run inline math"}]
            for i in range(4):
                steps.append({
                    "step_index": i + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"python3 -c \"print({i})\""}}]
                })
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-probe-nopath",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "python3 -c \"print(5)\""}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_PROBE_MODE": "enforce"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_probe_tripwire_single_quotes_and_pipe_regex(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            cmd_pipe = "python3 -c 'import re; print(re.search(r\"foo|bar\", open(\"/tmp/output.txt\").read()))'"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "pipe regex"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": cmd_pipe}}
                ]},
                {"step_index": 2, "source": "SYSTEM", "type": "GENERIC", "content": "match"},
                {"step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "python3 -c 'print(open(\"/tmp/output.txt\").read(10))'"}}
                ]},
                {"step_index": 4, "source": "SYSTEM", "type": "GENERIC", "content": "done"},
            ])
            payload = {
                "conversationId": "test-probe-pipe",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": cmd_pipe}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_PROBE_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            data = json.loads(res.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertIn("3rd inline probe of /tmp/output.txt", data.get("reason", ""))

    def test_probe_tripwire_edit_resets(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "probe and edit"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "python3 -c \"open('/tmp/output.txt')\""}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "python3 -c \"len(open('/tmp/output.txt').read())\""}}
                ]},
                {"step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "edit_file", "args": {"Path": "/tmp/output.txt", "Target": "old", "Replacement": "new"}}
                ]},
            ])
            payload = {
                "conversationId": "test-probe-edit-reset",
                "transcriptPath": str(t_path),
                "stepIdx": 5,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "python3 -c \"open('/tmp/output.txt').read()\""}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_PROBE_MODE": "enforce"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_probe_tripwire_default_log_mode_allows(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            log_path = Path(tmpdir) / "breaker.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "probe default mode"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "python3 -c \"open('/tmp/output.txt')\""}}
                ]},
                {"step_index": 2, "source": "SYSTEM", "type": "GENERIC", "content": "1"},
                {"step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "python3 -c \"open('/tmp/output.txt').read()\""}}
                ]},
                {"step_index": 4, "source": "SYSTEM", "type": "GENERIC", "content": "2"},
            ])
            payload = {
                "conversationId": "test-probe-default-log",
                "transcriptPath": str(t_path),
                "stepIdx": 5,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "python3 -c \"open('/tmp/output.txt').close()\""}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_LOG_PATH": str(log_path), "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")
            self.assertTrue(log_path.exists())
            with open(log_path, "r", encoding="utf-8") as f:
                logs = [json.loads(line) for line in f if line.strip()]
            self.assertEqual(len(logs), 1)
            self.assertEqual(logs[0]["rule"], "PROBE")
            self.assertEqual(logs[0]["mode"], "log")

    def test_true_loop_5_distinct_ranges_allow(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [{"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "distinct ranges"}]
            ranges = [(1, 50), (51, 100), (101, 150), (151, 200)]
            for i, r in enumerate(ranges):
                steps.append({
                    "step_index": i + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "read_file", "args": {"Path": "/tmp/source.py", "StartLine": r[0], "EndLine": r[1]}}]
                })
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-distinct-ranges",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {
                    "name": "read_file",
                    "args": {"Path": "/tmp/source.py", "StartLine": 201, "EndLine": 250}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_TRUELOOP_MODE": "enforce"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_true_loop_overlapping_reads_3rd_denies(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "overlap"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/source.py", "StartLine": 1, "EndLine": 50}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/source.py", "StartLine": 10, "EndLine": 50}}
                ]},
            ])
            payload = {
                "conversationId": "test-overlap-3rd",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {
                    "name": "read_file",
                    "args": {"Path": "/tmp/source.py", "StartLine": 10, "EndLine": 50}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_TRUELOOP_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            data = json.loads(res.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (True-Loop Tripwire)", data.get("reason", ""))
            self.assertIn("3rd overlapping read of /tmp/source.py", data.get("reason", ""))

    def test_true_loop_edit_resets_overlapping_reads(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "overlap reset"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/source.py", "StartLine": 1, "EndLine": 50}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/source.py", "StartLine": 10, "EndLine": 50}}
                ]},
                {"step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "edit_file", "args": {"Path": "/tmp/source.py", "Target": "a", "Replacement": "b"}}
                ]},
            ])
            payload = {
                "conversationId": "test-overlap-edit-reset",
                "transcriptPath": str(t_path),
                "stepIdx": 5,
                "toolCall": {
                    "name": "read_file",
                    "args": {"Path": "/tmp/source.py", "StartLine": 10, "EndLine": 50}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_TRUELOOP_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_true_loop_same_command_3rd_denies(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "run same cmd"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "git status"}}
                ]},
                {"step_index": 2, "source": "SYSTEM", "type": "GENERIC", "content": "clean"},
                {"step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "git   status"}}
                ]},
                {"step_index": 4, "source": "SYSTEM", "type": "GENERIC", "content": "clean"},
            ])
            payload = {
                "conversationId": "test-cmd-loop",
                "transcriptPath": str(t_path),
                "stepIdx": 5,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "git status"}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_TRUELOOP_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            data = json.loads(res.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (True-Loop Tripwire)", data.get("reason", ""))
            self.assertIn("3rd execution of 'git status'", data.get("reason", ""))

    def test_true_loop_same_command_after_edit_allows(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "run cmd edit"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "git status"}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "git status"}}
                ]},
                {"step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "edit_file", "args": {"Path": "/tmp/foo.py", "Target": "x", "Replacement": "y"}}
                ]},
            ])
            payload = {
                "conversationId": "test-cmd-edit-reset",
                "transcriptPath": str(t_path),
                "stepIdx": 5,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "git status"}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_TRUELOOP_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_true_loop_edits_and_manage_task_allow(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [{"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "edits and tasks"}]
            for i in range(10):
                steps.append({
                    "step_index": i + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [
                        {"name": "manage_task", "args": {"Action": "list"}}
                    ]
                })
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-manage-task",
                "transcriptPath": str(t_path),
                "stepIdx": 12,
                "toolCall": {
                    "name": "manage_task",
                    "args": {"Action": "list"}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_TRUELOOP_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_true_loop_different_grep_patterns_allow(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "grep patterns"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "grep -rn 'pattern_one' /repo"}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "grep -rn 'pattern_two' /repo"}}
                ]},
            ])
            payload = {
                "conversationId": "test-grep-diff",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "grep -rn 'pattern_three' /repo"}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_TRUELOOP_MODE": "enforce"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_true_loop_test_runner_excluded(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "run tests"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "pytest tests/"}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "pytest tests/"}}
                ]},
            ])
            payload = {
                "conversationId": "test-runner-excluded",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "pytest tests/"}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_TRUELOOP_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_true_loop_default_log_mode_allows(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            log_path = Path(tmpdir) / "breaker.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "overlap log mode"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/source.py", "StartLine": 1, "EndLine": 50}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/source.py", "StartLine": 10, "EndLine": 50}}
                ]},
            ])
            payload = {
                "conversationId": "test-overlap-log-mode",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {
                    "name": "read_file",
                    "args": {"Path": "/tmp/source.py", "StartLine": 10, "EndLine": 50}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_LOG_PATH": str(log_path), "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")
            self.assertTrue(log_path.exists())
            with open(log_path, "r", encoding="utf-8") as f:
                logs = [json.loads(line) for line in f if line.strip()]
            self.assertEqual(len(logs), 1)
            self.assertEqual(logs[0]["rule"], "TRUELOOP")
            self.assertEqual(logs[0]["mode"], "log")

    def test_checkpoint_unauthorized_call_n_denies_n_plus_1_allows_2n_denies(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [{"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "just some question"}]
            for i in range(4):
                steps.append({
                    "step_index": i + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"echo {i}"}}]
                })
            self.write_jsonl(t_path, steps)
            payload_n = {
                "conversationId": "test-checkpoint-unauth",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo call_5"}}
            }
            env = {"BREAKER_CHECKPOINT_MODE": "enforce", "BREAKER_CHECKPOINT_CALLS": "5", "BREAKER_DUP_MODE": "off"}
            res_n = self.execute_breaker(payload_n, env=env)
            self.assertEqual(res_n.returncode, 0)
            data_n = json.loads(res_n.stdout)
            self.assertEqual(data_n.get("decision"), "deny")
            self.assertIn("Checkpoint: 5 calls since the user's last message", data_n.get("reason", ""))
            self.assertIn("Send found / unknown / options and wait", data_n.get("reason", ""))

            steps.append({
                "step_index": 5, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "tool_calls": [{"name": "run_command", "args": {"CommandLine": "echo call_5"}}]
            })
            self.write_jsonl(t_path, steps)
            payload_n1 = {
                "conversationId": "test-checkpoint-unauth",
                "transcriptPath": str(t_path),
                "stepIdx": 7,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo call_6"}}
            }
            res_n1 = self.execute_breaker(payload_n1, env=env)
            self.assertEqual(res_n1.returncode, 0)
            self.assertEqual(res_n1.stdout, "")

            for i in range(6, 10):
                steps.append({
                    "step_index": i, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"echo call_{i}"}}]
                })
            self.write_jsonl(t_path, steps)
            payload_2n = {
                "conversationId": "test-checkpoint-unauth",
                "transcriptPath": str(t_path),
                "stepIdx": 12,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo call_10"}}
            }
            res_2n = self.execute_breaker(payload_2n, env=env)
            self.assertEqual(res_2n.returncode, 0)
            data_2n = json.loads(res_2n.stdout)
            self.assertEqual(data_2n.get("decision"), "deny")
            self.assertIn("Checkpoint: 10 calls since the user's last message", data_2n.get("reason", ""))

    def test_checkpoint_helper_agents_exempt(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [{"step_index": 0, "source": "SYSTEM", "type": "SYSTEM_MESSAGE", "content": "helper subagent start"}]
            for i in range(4):
                steps.append({
                    "step_index": i + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"echo {i}"}}]
                })
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-checkpoint-helper",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo call_5"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_CHECKPOINT_MODE": "enforce", "BREAKER_CHECKPOINT_CALLS": "5", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_checkpoint_authorized_phrases_never_deny(self):
        phrases = [
            "/goal do this",
            "run in autonomous mode",
            "act as lead orchestrator",
            "continue from pending steps",
            "ingest ~/.gemini/handoffs/dev/active_context.md and go",
            "implement phase 2",
            "keep going until done"
        ]
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            for phrase in phrases:
                steps = [{"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": phrase}]
                for i in range(4):
                    steps.append({
                        "step_index": i + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                        "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"echo {phrase}_{i}"}}]
                    })
                self.write_jsonl(t_path, steps)
                payload = {
                    "conversationId": "test-checkpoint-auth",
                    "transcriptPath": str(t_path),
                    "stepIdx": 6,
                    "toolCall": {"name": "run_command", "args": {"CommandLine": "echo auth_5"}}
                }
                res = self.execute_breaker(payload, env={"BREAKER_CHECKPOINT_MODE": "enforce", "BREAKER_CHECKPOINT_CALLS": "5", "BREAKER_DUP_MODE": "off"})
                self.assertEqual(res.returncode, 0)
                self.assertEqual(res.stdout, "", f"Failed for authorized phrase: {phrase}")

    def test_checkpoint_flag_file_never_denies(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            state_dir = Path(tmpdir) / "state"
            state_dir.mkdir()
            conv_id = "test-flag-conv"
            flag_file = state_dir / f"autonomous-{conv_id}"
            flag_file.write_text("1")

            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [{"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "plain question"}]
            for i in range(4):
                steps.append({
                    "step_index": i + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"echo {i}"}}]
                })
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": conv_id,
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo call_5"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_CHECKPOINT_MODE": "enforce", "BREAKER_CHECKPOINT_CALLS": "5", "BREAKER_STATE_DIR": str(state_dir), "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_checkpoint_helper_reply_after_15_min_allows(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "wait for helper"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [{"name": "run_command", "args": {"CommandLine": "echo 1"}}]},
                {"step_index": 2, "source": "SYSTEM", "type": "SYSTEM_MESSAGE", "content": "helper woke up after 15 minutes"},
            ]
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-checkpoint-wake",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo 2"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_CHECKPOINT_MODE": "enforce", "BREAKER_CHECKPOINT_CALLS": "5"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_checkpoint_new_user_input_resets(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [{"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "turn 1"}]
            for i in range(4):
                steps.append({
                    "step_index": i + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"echo {i}"}}]
                })
            steps.append({"step_index": 5, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "turn 2"})
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-checkpoint-reset",
                "transcriptPath": str(t_path),
                "stepIdx": 7,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo new_call_1"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_CHECKPOINT_MODE": "enforce", "BREAKER_CHECKPOINT_CALLS": "5"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_checkpoint_zero_disables(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [{"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "turn 1"}]
            for i in range(4):
                steps.append({
                    "step_index": i + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"echo {i}"}}]
                })
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-checkpoint-zero",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo call_5"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_CHECKPOINT_MODE": "enforce", "BREAKER_CHECKPOINT_CALLS": "0"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_checkpoint_default_log_mode_allows(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            log_path = Path(tmpdir) / "breaker.jsonl"
            steps = [{"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "turn 1"}]
            for i in range(4):
                steps.append({
                    "step_index": i + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"echo {i}"}}]
                })
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-checkpoint-log-mode",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo call_5"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_LOG_PATH": str(log_path), "BREAKER_CHECKPOINT_CALLS": "5", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")
            self.assertTrue(log_path.exists())
            with open(log_path, "r", encoding="utf-8") as f:
                logs = [json.loads(line) for line in f if line.strip()]
            self.assertEqual(len(logs), 1)
            self.assertEqual(logs[0]["rule"], "CHECKPOINT")
            self.assertEqual(logs[0]["mode"], "log")


    def test_default_env_helper_does_not_write_to_real_metrics_file(self):
        real_log = Path.home() / ".gemini" / "metrics" / "breaker.jsonl"
        size_before = real_log.stat().st_size if real_log.exists() else 0
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "review"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "view_file", "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}}
                ]},
            ])
            payload = {
                "conversationId": "test-no-leak",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {
                    "name": "read_file",
                    "args": {"Path": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}
                }
            }
            res = self.execute_breaker(payload)
            self.assertEqual(res.returncode, 0)
        size_after = real_log.stat().st_size if real_log.exists() else 0
        self.assertEqual(size_before, size_after)

    def test_dup_log_mode_multiple_prior_matches_logs_once(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            log_path = Path(tmpdir) / "breaker.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "review"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "view_file", "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "view_file", "args": {"AbsolutePath": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}}
                ]},
            ])
            payload = {
                "conversationId": "test-dup-multimatch-log",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {
                    "name": "read_file",
                    "args": {"Path": "/tmp/x.rb", "StartLine": 1, "EndLine": 800}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_LOG_PATH": str(log_path), "BREAKER_DUP_MODE": "log", "BREAKER_TRUELOOP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")
            self.assertTrue(log_path.exists())
            with open(log_path, "r", encoding="utf-8") as f:
                logs = [json.loads(line) for line in f if line.strip()]
            self.assertEqual(len(logs), 1)
            self.assertEqual(logs[0]["rule"], "DUP")

    def test_probe_tripwire_edit_tilde_path_resets(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "probe tilde"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "python3 -c \"open('~/.gemini/tmp_probe.txt')\""}}
                ]},
                {"step_index": 2, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "python3 -c \"len(open('~/.gemini/tmp_probe.txt').read())\""}}
                ]},
                {"step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "edit_file", "args": {"Path": os.path.expanduser("~/.gemini/tmp_probe.txt"), "Target": "a", "Replacement": "b"}}
                ]},
            ])
            payload = {
                "conversationId": "test-probe-tilde-reset",
                "transcriptPath": str(t_path),
                "stepIdx": 5,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "python3 -c \"open('~/.gemini/tmp_probe.txt').read()\""}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_PROBE_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_subagent_explore_budget_exempts_python_unittest(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            subagent_steps = [
                {"step_index": 0, "source": "SYSTEM", "type": "SYSTEM_MESSAGE", "content": "subagent start"}
            ]
            for i in range(25):
                subagent_steps.append({
                    "step_index": i + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"python3 -m unittest test_{i}"}}]
                })
            self.write_jsonl(t_path, subagent_steps)
            payload = {
                "conversationId": "subagent-unittest-explore",
                "transcriptPath": str(t_path),
                "stepIdx": 28,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "python3 -m unittest test_final"}
                }
            }
            res = self.execute_breaker(payload)
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_subagent_strikes_triggers_on_python_unittest(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "SYSTEM", "type": "SYSTEM_MESSAGE", "content": "subagent start"}
            ]
            step_idx = 1
            for _ in range(4):
                steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": "python3 -m unittest test_a"}}]
                })
                step_idx += 1
                steps.append({
                    "step_index": step_idx, "source": "SYSTEM", "type": "GENERIC",
                    "content": "The command exited with code 1."
                })
                step_idx += 1
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "subagent-strikes-unittest",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "python3 -m unittest test_a"}
                }
            }
            res = self.execute_breaker(payload)
            self.assertEqual(res.returncode, 0)
            data = json.loads(res.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (4-Strike Test-Fix Escalation)", data.get("reason", ""))

    def test_dup_allows_identical_test_runner(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "run tests"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "python3 -m unittest test_x"}}
                ]},
            ])
            payload = {
                "conversationId": "test-dup-test-runner",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": "python3 -m unittest test_x"}
                }
            }
            res = self.execute_breaker(payload)
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_checkpoint_not_double_counting_with_trailing_generic_step(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "unauth question"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": "echo 1"}},
                    {"name": "run_command", "args": {"CommandLine": "echo 2"}},
                    {"name": "run_command", "args": {"CommandLine": "echo 3"}},
                    {"name": "run_command", "args": {"CommandLine": "echo 4"}},
                ]},
                {"step_index": 2, "source": "SYSTEM", "type": "GENERIC", "content": "command output"}
            ]
            self.write_jsonl(t_path, steps)
            payload_n_minus_1 = {
                "conversationId": "test-checkpoint-trailing",
                "transcriptPath": str(t_path),
                "stepIdx": 5,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo 4"}}
            }
            env = {"BREAKER_CHECKPOINT_MODE": "enforce", "BREAKER_CHECKPOINT_CALLS": "5", "BREAKER_DUP_MODE": "off"}
            res_4 = self.execute_breaker(payload_n_minus_1, env=env)
            self.assertEqual(res_4.returncode, 0)
            self.assertEqual(res_4.stdout, "")

            steps[1]["tool_calls"].append({"name": "run_command", "args": {"CommandLine": "echo 5"}})
            self.write_jsonl(t_path, steps)
            payload_n = {
                "conversationId": "test-checkpoint-trailing",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo 5"}}
            }
            res_5 = self.execute_breaker(payload_n, env=env)
            self.assertEqual(res_5.returncode, 0)
            data_5 = json.loads(res_5.stdout)
            self.assertEqual(data_5.get("decision"), "deny")
            self.assertIn("Checkpoint: 5 calls since the user's last message", data_5.get("reason", ""))

    def test_h1_authorized_main_6_edits_allow_7th_denies_enforce_mode(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "/goal implement feature"},
            ]
            step_idx = 1
            for i in range(6):
                steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "edit_file", "args": {"Path": f"/repo/file_{i}.py", "Target": "a", "Replacement": "b"}}]
                })
                step_idx += 2
            self.write_jsonl(t_path, steps)

            payload_6th = {
                "conversationId": "test-h1-main",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx - 1,
                "toolCall": {"name": "edit_file", "args": {"Path": "/repo/file_5.py", "Target": "a", "Replacement": "b"}}
            }
            res_6th = self.execute_breaker(payload_6th, env={"BREAKER_H1_MODE": "enforce", "BREAKER_MAIN_EDIT_MAX": "6"})
            self.assertEqual(res_6th.returncode, 0)
            self.assertEqual(res_6th.stdout, "")

            payload_7th = {
                "conversationId": "test-h1-main",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {"name": "write_file", "args": {"Path": "/repo/file_6.py", "Content": "code"}}
            }
            res_7th = self.execute_breaker(payload_7th, env={"BREAKER_H1_MODE": "enforce", "BREAKER_MAIN_EDIT_MAX": "6"})
            self.assertEqual(res_7th.returncode, 0)
            data_7th = json.loads(res_7th.stdout)
            self.assertEqual(data_7th.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Main-Session Code-Edit Cap)", data_7th.get("reason", ""))
            self.assertIn("send this to an implementation helper agent via invoke_subagent", data_7th.get("reason", "").lower())
            self.assertIn("6/6", data_7th.get("reason", ""))

    def test_h1_default_log_mode_allows_and_logs(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            log_path = Path(tmpdir) / "breaker.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "/goal implement feature"},
            ]
            step_idx = 1
            for i in range(6):
                steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "write_to_file", "args": {"TargetFile": f"/repo/code_{i}.go", "CodeContent": "package main"}}]
                })
                step_idx += 2
            self.write_jsonl(t_path, steps)

            payload_7th = {
                "conversationId": "test-h1-log",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {"name": "replace_file_content", "args": {"TargetFile": "/repo/code_6.go", "TargetContent": "a", "ReplacementContent": "b"}}
            }
            res_7th = self.execute_breaker(payload_7th, env={"BREAKER_MAIN_EDIT_MAX": "6", "BREAKER_LOG_PATH": str(log_path)})
            self.assertEqual(res_7th.returncode, 0)
            self.assertEqual(res_7th.stdout, "")
            self.assertTrue(log_path.exists())
            with open(log_path, "r", encoding="utf-8") as f:
                logs = [json.loads(line) for line in f if line.strip()]
            h1_logs = [entry for entry in logs if entry.get("rule") == "H1"]
            self.assertEqual(len(h1_logs), 1)
            self.assertEqual(h1_logs[0]["mode"], "log")
            self.assertEqual(h1_logs[0]["conversation_id"], "test-h1-log")

    def test_h1_md_edits_never_count(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "/goal write docs"},
            ]
            step_idx = 1
            for i in range(10):
                steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "write_file", "args": {"Path": f"/repo/doc_{i}.md", "Content": "# doc"}}]
                })
                step_idx += 2
            self.write_jsonl(t_path, steps)

            payload = {
                "conversationId": "test-h1-md",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {"name": "write_file", "args": {"Path": "/repo/another.md", "Content": "# doc"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_H1_MODE": "enforce", "BREAKER_MAIN_EDIT_MAX": "6"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_h1_invoke_subagent_resets_count(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "/goal implement stream"},
            ]
            step_idx = 1
            for i in range(6):
                steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "edit_file", "args": {"Path": f"/repo/f_{i}.rb", "Target": "1", "Replacement": "2"}}]
                })
                step_idx += 2
            steps.append({
                "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "tool_calls": [{"name": "invoke_subagent", "args": {"TypeName": "research", "Role": "tester", "Prompt": "run tests"}}]
            })
            step_idx += 2
            for i in range(5):
                steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "edit_file", "args": {"Path": f"/repo/after_{i}.rb", "Target": "1", "Replacement": "2"}}]
                })
                step_idx += 2
            self.write_jsonl(t_path, steps)

            payload_6th = {
                "conversationId": "test-h1-reset",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {"name": "edit_file", "args": {"Path": "/repo/after_5.rb", "Target": "1", "Replacement": "2"}}
            }
            res_6th = self.execute_breaker(payload_6th, env={"BREAKER_H1_MODE": "enforce", "BREAKER_MAIN_EDIT_MAX": "6"})
            self.assertEqual(res_6th.returncode, 0)
            self.assertEqual(res_6th.stdout, "")

            steps.append({
                "step_index": step_idx + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "tool_calls": [{"name": "edit_file", "args": {"Path": "/repo/after_5.rb", "Target": "1", "Replacement": "2"}}]
            })
            self.write_jsonl(t_path, steps)

            payload_7th = {
                "conversationId": "test-h1-reset",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 3,
                "toolCall": {"name": "edit_file", "args": {"Path": "/repo/after_6.rb", "Target": "1", "Replacement": "2"}}
            }
            res_7th = self.execute_breaker(payload_7th, env={"BREAKER_H1_MODE": "enforce", "BREAKER_MAIN_EDIT_MAX": "6"})
            self.assertEqual(res_7th.returncode, 0)
            data_7th = json.loads(res_7th.stdout)
            self.assertEqual(data_7th.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Main-Session Code-Edit Cap)", data_7th.get("reason", ""))

    def test_h1_unauthorized_turn_no_h1(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "can you fix this error?"},
            ]
            step_idx = 1
            for i in range(10):
                steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "edit_file", "args": {"Path": f"/repo/code_{i}.py", "Target": "x", "Replacement": "y"}}]
                })
                step_idx += 2
            self.write_jsonl(t_path, steps)

            payload = {
                "conversationId": "test-h1-unauth",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {"name": "edit_file", "args": {"Path": "/repo/code_11.py", "Target": "x", "Replacement": "y"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_H1_MODE": "enforce", "BREAKER_MAIN_EDIT_MAX": "6"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_h1_disabled_when_max_zero(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "/goal build"},
            ]
            step_idx = 1
            for i in range(10):
                steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "edit_file", "args": {"Path": f"/repo/code_{i}.py", "Target": "x", "Replacement": "y"}}]
                })
                step_idx += 2
            self.write_jsonl(t_path, steps)

            payload = {
                "conversationId": "test-h1-disabled",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {"name": "edit_file", "args": {"Path": "/repo/code_11.py", "Target": "x", "Replacement": "y"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_H1_MODE": "enforce", "BREAKER_MAIN_EDIT_MAX": "0"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_h2_helper_step_cap(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "SYSTEM", "type": "SYSTEM_MESSAGE", "content": "subagent start"}
            ]
            step_idx = 1
            for i in range(129):
                steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"pytest test_{i}.py"}}]
                })
                step_idx += 2
            self.write_jsonl(t_path, steps)

            payload_129 = {
                "conversationId": "test-h2-helper",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "pytest test_run.py"}}
            }
            res_129 = self.execute_breaker(payload_129, env={"BREAKER_H2_MODE": "enforce", "BREAKER_SUBAGENT_STEP_MAX": "130"})
            self.assertEqual(res_129.returncode, 0)
            self.assertEqual(res_129.stdout, "")

            steps.append({
                "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "tool_calls": [{"name": "run_command", "args": {"CommandLine": "pytest test_129.py"}}]
            })
            step_idx += 2
            self.write_jsonl(t_path, steps)

            payload_130_cmd = {
                "conversationId": "test-h2-helper",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "pytest"}}
            }
            res_130_cmd = self.execute_breaker(payload_130_cmd, env={"BREAKER_H2_MODE": "enforce", "BREAKER_SUBAGENT_STEP_MAX": "130"})
            self.assertEqual(res_130_cmd.returncode, 0)
            data_130 = json.loads(res_130_cmd.stdout)
            self.assertEqual(data_130.get("decision"), "deny")
            self.assertIn("130/130 planner turns used: write your 4-section handoff and return via send_message", data_130.get("reason", ""))

            payload_130_send = {
                "conversationId": "test-h2-helper",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {"name": "send_message", "args": {"Recipient": "parent", "Message": "done"}}
            }
            res_130_send = self.execute_breaker(payload_130_send, env={"BREAKER_H2_MODE": "enforce", "BREAKER_SUBAGENT_STEP_MAX": "130"})
            self.assertEqual(res_130_send.returncode, 0)
            self.assertEqual(res_130_send.stdout, "")

            payload_130_handoff = {
                "conversationId": "test-h2-helper",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {"name": "write_file", "args": {"Path": "/tmp/subagent-handoff.md", "Content": "# handoff"}}
            }
            res_130_handoff = self.execute_breaker(payload_130_handoff, env={"BREAKER_H2_MODE": "enforce", "BREAKER_SUBAGENT_STEP_MAX": "130"})
            self.assertEqual(res_130_handoff.returncode, 0)
            self.assertEqual(res_130_handoff.stdout, "")

            payload_130_brain = {
                "conversationId": "test-h2-helper",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {"name": "write_file", "args": {"Path": "/brain/session/notes.md", "Content": "# notes"}}
            }
            res_130_brain = self.execute_breaker(payload_130_brain, env={"BREAKER_H2_MODE": "enforce", "BREAKER_SUBAGENT_STEP_MAX": "130"})
            self.assertEqual(res_130_brain.returncode, 0)
            self.assertEqual(res_130_brain.stdout, "")

    def test_h2_parallel_reads_count_as_one_step(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "SYSTEM", "type": "SYSTEM_MESSAGE", "content": "subagent start"},
                {
                    "step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "read_file", "args": {"Path": f"/tmp/f_{i}.txt"}} for i in range(8)]
                }
            ]
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-h2-parallel",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "ls"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_H2_MODE": "enforce", "BREAKER_SUBAGENT_STEP_MAX": "2"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_h2_default_log_mode_allows_and_logs(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            log_path = Path(tmpdir) / "breaker.jsonl"
            steps = [
                {"step_index": 0, "source": "SYSTEM", "type": "SYSTEM_MESSAGE", "content": "subagent start"}
            ]
            for i in range(130):
                steps.append({
                    "step_index": i + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"pytest test_{i}.py"}}]
                })
            self.write_jsonl(t_path, steps)

            payload = {
                "conversationId": "test-h2-log",
                "transcriptPath": str(t_path),
                "stepIdx": 132,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "pytest test_final.py"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_SUBAGENT_STEP_MAX": "130", "BREAKER_LOG_PATH": str(log_path)})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")
            self.assertTrue(log_path.exists())
            with open(log_path, "r", encoding="utf-8") as f:
                logs = [json.loads(line) for line in f if line.strip()]
            h2_logs = [entry for entry in logs if entry.get("rule") == "H2"]
            self.assertEqual(len(h2_logs), 1)
            self.assertEqual(h2_logs[0]["mode"], "log")

    def test_h2_disabled_when_max_zero(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "SYSTEM", "type": "SYSTEM_MESSAGE", "content": "subagent start"}
            ]
            for i in range(130):
                steps.append({
                    "step_index": i + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"pytest test_{i}.py"}}]
                })
            self.write_jsonl(t_path, steps)

            payload = {
                "conversationId": "test-h2-disabled",
                "transcriptPath": str(t_path),
                "stepIdx": 132,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "pytest test_final.py"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_H2_MODE": "enforce", "BREAKER_SUBAGENT_STEP_MAX": "0"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_baton_104k_allows(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            padding = "a" * 415000
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "do task"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": padding, "tool_calls": [{"name": "run_command", "args": {"CommandLine": "ls"}}]},
            ]
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload = {
                "conversationId": "test-baton-104k",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "pwd"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_baton_105k_warn_unauthorized_then_retry_at_110k_allows(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            padding = "a" * 420000
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "unauth question"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": padding, "tool_calls": [{"name": "run_command", "args": {"CommandLine": "echo 1"}}]},
            ]
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload_warn = {
                "conversationId": "test-baton-warn",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo 2"}}
            }
            res_warn = self.execute_breaker(payload_warn, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_warn.returncode, 0)
            data_warn = json.loads(res_warn.stdout)
            self.assertEqual(data_warn.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Baton Warn)", data_warn.get("reason", ""))

            state_file = Path(tmpdir) / "baton-test-baton-warn.json"
            self.assertTrue(state_file.exists())
            with open(state_file, "r") as f:
                state_data = json.load(f)
            self.assertIsNotNone(state_data.get("warned_at_estimate"))

            steps[1]["content"] = "a" * 440000
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload_retry = {
                "conversationId": "test-baton-warn",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo 3"}}
            }
            res_retry = self.execute_breaker(payload_retry, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_retry.returncode, 0)
            self.assertEqual(res_retry.stdout, "")

    def test_baton_130k_blocks_run_command_and_invoke_subagent(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            padding = "a" * 525000
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "do work"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": padding, "tool_calls": [{"name": "run_command", "args": {"CommandLine": "ls"}}]},
            ]
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload_cmd = {
                "conversationId": "test-baton-block",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "pwd"}}
            }
            res_cmd = self.execute_breaker(payload_cmd, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_cmd.returncode, 0)
            data_cmd = json.loads(res_cmd.stdout)
            self.assertEqual(data_cmd.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Baton Block)", data_cmd.get("reason", ""))
            self.assertIn("Only ACTIVE_CONTEXT.md handoff and Prism saves are allowed", data_cmd.get("reason", ""))

            payload_sub = {
                "conversationId": "test-baton-block",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {"name": "invoke_subagent", "args": {"TypeName": "research", "Role": "researcher", "Prompt": "help"}}
            }
            res_sub = self.execute_breaker(payload_sub, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_sub.returncode, 0)
            data_sub = json.loads(res_sub.stdout)
            self.assertEqual(data_sub.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Baton Block)", data_sub.get("reason", ""))

    def test_baton_130k_allows_active_context_write_and_prism_call_and_active_context_read(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            padding = "a" * 525000
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "do work"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": padding, "tool_calls": [{"name": "run_command", "args": {"CommandLine": "ls"}}]},
            ]
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload_write = {
                "conversationId": "test-baton-exempt",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {"name": "write_file", "args": {"Path": "/handoffs/2026-10-21-ACTIVE_CONTEXT.md", "Content": "# Context"}}
            }
            res_write = self.execute_breaker(payload_write, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_write.returncode, 0)
            self.assertEqual(res_write.stdout, "")

            payload_prism = {
                "conversationId": "test-baton-exempt",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {"name": "call_mcp_tool", "args": {"ServerName": "prism", "ToolName": "session_save_handoff", "Arguments": {}}}
            }
            res_prism = self.execute_breaker(payload_prism, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_prism.returncode, 0)
            self.assertEqual(res_prism.stdout, "")

            payload_read = {
                "conversationId": "test-baton-exempt",
                "transcriptPath": str(t_path),
                "stepIdx": 5,
                "toolCall": {"name": "read_file", "args": {"Path": "/handoffs/2026-10-21-ACTIVE_CONTEXT.md"}}
            }
            res_read = self.execute_breaker(payload_read, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_read.returncode, 0)
            self.assertEqual(res_read.stdout, "")

            payload_view = {
                "conversationId": "test-baton-exempt",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {"name": "view_file", "args": {"AbsolutePath": "/handoffs/2026-10-21-ACTIVE_CONTEXT.md"}}
            }
            res_view = self.execute_breaker(payload_view, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_view.returncode, 0)
            self.assertEqual(res_view.stdout, "")

    def test_baton_active_context_write_records_handoff_and_unlocks_plus_50k(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            padding = "a" * 525000
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "do work"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": padding, "tool_calls": [{"name": "run_command", "args": {"CommandLine": "ls"}}]},
            ]
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload_write = {
                "conversationId": "test-baton-unlock",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {"name": "write_to_file", "args": {"TargetFile": "~/.gemini/handoffs/dev/2026-10-21-ACTIVE_CONTEXT.md", "CodeContent": "# Context"}}
            }
            res_write = self.execute_breaker(payload_write, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_write.returncode, 0)
            self.assertEqual(res_write.stdout, "")

            state_file = Path(tmpdir) / "baton-test-baton-unlock.json"
            self.assertTrue(state_file.exists())
            with open(state_file, "r") as f:
                state_data = json.load(f)
            handoff_est = state_data.get("handoff_at_estimate")
            self.assertIsNotNone(handoff_est)

            steps[1]["content"] = "a" * int((handoff_est + 40000) * 4)
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload_unlocked = {
                "conversationId": "test-baton-unlock",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "git status"}}
            }
            res_unlocked = self.execute_breaker(payload_unlocked, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_unlocked.returncode, 0)
            self.assertEqual(res_unlocked.stdout, "")

            steps[1]["content"] = "a" * int((handoff_est + 51000) * 4)
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload_reblock = {
                "conversationId": "test-baton-unlock",
                "transcriptPath": str(t_path),
                "stepIdx": 5,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "git status"}}
            }
            res_reblock = self.execute_breaker(payload_reblock, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_reblock.returncode, 0)
            data_reblock = json.loads(res_reblock.stdout)
            self.assertEqual(data_reblock.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Baton Block)", data_reblock.get("reason", ""))

    def test_baton_compaction_checkpoint_resets_estimate(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "long task"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": "a" * 1400000, "tool_calls": []},
                {"step_index": 2, "source": "SYSTEM", "type": "CHECKPOINT", "content": "# Resuming from a compaction\n\nState summary..."},
                {"step_index": 3, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "continue"},
                {"step_index": 4, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": "small content", "tool_calls": []},
            ]
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload = {
                "conversationId": "test-baton-compaction",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "ls"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_baton_non_compaction_checkpoint_does_not_reset(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "long task"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": "a" * 525000, "tool_calls": []},
                {"step_index": 2, "source": "SYSTEM", "type": "CHECKPOINT", "content": "{{ CHECKPOINT 0 }}"},
                {"step_index": 3, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "continue"},
                {"step_index": 4, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": "small content", "tool_calls": []},
            ]
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload = {
                "conversationId": "test-baton-no-reset",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "ls"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res.returncode, 0)
            data = json.loads(res.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Baton Block)", data.get("reason", ""))

    def test_baton_literal_resuming_from_compaction_in_generic_or_thought_does_not_reset(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "long task"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": "a" * 525000, "tool_calls": []},
                {"step_index": 2, "source": "MODEL", "type": "GENERIC", "content": "# Resuming from a compaction in output"},
                {"step_index": 3, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "continue"},
                {"step_index": 4, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": "small content", "tool_calls": []},
            ]
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload = {
                "conversationId": "test-baton-literal-no-reset",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "ls"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res.returncode, 0)
            data = json.loads(res.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Baton Block)", data.get("reason", ""))

    def test_baton_another_conversation_state_file_does_not_unlock(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "long task"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": "a" * 525000, "tool_calls": []},
            ]
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            other_state = Path(tmpdir) / "baton-other-conv.json"
            with open(other_state, "w") as f:
                json.dump({"warned_at_estimate": 105000, "handoff_at_estimate": 131000, "segment": 0}, f)

            payload = {
                "conversationId": "current-conv",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "ls"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res.returncode, 0)
            data = json.loads(res.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Baton Block)", data.get("reason", ""))

    def test_baton_authorized_run_warn_timing(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            padding = "a" * 420000
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "/goal do big task"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": padding, "tool_calls": [{"name": "run_command", "args": {"CommandLine": "echo 1"}}]},
            ]
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload_cmd = {
                "conversationId": "test-baton-auth-warn",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo 2"}}
            }
            res_cmd = self.execute_breaker(payload_cmd, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_cmd.returncode, 0)
            self.assertEqual(res_cmd.stdout, "")

            payload_sub = {
                "conversationId": "test-baton-auth-warn",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {"name": "invoke_subagent", "args": {"TypeName": "research", "Role": "worker", "Prompt": "go"}}
            }
            res_sub = self.execute_breaker(payload_sub, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_sub.returncode, 0)
            data_sub = json.loads(res_sub.stdout)
            self.assertEqual(data_sub.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Baton Warn)", data_sub.get("reason", ""))

            res_retry = self.execute_breaker(payload_sub, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_retry.returncode, 0)
            self.assertEqual(res_retry.stdout, "")

            steps.append({
                "step_index": 2, "source": "SYSTEM", "type": "SYSTEM_MESSAGE", "content": "subagent finished"
            })
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            state_file = Path(tmpdir) / "baton-test-baton-sys-warn.json"
            payload_sys_prev = {
                "conversationId": "test-baton-sys-warn",
                "transcriptPath": str(t_path),
                "stepIdx": 5,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "pwd"}}
            }
            res_sys_prev = self.execute_breaker(payload_sys_prev, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_sys_prev.returncode, 0)
            data_sys = json.loads(res_sys_prev.stdout)
            self.assertEqual(data_sys.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Baton Warn)", data_sys.get("reason", ""))

    def test_baton_helper_agent_exempt(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            padding = "a" * 800000
            steps = [
                {"step_index": 0, "source": "SYSTEM", "type": "SYSTEM_MESSAGE", "content": "helper start"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": padding, "tool_calls": [{"name": "run_command", "args": {"CommandLine": "pytest test_1.py"}}]},
            ]
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload = {
                "conversationId": "test-baton-helper",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "pytest test_2.py"}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_BATON_MODE": "enforce", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_performance_under_300ms_on_large_transcript(self):
        large_transcript = os.path.expanduser("~/.gemini/antigravity/brain/a104d6ed-7b67-4464-a0a9-a5ec607b5e51/.system_generated/logs/transcript.jsonl")
        if not os.path.exists(large_transcript):
            return
        import time
        payload = {
            "conversationId": "a104d6ed-7b67-4464-a0a9-a5ec607b5e51",
            "transcriptPath": large_transcript,
            "stepIdx": 1000,
            "toolCall": {"name": "run_command", "args": {"CommandLine": "git status"}}
        }
        t0 = time.perf_counter()
        res = self.execute_breaker(payload)
        t1 = time.perf_counter()
        elapsed = t1 - t0
        self.assertEqual(res.returncode, 0)
        print(f"\n[PERF] 5MB transcript execution time: {elapsed:.4f}s")
        self.assertLess(elapsed, 0.30)

    def test_tool_args_as_string_allows(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            self.write_jsonl(t_path, [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "hi"}
            ])
            payload = {
                "conversationId": "test-string-args",
                "transcriptPath": str(t_path),
                "stepIdx": 1,
                "toolCall": {
                    "name": "run_command",
                    "args": "not a dict"
                }
            }
            res = self.execute_breaker(payload)
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_baton_default_enforce_mode_warns_once_then_allows(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            padding = "a" * 420000
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "unauth question"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": padding, "tool_calls": [{"name": "run_command", "args": {"CommandLine": "echo 1"}}]},
            ]
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload_1 = {
                "conversationId": "test-baton-enforce-mode",
                "transcriptPath": str(t_path),
                "stepIdx": 3,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo 2"}}
            }
            res_1 = self.execute_breaker(payload_1)
            self.assertEqual(res_1.returncode, 0)
            self.assertIn('"decision": "deny"', res_1.stdout)
            self.assertIn("Baton Warn", res_1.stdout)

            log_file = os.path.join(self.test_tmpdir, "breaker.jsonl")
            self.assertTrue(os.path.exists(log_file))
            with open(log_file, "r", encoding="utf-8") as f:
                logs_1 = [json.loads(line) for line in f if line.strip()]
            baton_logs_1 = [entry for entry in logs_1 if entry.get("rule") == "BATON"]
            self.assertEqual(len(baton_logs_1), 1)
            self.assertEqual(baton_logs_1[0]["mode"], "enforce")

            payload_2 = {
                "conversationId": "test-baton-enforce-mode",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo 3"}}
            }
            res_2 = self.execute_breaker(payload_2)
            self.assertEqual(res_2.returncode, 0)
            self.assertEqual(res_2.stdout, "")

            with open(log_file, "r", encoding="utf-8") as f:
                logs_2 = [json.loads(line) for line in f if line.strip()]
            baton_logs_2 = [entry for entry in logs_2 if entry.get("rule") == "BATON"]
            self.assertEqual(len(baton_logs_2), 1)

    def test_checkpoint_default_is_45(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [{"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "unauth question"}]
            curr = 1
            for i in range(43):
                steps.append({
                    "step_index": curr, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"echo {i}"}}]
                })
                curr += 1
                steps.append({"step_index": curr, "source": "TOOL", "type": "GENERIC", "content": "ok"})
                curr += 1

            self.write_jsonl(t_path, steps)
            payload_44 = {
                "conversationId": "test-cp-default-45",
                "transcriptPath": str(t_path),
                "stepIdx": curr,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo 43_again"}}
            }
            res_44 = self.execute_breaker(payload_44, env={"BREAKER_CHECKPOINT_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res_44.returncode, 0)
            self.assertEqual(res_44.stdout, "")

            steps.append({
                "step_index": curr, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "tool_calls": [{"name": "run_command", "args": {"CommandLine": "echo 43_again"}}]
            })
            curr += 1
            steps.append({"step_index": curr, "source": "TOOL", "type": "GENERIC", "content": "ok"})
            curr += 1
            self.write_jsonl(t_path, steps)

            payload_45 = {
                "conversationId": "test-cp-default-45",
                "transcriptPath": str(t_path),
                "stepIdx": curr,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo 44_final"}}
            }
            res_45 = self.execute_breaker(payload_45, env={"BREAKER_CHECKPOINT_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res_45.returncode, 0)
            data_45 = json.loads(res_45.stdout)
            self.assertEqual(data_45.get("decision"), "deny")
            self.assertIn("Checkpoint: 45 calls since the user's last message", data_45.get("reason", ""))

    def test_explore_default_is_100(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            subagent_steps = [
                {"step_index": 0, "source": "SYSTEM", "type": "SYSTEM_MESSAGE", "content": "subagent start"}
            ]
            step_idx = 1
            for i in range(99):
                subagent_steps.append({
                    "step_index": step_idx, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "read_file", "args": {"Path": f"/tmp/source_{i}.rb"}}]
                })
                step_idx += 2
            self.write_jsonl(t_path, subagent_steps)

            payload_99 = {
                "conversationId": "subagent-default-100",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "ls /tmp"}}
            }
            res_99 = self.execute_breaker(payload_99, env={"BREAKER_EXPLORE_MODE": "enforce"})
            self.assertEqual(res_99.returncode, 0)
            self.assertEqual(res_99.stdout, "")

            subagent_steps.append({
                "step_index": step_idx + 1, "source": "MODEL", "type": "PLANNER_RESPONSE",
                "tool_calls": [{"name": "run_command", "args": {"CommandLine": "ls /tmp"}}]
            })
            step_idx += 2
            self.write_jsonl(t_path, subagent_steps)

            payload_100 = {
                "conversationId": "subagent-default-100",
                "transcriptPath": str(t_path),
                "stepIdx": step_idx + 1,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "ls /tmp"}}
            }
            res_100 = self.execute_breaker(payload_100, env={"BREAKER_EXPLORE_MODE": "enforce"})
            self.assertEqual(res_100.returncode, 0)
            data_100 = json.loads(res_100.stdout)
            self.assertEqual(data_100.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Subagent Exploration Budget Reached)", data_100.get("reason", ""))
            self.assertIn("100", data_100.get("reason", ""))

    def test_r2_trueloop_chunked_walk_allows(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "walk"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/chunk.py", "StartLine": 400, "EndLine": 500}}
                ]},
                {"step_index": 2, "source": "TOOL", "type": "GENERIC", "content": "ok"},
                {"step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/chunk.py", "StartLine": 300, "EndLine": 400}}
                ]},
                {"step_index": 4, "source": "TOOL", "type": "GENERIC", "content": "ok"},
                {"step_index": 5, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/chunk.py", "StartLine": 200, "EndLine": 300}}
                ]},
                {"step_index": 6, "source": "TOOL", "type": "GENERIC", "content": "ok"},
            ]
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-chunked-walk",
                "transcriptPath": str(t_path),
                "stepIdx": 8,
                "toolCall": {"name": "read_file", "args": {"Path": "/tmp/chunk.py", "StartLine": 110, "EndLine": 199}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_TRUELOOP_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_r2_trueloop_small_overlap_allows(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "small overlap"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/source.py", "StartLine": 1, "EndLine": 100}}
                ]},
                {"step_index": 2, "source": "TOOL", "type": "GENERIC", "content": "ok"},
            ]
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-small-overlap",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {"name": "read_file", "args": {"Path": "/tmp/source.py", "StartLine": 98, "EndLine": 153}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_TRUELOOP_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_r2_trueloop_stale_read_outside_window_allows(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "window test"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/source.py", "StartLine": 1, "EndLine": 80}}
                ]},
            ]
            curr = 2
            for i in range(25):
                steps.append({"step_index": curr, "source": "TOOL", "type": "GENERIC", "content": "ok"})
                curr += 1
                steps.append({
                    "step_index": curr, "source": "MODEL", "type": "PLANNER_RESPONSE",
                    "tool_calls": [{"name": "run_command", "args": {"CommandLine": f"echo {i}"}}]
                })
                curr += 1

            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-window-stale",
                "transcriptPath": str(t_path),
                "stepIdx": curr + 1,
                "toolCall": {"name": "read_file", "args": {"Path": "/tmp/source.py", "StartLine": 45, "EndLine": 77}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_TRUELOOP_MODE": "enforce", "BREAKER_DUP_MODE": "off", "BREAKER_TRUELOOP_WINDOW": "20"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_r2_trueloop_three_reads_trigger_on_third(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "loop 10-60"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/source.py", "StartLine": 10, "EndLine": 60}}
                ]},
                {"step_index": 2, "source": "TOOL", "type": "GENERIC", "content": "ok"},
                {"step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/source.py", "StartLine": 10, "EndLine": 60}}
                ]},
                {"step_index": 4, "source": "TOOL", "type": "GENERIC", "content": "ok"},
            ]
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-loop-10-60",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {"name": "read_file", "args": {"Path": "/tmp/source.py", "StartLine": 10, "EndLine": 60}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_TRUELOOP_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            data = json.loads(res.stdout)
            self.assertEqual(data.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (True-Loop Tripwire)", data.get("reason", ""))

    def test_r2_trueloop_whole_file_then_bounded_reads(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "whole file loop"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/whole.py"}}
                ]},
                {"step_index": 2, "source": "TOOL", "type": "GENERIC", "content": "ok"},
                {"step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/whole.py", "StartLine": 10, "EndLine": 60}}
                ]},
                {"step_index": 4, "source": "TOOL", "type": "GENERIC", "content": "ok"},
            ]
            self.write_jsonl(t_path, steps)
            payload_2nd = {
                "conversationId": "test-whole-file-bounded",
                "transcriptPath": str(t_path),
                "stepIdx": 6,
                "toolCall": {"name": "read_file", "args": {"Path": "/tmp/whole.py", "StartLine": 20, "EndLine": 50}}
            }
            res_2nd = self.execute_breaker(payload_2nd, env={"BREAKER_TRUELOOP_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res_2nd.returncode, 0)
            data_2nd = json.loads(res_2nd.stdout)
            self.assertEqual(data_2nd.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (True-Loop Tripwire)", data_2nd.get("reason", ""))

    def test_r3_baton_authorized_warn_at_plus_10k(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            t_full_path = Path(tmpdir) / "transcript_full.jsonl"
            padding_112k = "x" * 448000
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "lead orchestrator run"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "content": padding_112k, "tool_calls": [{"name": "run_command", "args": {"CommandLine": "ls"}}]},
                {"step_index": 2, "source": "TOOL", "type": "GENERIC", "content": "ok"},
            ]
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload_112k = {
                "conversationId": "test-baton-r3",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo 1"}}
            }
            res_112k = self.execute_breaker(payload_112k, env={"BREAKER_BATON_MODE": "enforce", "BATON_WARN_TOKENS": "105000", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_112k.returncode, 0)
            self.assertEqual(res_112k.stdout, "")

            padding_116k = "x" * 464000
            steps[1]["content"] = padding_116k
            self.write_jsonl(t_path, steps)
            self.write_jsonl(t_full_path, steps)

            payload_116k = {
                "conversationId": "test-baton-r3",
                "transcriptPath": str(t_path),
                "stepIdx": 4,
                "toolCall": {"name": "run_command", "args": {"CommandLine": "echo 2"}}
            }
            res_116k = self.execute_breaker(payload_116k, env={"BREAKER_BATON_MODE": "enforce", "BATON_WARN_TOKENS": "105000", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_116k.returncode, 0)
            data_116k = json.loads(res_116k.stdout)
            self.assertEqual(data_116k.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (Baton Warn)", data_116k.get("reason", ""))

            res_retry = self.execute_breaker(payload_116k, env={"BREAKER_BATON_MODE": "enforce", "BATON_WARN_TOKENS": "105000", "BREAKER_STATE_DIR": tmpdir})
            self.assertEqual(res_retry.returncode, 0)
            self.assertEqual(res_retry.stdout, "")

    def test_r2_trueloop_range_re_read_count_per_range(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "analyze code"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/code.py", "StartLine": 200, "EndLine": 300}}
                ]},
                {"step_index": 2, "source": "TOOL", "type": "GENERIC", "content": "ok"},
                {"step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/code.py", "StartLine": 200, "EndLine": 300}}
                ]},
                {"step_index": 4, "source": "TOOL", "type": "GENERIC", "content": "ok"},
                {"step_index": 5, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/code.py", "StartLine": 1, "EndLine": 100}}
                ]},
                {"step_index": 6, "source": "TOOL", "type": "GENERIC", "content": "ok"},
            ]
            self.write_jsonl(t_path, steps)
            payload_a_2nd = {
                "conversationId": "test-range-loop",
                "transcriptPath": str(t_path),
                "stepIdx": 8,
                "toolCall": {"name": "read_file", "args": {"Path": "/tmp/code.py", "StartLine": 1, "EndLine": 100}}
            }
            res_2nd = self.execute_breaker(payload_a_2nd, env={"BREAKER_TRUELOOP_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res_2nd.returncode, 0)
            self.assertEqual(res_2nd.stdout, "")

            steps.append({
                "step_index": 7, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": "/tmp/code.py", "StartLine": 1, "EndLine": 100}}
                ]
            })
            steps.append({"step_index": 8, "source": "TOOL", "type": "GENERIC", "content": "ok"})
            self.write_jsonl(t_path, steps)

            payload_a_3rd = {
                "conversationId": "test-range-loop",
                "transcriptPath": str(t_path),
                "stepIdx": 10,
                "toolCall": {"name": "read_file", "args": {"Path": "/tmp/code.py", "StartLine": 1, "EndLine": 100}}
            }
            res_3rd = self.execute_breaker(payload_a_3rd, env={"BREAKER_TRUELOOP_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res_3rd.returncode, 0)
            data_3rd = json.loads(res_3rd.stdout)
            self.assertEqual(data_3rd.get("decision"), "deny")
            self.assertIn("CIRCUIT BREAKER (True-Loop Tripwire)", data_3rd.get("reason", ""))

            log_path = Path(tmpdir) / "breaker.jsonl"
            res_log = self.execute_breaker(payload_a_3rd, env={"BREAKER_TRUELOOP_MODE": "log", "BREAKER_DUP_MODE": "off", "BREAKER_LOG_PATH": str(log_path)})
            self.assertEqual(res_log.returncode, 0)
            self.assertEqual(res_log.stdout, "")
            self.assertTrue(log_path.exists())
            with open(log_path, "r", encoding="utf-8") as f:
                logs = [json.loads(line) for line in f if line.strip()]
            self.assertEqual(len(logs), 1)
            self.assertEqual(logs[0]["rule"], "TRUELOOP")

    def test_probe_url_five_times_no_probe(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            cmd_url = 'python3 -c "import urllib.request; urllib.request.urlopen(\'https://api.github.com/repos/foo/bar\')"'
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "url calls"},
            ]
            curr = 1
            for i in range(4):
                steps.append({"step_index": curr, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "run_command", "args": {"CommandLine": cmd_url}}
                ]})
                curr += 1
                steps.append({"step_index": curr, "source": "TOOL", "type": "GENERIC", "content": "ok"})
                curr += 1
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-url-no-probe",
                "transcriptPath": str(t_path),
                "stepIdx": curr + 1,
                "toolCall": {
                    "name": "run_command",
                    "args": {"CommandLine": cmd_url}
                }
            }
            res = self.execute_breaker(payload, env={"BREAKER_PROBE_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

    def test_evaluate_defaults_and_env_isolation(self):
        import autonomous_breaker
        with tempfile.TemporaryDirectory() as fake_home:
            orig_environ = os.environ.copy()
            os.environ["HOME"] = fake_home
            t_path = Path(fake_home) / "transcript.jsonl"
            t_path.write_text(json.dumps({"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "test"}) + "\n")
            payload = {
                "conversationId": "test-eval-isolation",
                "transcriptPath": str(t_path),
                "stepIdx": 1,
                "toolCall": {"name": "read_file", "args": {"Path": "/tmp/foo.py"}}
            }
            try:
                res = autonomous_breaker.evaluate(payload)
                self.assertIsInstance(res, list)
                fake_home_gemini = Path(fake_home) / ".gemini"
                if fake_home_gemini.exists():
                    self.assertFalse((fake_home_gemini / "state").exists())
                    self.assertFalse((fake_home_gemini / "metrics").exists())
                self.assertEqual(dict(os.environ), orig_environ | {"HOME": fake_home})
            finally:
                os.environ.clear()
                os.environ.update(orig_environ)

    def test_r2_trueloop_edit_tilde_path_resets_absolute_reads(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            t_path = Path(tmpdir) / "transcript.jsonl"
            abs_path = os.path.expanduser("~/x.py")
            steps = [
                {"step_index": 0, "source": "USER_EXPLICIT", "type": "USER_INPUT", "content": "tilde reset test"},
                {"step_index": 1, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": abs_path, "StartLine": 1, "EndLine": 100}}
                ]},
                {"step_index": 2, "source": "TOOL", "type": "GENERIC", "content": "ok"},
                {"step_index": 3, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "read_file", "args": {"Path": abs_path, "StartLine": 1, "EndLine": 100}}
                ]},
                {"step_index": 4, "source": "TOOL", "type": "GENERIC", "content": "ok"},
                {"step_index": 5, "source": "MODEL", "type": "PLANNER_RESPONSE", "tool_calls": [
                    {"name": "edit_file", "args": {"Path": "~/x.py", "Target": "old", "Replacement": "new"}}
                ]},
                {"step_index": 6, "source": "TOOL", "type": "GENERIC", "content": "ok"},
            ]
            self.write_jsonl(t_path, steps)
            payload = {
                "conversationId": "test-tilde-reset",
                "transcriptPath": str(t_path),
                "stepIdx": 8,
                "toolCall": {"name": "read_file", "args": {"Path": abs_path, "StartLine": 1, "EndLine": 100}}
            }
            res = self.execute_breaker(payload, env={"BREAKER_TRUELOOP_MODE": "enforce", "BREAKER_DUP_MODE": "off"})
            self.assertEqual(res.returncode, 0)
            self.assertEqual(res.stdout, "")

if __name__ == "__main__":
    unittest.main()
