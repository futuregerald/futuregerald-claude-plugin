import re
import subprocess
import sys
from pathlib import Path

import check_plan_page

SCRIPT = Path(__file__).resolve().parent / "check_plan_page.py"

VALID = """<title>Delivery Alerts</title>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans&display=swap">
<style>
:root { --ground: #f6f8f9; --ink: #16202a; }
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) { color-scheme: dark; --ground: #10161b; }
}
:root[data-theme="dark"] { color-scheme: dark; --ground: #10161b; }
body { background: var(--ground); color: var(--ink); }
</style>
<div class="wrap">
<div class="diagram"><svg viewBox="0 0 10 10"><rect style="fill: var(--surface)"></rect></svg></div>
<div class="diagram erd"><pre class="mermaid">
erDiagram
  parcels ||--o{ delivery_alerts : "alerts"
</pre></div>
</div>
"""


def problems_mentioning(text, word):
    return [p for p in check_plan_page.check(text) if word.lower() in p.lower()]


def test_valid_page_has_no_problems():
    assert check_plan_page.check(VALID) == []


def test_missing_title():
    text = VALID.replace("<title>Delivery Alerts</title>", "")
    assert problems_mentioning(text, "title")


def test_title_after_first_8kb():
    text = VALID.replace("<title>Delivery Alerts</title>", "") + " " * 9000 + "<title>Late</title>"
    assert problems_mentioning(text, "title")


def test_skeleton_tags():
    text = "<!doctype html><html><body>" + VALID + "</body></html>"
    assert problems_mentioning(text, "fragment")


def test_placeholder_left_in():
    text = VALID.replace("Delivery Alerts</title>", "{{TITLE}}</title>")
    assert problems_mentioning(text, "{{TITLE}}")


def test_filler_left_in():
    assert problems_mentioning(VALID + "<p>TODO</p>", "filler")
    assert problems_mentioning(VALID + "<p>Lorem ipsum dolor</p>", "filler")


def test_ordinary_word_todo_is_not_filler():
    assert check_plan_page.check(VALID + "<p>Add a todo to the inbox. Todo items sync hourly.</p>") == []


def test_filled_template_passes():
    template = (Path(__file__).resolve().parent.parent / "assets" / "base.html").read_text(encoding="utf-8")
    filled = re.sub(r"\{\{[A-Z_]+\}\}", "Delivery Alerts", template)
    assert check_plan_page.check(filled) == []


def test_missing_dark_palette():
    text = VALID.replace(':root[data-theme="dark"] { color-scheme: dark; --ground: #10161b; }', "")
    assert problems_mentioning(text, "data-theme")


def test_missing_body_background():
    text = VALID.replace("body { background: var(--ground); color: var(--ink); }", "body { color: var(--ink); }")
    assert problems_mentioning(text, "body background")


def test_blocked_script_host():
    text = VALID + '<script src="https://unpkg.com/lib.js"></script>'
    assert problems_mentioning(text, "unpkg.com")


def test_blocked_script_host_single_quoted():
    text = VALID + "<script src='https://unpkg.com/lib.js'></script>"
    assert problems_mentioning(text, "unpkg.com")


def test_allowed_script_host():
    text = VALID + '<script src="https://cdnjs.cloudflare.com/ajax/libs/d3/7.9.0/d3.min.js"></script>'
    assert check_plan_page.check(text) == []


def test_mermaid_script_in_fragment():
    text = VALID + '<script src="https://cdn.jsdelivr.net/npm/mermaid@11.4.1/dist/mermaid.min.js"></script>'
    assert problems_mentioning(text, "draws")


def test_blocked_stylesheet_host():
    text = VALID + '<link rel="stylesheet" href="https://example.com/site.css">'
    assert problems_mentioning(text, "example.com")


def test_empty_mermaid_block():
    text = VALID + '<pre class="mermaid">   </pre>'
    assert problems_mentioning(text, "empty")


def test_hex_colour_in_svg():
    text = VALID.replace("fill: var(--surface)", "fill: #ffffff")
    assert problems_mentioning(text, "hex")


def test_hex_attribute_in_svg():
    text = VALID.replace('style="fill: var(--surface)"', 'fill="#fff"')
    assert problems_mentioning(text, "hex")


def test_single_quoted_hex_attribute_in_svg():
    text = VALID.replace('style="fill: var(--surface)"', "fill='#fff'")
    assert problems_mentioning(text, "hex")


def test_blocked_stylesheet_single_quoted_href_first():
    text = VALID + "<link href='https://example.com/site.css' rel='stylesheet'>"
    assert problems_mentioning(text, "example.com")


def run_cli(*args, stdin=None):
    return subprocess.run([sys.executable, str(SCRIPT), *args], input=stdin, capture_output=True, text=True)


def test_cli_exit_zero_when_valid(tmp_path):
    page = tmp_path / "page.html"
    page.write_text(VALID, encoding="utf-8")
    assert run_cli(str(page)).returncode == 0


def test_cli_exit_one_with_problems():
    result = run_cli("-", stdin="<p>no title</p>")
    assert result.returncode == 1
    assert "title" in result.stdout


def test_cli_exit_two_on_usage_error():
    assert run_cli().returncode == 2
    assert run_cli("/no/such/file.html").returncode == 2
