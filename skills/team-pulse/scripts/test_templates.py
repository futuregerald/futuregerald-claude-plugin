import html
import re
from pathlib import Path

import pytest

SKILL = Path(__file__).resolve().parent.parent
ASSETS = SKILL / "assets"
REPORT_FORMAT = SKILL / "references" / "report-format.md"

ONE_ON_ONE_MD = ASSETS / "template-1on1.md"
ONE_ON_ONE_HTML = ASSETS / "template-1on1.html"
TEAM_HTML = ASSETS / "template.html"
FORECAST_HTML = ASSETS / "forecast.html"

MD_SECTIONS = [
    "🧭 Scorecard (this week)",
    "01. Wins to Recognise",
    "02. Epics",
    "02b. Work Breakdown by Epic",
    "03. Open PRs",
    "04. Talking Points",
    "05. Bottom Line",
]
HTML_SECTIONS = [
    "Scorecard (this week)",
    "Wins to Recognise",
    "Epics",
    "Work Breakdown by Epic",
    "Open PRs",
    "Talking Points",
]
SCORECARD_ROWS = [
    "Overall",
    "PRs this week",
    "Open PRs now",
    "Tickets done this week",
    "Reviews given this week",
]
OPEN_PR_COLUMNS = ["PR", "Opened", "Merged", "Title", "Age", "Status", "Action"]


def read(path):
    return path.read_text(encoding="utf-8")


def css_block(text, selector):
    start = text.index(selector)
    open_brace = text.index("{", start)
    depth = 0
    for i in range(open_brace, len(text)):
        if text[i] == "{":
            depth += 1
        elif text[i] == "}":
            depth -= 1
            if depth == 0:
                return text[start : i + 1]
    raise AssertionError(f"unterminated block for {selector!r}")


def stylesheet(text):
    body = re.search(r"<style>(.*?)</style>", text, re.S).group(1)
    body = re.sub(r"/\*.*?\*/", "", body, flags=re.S)
    return re.sub(r"\s+", " ", body).strip()


def table_header(text, first_column):
    for line in text.splitlines():
        cells = [c.strip() for c in line.strip().strip("|").split("|")]
        if line.startswith("|") and cells[0] == first_column:
            return cells
    raise AssertionError(f"no table whose first column is {first_column!r}")


def test_1on1_md_sections_in_order():
    headings = [l[3:].strip() for l in read(ONE_ON_ONE_MD).splitlines() if l.startswith("## ")]
    assert headings == MD_SECTIONS


def test_1on1_html_sections_in_order():
    raw = re.findall(r"<h2[^>]*>(.*?)</h2>", read(ONE_ON_ONE_HTML), re.S)
    headings = [html.unescape(re.sub(r"<[^>]+>", "", h)).strip() for h in raw]
    assert headings == HTML_SECTIONS


def test_1on1_html_has_bottom_line():
    assert re.search(r'class="bottom-line".*?<h3>Bottom Line</h3>', read(ONE_ON_ONE_HTML), re.S)


def test_1on1_md_scorecard_rows():
    rows = re.findall(r"^\| \*\*([^*]+)\*\* \|", read(ONE_ON_ONE_MD), re.M)
    assert rows == SCORECARD_ROWS


def test_open_pr_tables_have_opened_and_merged():
    assert table_header(read(ONE_ON_ONE_MD), "PR") == OPEN_PR_COLUMNS
    html_columns = re.findall(r"<th>([^<]+)</th>", read(ONE_ON_ONE_HTML))
    assert html_columns == OPEN_PR_COLUMNS


def test_1on1_placeholders_documented_both_ways():
    text = read(REPORT_FORMAT)
    start = text.index("### Bundled 1:1 Templates")
    section = text[start : text.index("\n### ", start + 1)]
    placeholder = r"\{\{[A-Z_]+\}\}"
    used = set(re.findall(placeholder, read(ONE_ON_ONE_MD))) | set(re.findall(placeholder, read(ONE_ON_ONE_HTML)))
    documented = set(re.findall(placeholder, section))
    assert sorted(used - documented) == [], "used in a 1:1 template but not documented"
    assert sorted(documented - used) == [], "documented for 1:1 but used in neither template"


@pytest.mark.parametrize("path", [ONE_ON_ONE_HTML, FORECAST_HTML], ids=lambda p: p.name)
@pytest.mark.parametrize("selector", [":root {", ":root[data-theme=light] {", "@media print {"])
def test_html_templates_share_tokens(path, selector):
    assert css_block(read(path), selector) == css_block(read(TEAM_HTML), selector)


def test_1on1_stylesheet_matches_team():
    assert stylesheet(read(ONE_ON_ONE_HTML)) == stylesheet(read(TEAM_HTML))


def test_1on1_html_offline():
    text = read(ONE_ON_ONE_HTML)
    remote = re.compile(
        r"""<link\b|<script\b[^>]*\bsrc\s*=|@import|<iframe\b"""
        r"""|\bsrc\s*=\s*['"]?(?!data:)|url\(\s*['"]?(?!data:)""",
        re.I,
    )
    assert remote.findall(text) == []


@pytest.mark.parametrize("path", [ONE_ON_ONE_MD, ONE_ON_ONE_HTML], ids=lambda p: p.name)
def test_1on1_templates_have_no_comments(path):
    text = read(path)
    assert "<!--" not in text
    assert "/*" not in text
