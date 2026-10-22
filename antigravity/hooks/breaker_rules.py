import json
import os
import re

AUTHORIZED_PATTERNS = [
    re.compile(r"/goal", re.IGNORECASE),
    re.compile(r"autonomous", re.IGNORECASE),
    re.compile(r"lead orchestrator", re.IGNORECASE),
    re.compile(r"continue from pending steps", re.IGNORECASE),
    re.compile(r"ingest .*active_context", re.IGNORECASE),
    re.compile(r"implement .*(phase|stream)", re.IGNORECASE),
    re.compile(r"keep going", re.IGNORECASE),
    re.compile(r"\b(start|begin|execute|run|continue)\s+phase\s+\d+", re.IGNORECASE),
    re.compile(r"brief-phase", re.IGNORECASE),
    re.compile(r"handoff-phase", re.IGNORECASE),
    re.compile(r"complete instructions for phase\s+\w+", re.IGNORECASE),
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
    low = path.lower()
    for ext in (".md", ".txt", ".json", ".yaml", ".yml"):
        if low.endswith(ext):
            return False
    for sub in ("/brain/", "/tmp/", "docs/plans/"):
        if sub in path:
            return False
    return True

def is_complaint(user_text: str) -> bool:
    if not user_text or not isinstance(user_text, str):
        return False
    return bool(COMPLAINT_PATTERN.search(user_text))

def compaction_step_indices(steps: list[dict]) -> list[int]:
    res = []
    for idx, s in enumerate(steps):
        if not isinstance(s, dict):
            continue
        if s.get("type") == "CHECKPOINT":
            content = s.get("content") or ""
            if isinstance(content, str) and content.lstrip().startswith("# Resuming from a compaction"):
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
    if r in ("PROBE", "TRUELOOP", "CHECKPOINT", "H1", "H2"):
        return "log"
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
        norm = str(t_path_str)
        if "/brain/" in norm:
            parts = norm.split("/brain/", 1)[1].split("/")
            if parts and parts[0]:
                return parts[0]
    return str(payload.get("conversationId") or "")

def context_estimate_tokens(full_lines_bytes: list[int], compaction_positions: list[int]) -> int:
    if not full_lines_bytes:
        return 0
    if not compaction_positions:
        return sum(full_lines_bytes) // 4
    last_cp = compaction_positions[-1]
    if last_cp + 1 >= len(full_lines_bytes):
        return 0
    return sum(full_lines_bytes[last_cp + 1:]) // 4

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


