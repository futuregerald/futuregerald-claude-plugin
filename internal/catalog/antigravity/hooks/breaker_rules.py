from __future__ import annotations

import json
import os
import re
import tempfile

AUTHORIZED_PATTERNS = [
    re.compile(r"/goal", re.IGNORECASE),
    re.compile(r"autonomous", re.IGNORECASE),
    re.compile(r"lead orchestrator", re.IGNORECASE),
    re.compile(r"continue\b.*pending steps", re.IGNORECASE),
    re.compile(r"(ingest|continue from)\b.*active_context", re.IGNORECASE),
    re.compile(r"implement .*(phase|stream)", re.IGNORECASE),
    re.compile(r"keep going", re.IGNORECASE),
    re.compile(r"\b(start|begin|execute|run|continue)\s+phase\s+\d+", re.IGNORECASE),
    re.compile(r"brief-phase", re.IGNORECASE),
    re.compile(r"handoff-phase", re.IGNORECASE),
    re.compile(r"complete instructions for phase\s+\w+", re.IGNORECASE),
    re.compile(r"subagent[- ]driven development", re.IGNORECASE),
    re.compile(r"/start\b", re.IGNORECASE),
    re.compile(r"/begin\b", re.IGNORECASE),
    re.compile(r"/lifecycle\b", re.IGNORECASE),
    re.compile(r"/peer-pipeline-swarm\b", re.IGNORECASE),
]

COMPLAINT_PATTERN = re.compile(r"why .*(long|slow)|taking so long|\bstop\b", re.IGNORECASE)

TEST_RUNNER_PATTERN = re.compile(
    r"(^|[\s;&|])(pytest|python3?\s+-m\s+unittest|vitest|jest|go\s+test|rspec|npm\s+test|npx\s+vitest)(\s|$|[;&|])"
)

def is_authorized(user_text: str, conversation_id: str | None = None, state_dir: str = "~/.gemini/state") -> bool:
    if user_text:
        for pat in AUTHORIZED_PATTERNS:
            if pat.search(user_text):
                return True
    if conversation_id:
        exp_dir = os.path.expanduser(state_dir)
        flag_path = os.path.join(exp_dir, f"autonomous-{conversation_id}")
        if os.path.exists(flag_path):
            return True
    return False

def read_ranges_overlap(a: tuple[int | None, int | None], b: tuple[int | None, int | None]) -> bool:
    start_a = 1 if a[0] is None else a[0]
    end_a = float("inf") if a[1] is None else a[1]
    start_b = 1 if b[0] is None else b[0]
    end_b = float("inf") if b[1] is None else b[1]
    return max(start_a, start_b) <= min(end_a, end_b)

def read_coverage(incoming_rng: tuple[int | None, int | None], prior_ranges: list[tuple[int | None, int | None]]) -> float:
    s = 1 if incoming_rng[0] is None else incoming_rng[0]
    e = float("inf") if incoming_rng[1] is None else incoming_rng[1]
    if e == float("inf"):
        for ps, pe in prior_ranges:
            p_start = 1 if ps is None else ps
            p_end = float("inf") if pe is None else pe
            if p_start <= 1 and p_end == float("inf"):
                return 1.0
        return 0.0
    total_lines = e - s + 1
    if total_lines <= 0:
        return 0.0
    intervals = []
    for ps, pe in prior_ranges:
        p_start = 1 if ps is None else ps
        p_end = float("inf") if pe is None else pe
        ov_s = max(s, p_start)
        ov_e = min(e, p_end)
        if ov_s <= ov_e:
            intervals.append((ov_s, ov_e))
    if not intervals:
        return 0.0
    intervals.sort(key=lambda x: x[0])
    merged = []
    for start, end in intervals:
        if not merged or merged[-1][1] < start - 1:
            merged.append([start, end])
        else:
            merged[-1][1] = max(merged[-1][1], end)
    covered_lines = sum(min(end, e) - max(start, s) + 1 for start, end in merged)
    return covered_lines / total_lines

def is_re_read(incoming_rng: tuple[int | None, int | None], prior_ranges: list[tuple[int | None, int | None]], threshold: float = 0.5) -> bool:
    return read_coverage(incoming_rng, prior_ranges) >= threshold

def normalize_command(cmd: str) -> str:
    if not cmd:
        return ""
    return " ".join(cmd.strip().split())

def is_test_runner(cmd: str) -> bool:
    if not cmd:
        return False
    norm = normalize_command(cmd)
    return bool(TEST_RUNNER_PATTERN.search(norm))

