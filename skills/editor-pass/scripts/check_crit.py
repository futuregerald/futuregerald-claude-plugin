#!/usr/bin/env python3
import re
import sys
from pathlib import Path

ITEM_START = re.compile(r"^\s*(?:[-*]\s+)?([A-Z])\.\s+\[([^\]]*)\]\s+(.+)$")
UNPINNED_ITEM_START = re.compile(r"^\s*(?:[-*]\s+)?([A-Z])\.\s+(?!\[)\S")
OVERALL = re.compile(r"(?im)^\s*(?:#+\s*|\*\*)?overall\b")
CROSS_REF = re.compile(r"\b(?i:same (?:note |treatment )?as|similar to|see|match(?:es)?)\s+([A-Z])\b")
PRAISE = ("looks great", "looks good", "nice work", "great job", "love it")
SCORE = re.compile(r"\b\d{1,2}\s*/\s*10\b|\bscore\b", re.IGNORECASE)
DIRECTION = ("→", "->")


def split_sections(text):
    overall_lines = []
    items = []
    in_overall = False
    current = None
    for line in text.splitlines():
        start = ITEM_START.match(line)
        unpinned = None if start else UNPINNED_ITEM_START.match(line)
        if start or unpinned:
            letter = (start or unpinned).group(1)
            pin = start.group(2).strip() if start else ""
            body = start.group(3) if start else line.split(".", 1)[1]
            current = {"letter": letter, "pin": pin, "body": [body]}
            items.append(current)
            in_overall = False
        elif not line.strip():
            current = None
            in_overall = False
        elif current is not None:
            current["body"].append(line.strip())
        elif OVERALL.match(line) or in_overall:
            in_overall = True
            overall_lines.append(line)
    for item in items:
        item["body"] = " ".join(item["body"])
    return " ".join(overall_lines), items


def wording_problems(label, text):
    problems = []
    lowered = text.lower()
    praise = [phrase for phrase in PRAISE if phrase in lowered]
    if praise:
        problems.append(f"{label}: praise is not a fix ({', '.join(praise)})")
    if SCORE.search(text):
        problems.append(f"{label}: no scores — say what is off and what it should be")
    return problems


def check_crit(text):
    overall, items = split_sections(text)
    problems = []
    if not OVERALL.search(text):
        problems.append("Missing the **Overall:** note before the lettered items")
    else:
        problems.extend(wording_problems("Overall", overall))
    if not items:
        problems.append("Crit sheet has no lettered items (A. [pin] what is off → what it should be)")
        return problems

    letters = [item["letter"] for item in items]
    for letter in sorted({l for l in letters if letters.count(l) > 1}):
        problems.append(f"Item {letter}: duplicate letter")
    present = set(letters)
    for code in range(ord("A"), ord(max(letters))):
        if chr(code) not in present:
            problems.append(f"Item {chr(code)}: missing — letters must run A, B, C… with no gaps")

    for item in items:
        label = f"Item {item['letter']}"
        if not item["pin"]:
            problems.append(f"{label}: no pin — say where, e.g. [Hero headline] or [file.py:42]")
        if not any(arrow in item["body"] for arrow in DIRECTION):
            problems.append(f"{label}: no direction — add → and what it should be or feel like")
        for ref in CROSS_REF.findall(item["body"]):
            if ref not in present:
                problems.append(f"{label}: refers to item {ref}, which does not exist")
        problems.extend(wording_problems(label, item["body"]))
    return problems


def main(argv):
    if len(argv) != 1:
        print("usage: check_crit.py FILE|-", file=sys.stderr)
        return 2
    if argv[0] == "-":
        text = sys.stdin.read()
    else:
        path = Path(argv[0])
        if not path.is_file():
            print(f"check_crit.py: no such file: {path}", file=sys.stderr)
            return 2
        text = path.read_text(encoding="utf-8")
    problems = check_crit(text)
    for problem in problems:
        print(problem)
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
