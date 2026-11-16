#!/usr/bin/env python3
import argparse
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path, PurePosixPath


POST_CODE_MARKER = "## Post-Code Verification & Diff"
INDEX_MARKER = "## Read Chunk Index"
MANIFEST_MARKER = "## Turn-1 Parallel Read Manifest"

RETIRED_V3_SKILLS = frozenset(
    {
        "to-spec",
        "to-tickets",
        "implement",
        "domain-modeling",
        "grill-with-docs",
        "architecture-decision-records",
        "javascript-testing-patterns",
        "improve-codebase-architecture",
        "codebase-design",
    }
)

ORCHESTRATION_AND_META_SKILLS = frozenset(
    {
        "begin",
        "brainstorming",
        "code-simplifier",
        "comprehensive-code-review",
        "craftsmans-wisdom",
        "dispatching-parallel-agents",
        "editor-pass",
        "fix-check",
        "future-grilling",
        "future-model-router",
        "future-wayfinder",
        "future-writing-plans",
        "grilling-page",
        "handoff",
        "huashu-design",
        "lifecycle",
        "meeting-debrief",
        "plan-review",
        "plugin-updater",
        "prototype",
        "pull-request-description",
        "qa-verification",
        "research",
        "review-lens",
        "start",
        "subagent-driven-development",
        "teach",
        "team-pulse",
        "ticket-grooming",
        "ui-design",
        "using-git-worktrees",
        "visualize-plan",
        "write-a-prd",
        "writing-tickets",
    }
)
META_SKILL_PREFIXES = ("baton-", "start-")
META_SKILL_SUFFIXES = ("-orchestration", "-swarm")

MAX_DECISION_DOCS = 4

SKILL_SLUG_RE = re.compile(r"^[a-z0-9][a-z0-9-]{0,63}$")
EXPLICIT_SKILL_MD_RE = re.compile(r"(?:^|/)([a-z0-9][a-z0-9-]{0,63})/SKILL\.md$")
UI_EXTENSIONS = frozenset({".tsx", ".jsx", ".html", ".css", ".scss", ".vue", ".svelte"})
CODE_EXTENSIONS = frozenset(
    {
        ".py",
        ".go",
        ".ts",
        ".tsx",
        ".js",
        ".jsx",
        ".mjs",
        ".cjs",
        ".rb",
        ".rs",
        ".java",
        ".kt",
        ".swift",
        ".c",
        ".cc",
        ".cpp",
        ".h",
        ".sh",
        ".sql",
        ".vue",
        ".svelte",
    }
)
MOTION_TOKENS = ("anim", "motion", "transition", "spring", "keyframe", "rive", "gsap")


def fail(message: str) -> int:
    sys.stderr.write(f"error: {message}\n")
    return 1


def max_backtick_run(lines: list[str]) -> int:
    best = 0
    for line in lines:
        cur = 0
        for ch in line:
            if ch == "`":
                cur += 1
                if cur > best:
                    best = cur
            else:
                cur = 0
    return best


def fence_for(lines: list[str], info: str = "") -> tuple[str, str]:
    ticks = "`" * max(3, max_backtick_run(lines) + 1)
    return f"{ticks}{info}", ticks


def line_utf8_bytes(line: str) -> int:
    return len((line + "\n").encode("utf-8"))


def lines_utf8_bytes(lines: list[str]) -> int:
    return sum(line_utf8_bytes(ln) for ln in lines)


def split_long_utf8_line(line: str, max_bytes: int) -> list[str]:
    budget = max(64, max_bytes - 1)
    if len(line.encode("utf-8")) <= budget:
        return [line]
    segments: list[str] = []
    buf: list[str] = []
    used = 0
    for ch in line:
        ch_len = len(ch.encode("utf-8"))
        if buf and used + ch_len > budget:
            segments.append("".join(buf))
            buf = [ch]
            used = ch_len
        else:
            buf.append(ch)
            used += ch_len
    if buf:
        segments.append("".join(buf))
    return segments


def wrap_oversized_lines(lines: list[str], max_bytes: int) -> list[str]:
    wrapped: list[str] = []
    for line in lines:
        wrapped.extend(split_long_utf8_line(line, max_bytes))
    return wrapped


def extract_acceptance_criteria(plan_text: str) -> list[str]:
    lines = plan_text.splitlines()
    start_idx = None
    for idx, line in enumerate(lines):
        if line.startswith("## Acceptance Criteria"):
            start_idx = idx
            break
    if start_idx is None:
        return []
    collected = [lines[start_idx]]
    for line in lines[start_idx + 1 :]:
        if line.startswith("## ") or line.strip() == "---":
            break
        collected.append(line)
    while collected and collected[-1].strip() == "":
        collected.pop()
    return collected