def is_code_file(path: str) -> bool:
    if not path or not isinstance(path, str):
        return False
    norm = path.replace("\\", "/")
    low = norm.lower()
    for ext in (".md", ".txt", ".json", ".yaml", ".yml"):
        if low.endswith(ext):
            return False
    for sub in ("/brain/", "/tmp/", "/temp/", "docs/plans/", "/docs/", "/appdata/local/temp/"):
        if sub in low:
            return False
    sys_tmp = tempfile.gettempdir().replace("\\", "/").lower().rstrip("/")
    if sys_tmp and (low == sys_tmp or low.startswith(sys_tmp + "/")):
        return False
    return True

def is_complaint(user_text: str) -> bool:
    if not user_text or not isinstance(user_text, str):
        return False
    return bool(COMPLAINT_PATTERN.search(user_text))

def classify_swarm_steps(
    steps: list[dict],
    inflight_name: str = "",
    inflight_args: dict | None = None,
    tool_calls_extractor=None,
) -> tuple[bool, bool]:
    last_user_content = ""
    for s in steps:
        if isinstance(s, dict) and s.get("type") == "USER_INPUT":
            last_user_content = s.get("content") or ""
    has_prior_swarm_add = False
    has_prior_worker_board_calls = False
    for idx, s in enumerate(steps):
        if isinstance(s, dict) and s.get("type") == "PLANNER_RESPONSE":
            t_calls = tool_calls_extractor(idx, s) if tool_calls_extractor is not None else (s.get("tool_calls") or [])
            for c in t_calls:
                if not isinstance(c, dict):
                    continue
                c_name = c.get("name", "")
                raw_args = c.get("args") or {}
                c_args = decode_json_strings(raw_args) if isinstance(raw_args, dict) else {}
                if c_name == "swarm" and isinstance(c_args, dict) and c_args.get("action") == "add" and not has_prior_worker_board_calls:
                    has_prior_swarm_add = True
                elif c_name in ("read_board", "post_message", "swarm_wait") and not has_prior_swarm_add:
                    has_prior_worker_board_calls = True
    in_args = inflight_args if isinstance(inflight_args, dict) else {}
    has_coordinator_swarm_add = has_prior_swarm_add or (
        inflight_name == "swarm" and in_args.get("action") == "add" and not has_prior_worker_board_calls
    )
    has_worker_board_calls = has_prior_worker_board_calls or (
        not has_coordinator_swarm_add and inflight_name in ("read_board", "post_message", "swarm_wait")
    )
    is_swarm_worker = (not has_coordinator_swarm_add) and (
        has_worker_board_calls
        or bool(re.match(r"^@[a-z0-9-]+(?:\s|$)", str(last_user_content).lstrip()))
    )
    return has_coordinator_swarm_add, is_swarm_worker

def compaction_step_indices(steps: list[dict]) -> list[int]:
    res = []
    for idx, s in enumerate(steps):
        if not isinstance(s, dict):
            continue
        if s.get("type") == "CHECKPOINT":
            content = s.get("content") or ""
            if isinstance(content, str):
                stripped = content.lstrip()
                if stripped.startswith("# Resuming from a compaction") or "<CONTEXT_SUMMARY>" in stripped:
                    res.append(idx)
    return res

def decode_json_strings(val):
    if isinstance(val, str):
        s = val.strip()
        if ((s.startswith("{") and s.endswith("}")) or
            (s.startswith("[") and s.endswith("]")) or
            (s.startswith('"') and s.endswith('"') and len(s) >= 2)):
            try:
                decoded = json.loads(s)
                return decode_json_strings(decoded)
            except Exception:
                pass
        return s
    elif isinstance(val, dict):
        return {k: decode_json_strings(v) for k, v in val.items()}
    elif isinstance(val, list):
        return [decode_json_strings(v) for v in val]
    return val

def rule_mode(rule: str) -> str:
    if not rule or not isinstance(rule, str):
        return "enforce"
    r = rule.strip().upper()
    env_val = os.environ.get(f"BREAKER_{r}_MODE", "").strip().lower()
    if env_val in ("log", "enforce", "off"):
        return env_val
    return "enforce"

