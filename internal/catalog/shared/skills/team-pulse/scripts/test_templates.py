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
@pytest.mark.parametrize(
    "selector",
    [":root {", ":root[data-theme=light] {", "@media (prefers-color-scheme: light) {", "@media print {"],
)
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


HTML_TEMPLATES = [TEAM_HTML, ONE_ON_ONE_HTML, FORECAST_HTML]
CHART_EDGE_SELECTORS = {".gauge .band", ".run .band"}
LIGHT_BLOCKS = [":root[data-theme=light] {", "@media (prefers-color-scheme: light) {", "@media print {"]


def css_rules(text):
    body = re.search(r"<style>(.*?)</style>", text, re.S).group(1)
    body = re.sub(r"/\*.*?\*/", "", body, flags=re.S)
    return [(s.strip(), d) for s, d in re.findall(r"([^{}]+)\{([^{}]*)\}", body)]


def token(block, name):
    return re.search(rf"{re.escape(name)}\s*:\s*([^;]+);", block).group(1).strip().upper()


def test_forecast_stylesheet_extends_team():
    assert stylesheet(read(FORECAST_HTML)).startswith(stylesheet(read(TEAM_HTML)))


@pytest.mark.parametrize("path", HTML_TEMPLATES, ids=lambda p: p.name)
def test_html_templates_have_no_inline_styles(path):
    assert not re.search(r"""\bstyle\s*=\s*["']""", read(path))


@pytest.mark.parametrize("path", HTML_TEMPLATES, ids=lambda p: p.name)
def test_no_coloured_bars(path):
    bar = re.compile(r"border(-(left|top|right|bottom|inline-start|inline-end))?(-color)?\s*:[^;]*var\(--(acc|link|warn|crit|good)\)")
    side = re.compile(r"border-left\s*:\s*\d+px\s+solid|box-shadow\s*:\s*inset")
    offenders = [
        selector
        for selector, decl in css_rules(read(path))
        if selector not in CHART_EDGE_SELECTORS and (bar.search(decl) or side.search(decl))
    ]
    assert offenders == []


@pytest.mark.parametrize("path", HTML_TEMPLATES, ids=lambda p: p.name)
def test_single_radius(path):
    values = set(re.findall(r"border-radius\s*:\s*([^;]+);", read(path)))
    assert values <= {"4px", "0"}


@pytest.mark.parametrize("path", HTML_TEMPLATES, ids=lambda p: p.name)
def test_no_gradients(path):
    assert not re.search(r"(linear|radial)-gradient", read(path))


@pytest.mark.parametrize("path", HTML_TEMPLATES, ids=lambda p: p.name)
@pytest.mark.parametrize("selector", LIGHT_BLOCKS)
def test_no_light_cream(path, selector):
    assert token(css_block(read(path), selector), "--ground") == "#FFFFFF"


@pytest.mark.parametrize("path", HTML_TEMPLATES, ids=lambda p: p.name)
@pytest.mark.parametrize("selector", [":root {", *LIGHT_BLOCKS])
def test_status_hues_not_reused_for_accent(path, selector):
    block = css_block(read(path), selector)
    status = {token(block, "--warn"), token(block, "--good"), token(block, "--crit")}
    assert token(block, "--link") not in status
    assert token(block, "--acc") not in status


def custom_properties(block):
    return dict(re.findall(r"(--[\w-]+)\s*:\s*([^;]+);", block))


@pytest.mark.parametrize("path", HTML_TEMPLATES, ids=lambda p: p.name)
def test_light_blocks_agree(path):
    text = read(path)
    first, *rest = [custom_properties(css_block(text, s)) for s in LIGHT_BLOCKS]
    for other in rest:
        assert other == first


MONO_SELECTORS = {".key", ".bd-wrap code", ".mono"}


@pytest.mark.parametrize("path", HTML_TEMPLATES, ids=lambda p: p.name)
def test_monospace_only_for_keys_and_code(path):
    users = {selector for selector, decl in css_rules(read(path)) if "var(--mono)" in decl}
    assert users <= MONO_SELECTORS
