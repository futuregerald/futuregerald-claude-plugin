import subprocess
import sys
from pathlib import Path

import build_standalone

SCRIPT = Path(__file__).resolve().parent / "build_standalone.py"

FRAGMENT = """<title>Delivery Alerts</title>
<style>
body { background: var(--ground); }
</style>
<div class="wrap"><h1>Delivery Alerts</h1></div>
"""

MERMAID = '<div class="diagram erd"><pre class="mermaid">erDiagram\n  a ||--o{ b : "has"\n</pre></div>'


def test_wraps_fragment_in_a_full_document():
    page = build_standalone.build(FRAGMENT)
    assert page.startswith("<!doctype html>")
    assert page.rstrip().endswith("</html>")


def test_title_and_style_go_in_head():
    page = build_standalone.build(FRAGMENT)
    head = page.split("</head>")[0]
    assert "<title>Delivery Alerts</title>" in head
    assert "body { background: var(--ground); }" in head


def test_content_goes_in_body():
    page = build_standalone.build(FRAGMENT)
    body = page.split("<body>")[1]
    assert '<div class="wrap"><h1>Delivery Alerts</h1></div>' in body


def test_no_mermaid_script_without_a_diagram():
    assert "mermaid" not in build_standalone.build(FRAGMENT)


def test_mermaid_script_added_for_a_diagram():
    page = build_standalone.build(FRAGMENT + MERMAID)
    assert build_standalone.MERMAID_SRC in page
    assert "mermaid.initialize" in page
    assert page.index(build_standalone.MERMAID_SRC) > page.index("</pre>")


def test_mermaid_script_added_for_single_quoted_diagram():
    page = build_standalone.build(FRAGMENT + "<pre class='mermaid'>erDiagram\n</pre>")
    assert build_standalone.MERMAID_SRC in page


def test_fragment_without_style_goes_in_body():
    page = build_standalone.build("<p>Only content</p>")
    assert "<p>Only content</p>" in page.split("<body>")[1]


def test_default_output_name():
    assert build_standalone.default_output(Path("/tmp/plan.html")) == Path("/tmp/plan.standalone.html")


def run_cli(*args):
    return subprocess.run([sys.executable, str(SCRIPT), *args], capture_output=True, text=True)


def test_cli_writes_default_output(tmp_path):
    source = tmp_path / "plan.html"
    source.write_text(FRAGMENT, encoding="utf-8")
    result = run_cli(str(source))
    assert result.returncode == 0
    assert (tmp_path / "plan.standalone.html").read_text(encoding="utf-8").startswith("<!doctype html>")


def test_cli_writes_named_output(tmp_path):
    source = tmp_path / "plan.html"
    source.write_text(FRAGMENT, encoding="utf-8")
    target = tmp_path / "out.html"
    assert run_cli(str(source), str(target)).returncode == 0
    assert target.exists()


def test_cli_usage_errors():
    assert run_cli().returncode == 2
    assert run_cli("/no/such/plan.html").returncode == 2