def parse_slice_spec(spec: str, root: Path) -> tuple[str, int, int, list[str]]:
    if ":" not in spec:
        raise ValueError(f"malformed slice '{spec}', expected <file>:<start>-<end>")
    file_part, range_part = spec.rsplit(":", 1)
    if not file_part or "-" not in range_part:
        raise ValueError(f"malformed slice '{spec}', expected <file>:<start>-<end>")
    start_str, end_str = range_part.split("-", 1)
    if not start_str.isdigit() or not end_str.isdigit():
        raise ValueError(f"malformed slice line range in '{spec}', expected positive integers")
    start_line = int(start_str)
    end_line = int(end_str)
    if start_line < 1 or start_line > end_line:
        raise ValueError(f"invalid slice range in '{spec}': require 1 <= start <= end")
    try:
        root_real = root.resolve(strict=True)
    except OSError as exc:
        raise ValueError(f"invalid root directory '{root}': {exc}") from exc
    raw_path = Path(file_part)
    candidate = raw_path if raw_path.is_absolute() else (root_real / raw_path)
    try:
        target_real = candidate.resolve(strict=True)
    except OSError as exc:
        raise ValueError(f"slice file does not exist: {file_part}") from exc
    if not target_real.is_relative_to(root_real):
        raise ValueError(f"slice path '{file_part}' escapes root '{root_real}'")
    if not target_real.is_file():
        raise ValueError(f"slice file does not exist: {file_part}")
    file_lines = target_real.read_text(encoding="utf-8", errors="replace").splitlines()
    if end_line > len(file_lines):
        raise ValueError(
            f"slice '{spec}' out of bounds: file has {len(file_lines)} lines"
        )
    rel_label = target_real.relative_to(root_real).as_posix()
    return rel_label, start_line, end_line, file_lines[start_line - 1 : end_line]


def render_index(chunks: list[tuple[int, int]]) -> list[str]:
    out = [INDEX_MARKER]
    for idx, (start_line, end_line) in enumerate(chunks, start=1):
        out.append(
            f"- Chunk {idx}: StartLine={start_line} EndLine={end_line} ({end_line - start_line + 1} lines)"
        )
    return out


def leading_backtick_count(line: str) -> int:
    count = 0
    for ch in line:
        if ch == "`":
            count += 1
        else:
            break
    return count


def partition_lines(
    full_lines: list[str], max_chunk_lines: int, max_chunk_bytes: int
) -> list[tuple[int, int]]:
    total = len(full_lines)
    if total == 0:
        return [(1, 1)]

    fence_before = [False] * total
    open_fence_len = 0
    for idx, line in enumerate(full_lines):
        fence_before[idx] = open_fence_len > 0
        ticks = leading_backtick_count(line)
        if open_fence_len == 0:
            if ticks >= 3:
                open_fence_len = ticks
        else:
            stripped_right = line.rstrip()
            if ticks >= open_fence_len and len(stripped_right) == ticks:
                open_fence_len = 0

    prefix_bytes = [0] * (total + 1)
    for idx, line in enumerate(full_lines):
        prefix_bytes[idx + 1] = prefix_bytes[idx] + line_utf8_bytes(line)

    chunks: list[tuple[int, int]] = []
    pos = 0
    while pos < total:
        line_cap_end = min(total, pos + max_chunk_lines)
        hard_end = pos + 1
        while (
            hard_end < line_cap_end
            and (prefix_bytes[hard_end + 1] - prefix_bytes[pos]) <= max_chunk_bytes
        ):
            hard_end += 1

        if hard_end == total:
            chunks.append((pos + 1, total))
            break

        window_len = hard_end - pos
        min_split = pos + max(1, window_len // 2)
        split_at = None
        for cand in range(hard_end, min_split - 1, -1):
            if not fence_before[cand] and (
                full_lines[cand].startswith("### Diff Part ")
                or full_lines[cand].startswith("## ")
                or full_lines[cand].startswith("### ")
            ):
                split_at = cand
                break
        if split_at is None:
            for cand in range(hard_end, min_split - 1, -1):
                if not fence_before[cand] and full_lines[cand].strip() == "":
                    split_at = cand
                    break
        if split_at is None or split_at <= pos:
            split_at = hard_end
        chunks.append((pos + 1, split_at))
        pos = split_at
    return chunks


def build_indexed_document(
    title: str, body_lines: list[str], max_chunk_lines: int, max_chunk_bytes: int
) -> str:
    raw_body = "\n".join(body_lines).splitlines()
    cleaned_body = wrap_oversized_lines(raw_body, max(128, max_chunk_bytes - 256))
    while cleaned_body and cleaned_body[0].strip() == "":
        cleaned_body.pop(0)
    while cleaned_body and cleaned_body[-1].strip() == "":
        cleaned_body.pop()

    index_lines = render_index([(1, max(1, len(cleaned_body) + 5))])
    full_lines: list[str] = []
    for _ in range(10):
        full_lines = [title, ""] + index_lines + [""] + cleaned_body
        chunks = partition_lines(full_lines, max_chunk_lines, max_chunk_bytes)
        next_index = render_index(chunks)
        if next_index == index_lines:
            break
        index_lines = next_index
    full_lines = [title, ""] + index_lines + [""] + cleaned_body
    return "\n".join(full_lines) + "\n"


def atomic_write_private(out_path: Path, rendered: str) -> None:
    if out_path.is_symlink():
        raise ValueError(f"refusing to write symlink output path: {out_path}")
    if out_path.exists() and not out_path.is_file():
        raise ValueError(f"output path is not a regular file: {out_path}")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    tmp_path: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            dir=out_path.parent,
            prefix=".subagent-context-",
            suffix=".tmp",
            mode="w",
            encoding="utf-8",
            delete=False,
        ) as tmp_fh:
            tmp_path = Path(tmp_fh.name)
            tmp_fh.write(rendered)
            tmp_fh.flush()
            os.fsync(tmp_fh.fileno())
        os.chmod(tmp_path, 0o600)
        if out_path.is_symlink():
            raise ValueError(f"refusing to overwrite symlink output path: {out_path}")
        os.replace(tmp_path, out_path)
        tmp_path = None
    finally:
        if tmp_path is not None and tmp_path.exists():
            tmp_path.unlink(missing_ok=True)


