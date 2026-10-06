#!/usr/bin/env python3
import re
import sys
from pathlib import Path

ITEM_START = re.compile(r"^\s*(?:[-*]\s+)?([A-Z])\.\s+\[([^\]]*)\]\s*(.*)$")
UNPINNED_ITEM_START = re.compile(r"^\s*(?:[-*]\s+)?([A-Z])\.\s+(?!\[)(\S.*)$")
MULTI_LETTER_ITEM = re.compile(r"^\s*(?:[-*]\s+)?([A-Z]{2,})\.\s")
OVERALL_HEADING = re.compile(r"^\s*(?:#+\s*|\*\*)?overall\b(?::?\*\*)?:?\s*(.*)$", re.IGNORECASE)
CROSS_REF = re.compile(r"\b(?i:same (?:note |treatment )?as|similar to|see|match(?:es|ing)?)\s+([A-Z])\b")
PRAISE = re.compile(r"\b(?:looks great|looks good|nice work|great job|love it)\b", re.IGNORECASE)
SCORE = re.compile(r"(?<![\d/])\b\d{1,2}\s*/\s*10\b(?!\s*/)|\b(?:score|rating)\s*[:=]\s*\d", re.IGNORECASE)
DIRECTION = ("→", "->")


def parse_sheet(text):
    overall = None
    items = []
    oversized = []
    current = None
    for line in text.splitlines():
        pinned = ITEM_START.match(line)
        unpinned = None if pinned else UNPINNED_ITEM_START.match(line)
        if pinned or unpinned:
            match = pinned or unpinned
            current = {
                "letter": match.group(1),
                "pin": pinned.group(2).strip() if pinned else "",
                "body": [pinned.group(3) if pinned else unpinned.group(2)],
            }
            items.append(current)
            continue
        if MULTI_LETTER_ITEM.match(line):
            oversized.append(MULTI_LETTER_ITEM.match(line).group(1))
            current = None
            continue
        if current is not None:
            if line.strip():
                current["body"].append(line.strip())
            else:
                current = None
            continue
        heading = OVERALL_HEADING.match(line)
        if overall is None and heading:
            overall = [heading.group(1)] if heading.group(1).strip() else []
        elif overall is not None and not items and line.strip():
            overall.append(line.strip())
    for item in items:
        item["body"] = " ".join(part for part in item["body"] if part).strip()
    return overall, items, oversized


def wording_problems(label, text):
    problems = []
    praise = sorted({match.lower() for match in PRAISE.findall(text)})
    if praise:
        problems.append(f"{label}: praise is not a fix ({', '.join(praise)})")
    if SCORE.search(text):
        problems.append(f"{label}: no scores — say what is off and what it should be")
    return problems


def check_crit(text):
    overall, items, oversized = parse_sheet(text)
    problems = []
    if overall is None:
        problems.append("Missing the **Overall:** note before the lettered items")
    else:
        problems.extend(wording_problems("Overall", " ".join(overall)))
    if oversized:
        problems.append(
            f"Items {', '.join(oversized)}: more than 26 items — merge related ones or split the sheet"
        )
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