def probe_target_path(cmd: str) -> str | None:
    if not cmd or not isinstance(cmd, str):
        return None
    m = re.search(r"(?:^|[\s;&|])(?:/[^\s]+/)?python3?\s+(-c\b|-\s*<<)", cmd)
    if not m:
        return None
    body = cmd[m.end():]
    cand_matches = re.findall(r"(?<![\w.~/:-])(~?/[\w.*?-]+(?:/[\w.*?-]+)*)", body)
    for cand in cand_matches:
        if "*" in cand or "?" in cand:
            continue
        if cand.startswith("//"):
            continue
        if cand.startswith("/") or cand.startswith("~/"):
            parts = [p for p in cand.strip("~").strip("/").split("/") if p]
            if len(parts) < 2:
                continue
            if parts[0] == "dev":
                continue
            return os.path.expanduser(cand)
    return None

def append_breaker_log(entry: dict, log_path: str | None = None) -> None:
    try:
        path = log_path or os.environ.get("BREAKER_LOG_PATH") or os.path.expanduser("~/.gemini/metrics/breaker.jsonl")
        os.makedirs(os.path.dirname(os.path.abspath(path)), exist_ok=True)
        with open(path, "a", encoding="utf-8") as f:
            f.write(json.dumps(entry) + "\n")
    except Exception:
        pass

def extract_conversation_id(payload: dict) -> str:
    t_path_str = payload.get("transcriptPath")
    if t_path_str:
        norm = str(t_path_str).replace("\\", "/")
        if "/brain/" in norm:
            parts = norm.split("/brain/", 1)[1].split("/")
            if parts and parts[0]:
                return parts[0]
    return str(payload.get("conversationId") or "")

def _parse_token_field(val) -> int | None:
    if val is None or val == "":
        return 0
    if isinstance(val, bool):
        return None
    if isinstance(val, str) and len(val.strip()) > 12:
        return None
    try:
        n = int(val)
    except (TypeError, ValueError, OverflowError):
        return None
    if n < 0 or n > 100_000_000:
        return None
    return n

def context_estimate_tokens(
    full_lines_bytes: list[int],
    compaction_positions: list[int],
    steps: list[dict] | None = None,
) -> int:
    seg_start = (compaction_positions[-1] + 1) if compaction_positions else 0
    byte_est = 0
    if full_lines_bytes and seg_start < len(full_lines_bytes):
        byte_est = sum(full_lines_bytes[seg_start:]) // 4

    if steps and seg_start < len(steps):
        for idx in range(len(steps) - 1, seg_start - 1, -1):
            s = steps[idx]
            if not isinstance(s, dict) or s.get("type") != "PLANNER_RESPONSE":
                continue
            inp = _parse_token_field(s.get("input_tokens"))
            cache = _parse_token_field(s.get("cache_read_tokens"))
            api_out = _parse_token_field(s.get("output_tokens"))
            if inp is None or cache is None or api_out is None:
                continue
            api_in = inp + cache
            if api_in > 0:
                trailing_bytes = 0
                if full_lines_bytes and idx + 1 < len(full_lines_bytes):
                    trailing_bytes = sum(full_lines_bytes[idx + 1:]) // 4
                return max(byte_est, api_in + api_out + trailing_bytes)

    return byte_est

def load_baton_state(conv_id: str, current_segment: int, state_dir: str = "~/.gemini/state") -> dict:
    if not conv_id:
        return {"warned_at_estimate": None, "handoff_at_estimate": None, "segment": current_segment}
    exp_dir = os.path.expanduser(state_dir)
    state_file = os.path.join(exp_dir, f"baton-{conv_id}.json")
    try:
        if os.path.exists(state_file):
            with open(state_file, "r", encoding="utf-8") as f:
                data = json.load(f)
            stored_segment = data.get("segment", 0)
            if current_segment > stored_segment:
                return {"warned_at_estimate": None, "handoff_at_estimate": None, "segment": current_segment}
            return {
                "warned_at_estimate": data.get("warned_at_estimate"),
                "handoff_at_estimate": data.get("handoff_at_estimate"),
                "segment": stored_segment,
            }
    except Exception:
        pass
    return {"warned_at_estimate": None, "handoff_at_estimate": None, "segment": current_segment}

def save_baton_state(conv_id: str, state: dict, state_dir: str = "~/.gemini/state") -> None:
    if not conv_id:
        return
    exp_dir = os.path.expanduser(state_dir)
    state_file = os.path.join(exp_dir, f"baton-{conv_id}.json")
    tmp_file = os.path.join(exp_dir, f"baton-{conv_id}.json.tmp.{os.getpid()}")
    try:
        os.makedirs(exp_dir, exist_ok=True)
        with open(tmp_file, "w", encoding="utf-8") as f:
            json.dump(state, f)
        os.replace(tmp_file, state_file)
    except Exception:
        try:
            if os.path.exists(tmp_file):
                os.remove(tmp_file)
        except Exception:
            pass