def strip_index_and_post_code(existing_text: str) -> tuple[str, list[str]]:
    lines = existing_text.splitlines()
    title = "# Subagent Context"
    rest = lines
    if lines and lines[0].startswith("# "):
        title = lines[0]
        rest = lines[1:]
    idx = 0
    while idx < len(rest) and rest[idx].strip() == "":
        idx += 1
    if idx < len(rest) and rest[idx].strip() == INDEX_MARKER:
        idx += 1
        while idx < len(rest):
            stripped = rest[idx].strip()
            if stripped == "" or rest[idx].startswith("- Chunk "):
                idx += 1
                continue
            break
    trimmed = rest[idx:]
    cut_idx = None
    for pos, line in enumerate(trimmed):
        if line.strip() == POST_CODE_MARKER:
            cut_idx = pos
            break
    if cut_idx is not None:
        trimmed = trimmed[:cut_idx]
    while trimmed and trimmed[0].strip() == "":
        trimmed.pop(0)
    while trimmed and trimmed[-1].strip() == "":
        trimmed.pop()
    return title, trimmed


def split_unified_diff(
    diff_text: str, max_chunk_lines: int, max_chunk_bytes: int
) -> list[list[str]]:
    payload_line_limit = max(10, max_chunk_lines - 8)
    payload_byte_limit = max(256, max_chunk_bytes - 320)
    raw_lines = wrap_oversized_lines(
        diff_text.splitlines(), max(128, payload_byte_limit // 2)
    )
    if not raw_lines:
        return [["(no diff changes)"]]

    file_blocks: list[list[str]] = []
    current_file: list[str] = []
    for line in raw_lines:
        if line.startswith("diff --git ") and current_file:
            file_blocks.append(current_file)
            current_file = [line]
        else:
            current_file.append(line)
    if current_file:
        file_blocks.append(current_file)

    parts: list[list[str]] = []
    current_part: list[str] = []
    current_bytes = 0

    def flush_part() -> None:
        nonlocal current_part, current_bytes
        if current_part:
            parts.append(current_part)
            current_part = []
            current_bytes = 0

    def stream_lines_with_prefix(prefix: list[str], body_seq: list[str]) -> None:
        prefix_bytes = lines_utf8_bytes(prefix)
        effective_prefix = prefix if (
            len(prefix) < payload_line_limit and prefix_bytes < payload_byte_limit // 2
        ) else []
        eff_prefix_bytes = lines_utf8_bytes(effective_prefix)
        buf = list(effective_prefix)
        buf_bytes = eff_prefix_bytes
        for ln in body_seq:
            ln_b = line_utf8_bytes(ln)
            if len(buf) > len(effective_prefix) and (
                len(buf) + 1 > payload_line_limit
                or buf_bytes + ln_b > payload_byte_limit
            ):
                parts.append(buf)
                buf = list(effective_prefix)
                buf_bytes = eff_prefix_bytes
            buf.append(ln)
            buf_bytes += ln_b
        if len(buf) > len(effective_prefix):
            parts.append(buf)

    for block in file_blocks:
        block_bytes = lines_utf8_bytes(block)
        if len(block) <= payload_line_limit and block_bytes <= payload_byte_limit:
            if current_part and (
                len(current_part) + len(block) > payload_line_limit
                or current_bytes + block_bytes > payload_byte_limit
            ):
                flush_part()
            current_part.extend(block)
            current_bytes += block_bytes
            continue

        flush_part()
        header: list[str] = []
        hunks: list[list[str]] = []
        cur_hunk: list[str] = []
        seen_hunk = False
        for line in block:
            if line.startswith("@@ "):
                seen_hunk = True
                if cur_hunk:
                    hunks.append(cur_hunk)
                cur_hunk = [line]
            elif not seen_hunk:
                header.append(line)
            else:
                cur_hunk.append(line)
        if cur_hunk:
            hunks.append(cur_hunk)

        if not hunks:
            stream_lines_with_prefix([], block)
            continue

        header_bytes = lines_utf8_bytes(header)
        part_buf: list[str] = list(header)
        part_buf_bytes = header_bytes
        for hunk in hunks:
            hunk_bytes = lines_utf8_bytes(hunk)
            if (
                len(header) + len(hunk) > payload_line_limit
                or header_bytes + hunk_bytes > payload_byte_limit
            ):
                if len(part_buf) > len(header):
                    parts.append(part_buf)
                    part_buf = list(header)
                    part_buf_bytes = header_bytes
                stream_lines_with_prefix(header, hunk)
            elif (
                len(part_buf) + len(hunk) > payload_line_limit
                or part_buf_bytes + hunk_bytes > payload_byte_limit
            ):
                parts.append(part_buf)
                part_buf = list(header) + hunk
                part_buf_bytes = header_bytes + hunk_bytes
            else:
                part_buf.extend(hunk)
                part_buf_bytes += hunk_bytes
        if len(part_buf) > len(header):
            parts.append(part_buf)

    flush_part()
    return parts if parts else [["(no diff changes)"]]


def _safe_in_root_regular_file(cand: Path, root_real: Path) -> Path | None:
    try:
        rel = cand.relative_to(root_real)
    except ValueError:
        return None
    cur = root_real
    for part in rel.parts:
        cur = cur / part
        if cur.is_symlink():
            return None
    try:
        real_p = cur.resolve(strict=True)
    except OSError:
        return None
    if not real_p.is_relative_to(root_real) or not real_p.is_file():
        return None
    return real_p


def validate_touch_path(raw_touch: str, root: Path) -> str:
    if not isinstance(raw_touch, str) or not raw_touch.strip():
        raise ValueError("empty --touch path")
    s = raw_touch.strip()
    norm_slashes = s.replace("\\", "/")
    if ".." in re.split(r"[\\/]+", s) or ".." in PurePosixPath(norm_slashes).parts:
        raise ValueError(f"--touch path '{raw_touch}' must not contain '..'")
    try:
        root_real = root.resolve(strict=True)
    except OSError as exc:
        raise ValueError(f"invalid root directory '{root}': {exc}") from exc

    raw_p = Path(s)
    if not raw_p.is_absolute():
        cur = root_real
        for pt in PurePosixPath(norm_slashes).parts:
            if pt in ("", "."):
                continue
            cur = cur / pt
            if cur.is_symlink():
                try:
                    lt = cur.resolve(strict=True)
                except OSError as exc:
                    raise ValueError(f"--touch symlink does not resolve: {raw_touch}") from exc
                if not lt.is_relative_to(root_real):
                    raise ValueError(f"--touch path '{raw_touch}' escapes root '{root_real}'")
        candidate = root_real / raw_p
    else:
        candidate = raw_p
        anc = candidate
        while not anc.exists() and not anc.is_symlink() and anc.parent != anc:
            anc = anc.parent
        if anc.is_symlink():
            try:
                anc_real = anc.resolve(strict=True)
            except OSError as exc:
                raise ValueError(f"--touch symlink ancestor invalid: {raw_touch}") from exc
            if not anc_real.is_relative_to(root_real) and not root_real.is_relative_to(anc_real):
                raise ValueError(f"--touch path '{raw_touch}' escapes root '{root_real}'")

    resolved = candidate.resolve(strict=False)
    if not resolved.is_relative_to(root_real) or resolved == root_real:
        raise ValueError(f"--touch path '{raw_touch}' escapes root '{root_real}'")

    cur_check = root_real
    for pt in resolved.relative_to(root_real).parts:
        cur_check = cur_check / pt
        if cur_check.is_symlink():
            try:
                lt2 = cur_check.resolve(strict=True)
            except OSError as exc:
                raise ValueError(f"--touch symlink does not resolve: {raw_touch}") from exc
            if not lt2.is_relative_to(root_real):
                raise ValueError(f"--touch path '{raw_touch}' escapes root '{root_real}'")

    return resolved.relative_to(root_real).as_posix()


def extract_plan_paths_and_skills(plan_text: str, root: Path) -> tuple[list[str], list[str]]:
    try:
        root_real = root.resolve(strict=True)
    except OSError:
        return [], []

    plan_paths: list[str] = []
    seen_paths: set[str] = set()
    skill_tokens: list[str] = []
    seen_skills: set[str] = set()

    for raw_span in re.findall(r"`([^`\n]+)`", plan_text):
        tok = raw_span.strip()
        if not tok:
            continue

        m_skill_md = EXPLICIT_SKILL_MD_RE.search(tok)
        if m_skill_md:
            explicit_tok = f"{m_skill_md.group(1)}/SKILL.md"
            if explicit_tok not in seen_skills:
                seen_skills.add(explicit_tok)
                skill_tokens.append(explicit_tok)

        if SKILL_SLUG_RE.match(tok) and tok not in seen_skills:
            seen_skills.add(tok)
            skill_tokens.append(tok)

        path_cand = tok.split(":", 1)[0].strip()
        while path_cand.startswith("./"):
            path_cand = path_cand[2:]
        if (
            not path_cand
            or path_cand.startswith(("/", "~", "-"))
            or ".." in re.split(r"[\\/]+", path_cand)
        ):
            continue
        safe_p = _safe_in_root_regular_file(root_real / path_cand, root_real)
        if safe_p is not None:
            rel_posix = safe_p.relative_to(root_real).as_posix()
            if rel_posix not in seen_paths:
                seen_paths.add(rel_posix)
                plan_paths.append(rel_posix)

    return plan_paths, skill_tokens


def _is_ui_or_frontend_path(rel_path: str) -> bool:
    low = rel_path.lower().replace("\\", "/")
    pp = PurePosixPath(low)
    if pp.suffix in UI_EXTENSIONS:
        return True
    ui_dirs = {"mobile", "web", "frontend", "ui", "extension", "components", "screens", "pages", "design-demos"}
    return any(part in ui_dirs for part in pp.parts[:-1])


def discover_product_and_design_docs(
    root: Path, touched_paths: list[str], plan_text: str
) -> list[Path]:
    try:
        root_real = root.resolve(strict=True)
    except OSError:
        return []

    candidates: list[Path] = [root_real / "docs" / "prd.md"]
    docs_dir = root_real / "docs"
    if not docs_dir.is_symlink() and docs_dir.is_dir():
        decisions_dir = docs_dir / "decisions"
        if not decisions_dir.is_symlink() and decisions_dir.is_dir():
            safe_decisions = [
                sd
                for p in sorted(decisions_dir.glob("*.md"))
                if (sd := _safe_in_root_regular_file(p, root_real)) is not None
            ]
            prioritized: list[Path] = []
            seen_dec: set[str] = set()
            for sd in safe_decisions:
                if sd.name == "product-scope.md" or sd.name in plan_text:
                    prioritized.append(sd)
                    seen_dec.add(sd.as_posix())
            for sd in safe_decisions:
                if len(prioritized) >= MAX_DECISION_DOCS:
                    break
                if sd.as_posix() not in seen_dec:
                    prioritized.append(sd)
                    seen_dec.add(sd.as_posix())
            candidates.extend(prioritized[:MAX_DECISION_DOCS])

        pov_dir = docs_dir / "point-of-view"
        if not pov_dir.is_symlink() and pov_dir.is_dir():
            candidates.extend(sorted(pov_dir.glob("*.md")))

    include_design = "direction-approved.md" in plan_text or any(
        _is_ui_or_frontend_path(p) for p in touched_paths
    )
    if include_design:
        demos_dir = root_real / "design-demos"
        if not demos_dir.is_symlink() and demos_dir.is_dir():
            candidates.append(demos_dir / "direction-approved.md")
            for child in sorted(demos_dir.iterdir()):
                if not child.is_symlink() and child.is_dir():
                    candidates.append(child / "direction-approved.md")

    found: dict[str, Path] = {}
    for cand in candidates:
        safe_doc = _safe_in_root_regular_file(cand, root_real)
        if safe_doc is not None:
            found[safe_doc.as_posix()] = safe_doc
    return [found[k] for k in sorted(found)]


def resolve_skills_dirs(explicit_dirs: list[str]) -> list[Path]:
    resolved_dirs: list[Path] = []
    seen: set[str] = set()
    if explicit_dirs:
        for raw in explicit_dirs:
            if not isinstance(raw, str) or not raw.strip():
                raise ValueError("empty --skills-dir path")
            try:
                real_d = Path(raw.strip()).resolve(strict=True)
            except OSError as exc:
                raise ValueError(f"skills-dir does not exist: {raw}") from exc
            if not real_d.is_dir():
                raise ValueError(f"skills-dir is not a directory: {raw}")
            key = real_d.as_posix()
            if key not in seen:
                seen.add(key)
                resolved_dirs.append(real_d)
        return resolved_dirs

    home_dir = Path.home()
    for default_cand in (
        home_dir / ".gemini" / "config" / "skills",
        home_dir / ".claude" / "skills",
    ):
        if not default_cand.is_dir():
            continue
        try:
            real_d = default_cand.resolve(strict=True)
        except OSError:
            continue
        if real_d.is_dir():
            key = real_d.as_posix()
            if key not in seen:
                seen.add(key)
                resolved_dirs.append(real_d)
    return resolved_dirs


def resolve_turn1_skills(
    touched_paths: list[str], plan_skill_tokens: list[str], skills_dirs: list[Path]
) -> list[tuple[str, Path]]:
    if not skills_dirs:
        return []

    wanted: list[str] = []
    seen_wanted: set[str] = set()

    def add_candidate(slug: str) -> None:
        if slug in RETIRED_V3_SKILLS or slug in seen_wanted:
            return
        if not SKILL_SLUG_RE.match(slug):
            return
        seen_wanted.add(slug)
        wanted.append(slug)

    has_go = False
    has_ui = False
    has_motion = False
    has_sql = False
    has_skill_md = False
    has_datadog = False
    has_code = False

    for raw_p in touched_paths:
        norm_p = raw_p.replace("\\", "/")
        low = norm_p.lower()
        pp = PurePosixPath(norm_p)
        ext = pp.suffix.lower()
        if ext == ".go":
            has_go = True
        if ext in UI_EXTENSIONS:
            has_ui = True
        if any(tok in low for tok in MOTION_TOKENS):
            has_motion = True
        if ext == ".sql" or "migrations/" in low or low.startswith("migrations/") or "db/migrate/" in low or low.startswith("db/migrate/"):
            has_sql = True
        if pp.name == "SKILL.md":
            has_skill_md = True
        if "datadog" in low or ("dashboard" in low and ext == ".json"):
            has_datadog = True
        if ext in CODE_EXTENSIONS:
            has_code = True

    if any(any(tok in st.lower() for tok in MOTION_TOKENS) for st in plan_skill_tokens):
        has_motion = True

    if has_go:
        add_candidate("go-api-structure")
    if has_ui:
        add_candidate("impeccable")
        add_candidate("no-slop-ui")
        if has_motion:
            add_candidate("animation-principles")
    if has_sql:
        add_candidate("sql-optimization-patterns")
    if has_skill_md:
        add_candidate("future-skill-creator")
        add_candidate("skill-reviewer")
    if has_datadog:
        add_candidate("datadog-dashboards")
    if has_code:
        add_candidate("test-driven-development")

    for tok in plan_skill_tokens:
        if tok.endswith("/SKILL.md"):
            slug = tok[: -len("/SKILL.md")]
            add_candidate(slug)
        else:
            if (
                tok in ORCHESTRATION_AND_META_SKILLS
                or tok.startswith(META_SKILL_PREFIXES)
                or tok.endswith(META_SKILL_SUFFIXES)
            ):
                continue
            add_candidate(tok)

    resolved: list[tuple[str, Path]] = []
    for slug in wanted:
        for s_dir in skills_dirs:
            try:
                dir_real = s_dir.resolve(strict=True)
            except OSError:
                continue
            skill_dir = dir_real / slug
            skill_md = skill_dir / "SKILL.md"
            if skill_dir.is_symlink() or not skill_dir.is_dir():
                continue
            if skill_md.is_symlink() or not skill_md.is_file():
                continue
            try:
                md_real = skill_md.resolve(strict=True)
            except OSError:
                continue
            if md_real.is_relative_to(dir_real) and md_real.is_file():
                resolved.append((slug, md_real))
                break
    return resolved


def render_turn1_manifest(
    args_plan: str,
    plan_abs: Path,
    doc_paths: list[Path],
    resolved_skills: list[tuple[str, Path]],
) -> tuple[list[str], list[str]]:
    manifest_body = [
        MANIFEST_MARKER,
        f"- Plan: {args_plan}",
        *[f"- Doc: {doc_p.as_posix()}" for doc_p in doc_paths],
        *[f"- Skill ({name}): {md_p.as_posix()}" for name, md_p in resolved_skills],
    ]
    stdout_lines = [
        "TURN1_READ_MANIFEST:",
        f"- {plan_abs.as_posix()}",
        *[f"- {doc_p.as_posix()}" for doc_p in doc_paths],
        *[f"- {md_p.as_posix()}" for _, md_p in resolved_skills],
    ]
    return manifest_body, stdout_lines


def run_pre_code(args: argparse.Namespace) -> int:
    if args.max_chunk_lines < 25:
        return fail("--max-chunk-lines must be >= 25")
    if args.max_chunk_bytes < 512:
        return fail("--max-chunk-bytes must be >= 512")
    out_path = Path(args.output)
    if out_path.is_symlink():
        return fail(f"refusing to write symlink output path: {out_path}")
    plan_path = Path(args.plan)
    if not plan_path.is_file():
        return fail(f"plan file does not exist: {args.plan}")
    root_dir = Path(args.root)
    if not root_dir.is_dir():
        return fail(f"root directory does not exist: {args.root}")

    parsed_slices = []
    for spec in args.slice or []:
        try:
            parsed_slices.append(parse_slice_spec(spec, root_dir))
        except ValueError as exc:
            return fail(str(exc))

    explicit_touches: list[str] = []
    for raw_touch in args.touch or []:
        try:
            explicit_touches.append(validate_touch_path(raw_touch, root_dir))
        except ValueError as exc:
            return fail(str(exc))

    try:
        skills_dirs = resolve_skills_dirs(args.skills_dir or [])
        plan_abs = plan_path.resolve(strict=True)
    except (OSError, ValueError) as exc:
        return fail(str(exc))

    plan_text = plan_path.read_text(encoding="utf-8", errors="replace")
    ac_lines = extract_acceptance_criteria(plan_text)
    plan_paths, plan_skill_tokens = extract_plan_paths_and_skills(plan_text, root_dir)

    touched_paths: list[str] = []
    seen_touched: set[str] = set()
    for rel_p in [
        *(s[0] for s in parsed_slices),
        *explicit_touches,
        *plan_paths,
    ]:
        if rel_p not in seen_touched:
            seen_touched.add(rel_p)
            touched_paths.append(rel_p)

    doc_paths = discover_product_and_design_docs(root_dir, touched_paths, plan_text)
    resolved_skills = resolve_turn1_skills(touched_paths, plan_skill_tokens, skills_dirs)
    manifest_lines, stdout_lines = render_turn1_manifest(
        args.plan, plan_abs, doc_paths, resolved_skills
    )

    body: list[str] = list(manifest_lines)
    if args.gate_cmd:
        body.extend(["", "## Gate Commands"])
        for cmd in args.gate_cmd:
            body.append(f"- `{cmd}`")

    if ac_lines:
        body.append("")
        body.extend(ac_lines)

    if parsed_slices:
        body.extend(["", "## Code Slices"])
        for file_label, start_line, end_line, snippet in parsed_slices:
            numbered = [
                f"{start_line + offset}: {text_line}"
                for offset, text_line in enumerate(snippet)
            ]
            open_fence, close_fence = fence_for(numbered, "text")
            body.extend(
                [
                    "",
                    f"### {file_label}:{start_line}-{end_line}",
                    open_fence,
                    *numbered,
                    close_fence,
                ]
            )

    rendered = build_indexed_document(
        "# Subagent Context (Pre-Code)",
        body,
        args.max_chunk_lines,
        args.max_chunk_bytes,
    )
    try:
        atomic_write_private(out_path, rendered)
    except (OSError, ValueError) as exc:
        return fail(str(exc))
    sys.stdout.write("\n".join(stdout_lines) + "\n")
    return 0


def run_git_checked(root: Path, git_args: list[str]) -> tuple[bool, str, str]:
    proc = subprocess.run(
        ["git", "-C", str(root), *git_args],
        capture_output=True,
        text=True,
    )
    return proc.returncode == 0, proc.stdout, proc.stderr


def run_post_code(args: argparse.Namespace) -> int:
    if args.max_chunk_lines < 25:
        return fail("--max-chunk-lines must be >= 25")
    if args.max_chunk_bytes < 512:
        return fail("--max-chunk-bytes must be >= 512")
    if args.base.startswith("-") or (args.head and args.head.startswith("-")):
        return fail("git refs must not start with '-'")
    out_path = Path(args.output)
    if out_path.is_symlink():
        return fail(f"refusing to read or write symlink output path: {out_path}")
    share_root = Path(args.share_root)
    if not share_root.is_dir():
        return fail(f"share-root directory does not exist: {args.share_root}")

    gate_tail: list[str] = []
    if args.gate_log:
        gate_path = Path(args.gate_log)
        if not gate_path.is_file():
            return fail(f"gate-log file does not exist: {args.gate_log}")
        all_gate_lines = gate_path.read_text(encoding="utf-8", errors="replace").splitlines()
        gate_tail = all_gate_lines[-40:]

    diff_spec = f"{args.base}...{args.head}" if args.head else args.base
    range_desc = diff_spec if args.head else f"{args.base} (working tree)"

    safe_diff_base = [
        "diff",
        "--no-ext-diff",
        "--no-textconv",
        "--src-prefix=a/",
        "--dst-prefix=b/",
    ]
    ok_stat, stat_out, stat_err = run_git_checked(
        share_root, [*safe_diff_base, "--stat", diff_spec, "--"]
    )
    if not ok_stat:
        return fail(f"git diff --stat failed: {stat_err.strip() or 'unknown git error'}")

    ok_diff, diff_out, diff_err = run_git_checked(
        share_root, [*safe_diff_base, "-U3", diff_spec, "--"]
    )
    if not ok_diff:
        return fail(f"git diff -U3 failed: {diff_err.strip() or 'unknown git error'}")

    title = "# Subagent Context (Review Bundle)"
    preserved_body: list[str] = []
    if out_path.is_file():
        existing_text = out_path.read_text(encoding="utf-8", errors="replace")
        title, preserved_body = strip_index_and_post_code(existing_text)

    stat_lines = (
        stat_out.rstrip().splitlines() if stat_out.strip() else ["(no changes)"]
    )
    stat_open, stat_close = fence_for(stat_lines, "text")
    post_section: list[str] = [
        POST_CODE_MARKER,
        "",
        f"- Diff Range: `{range_desc}`",
        "",
        "### Diff Stat",
        stat_open,
        *stat_lines,
        stat_close,
    ]

    if args.gate_log:
        gate_lines = gate_tail if gate_tail else ["(empty gate log)"]
        gate_open, gate_close = fence_for(gate_lines, "text")
        post_section.extend(
            [
                "",
                "## Gate Summary",
                gate_open,
                *gate_lines,
                gate_close,
            ]
        )

    diff_parts = split_unified_diff(
        diff_out, args.max_chunk_lines, args.max_chunk_bytes
    )
    post_section.extend(["", "## Unified Diff"])
    total_parts = len(diff_parts)
    for part_idx, part_lines in enumerate(diff_parts, start=1):
        diff_open, diff_close = fence_for(part_lines, "diff")
        post_section.extend(
            [
                "",
                f"### Diff Part {part_idx} (of {total_parts})",
                diff_open,
                *part_lines,
                diff_close,
            ]
        )

    combined_body = list(preserved_body)
    if combined_body:
        combined_body.append("")
    combined_body.extend(post_section)

    rendered = build_indexed_document(
        title, combined_body, args.max_chunk_lines, args.max_chunk_bytes
    )
    try:
        atomic_write_private(out_path, rendered)
    except (OSError, ValueError) as exc:
        return fail(str(exc))
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="build_subagent_context.py")
    subparsers = parser.add_subparsers(dest="subcommand", required=True)

    pre = subparsers.add_parser("pre-code")
    pre.add_argument("--output", "--context", dest="output", required=True)
    pre.add_argument("--plan", required=True)
    pre.add_argument("--slice", action="append", default=[])
    pre.add_argument("--touch", action="append", default=[])
    pre.add_argument("--skills-dir", action="append", default=[])
    pre.add_argument("--gate-cmd", action="append", default=[])
    pre.add_argument("--root", default=".")
    pre.add_argument("--max-chunk-lines", type=int, default=700)
    pre.add_argument("--max-chunk-bytes", type=int, default=38000)

    post = subparsers.add_parser("post-code")
    post.add_argument("--output", "--context", dest="output", required=True)
    post.add_argument("--base", required=True)
    post.add_argument("--head", default=None)
    post.add_argument("--share-root", default=".")
    post.add_argument("--gate-log", default=None)
    post.add_argument("--max-chunk-lines", type=int, default=700)
    post.add_argument("--max-chunk-bytes", type=int, default=38000)

    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    if args.subcommand == "pre-code":
        return run_pre_code(args)
    if args.subcommand == "post-code":
        return run_post_code(args)
    return fail(f"unknown subcommand: {args.subcommand}")


if __name__ == "__main__":
    sys.exit(main())
