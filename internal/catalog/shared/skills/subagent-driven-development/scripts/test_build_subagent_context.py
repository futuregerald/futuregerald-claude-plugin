import os
import stat
import subprocess
import sys
import tokenize
from pathlib import Path

import pytest

SCRIPTS_DIR = Path(__file__).resolve().parent
SCRIPT_PATH = SCRIPTS_DIR / "build_subagent_context.py"


def load_module():
    if str(SCRIPTS_DIR) not in sys.path:
        sys.path.insert(0, str(SCRIPTS_DIR))
    import build_subagent_context
    return build_subagent_context


def run_cli(*args, cwd=None, env=None):
    proc_env = os.environ.copy()
    if env:
        proc_env.update(env)
    return subprocess.run(
        [sys.executable, str(SCRIPT_PATH), *args],
        cwd=cwd,
        env=proc_env,
        capture_output=True,
        text=True,
    )


@pytest.fixture(autouse=True)
def isolate_home(tmp_path, monkeypatch):
    fake_home = tmp_path / "isolated_home"
    fake_home.mkdir(parents=True, exist_ok=True)
    monkeypatch.setenv("HOME", str(fake_home))
    monkeypatch.setenv("USERPROFILE", str(fake_home))
    return fake_home


def init_git_repo(repo_dir: Path) -> str:
    subprocess.run(["git", "init", "-q"], cwd=repo_dir, check=True)
    subprocess.run(["git", "config", "user.email", "test@example.com"], cwd=repo_dir, check=True)
    subprocess.run(["git", "config", "user.name", "Test Runner"], cwd=repo_dir, check=True)
    alpha = repo_dir / "src" / "alpha.py"
    alpha.parent.mkdir(parents=True, exist_ok=True)
    alpha.write_text(
        "\n".join(f"def fn_{idx}(): return {idx}" for idx in range(1, 31)) + "\n",
        encoding="utf-8",
    )
    subprocess.run(["git", "add", "."], cwd=repo_dir, check=True)
    subprocess.run(["git", "commit", "-q", "-m", "chore: initial"], cwd=repo_dir, check=True)
    res = subprocess.run(
        ["git", "rev-parse", "HEAD"],
        cwd=repo_dir,
        check=True,
        capture_output=True,
        text=True,
    )
    return res.stdout.strip()


def assert_read_chunk_index_reconstructs_exact_file(
    content: str, max_chunk_lines: int, max_chunk_bytes: int = 38000
):
    lines = content.splitlines()
    assert "## Read Chunk Index" in lines
    chunk_ranges = []
    in_index = False
    for line in lines:
        if line.strip() == "## Read Chunk Index":
            in_index = True
            continue
        if in_index:
            if line.startswith("## ") or (line.strip() == "" and chunk_ranges):
                if line.startswith("## "):
                    break
                continue
            if line.startswith("- Chunk "):
                parts = line.split()
                start_kv = [p for p in parts if p.startswith("StartLine=")][0]
                end_kv = [p for p in parts if p.startswith("EndLine=")][0]
                start_line = int(start_kv.split("=", 1)[1])
                end_line = int(end_kv.split("=", 1)[1])
                chunk_ranges.append((start_line, end_line))
    assert len(chunk_ranges) >= 1
    assert chunk_ranges[0][0] == 1
    assert chunk_ranges[-1][1] == len(lines)
    reconstructed = []
    expected_start = 1
    for start_line, end_line in chunk_ranges:
        assert start_line == expected_start
        window = lines[start_line - 1 : end_line]
        assert 1 <= len(window) <= max_chunk_lines
        window_utf8_bytes = sum(len((ln + "\n").encode("utf-8")) for ln in window)
        assert window_utf8_bytes <= max_chunk_bytes, (
            f"Chunk {start_line}-{end_line} is {window_utf8_bytes} bytes > {max_chunk_bytes}"
        )
        reconstructed.extend(window)
        expected_start = end_line + 1
    assert reconstructed == lines
    return chunk_ranges


