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


def run_cli(*args, cwd=None):
    return subprocess.run(
        [sys.executable, str(SCRIPT_PATH), *args],
        cwd=cwd,
        capture_output=True,
        text=True,
    )


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
