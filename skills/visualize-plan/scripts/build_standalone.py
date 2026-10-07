#!/usr/bin/env python3
import re
import sys
from pathlib import Path

MERMAID_CLASS = re.compile(r"class=[\"']mermaid[\"']", re.IGNORECASE)

MERMAID_SRC = "https://cdn.jsdelivr.net/npm/mermaid@11.4.1/dist/mermaid.min.js"
MERMAID_INIT = (
    "mermaid.initialize({ startOnLoad: true, theme: "
    "window.matchMedia(\"(prefers-color-scheme: dark)\").matches ? \"dark\" : \"default\" });"
)


def split_head(fragment):
    end = fragment.find("</style>")
    if end == -1:
        return "", fragment
    end += len("</style>")
    return fragment[:end].strip(), fragment[end:].strip()


def build(fragment):
    head, body = split_head(fragment)
    lines = [
        "<!doctype html>",
        "<html lang=\"en\">",
        "<head>",
        "<meta charset=\"utf-8\">",
        "<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">",
        "<style>body{margin:0}</style>",
    ]
    if head:
        lines.append(head)
    lines += ["</head>", "<body>", body]
    if MERMAID_CLASS.search(fragment):
        lines += [f"<script src=\"{MERMAID_SRC}\"></script>", f"<script>{MERMAID_INIT}</script>"]
    lines += ["</body>", "</html>", ""]
    return "\n".join(lines)


def default_output(source):
    return source.with_name(f"{source.stem}.standalone.html")


def main(argv):
    if len(argv) not in (2, 3):
        print("usage: build_standalone.py FRAGMENT.html [OUT.html]", file=sys.stderr)
        return 2
    source = Path(argv[1])
    try:
        fragment = source.read_text(encoding="utf-8")
    except OSError as error:
        print(f"cannot read {source}: {error}", file=sys.stderr)
        return 2
    output = Path(argv[2]) if len(argv) == 3 else default_output(source)
    output.write_text(build(fragment), encoding="utf-8")
    print(output)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