def test_pre_code_extracts_acceptance_criteria_and_numbered_slices(tmp_path):
    repo_dir = tmp_path / "repo"
    repo_dir.mkdir()
    target = repo_dir / "pkg" / "service.go"
    target.parent.mkdir(parents=True)
    target.write_text(
        "\n".join(
            [
                "package pkg",
                "",
                "func Add(a, b int) int {",
                "\treturn a + b",
                "}",
                "",
                "func Sub(a, b int) int {",
                "\treturn a - b",
                "}",
            ]
        )
        + "\n",
        encoding="utf-8",
    )
    plan_file = tmp_path / "PLAN.md"
    plan_file.write_text(
        "\n".join(
            [
                "# Feature Plan",
                "",
                "## Acceptance Criteria & Success Metrics (Required Gate)",
                "",
                "| ID | Target | Entry |",
                "|----|--------|-------|",
                "| AC-1 | Fast sum | `pkg/service.go:Add` |",
                "",
                "## Affected Callers",
                "- None",
                "---",
                "### Task 1",
            ]
        )
        + "\n",
        encoding="utf-8",
    )
    out_file = tmp_path / "context.md"
    res = run_cli(
        "pre-code",
        "--output",
        str(out_file),
        "--plan",
        str(plan_file),
        "--root",
        str(repo_dir),
        "--slice",
        "pkg/service.go:3-5",
        "--gate-cmd",
        "go test ./pkg/...",
    )
    assert res.returncode == 0, res.stderr
    assert stat.S_IMODE(out_file.stat().st_mode) == 0o600
    rendered = out_file.read_text(encoding="utf-8")
    assert "# Subagent Context (Pre-Code)" in rendered
    assert f"- Plan: {plan_file}" in rendered
    assert "go test ./pkg/..." in rendered
    assert "| AC-1 | Fast sum | `pkg/service.go:Add` |" in rendered
    assert "## Affected Callers" not in rendered
    assert "### pkg/service.go:3-5" in rendered
    assert "3: func Add(a, b int) int {" in rendered
    assert "4: \treturn a + b" in rendered
    assert "5: }" in rendered
    assert "7: func Sub" not in rendered
    assert_read_chunk_index_reconstructs_exact_file(rendered, 700, 38000)


