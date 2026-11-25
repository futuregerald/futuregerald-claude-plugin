import sys
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
if str(SCRIPTS_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPTS_DIR))

import md2adf


def text_nodes(node, inside_code_block=False):
    in_code = inside_code_block or node.get("type") == "codeBlock"
    if node.get("type") == "text":
        yield node["text"], in_code
    for child in node.get("content", []):
        yield from text_nodes(child, in_code)


def prose_with_line_breaks(nodes):
    return [
        text
        for node in nodes
        for text, in_code in text_nodes(node)
        if "\n" in text and not in_code
    ]


def list_item_texts(list_node):
    return [
        "".join(text for text, _ in text_nodes(item["content"][0]))
        for item in list_node["content"]
    ]


def test_hard_wrapped_paragraph_becomes_one_line():
    nodes = md2adf.blocks("The list loads every\norder with no paging.")

    assert nodes == [md2adf.para("The list loads every order with no paging.")]


def test_unindented_wrapped_bullet_stays_in_its_item():
    nodes = md2adf.blocks(
        "- The preload loads a\nseparate copy of the records.\n- Second bullet"
    )

    assert len(nodes) == 1
    assert list_item_texts(nodes[0]) == [
        "The preload loads a separate copy of the records.",
        "Second bullet",
    ]


def test_indented_wrapped_bullet_stays_in_its_item():
    nodes = md2adf.blocks("- First bullet\n  wrapped with indent.")

    assert list_item_texts(nodes[0]) == ["First bullet wrapped with indent."]


def test_blank_line_ends_a_list_before_the_next_paragraph():
    nodes = md2adf.blocks("- Only bullet\n\nA new paragraph.")

    assert [n["type"] for n in nodes] == ["bulletList", "paragraph"]


def test_heading_after_a_bullet_is_not_folded_into_it():
    nodes = md2adf.blocks("- Only bullet\n## Risks")

    assert [n["type"] for n in nodes] == ["bulletList", "heading"]


def test_wrapped_notes_post_no_line_breaks_outside_code_blocks():
    notes = (
        "## TLDR\n"
        "Root cause: `OrderSerializer`\n"
        "asks the database per row.\n\n"
        "## Key Findings\n"
        "- One grouped count (**L28**),\n"
        "one postings query per row.\n"
        "  1. nested step that\n"
        "  wraps.\n\n"
        "```ruby\n"
        "line_one\n"
        "line_two\n"
        "```\n"
    )

    nodes = md2adf.blocks(notes)

    assert prose_with_line_breaks(nodes) == []
    code = [n for n in nodes if n["type"] == "codeBlock"]
    assert code[0]["content"][0]["text"] == "line_one\nline_two"
