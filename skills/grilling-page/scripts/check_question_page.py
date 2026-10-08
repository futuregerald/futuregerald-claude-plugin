#!/usr/bin/env python3
import json
import re
import sys
from pathlib import Path

TITLE = re.compile(r"<title>(.*?)</title>", re.IGNORECASE | re.DOTALL)
SKELETON_TAG = re.compile(r"<!doctype|<html[\s>]|<head[\s>]|<body[\s>]", re.IGNORECASE)
PLACEHOLDER = re.compile(r"\{\{[A-Z_]+\}\}")
FILLER = (re.compile(r"\blorem ipsum\b", re.IGNORECASE), re.compile(r"\bTODO\b"))
PAGE_DATA = re.compile(
    r"<script type=\"application/json\" id=\"page-data\">(.*?)</script>", re.IGNORECASE | re.DOTALL
)
TAG = re.compile(r"</?([a-zA-Z][a-zA-Z0-9]*)[^>]*>")
ALLOWED_TAGS = {"code", "em", "strong"}
THEME_MARKERS = {
    "light tokens on :root": re.compile(r":root\s*\{[^}]*--ground\s*:", re.DOTALL),
    "a dark palette under prefers-color-scheme": re.compile(
        r"@media\s*\(prefers-color-scheme:\s*dark\)\s*\{\s*:root:not\(\[data-theme=\"light\"\]\)"
    ),
    "a dark palette under :root[data-theme=\"dark\"]": re.compile(r":root\[data-theme=\"dark\"\]\s*\{"),
    "a body background from a token": re.compile(r"body\s*\{[^}]*background\s*:\s*var\(", re.DOTALL),
}
PAGE_PARTS = {
    "the progress count": 'id="count"',
    "the answer block": 'id="answers"',
    "the Use recommendations button": 'id="use-recommended"',
    "the Copy answers button": 'id="copy"',
    "the clipboard copy": "navigator.clipboard.writeText",
}
SAFE_NAME = re.compile(r"[A-Za-z0-9_-]+")
MAX_WORDS = {"title": 16, "context": 130, "why": 50, "desc": 50}
QUESTION_TYPES = ("radio", "checkbox")


def words(html):
    return len(TAG.sub("", html).split())


def text_field(problems, where, item, field, required=True):
    value = item.get(field)
    if not isinstance(value, str) or not value.strip():
        if required:
            problems.append(f"{where}: no {field}")
        return ""
    for tag in sorted({name.lower() for name in TAG.findall(value)} - ALLOWED_TAGS):
        problems.append(f"{where}: <{tag}> in {field}; only <code>, <em> and <strong> are allowed")
    limit = MAX_WORDS.get(field)
    if limit and words(value) > limit:
        problems.append(f"{where}: {field} is {words(value)} words; keep it to {limit}")
    return value


def check_options(problems, where, question):
    options = question.get("options")
    if not isinstance(options, list) or len(options) < 2:
        problems.append(f"{where}: needs at least two options")
        return
    keys = [option.get("key") for option in options if isinstance(option, dict)]
    if len(keys) != len(options) or not all(isinstance(key, str) and key for key in keys):
        problems.append(f"{where}: every option needs a key")
    elif len(set(keys)) != len(keys):
        problems.append(f"{where}: option keys repeat")
    for key in keys:
        if isinstance(key, str) and key and not SAFE_NAME.fullmatch(key):
            problems.append(f"{where}: option key {key!r} may only use letters, digits, - and _")
    for index, option in enumerate(options, start=1):
        if not isinstance(option, dict):
            continue
        label = f"{where} option {option.get('key') or index}"
        text_field(problems, label, option, "title")
        if not text_field(problems, label, option, "desc", required=False):
            problems.append(f"{label}: no desc; say what picking it means for users")
    flags = [option.get("rec") is True for option in options if isinstance(option, dict)]
    recommended = sum(flags)
    if question.get("type") == "radio" and recommended != 1:
        problems.append(f"{where}: a radio question needs exactly one recommended option, has {recommended}")
    if question.get("type") == "checkbox" and recommended < 1:
        problems.append(f"{where}: no recommended option")
    if recommended and flags != sorted(flags, reverse=True):
        problems.append(f"{where}: recommended options must be listed first")


def check_questions(data):
    problems = []
    if not isinstance(data, dict):
        return ["page data must be a JSON object"]
    for field in ("storeKey", "answerHeading"):
        if not isinstance(data.get(field), str) or not data[field].strip():
            problems.append(f"page data has no {field}")
    questions = data.get("questions")
    if not isinstance(questions, list) or not questions:
        problems.append("page data has no questions")
        return problems
    seen = set()
    for index, question in enumerate(questions, start=1):
        if not isinstance(question, dict):
            problems.append(f"question {index} is not an object")
            continue
        qid = question.get("id")
        has_id = isinstance(qid, str) and bool(qid.strip())
        where = qid if has_id else f"question {index}"
        if not has_id:
            problems.append(f"{where}: no id")
        elif not SAFE_NAME.fullmatch(qid):
            problems.append(f"{where}: id may only use letters, digits, - and _")
        elif qid in seen:
            problems.append(f"{where}: id used twice")
        else:
            seen.add(qid)
        title = text_field(problems, where, question, "title")
        if title and not title.rstrip().endswith("?"):
            problems.append(f"{where}: title must be a question ending in ?")
        text_field(problems, where, question, "context")
        text_field(problems, where, question, "why")
        if question.get("type") not in QUESTION_TYPES:
            problems.append(f"{where}: type must be radio or checkbox")
        check_options(problems, where, question)
    return problems


def check(text):
    problems = []

    title = TITLE.search(text[:8192])
    if not title or not title.group(1).strip():
        problems.append("no <title> in the first 8KB")

    if SKELETON_TAG.search(text):
        problems.append("contains doctype/html/head/body tags; the artifact source must be a fragment")

    for placeholder in sorted(set(PLACEHOLDER.findall(text))):
        problems.append(f"template placeholder left in: {placeholder}")

    fillers = {match.group(0) for pattern in FILLER for match in pattern.finditer(text)}
    for filler in sorted(fillers):
        problems.append(f"filler text left in: {filler}")

    for label, pattern in THEME_MARKERS.items():
        if not pattern.search(text):
            problems.append(f"missing {label}")

    for label, marker in PAGE_PARTS.items():
        if marker not in text:
            problems.append(f"missing {label}; keep the template's markup and script as they are")

    block = PAGE_DATA.search(text)
    if not block:
        problems.append("no <script type=\"application/json\" id=\"page-data\"> block")
        return problems
    try:
        data = json.loads(block.group(1))
    except json.JSONDecodeError as error:
        problems.append(f"page data is not valid JSON: {error}")
        return problems
    return problems + check_questions(data)


def main(argv):
    if len(argv) != 2:
        print("usage: check_question_page.py FILE|-", file=sys.stderr)
        return 2
    try:
        text = sys.stdin.read() if argv[1] == "-" else Path(argv[1]).read_text(encoding="utf-8")
    except OSError as error:
        print(f"cannot read {argv[1]}: {error}", file=sys.stderr)
        return 2
    problems = check(text)
    for problem in problems:
        print(problem)
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
