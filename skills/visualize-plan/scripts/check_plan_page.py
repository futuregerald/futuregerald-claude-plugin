#!/usr/bin/env python3
import re
import sys
from pathlib import Path

TITLE = re.compile(r"<title>(.*?)</title>", re.IGNORECASE | re.DOTALL)
SKELETON_TAG = re.compile(r"<!doctype|<html[\s>]|<head[\s>]|<body[\s>]", re.IGNORECASE)
PLACEHOLDER = re.compile(r"\{\{[A-Z_]+\}\}")
FILLER = (re.compile(r"\blorem ipsum\b", re.IGNORECASE), re.compile(r"\bTODO\b"))
SCRIPT_SRC = re.compile(r"<script[^>]*\bsrc=[\"']([^\"']+)[\"']", re.IGNORECASE)
LINK_TAG = re.compile(r"<link\b[^>]*>", re.IGNORECASE)
STYLESHEET_REL = re.compile(r"\brel=[\"']stylesheet[\"']", re.IGNORECASE)
HREF = re.compile(r"\bhref=[\"']([^\"']+)[\"']", re.IGNORECASE)
MERMAID_BLOCK = re.compile(r"<pre class=[\"']mermaid[\"']>(.*?)</pre>", re.IGNORECASE | re.DOTALL)
SVG_BLOCK = re.compile(r"<svg\b.*?</svg>", re.IGNORECASE | re.DOTALL)
SVG_HEX = re.compile(r"(?:fill|stroke)\s*[:=]\s*[\"']?\s*#[0-9a-fA-F]{3,8}")
SCRIPT_HOSTS = ("https://cdnjs.cloudflare.com/", "https://cdn.jsdelivr.net/npm/")
STYLE_HOSTS = ("https://fonts.googleapis.com/",)
THEME_MARKERS = {
    "light tokens on :root": re.compile(r":root\s*\{[^}]*--ground\s*:", re.DOTALL),
    "a dark palette under prefers-color-scheme": re.compile(
        r"@media\s*\(prefers-color-scheme:\s*dark\)\s*\{\s*:root:not\(\[data-theme=\"light\"\]\)"
    ),
    "a dark palette under :root[data-theme=\"dark\"]": re.compile(r":root\[data-theme=\"dark\"\]\s*\{"),
    "a body background from a token": re.compile(r"body\s*\{[^}]*background\s*:\s*var\(", re.DOTALL),
}


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

    for src in SCRIPT_SRC.findall(text):
        if "mermaid" in src.lower():
            problems.append("loads Mermaid; the artifact draws <pre class=\"mermaid\"> itself")
        elif not src.startswith(SCRIPT_HOSTS):
            problems.append(f"script from a host the artifact blocks: {src}")

    for tag in LINK_TAG.findall(text):
        href = HREF.search(tag)
        if not STYLESHEET_REL.search(tag) or not href:
            continue
        href = href.group(1)
        if not href.startswith(STYLE_HOSTS):
            problems.append(f"stylesheet from a host the artifact blocks: {href}")

    for index, block in enumerate(MERMAID_BLOCK.findall(text), start=1):
        if not block.strip():
            problems.append(f"Mermaid block {index} is empty")

    for index, svg in enumerate(SVG_BLOCK.findall(text), start=1):
        if SVG_HEX.search(svg):
            problems.append(f"SVG {index} uses a hex colour; use a CSS variable so both themes work")

    return problems


def main(argv):
    if len(argv) != 2:
        print("usage: check_plan_page.py FILE|-", file=sys.stderr)
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