def test_chunk_byte_cap_enforced_on_700_lines_of_200_byte_prose(tmp_path):
    repo_dir = tmp_path / "repo"
    repo_dir.mkdir()
    base_sha = init_git_repo(repo_dir)

    prose_line = "word_token_" * 18 + "end_marker"
    assert len(prose_line.encode("utf-8")) >= 200
    heavy_doc = repo_dir / "docs" / "heavy.md"
    heavy_doc.parent.mkdir(parents=True, exist_ok=True)
    heavy_doc.write_text(
        "\n".join(f"{idx:03d}: {prose_line}" for idx in range(1, 701)) + "\n",
        encoding="utf-8",
    )
    subprocess.run(["git", "add", "."], cwd=repo_dir, check=True)
    subprocess.run(["git", "commit", "-q", "-m", "feat: heavy prose"], cwd=repo_dir, check=True)
    head_sha = subprocess.run(
        ["git", "rev-parse", "HEAD"],
        cwd=repo_dir,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()

    plan_file = tmp_path / "PLAN.md"
    plan_file.write_text(
        "\n".join(
            [
                "# Byte Cap Plan",
                "",
                "## Acceptance Criteria",
                *[f"- AC-{i}: {prose_line}" for i in range(1, 25)],
            ]
        )
        + "\n",
        encoding="utf-8",
    )
    out_file = tmp_path / "byte-capped-context.md"
    pre = run_cli(
        "pre-code",
        "--output",
        str(out_file),
        "--plan",
        str(plan_file),
        "--root",
        str(repo_dir),
        "--slice",
        "docs/heavy.md:1-700",
    )
    assert pre.returncode == 0, pre.stderr
    pre_rendered = out_file.read_text(encoding="utf-8")
    pre_chunks = assert_read_chunk_index_reconstructs_exact_file(pre_rendered, 700, 38000)
    assert len(pre_chunks) >= 4

    post = run_cli(
        "post-code",
        "--output",
        str(out_file),
        "--base",
        base_sha,
        "--head",
        head_sha,
        "--share-root",
        str(repo_dir),
    )
    assert post.returncode == 0, post.stderr
    post_rendered = out_file.read_text(encoding="utf-8")
    post_chunks = assert_read_chunk_index_reconstructs_exact_file(post_rendered, 700, 38000)
    assert len(post_chunks) >= 8


def test_pre_code_rejects_traversal_outside_root_and_symlink_escape(tmp_path):
    repo_dir = tmp_path / "repo"
    (repo_dir / "pkg").mkdir(parents=True)
    (repo_dir / "pkg" / "ok.py").write_text("x = 1\n", encoding="utf-8")
    secret_outside = tmp_path / "outside_secret.txt"
    secret_outside.write_text("TOP_SECRET_TOKEN=123\nsecond_line=456\n", encoding="utf-8")
    escape_link = repo_dir / "pkg" / "escape_link.txt"
    escape_link.symlink_to(secret_outside)

    plan_file = tmp_path / "PLAN.md"
    plan_file.write_text("# Plan\n", encoding="utf-8")
    out_file = tmp_path / "out.md"

    for bad_spec in [
        "../outside_secret.txt:1-2",
        f"{secret_outside}:1-2",
        "pkg/escape_link.txt:1-2",
    ]:
        res = run_cli(
            "pre-code",
            "--output",
            str(out_file),
            "--plan",
            str(plan_file),
            "--root",
            str(repo_dir),
            "--slice",
            bad_spec,
        )
        assert res.returncode != 0, f"expected failure for {bad_spec}"
        assert not out_file.exists()


def test_output_rejects_symlink_and_writes_0600_mode(tmp_path):
    repo_dir = tmp_path / "repo"
    repo_dir.mkdir()
    base_sha = init_git_repo(repo_dir)
    plan_file = tmp_path / "PLAN.md"
    plan_file.write_text("# Plan\n", encoding="utf-8")

    victim = tmp_path / "victim.txt"
    victim.write_text("DO_NOT_CLOBBER\n", encoding="utf-8")
    symlink_out = tmp_path / "symlink_context.md"
    symlink_out.symlink_to(victim)

    pre_sym = run_cli(
        "pre-code",
        "--output",
        str(symlink_out),
        "--plan",
        str(plan_file),
        "--root",
        str(repo_dir),
    )
    assert pre_sym.returncode != 0
    assert victim.read_text(encoding="utf-8") == "DO_NOT_CLOBBER\n"

    post_sym = run_cli(
        "post-code",
        "--output",
        str(symlink_out),
        "--base",
        base_sha,
        "--share-root",
        str(repo_dir),
    )
    assert post_sym.returncode != 0
    assert victim.read_text(encoding="utf-8") == "DO_NOT_CLOBBER\n"

    real_out = tmp_path / "private_context.md"
    ok_post = run_cli(
        "post-code",
        "--output",
        str(real_out),
        "--base",
        base_sha,
        "--share-root",
        str(repo_dir),
    )
    assert ok_post.returncode == 0, ok_post.stderr
    assert stat.S_IMODE(real_out.stat().st_mode) == 0o600


def test_dynamic_backtick_fences_and_canonical_git_diff_flags(tmp_path):
    repo_dir = tmp_path / "repo"
    repo_dir.mkdir()
    base_sha = init_git_repo(repo_dir)

    subprocess.run(["git", "config", "diff.noprefix", "true"], cwd=repo_dir, check=True)
    ext_marker = tmp_path / "ext_diff_ran"
    subprocess.run(
        ["git", "config", "diff.external", f"sh -c 'touch {ext_marker}'"],
        cwd=repo_dir,
        check=True,
    )

    tricky = repo_dir / "src" / "tricky.md"
    tricky.write_text(
        "\n".join(
            [
                "Example nested fences:",
                "```python",
                "print('triple')",
                "```",
                "````markdown",
                "quad fence line",
                "````",
            ]
        )
        + "\n",
        encoding="utf-8",
    )
    gate_log = tmp_path / "gate.log"
    gate_log.write_text("gate output:\n```\nfenced log\n```\nexit=0\n", encoding="utf-8")
    plan_file = tmp_path / "PLAN.md"
    plan_file.write_text("# Plan\n", encoding="utf-8")
    out_file = tmp_path / "fenced.md"

    pre = run_cli(
        "pre-code",
        "--output",
        str(out_file),
        "--plan",
        str(plan_file),
        "--root",
        str(repo_dir),
        "--slice",
        "src/tricky.md:1-7",
    )
    assert pre.returncode == 0, pre.stderr

    subprocess.run(["git", "add", "."], cwd=repo_dir, check=True)
    post = run_cli(
        "post-code",
        "--output",
        str(out_file),
        "--base",
        base_sha,
        "--share-root",
        str(repo_dir),
        "--gate-log",
        str(gate_log),
    )
    assert post.returncode == 0, post.stderr
    assert not ext_marker.exists()
    rendered = out_file.read_text(encoding="utf-8")
    assert "`````text" in rendered
    assert "````text" in rendered
    assert "`````diff" in rendered
    assert "diff --git a/src/tricky.md b/src/tricky.md" in rendered
    assert_read_chunk_index_reconstructs_exact_file(rendered, 700, 38000)


@pytest.mark.parametrize(
    "bad_slice",
    [
        "missing.go:1-5",
        "pkg/service.go:0-2",
        "pkg/service.go:5-3",
        "pkg/service.go:1-99",
        "pkg/service.go:bad",
        "no_range_colon",
    ],
)
def test_pre_code_fails_on_missing_file_or_invalid_slice(tmp_path, bad_slice):
    repo_dir = tmp_path / "repo"
    (repo_dir / "pkg").mkdir(parents=True)
    (repo_dir / "pkg" / "service.go").write_text("package pkg\n\nfunc Ok() {}\n", encoding="utf-8")
    plan_file = tmp_path / "PLAN.md"
    plan_file.write_text("# Plan\n", encoding="utf-8")
    out_file = tmp_path / "context.md"
    res = run_cli(
        "pre-code",
        "--output",
        str(out_file),
        "--plan",
        str(plan_file),
        "--root",
        str(repo_dir),
        "--slice",
        bad_slice,
    )
    assert res.returncode != 0
    assert "error:" in res.stderr.lower()


def test_pre_code_fails_when_plan_missing(tmp_path):
    out_file = tmp_path / "context.md"
    res = run_cli(
        "pre-code",
        "--output",
        str(out_file),
        "--plan",
        str(tmp_path / "nonexistent-plan.md"),
    )
    assert res.returncode != 0
    assert "error:" in res.stderr.lower()


def test_post_code_uncommitted_working_tree_when_head_omitted(tmp_path):
    repo_dir = tmp_path / "repo"
    repo_dir.mkdir()
    base_sha = init_git_repo(repo_dir)

    alpha = repo_dir / "src" / "alpha.py"
    alpha.write_text(
        alpha.read_text(encoding="utf-8") + "def uncommitted_helper(): return 42\n",
        encoding="utf-8",
    )
    gate_log = tmp_path / "gate.log"
    gate_log.write_text(
        "\n".join(f"log line {i}" for i in range(1, 61)) + "\nPASS exit=0\n",
        encoding="utf-8",
    )
    out_file = tmp_path / "context.md"
    plan_file = tmp_path / "PLAN.md"
    plan_file.write_text("# Plan\n\n## Acceptance Criteria\n- AC-1: helper works\n", encoding="utf-8")

    pre = run_cli(
        "pre-code",
        "--output",
        str(out_file),
        "--plan",
        str(plan_file),
        "--root",
        str(repo_dir),
        "--slice",
        "src/alpha.py:1-5",
    )
    assert pre.returncode == 0, pre.stderr

    post = run_cli(
        "post-code",
        "--output",
        str(out_file),
        "--base",
        base_sha,
        "--share-root",
        str(repo_dir),
        "--gate-log",
        str(gate_log),
    )
    assert post.returncode == 0, post.stderr
    rendered = out_file.read_text(encoding="utf-8")
    assert "# Subagent Context (Pre-Code)" in rendered
    assert "### src/alpha.py:1-5" in rendered
    assert "+def uncommitted_helper(): return 42" in rendered
    assert "PASS exit=0" in rendered
    assert "log line 1\n" not in rendered
    assert_read_chunk_index_reconstructs_exact_file(rendered, 700, 38000)

    rerun = run_cli(
        "post-code",
        "--output",
        str(out_file),
        "--base",
        base_sha,
        "--share-root",
        str(repo_dir),
        "--gate-log",
        str(gate_log),
    )
    assert rerun.returncode == 0, rerun.stderr
    assert out_file.read_text(encoding="utf-8") == rendered


def test_post_code_three_dot_head_diff_and_multi_chunk_two_pass_offsets(tmp_path):
    repo_dir = tmp_path / "repo"
    repo_dir.mkdir()
    base_sha = init_git_repo(repo_dir)

    subprocess.run(["git", "checkout", "-q", "-b", "feat/large-diff"], cwd=repo_dir, check=True)
    for file_idx in range(1, 7):
        mod_file = repo_dir / "src" / f"module_{file_idx}.py"
        mod_file.write_text(
            "\n".join(f"def generated_{file_idx}_{line_idx}(): return {line_idx}" for line_idx in range(1, 185))
            + "\n",
            encoding="utf-8",
        )
    subprocess.run(["git", "add", "."], cwd=repo_dir, check=True)
    subprocess.run(["git", "commit", "-q", "-m", "feat: add six modules"], cwd=repo_dir, check=True)
    head_sha = subprocess.run(
        ["git", "rev-parse", "HEAD"],
        cwd=repo_dir,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()

    (repo_dir / "src" / "module_1.py").write_text("dirty_uncommitted = True\n", encoding="utf-8")

    out_file = tmp_path / "review-context.md"
    res = run_cli(
        "post-code",
        "--output",
        str(out_file),
        "--base",
        base_sha,
        "--head",
        head_sha,
        "--share-root",
        str(repo_dir),
        "--max-chunk-lines",
        "320",
    )
    assert res.returncode == 0, res.stderr
    rendered = out_file.read_text(encoding="utf-8")
    total_lines = len(rendered.splitlines())
    assert total_lines > 900
    assert "dirty_uncommitted" not in rendered
    assert "### Diff Part 1" in rendered
    assert "### Diff Part 2" in rendered
    assert_read_chunk_index_reconstructs_exact_file(rendered, 320, 38000)


def test_post_code_fails_on_invalid_git_ref(tmp_path):
    repo_dir = tmp_path / "repo"
    repo_dir.mkdir()
    init_git_repo(repo_dir)
    out_file = tmp_path / "out.md"
    res = run_cli(
        "post-code",
        "--output",
        str(out_file),
        "--base",
        "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
        "--share-root",
        str(repo_dir),
    )
    assert res.returncode != 0
    assert res.stderr.strip() != ""


def test_zero_comments_in_script_and_test_files():
    mod = load_module()
    assert callable(mod.main)
    assert os.access(SCRIPT_PATH, os.X_OK)
    for idx, py_file in enumerate([SCRIPT_PATH, Path(__file__).resolve()]):
        raw_lines = py_file.read_text(encoding="utf-8").splitlines()
        if idx == 0:
            assert raw_lines[0] == "#!/usr/bin/env python3"
        with py_file.open("rb") as fh:
            for tok in tokenize.tokenize(fh.readline):
                if tok.type == tokenize.COMMENT:
                    if idx == 0 and tok.start[0] == 1 and tok.string == "#!/usr/bin/env python3":
                        continue
                    pytest.fail(f"Forbidden comment in {py_file.name}:{tok.start[0]}: {tok.string}")
        for line_no, line in enumerate(raw_lines, start=1):
            if idx == 0 and line_no == 1 and line == "#!/usr/bin/env python3":
                continue
            assert not line.lstrip().startswith("#"), f"Line {line_no} in {py_file.name} starts with #"


def _write_skill(skills_root: Path, name: str, text: str = "skill body\n") -> Path:
    md = skills_root / name / "SKILL.md"
    md.parent.mkdir(parents=True, exist_ok=True)
    md.write_text(text, encoding="utf-8")
    return md


def test_pre_code_manifest_happy_path_touch_skills_dir_docs_and_post_code_preserves(tmp_path):
    repo_dir = tmp_path / "repo"
    repo_dir.mkdir()
    base_sha = init_git_repo(repo_dir)

    (repo_dir / "pkg").mkdir(parents=True, exist_ok=True)
    (repo_dir / "pkg" / "handler.go").write_text("package pkg\n\nfunc Handle() {}\n", encoding="utf-8")
    (repo_dir / "docs").mkdir(parents=True, exist_ok=True)
    prd = repo_dir / "docs" / "prd.md"
    prd.write_text("PRD\n", encoding="utf-8")
    pov = repo_dir / "docs" / "point-of-view" / "app.md"
    pov.parent.mkdir(parents=True, exist_ok=True)
    pov.write_text("POV\n", encoding="utf-8")
    dec = repo_dir / "docs" / "decisions" / "product-scope.md"
    dec.parent.mkdir(parents=True, exist_ok=True)
    dec.write_text("Scope\n", encoding="utf-8")
    approved = repo_dir / "design-demos" / "mobile" / "direction-approved.md"
    approved.parent.mkdir(parents=True, exist_ok=True)
    approved.write_text("Approved\n", encoding="utf-8")

    skills_dir = tmp_path / "skills_bundle"
    skills_dir.mkdir(parents=True, exist_ok=True)
    for skill_name in (
        "go-api-structure",
        "impeccable",
        "no-slop-ui",
        "animation-principles",
        "sql-optimization-patterns",
        "test-driven-development",
        "javascript-testing-patterns",
    ):
        _write_skill(skills_dir, skill_name)

    plan_file = tmp_path / "PLAN.md"
    plan_file.write_text(
        "\n".join(
            [
                "# Feature Plan",
                "",
                "## Acceptance Criteria & Success Metrics (Required Gate)",
                "| ID | Goal | Entry |",
                "|----|------|-------|",
                "| AC-1 | Go + UI + SQL | `pkg/handler.go:Handle` |",
            ]
        )
        + "\n",
        encoding="utf-8",
    )
    out_file = tmp_path / "context.md"

    pre = run_cli(
        "pre-code",
        "--context",
        str(out_file),
        "--plan",
        str(plan_file),
        "--root",
        str(repo_dir),
        "--slice",
        "pkg/handler.go:1-3",
        "--touch",
        "mobile/src/MotionCard.tsx",
        "--touch",
        "db/migrate/001_init.sql",
        "--skills-dir",
        str(skills_dir),
    )
    assert pre.returncode == 0, pre.stderr
    rendered = out_file.read_text(encoding="utf-8")
    assert "## Turn-1 Parallel Read Manifest" in rendered
    assert rendered.count("- Plan:") == 1
    assert f"- Plan: {plan_file}" in rendered

    for expected_doc in (prd.resolve(), dec.resolve(), pov.resolve(), approved.resolve()):
        assert f"- Doc: {expected_doc.as_posix()}" in rendered
        assert f"- {expected_doc.as_posix()}" in pre.stdout

    for expected_skill in (
        "go-api-structure",
        "impeccable",
        "no-slop-ui",
        "animation-principles",
        "sql-optimization-patterns",
        "test-driven-development",
    ):
        resolved_md = (skills_dir / expected_skill / "SKILL.md").resolve()
        assert f"- Skill ({expected_skill}): {resolved_md.as_posix()}" in rendered
        assert f"- {resolved_md.as_posix()}" in pre.stdout

    assert "javascript-testing-patterns" not in rendered
    assert "javascript-testing-patterns" not in pre.stdout
    assert "TURN1_READ_MANIFEST:" in pre.stdout
    assert f"- {plan_file.resolve().as_posix()}" in pre.stdout
    assert_read_chunk_index_reconstructs_exact_file(rendered, 700, 38000)

    (repo_dir / "src" / "alpha.py").write_text("def fn_1(): return 99\n", encoding="utf-8")
    post = run_cli(
        "post-code",
        "--context",
        str(out_file),
        "--base",
        base_sha,
        "--share-root",
        str(repo_dir),
    )
    assert post.returncode == 0, post.stderr
    post_rendered = out_file.read_text(encoding="utf-8")
    assert "## Turn-1 Parallel Read Manifest" in post_rendered
    assert post_rendered.count("- Plan:") == 1
    assert f"- Doc: {prd.resolve().as_posix()}" in post_rendered
    assert_read_chunk_index_reconstructs_exact_file(post_rendered, 700, 38000)


def test_pre_code_explicit_skill_tokens_prose_backticks_and_touch_validation(tmp_path):
    repo_dir = tmp_path / "repo"
    (repo_dir / "cmd").mkdir(parents=True)
    (repo_dir / "cmd" / "main.go").write_text("package main\n", encoding="utf-8")
    outside_file = tmp_path / "outside.go"
    outside_file.write_text("package outside\n", encoding="utf-8")

    skills_dir = tmp_path / "skills"
    custom_md = _write_skill(skills_dir, "custom-local-skill")
    start_md = _write_skill(skills_dir, "start")
    begin_md = _write_skill(skills_dir, "begin")

    plan_file = tmp_path / "PLAN.md"
    plan_file.write_text(
        "\n".join(
            [
                "# Plan",
                "Use `custom-local-skill`, ignore wrapper `begin`, include explicit `start/SKILL.md`.",
                "Reject traversal like `../outside.go` in `--touch`, and touch `cmd/main.go:1-1`.",
            ]
        )
        + "\n",
        encoding="utf-8",
    )
    out_file = tmp_path / "ctx.md"
    ok = run_cli(
        "pre-code",
        "--output",
        str(out_file),
        "--plan",
        str(plan_file),
        "--root",
        str(repo_dir),
        "--skills-dir",
        str(skills_dir),
    )
    assert ok.returncode == 0, ok.stderr
    rendered = out_file.read_text(encoding="utf-8")
    assert f"- Skill (custom-local-skill): {custom_md.resolve().as_posix()}" in rendered
    assert f"- Skill (start): {start_md.resolve().as_posix()}" in rendered
    assert str(begin_md.resolve().as_posix()) not in rendered
    assert "outside.go" not in rendered

    for bad_touch in ("../outside.go", str(outside_file), ""):
        bad_out = tmp_path / "bad_ctx.md"
        bad = run_cli(
            "pre-code",
            "--output",
            str(bad_out),
            "--plan",
            str(plan_file),
            "--root",
            str(repo_dir),
            "--touch",
            bad_touch,
            "--skills-dir",
            str(skills_dir),
        )
        assert bad.returncode != 0
        assert not bad_out.exists()

    missing_skills = run_cli(
        "pre-code",
        "--output",
        str(tmp_path / "missing_skills_ctx.md"),
        "--plan",
        str(plan_file),
        "--root",
        str(repo_dir),
        "--skills-dir",
        str(tmp_path / "does_not_exist_skills"),
    )
    assert missing_skills.returncode != 0


def test_pre_code_default_home_precedence_and_symlink_escape_rejection(tmp_path, isolate_home):
    repo_dir = tmp_path / "repo"
    (repo_dir / "docs").mkdir(parents=True)
    (repo_dir / "pkg").mkdir(parents=True)
    (repo_dir / "pkg" / "svc.go").write_text("package pkg\n", encoding="utf-8")

    outside_secret = tmp_path / "secret.md"
    outside_secret.write_text("SECRET\n", encoding="utf-8")
    (repo_dir / "docs" / "prd.md").symlink_to(outside_secret)

    gemini_skills = isolate_home / ".gemini" / "config" / "skills"
    claude_skills = isolate_home / ".claude" / "skills"
    gem_go = _write_skill(gemini_skills, "go-api-structure", "gemini override\n")
    _write_skill(claude_skills, "go-api-structure", "claude fallback\n")
    claude_tdd = _write_skill(claude_skills, "test-driven-development", "claude tdd\n")

    evil_skill_dir = gemini_skills / "evil-skill"
    evil_skill_dir.mkdir(parents=True, exist_ok=True)
    (evil_skill_dir / "SKILL.md").symlink_to(outside_secret)

    plan_file = tmp_path / "PLAN.md"
    plan_file.write_text("Touch `pkg/svc.go` and `evil-skill`.\n", encoding="utf-8")
    out_file = tmp_path / "home_ctx.md"

    res = run_cli(
        "pre-code",
        "--output",
        str(out_file),
        "--plan",
        str(plan_file),
        "--root",
        str(repo_dir),
    )
    assert res.returncode == 0, res.stderr
    rendered = out_file.read_text(encoding="utf-8")
    assert f"- Skill (go-api-structure): {gem_go.resolve().as_posix()}" in rendered
    assert f"- Skill (test-driven-development): {claude_tdd.resolve().as_posix()}" in rendered
    assert "prd.md" not in rendered
    assert "evil-skill" not in rendered
    assert "secret.md" not in rendered


def test_pre_code_excludes_orchestration_slugs_caps_decisions_and_supports_symlinked_default_skills_dir(
    tmp_path, isolate_home
):
    repo_dir = tmp_path / "repo"
    (repo_dir / "pkg").mkdir(parents=True)
    (repo_dir / "pkg" / "api.go").write_text("package pkg\n", encoding="utf-8")
    decisions_dir = repo_dir / "docs" / "decisions"
    decisions_dir.mkdir(parents=True)
    (decisions_dir / "product-scope.md").write_text("scope\n", encoding="utf-8")
    for idx in range(1, 10):
        (decisions_dir / f"adr-{idx:03d}.md").write_text(f"adr {idx}\n", encoding="utf-8")

    dotfiles_skills = tmp_path / "dotfiles_gemini_skills"
    dotfiles_skills.mkdir(parents=True)
    for meta_slug in (
        "start",
        "start-lite",
        "lifecycle",
        "future-writing-plans",
        "pipeline-swarm",
        "subagent-driven-development",
        "session-orchestration",
        "future-model-router",
        "comprehensive-code-review",
        "baton-handoff",
    ):
        _write_skill(dotfiles_skills, meta_slug)
    go_md = _write_skill(dotfiles_skills, "go-api-structure")
    domain_md = _write_skill(dotfiles_skills, "error-handling-patterns")
    explicit_meta_md = dotfiles_skills / "pipeline-swarm" / "SKILL.md"

    gem_cfg = isolate_home / ".gemini" / "config"
    gem_cfg.mkdir(parents=True, exist_ok=True)
    (gem_cfg / "skills").symlink_to(dotfiles_skills)

    plan_file = tmp_path / "PLAN.md"
    plan_file.write_text(
        "\n".join(
            [
                "# Realistic Plan",
                " Wire `start`, `start-lite`, `lifecycle`, `future-writing-plans`, `session-orchestration`,",
                " `subagent-driven-development`, `future-model-router`, `comprehensive-code-review`, and `baton-handoff`.",
                " Apply `error-handling-patterns`, explicit `pipeline-swarm/SKILL.md`, and `adr-007.md` on `pkg/api.go`.",
            ]
        )
        + "\n",
        encoding="utf-8",
    )
    out_file = tmp_path / "realistic_ctx.md"
    res = run_cli(
        "pre-code",
        "--output",
        str(out_file),
        "--plan",
        str(plan_file),
        "--root",
        str(repo_dir),
    )
    assert res.returncode == 0, res.stderr
    rendered = out_file.read_text(encoding="utf-8")
    assert f"- Skill (go-api-structure): {go_md.resolve().as_posix()}" in rendered
    assert f"- Skill (error-handling-patterns): {domain_md.resolve().as_posix()}" in rendered
    assert f"- Skill (pipeline-swarm): {explicit_meta_md.resolve().as_posix()}" in rendered

    for excluded_meta in (
        "start-lite",
        "lifecycle",
        "future-writing-plans",
        "session-orchestration",
        "subagent-driven-development",
        "future-model-router",
        "comprehensive-code-review",
        "baton-handoff",
    ):
        assert f"- Skill ({excluded_meta}):" not in rendered

    doc_lines = [ln for ln in rendered.splitlines() if ln.startswith("- Doc: ")]
    assert len(doc_lines) == 4
    assert any("product-scope.md" in ln for ln in doc_lines)
    assert any("adr-007.md" in ln for ln in doc_lines)
