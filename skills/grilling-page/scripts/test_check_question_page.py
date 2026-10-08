import json
import re
import subprocess
import sys
from pathlib import Path

import check_question_page

SCRIPT = Path(__file__).resolve().parent / "check_question_page.py"
TEMPLATE = Path(__file__).resolve().parent.parent / "assets" / "question-page.html"
FILLED = {
    "{{TITLE}}": "Recipe App Decisions",
    "{{LABEL}}": "Recipe app · round 1",
    "{{HEADING}}": "Three decisions before the plan",
    "{{INTRO}}": "Decisions D1 to D3 are settled in <code>docs/decisions/</code>.",
    "{{STORE_KEY}}": "recipe-app-grilling-round-1",
    "{{ANSWER_HEADING}}": "Grilling round 1 answers (recipe app)",
}
DATA = re.compile(r"(<script type=\"application/json\" id=\"page-data\">)(.*?)(</script>)", re.DOTALL)


def valid_page():
    text = TEMPLATE.read_text(encoding="utf-8")
    for placeholder, value in FILLED.items():
        text = text.replace(placeholder, value)
    return text


def page_data(text):
    return json.loads(DATA.search(text).group(2))


def with_data(text, data):
    return DATA.sub(lambda m: m.group(1) + json.dumps(data) + m.group(3), text, count=1)


def edited(change):
    text = valid_page()
    data = page_data(text)
    change(data)
    return with_data(text, data)


def problems_mentioning(text, word):
    return [p for p in check_question_page.check(text) if word.lower() in p.lower()]


def test_filled_template_has_no_problems():
    assert check_question_page.check(valid_page()) == []


def test_unfilled_template_reports_placeholders():
    text = TEMPLATE.read_text(encoding="utf-8")
    assert problems_mentioning(text, "{{STORE_KEY}}")
    assert problems_mentioning(text, "{{TITLE}}")


def test_missing_title():
    text = re.sub(r"<title>.*?</title>", "", valid_page())
    assert problems_mentioning(text, "title")


def test_skeleton_tags():
    text = "<!doctype html><html><body>" + valid_page() + "</body></html>"
    assert problems_mentioning(text, "fragment")


def test_filler_left_in():
    assert problems_mentioning(valid_page() + "<p>TODO</p>", "filler")


def test_missing_dark_palette():
    text = valid_page().replace(':root[data-theme="dark"] {', ":root.other {")
    assert problems_mentioning(text, "data-theme")


def test_missing_data_block():
    text = DATA.sub("", valid_page())
    assert problems_mentioning(text, "page-data")


def test_invalid_json():
    text = DATA.sub(lambda m: m.group(1) + "{ not json" + m.group(3), valid_page())
    assert problems_mentioning(text, "valid JSON")


def test_missing_store_key():
    assert problems_mentioning(edited(lambda d: d.update(storeKey="")), "storeKey")


def test_no_questions():
    assert problems_mentioning(edited(lambda d: d.update(questions=[])), "no questions")


def test_repeated_id():
    def change(data):
        data["questions"][1]["id"] = data["questions"][0]["id"]

    assert problems_mentioning(edited(change), "used twice")


def test_missing_id():
    assert problems_mentioning(edited(lambda d: d["questions"][0].pop("id")), "no id")


def test_title_not_a_question():
    def change(data):
        data["questions"][0]["title"] = "Recipe visibility"

    assert problems_mentioning(edited(change), "ending in ?")


def test_title_too_long():
    def change(data):
        data["questions"][0]["title"] = " ".join(["word"] * 17) + "?"

    assert problems_mentioning(edited(change), "title is 17 words")


def test_missing_context_and_why():
    def change(data):
        data["questions"][0]["context"] = ""
        data["questions"][0].pop("why")

    text = edited(change)
    assert problems_mentioning(text, "no context")
    assert problems_mentioning(text, "no why")


def test_context_too_long():
    def change(data):
        data["questions"][0]["context"] = " ".join(["word"] * 131)

    assert problems_mentioning(edited(change), "context is 131 words")


def test_tags_do_not_count_as_words():
    def change(data):
        data["questions"][0]["why"] = " ".join(["<code>x</code>"] * 50)

    assert check_question_page.check(edited(change)) == []


def test_disallowed_markup():
    def change(data):
        data["questions"][0]["context"] += ' <a href="https://example.com">link</a>'

    assert problems_mentioning(edited(change), "<a>")


def test_bad_type():
    assert problems_mentioning(edited(lambda d: d["questions"][0].update(type="select")), "radio or checkbox")


def test_one_option():
    def change(data):
        data["questions"][0]["options"] = data["questions"][0]["options"][:1]

    assert problems_mentioning(edited(change), "at least two options")


def test_repeated_option_key():
    def change(data):
        data["questions"][0]["options"][1]["key"] = "A"

    assert problems_mentioning(edited(change), "keys repeat")


def test_option_without_desc():
    def change(data):
        data["questions"][0]["options"][2]["desc"] = ""

    assert problems_mentioning(edited(change), "no desc")


def test_option_desc_too_long():
    def change(data):
        data["questions"][0]["options"][1]["desc"] = " ".join(["word"] * 51)

    assert problems_mentioning(edited(change), "desc is 51 words")


def test_radio_with_two_recommended():
    def change(data):
        data["questions"][0]["options"][1]["rec"] = True

    assert problems_mentioning(edited(change), "exactly one recommended")


def test_checkbox_without_recommended():
    def change(data):
        for option in data["questions"][1]["options"]:
            option.pop("rec", None)

    assert problems_mentioning(edited(change), "no recommended")


def test_recommended_not_first():
    def change(data):
        data["questions"][0]["options"].reverse()

    assert problems_mentioning(edited(change), "listed first")


def run_cli(*args, stdin=""):
    return subprocess.run([sys.executable, str(SCRIPT), *args], input=stdin, capture_output=True, text=True)


def test_cli_exit_codes(tmp_path):
    good = tmp_path / "good.html"
    good.write_text(valid_page(), encoding="utf-8")
    assert run_cli(str(good)).returncode == 0
    assert run_cli("-", stdin=valid_page()).returncode == 0
    bad = run_cli(str(TEMPLATE))
    assert bad.returncode == 1
    assert "{{TITLE}}" in bad.stdout
    assert run_cli().returncode == 2
    assert run_cli(str(tmp_path / "missing.html")).returncode == 2
