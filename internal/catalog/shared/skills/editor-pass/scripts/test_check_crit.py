import subprocess
import sys
from pathlib import Path

import pytest

import check_crit

SCRIPT = Path(__file__).resolve().parent / "check_crit.py"

VALID = """**Overall:** The page reads as a generic SaaS template, not a restaurant.

A. [Hero headline] Reads like a software tagline → name the dish and the city
B. [Hero photo] Stock gradient where food should be → same treatment as A, lead with the grill
"""


def problems_mentioning(text, word):
    return [p for p in check_crit.check_crit(text) if word.lower() in p.lower()]


def test_valid_sheet_has_no_problems():
    assert check_crit.check_crit(VALID) == []


def test_missing_overall():
    text = VALID.replace("**Overall:**", "**Summary:**")
    assert len(check_crit.check_crit(text)) == 1
    assert problems_mentioning(text, "overall")


def test_overall_with_no_items():
    text = "**Overall:** Fine structure, nothing pinned yet.\n"
    assert len(check_crit.check_crit(text)) == 1
    assert problems_mentioning(text, "no lettered items")


def test_gap_in_letters():
    text = VALID.replace("B. [Hero photo]", "C. [Hero photo]")
    found = check_crit.check_crit(text)
    assert len(found) == 1
    assert "B" in found[0]


def test_duplicate_letter():
    text = VALID.replace("B. [Hero photo]", "A. [Hero photo]")
    assert len(check_crit.check_crit(text)) == 1
    assert problems_mentioning(text, "duplicate")


def test_empty_pin():
    text = "**Overall:** x\n\nA. [] edges too sharp → cloudier\n"
    assert len(check_crit.check_crit(text)) == 1
    assert problems_mentioning(text, "pin")


def test_dangling_cross_reference():
    text = "**Overall:** x\n\nA. [Edge] too sharp → cloudier\nB. [Bubble] same as Z → smaller\n"
    found = check_crit.check_crit(text)
    assert len(found) == 1
    assert "Z" in found[0]


def test_capitalised_cross_reference_is_checked():
    text = "**Overall:** x\n\nA. [Edge] too sharp → cloudier\nB. [Cube] Same note as Z → frost it\n"
    found = check_crit.check_crit(text)
    assert len(found) == 1
    assert "Z" in found[0]


def test_lowercase_letter_after_verb_is_not_a_reference():
    text = "**Overall:** x\n\nA. [Wall] see a crack near the base → fill it\n"
    assert check_crit.check_crit(text) == []


@pytest.mark.parametrize("score", ["8/10", "Score: 7", "a 9 / 10 page"])
def test_scores_rejected(score):
    text = VALID.replace("Reads like a software tagline", f"Reads like a software tagline, {score}")
    assert len(check_crit.check_crit(text)) == 1
    assert problems_mentioning(text, "score")


@pytest.mark.parametrize("praise", ["Looks great", "looks good", "Nice work", "great job", "love it"])
def test_praise_rejected(praise):
    text = VALID.replace("**Overall:** The page", f"**Overall:** {praise}. The page")
    assert len(check_crit.check_crit(text)) == 1
    assert problems_mentioning(text, "praise")


def test_item_without_direction():
    text = "**Overall:** x\n\nA. [Footer] the links are cramped\n"
    assert len(check_crit.check_crit(text)) == 1
    assert problems_mentioning(text, "direction")


def test_wrapped_item_reads_its_continuation_lines():
    text = "**Overall:** x\n\nA. [Edge] edges too sharp\n   and glassy → cloudier, with fine cracks\n"
    assert check_crit.check_crit(text) == []


def test_praise_on_a_continuation_line_is_caught():
    text = "**Overall:** x\n\nA. [Edge] edges too sharp → cloudier\n   though overall it looks great\n"
    assert len(check_crit.check_crit(text)) == 1
    assert problems_mentioning(text, "praise")


@pytest.mark.parametrize(
    "item",
    [
        "A. [Score card] The credit score is 11px grey → make it the largest number",
        "A. [Invoice date] 3/10/2026 reads as March or October → write 10 March 2026",
        "A. [Hero copy] Shoppers love items they can preview → lead with the preview",
        "A. [Product grid] The glove items lack sizes → show sizes on each card",
        "A. [Docs intro] Describes a nice workflow but no first step → open with the first command",
    ],
)
def test_subject_words_are_not_praise_or_scores(item):
    assert check_crit.check_crit(f"**Overall:** x\n\n{item}\n") == []


def test_dangling_matching_reference():
    text = "**Overall:** x\n\nA. [Edge] too sharp → cloudier\nB. [Link] one style → matching Z\n"
    found = check_crit.check_crit(text)
    assert len(found) == 1
    assert "Z" in found[0]


def test_pin_alone_on_its_line_keeps_the_item():
    text = "**Overall:** x\n\nA. [Hero]\n   Reads like a tagline → name the dish\n"
    assert check_crit.check_crit(text) == []


def test_item_with_no_body_needs_a_direction():
    text = "**Overall:** x\n\nA. [Edge] too sharp → cloudier\nB. [Footer]\n"
    found = check_crit.check_crit(text)
    assert len(found) == 1
    assert "Item B" in found[0] and "direction" in found[0]


def test_arrow_directly_after_pin_is_an_item():
    text = "**Overall:** x\n\nA. [Footer]→ one link style\n"
    assert check_crit.check_crit(text) == []


def test_overall_word_inside_an_item_is_not_the_overall_note():
    text = "A. [Hero] too generic\n   overall it reads as SaaS → name the dish\n"
    found = check_crit.check_crit(text)
    assert len(found) == 1
    assert "overall" in found[0].lower()


def test_overall_heading_then_paragraph_is_checked_for_praise():
    text = "## Overall\n\nLooks great, mostly.\n\nA. [Edge] too sharp → cloudier\n"
    found = check_crit.check_crit(text)
    assert len(found) == 1
    assert "praise" in found[0]


def test_more_than_26_items_is_reported():
    text = "**Overall:** x\n\nA. [Edge] too sharp → cloudier\nAA. [Extra] more → split it\n"
    assert problems_mentioning(text, "26")


def test_worked_example_in_crit_sheet_reference_passes():
    reference = Path(__file__).resolve().parent.parent / "references" / "crit-sheet.md"
    text = reference.read_text(encoding="utf-8")
    example = text.split("<!-- example:start -->", 1)[1].split("<!-- example:end -->", 1)[0]
    assert check_crit.check_crit(example) == []


def run_cli(*args, stdin=None):
    return subprocess.run(
        [sys.executable, str(SCRIPT), *args], input=stdin, capture_output=True, text=True
    )


def test_cli_valid_file(tmp_path):
    sheet = tmp_path / "crit.md"
    sheet.write_text(VALID)
    assert run_cli(str(sheet)).returncode == 0


def test_cli_invalid_file(tmp_path):
    sheet = tmp_path / "crit.md"
    sheet.write_text("**Overall:** looks great\n")
    result = run_cli(str(sheet))
    assert result.returncode == 1
    assert result.stdout.strip()


def test_cli_reads_stdin():
    assert run_cli("-", stdin=VALID).returncode == 0


@pytest.mark.parametrize("args", [(), ("does-not-exist.md",)])
def test_cli_usage_errors(args):
    assert run_cli(*args).returncode == 2
