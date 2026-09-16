#!/usr/bin/env python3
"""Convert a subset of Markdown to Atlassian Document Format (ADF).

Supports: headings, paragraphs, bullet lists, ordered lists, tables, rules,
and the inline marks that matter for triaging notes -- strong, em, code, link,
and the combination `[`code`](url)` which Jira's own markdown converter
silently strips.

The sub-agent writes ONE Markdown file with a marker line separating the visible summary from
the collapsed detail. This script splits it, converts both halves, and emits the ADF document.
It also writes the visible half beside the input so the orchestrator can read exactly the text
that will be posted -- never a sub-agent's summary of it.

Usage (cd into the ticket's scratchpad directory first; takes no path argument):
    md2adf.py [--title "Full Investigation Details"] > notes.adf.json
    md2adf.py --no-expand > notes.adf.json          # full mode: nothing collapsed

Reads ./notes.md and writes ./notes.visible.md -- both fixed names in the working
directory. The ticket key lives in the caller's `cd`, so no externally-supplied value
ever reaches a file path.

Both modes take the same two-block input. Short mode collapses the details into an `expand`
node; `--no-expand` renders them inline after a rule. Full mode still goes through this script
rather than posting raw Markdown, because Jira's own Markdown converter strips the
`[`code`](url)` link form these notes are built out of.

Writes `<stem>.visible.md` beside the input so the orchestrator can read exactly the text
that will be posted -- never a sub-agent's summary of it.

Known inline limits (all produce odd text, none produce invalid ADF):
  - a link href stops at the first ")", so `.../a_(b)_c` truncates
  - backticks inside **bold** render literally
  - an unclosed backtick pairs with the next one on the line

Exits non-zero with a message on stderr if the marker is missing or either half is empty.
"""
import json
import os
import re
import sys

# ---------- inline ----------

TOKEN = re.compile(
    r"""
    (?P<codelink>\[`(?P<cl_text>[^`]+)`\]\((?P<cl_href>[^)]+)\))
  | (?P<link>\[(?P<l_text>[^\]]+)\]\((?P<l_href>[^)]+)\))
  | (?P<strong>\*\*(?P<s_text>.+?)\*\*)
  | (?P<code>`(?P<c_text>[^`]+)`)
  | (?P<em>(?<![\w*])_(?P<e_text>[^_]+)_(?![\w*]))
    """,
    re.VERBOSE | re.DOTALL,
)


def _text(value, marks=None):
    node = {"type": "text", "text": value}
    if marks:
        node["marks"] = marks
    return node


def inline(src):
    """Markdown inline string -> list of ADF inline nodes."""
    out = []
    pos = 0
    for m in TOKEN.finditer(src):
        if m.start() > pos:
            out.append(_text(src[pos:m.start()]))
        if m.group("codelink"):
            out.append(_text(m.group("cl_text"), [
                {"type": "code"},
                {"type": "link", "attrs": {"href": m.group("cl_href")}},
            ]))
        elif m.group("link"):
            # the label may itself contain marks; keep it simple and bold-aware
            label = m.group("l_text")
            marks = [{"type": "link", "attrs": {"href": m.group("l_href")}}]
            bold = re.fullmatch(r"\*\*(.+)\*\*", label, re.DOTALL)
            if bold:
                label = bold.group(1)
                marks.append({"type": "strong"})
            out.append(_text(label, marks))
        elif m.group("strong"):
            inner = m.group("s_text")
            code = re.fullmatch(r"`([^`]+)`", inner)
            if code:
                # ADF's code mark combines only with link. Pairing it with strong is
                # rejected or silently stripped, so code wins and strong is dropped.
                out.append(_text(code.group(1), [{"type": "code"}]))
            else:
                out.append(_text(inner, [{"type": "strong"}]))
        elif m.group("code"):
            out.append(_text(m.group("c_text"), [{"type": "code"}]))
        elif m.group("em"):
            out.append(_text(m.group("e_text"), [{"type": "em"}]))
        pos = m.end()
    if pos < len(src):
        out.append(_text(src[pos:]))
    return [n for n in out if n["text"]] or [_text(" ")]


def para(src):
    return {"type": "paragraph", "content": inline(src)}


# ---------- block ----------

LIST_ITEM = re.compile(r"^(\s*)([-*]|\d+[.)])\s+(.*)$")
# A paragraph ends at any of these; the fence is here because a fenced block
# following a prose line with no blank between them must still be a code block.
BLOCK_START = re.compile(r"^(#{1,6}\s|\s*([-*]|\d+[.)])\s|\||`{3,}|~{3,})")


def _build_lists(items, pos, indent):
    """(indent, ordered, text) tuples -> (nested ADF list nodes, next position).

    A deeper-indented run becomes a child of the item above it, so an ordered
    list interrupted by nested bullets stays one list and keeps numbering.
    """
    nodes = []
    while pos < len(items) and items[pos][0] >= indent:
        level, ordered = items[pos][0], items[pos][1]
        node = {"type": "orderedList" if ordered else "bulletList"}
        if ordered:
            node["attrs"] = {"order": 1}
        node["content"] = []
        while (pos < len(items)
               and items[pos][0] == level
               and items[pos][1] == ordered):
            listitem = {"type": "listItem", "content": [para(items[pos][2])]}
            pos += 1
            if pos < len(items) and items[pos][0] > level:
                children, pos = _build_lists(items, pos, items[pos][0])
                listitem["content"].extend(children)
            node["content"].append(listitem)
        nodes.append(node)
        if pos < len(items) and items[pos][0] < level:
            break
    return nodes, pos


def _is_table_separator(line):
    """True for a GFM separator row: | --- | :--: | etc. Linear, no backtracking."""
    s = line.strip()
    if len(s) < 3 or not s.startswith("|") or not s.endswith("|"):
        return False
    body = s[1:-1]
    return bool(body) and all(c in " \t:|-" for c in body) and "-" in body


def _cell(kind, src):
    return {"type": kind, "attrs": {}, "content": [para(src)]}


def _split_row(line):
    return [c.strip() for c in line.strip().strip("|").split("|")]


def blocks(md):
    lines = md.split("\n")
    out = []
    i = 0
    while i < len(lines):
        line = lines[i]
        stripped = line.strip()

        if not stripped:
            i += 1
            continue

        if stripped in ("---", "***", "___"):
            out.append({"type": "rule"})
            i += 1
            continue

        # fenced code block -- consumed verbatim, newlines preserved
        fence = re.match(r"^\s*(`{3,}|~{3,})\s*([A-Za-z0-9_+-]*)\s*$", line)
        if fence:
            marker, lang = fence.group(1)[0] * 3, fence.group(2)
            body = []
            i += 1
            while i < len(lines) and not re.match(
                r"^\s*(`{3,}|~{3,})\s*$", lines[i]
            ):
                body.append(lines[i])
                i += 1
            i += 1  # closing fence (or EOF, which is fine)
            node = {"type": "codeBlock", "attrs": {}}
            if lang:
                node["attrs"]["language"] = lang
            text = "\n".join(body)
            if text:
                node["content"] = [{"type": "text", "text": text}]
            out.append(node)
            continue

        h = re.match(r"^(#{1,6})\s+(.*)$", stripped)
        if h:
            out.append({
                "type": "heading",
                "attrs": {"level": min(len(h.group(1)), 6)},
                "content": inline(h.group(2)),
            })
            i += 1
            continue

        # table: a header row followed by a separator row
        # The separator test is a character-set check, not a regex: the natural pattern
        # puts "|" inside a repeated class AND requires one after it, which backtracks.
        if stripped.startswith("|") and i + 1 < len(lines) and _is_table_separator(
            lines[i + 1]
        ):
            header = _split_row(stripped)
            rows = []
            i += 2
            while i < len(lines) and lines[i].strip().startswith("|"):
                rows.append(_split_row(lines[i].strip()))
                i += 1
            content = [{
                "type": "tableRow",
                "content": [_cell("tableHeader", c) for c in header],
            }]
            for r in rows:
                if len(r) > len(header):
                    print(
                        f"md2adf: warning: table row has {len(r)} cells but the "
                        f"header has {len(header)}; extra cells dropped: "
                        f"{r[len(header):]}",
                        file=sys.stderr,
                    )
                r = (r + [""] * len(header))[: len(header)]
                content.append({
                    "type": "tableRow",
                    "content": [_cell("tableCell", c) for c in r],
                })
            out.append({"type": "table", "attrs": {"isNumberColumnEnabled": False,
                                                   "layout": "default"}, "content": content})
            continue

        # lists (bullet or ordered), with nesting and wrapped continuation lines
        if LIST_ITEM.match(line):
            collected = []
            while i < len(lines):
                m2 = LIST_ITEM.match(lines[i])
                if m2:
                    collected.append((
                        len(m2.group(1).expandtabs(4)),
                        m2.group(2) not in ("-", "*"),
                        m2.group(3),
                    ))
                    i += 1
                    continue
                # a wrapped continuation line belongs to the item above it
                if lines[i].strip() and collected and lines[i][:1] in (" ", "\t"):
                    ind, ordered, text = collected[-1]
                    collected[-1] = (ind, ordered, text + " " + lines[i].strip())
                    i += 1
                    continue
                break
            built, _ = _build_lists(collected, 0, collected[0][0])
            out.extend(built)
            continue

        # paragraph: consume until blank or a new block starts
        buf = [stripped]
        i += 1
        while i < len(lines):
            nxt = lines[i]
            if not nxt.strip():
                break
            if BLOCK_START.match(nxt.strip()):
                break
            if nxt.strip() in ("---", "***", "___"):
                break
            buf.append(nxt.strip())
            i += 1
        out.append(para(" ".join(buf)))
    return out


MARKER = "# BLOCK 2 — FULL INVESTIGATION DETAILS"
VISIBLE_HEADER = re.compile(r"^#\s*BLOCK 1[^\n]*\n", re.MULTILINE)


def split_notes(md):
    """notes.md -> (visible, details). Raises SystemExit if the shape is wrong."""
    if MARKER not in md:
        sys.exit(
            "md2adf: marker not found. The notes file must contain this line verbatim:\n"
            f"  {MARKER}"
        )
    visible, details = md.split(MARKER, 1)
    visible = VISIBLE_HEADER.sub("", visible).strip()
    # Drop trailing horizontal rules left over from the split. Done line-by-line rather
    # than with a regex: the obvious pattern nests \s* inside a + group, and \s matches
    # the newline the group also consumes, which backtracks exponentially.
    vlines = visible.split("\n")
    while vlines and vlines[-1].strip() in ("", "---", "***", "___"):
        vlines.pop()
    visible = "\n".join(vlines).strip()
    details = details.strip()
    if not visible:
        sys.exit("md2adf: the visible summary is empty.")
    if not details:
        sys.exit("md2adf: the details block is empty.")
    return visible, details


LOCAL_PATH = re.compile(r"(?:/Users/|/home/|/private/tmp/|/var/folders/|[A-Z]:\\\\Users\\\\)\S*")


def _warn_local_paths(md):
    """The note is about to be posted to a tracker that may be public.

    An absolute local path leaks the operator's username and private repo names, and it is
    useless to every reader. output-templates.md forbids them; this is the check behind that
    rule, because the details block is the part nobody re-reads before it goes out.
    """
    hits = []
    for n, line in enumerate(md.split("\n"), 1):
        for m in LOCAL_PATH.finditer(line):
            hits.append((n, m.group(0)))
    for n, hit in hits[:10]:
        print(f"md2adf: warning: line {n} contains a local path: {hit}", file=sys.stderr)
    if len(hits) > 10:
        print(f"md2adf: warning: ...and {len(hits) - 10} more", file=sys.stderr)
    if hits:
        print(
            "md2adf: local paths leak your username and private repo names to whoever can read "
            "the ticket. Replace them with repo-relative paths or permalinks before posting.",
            file=sys.stderr,
        )
    return hits


# This script takes NO path argument. It reads NOTES_FILE from the directory it was
# started in and writes SIDECAR_FILE beside it -- both constants. The ticket key lives
# in the caller's `cd`, never in a value this process handles, so there is no path for
# a malformed key to travel into the filesystem and nothing to validate or escape.
NOTES_FILE = "notes.md"
SIDECAR_FILE = "notes.visible.md"


def _read_notes():
    if not os.path.isfile(NOTES_FILE):
        sys.exit(
            f"md2adf: no {NOTES_FILE} in {os.getcwd()}.\n"
            f"  cd into the ticket's scratchpad directory first, then run this with no path."
        )
    with open(NOTES_FILE, encoding="utf-8") as fh:
        return fh.read()


def _write_sidecar(visible):
    if os.path.exists(SIDECAR_FILE) and not os.path.isfile(SIDECAR_FILE):
        sys.exit(f"md2adf: refusing to write {SIDECAR_FILE} -- not a regular file")
    with open(SIDECAR_FILE, "w", encoding="utf-8") as fh:
        fh.write(visible + "\n")
    print(f"md2adf: wrote {os.path.join(os.getcwd(), SIDECAR_FILE)}", file=sys.stderr)


def main():
    args = list(sys.argv[1:])
    expand_title = "Full Investigation Details"
    no_expand = False
    if "--no-expand" in args:
        args.remove("--no-expand")
        no_expand = True
    if "--title" in args:
        idx = args.index("--title")
        if idx + 1 >= len(args):
            sys.exit("md2adf: --title needs a value")
        expand_title = args[idx + 1]
        del args[idx:idx + 2]
    if args:
        sys.exit(f"md2adf: unexpected argument {args[0]!r}\n{__doc__}")

    raw = _read_notes()
    visible, details = split_notes(raw)
    _warn_local_paths(raw)

    # The guard rail: write the visible half so the orchestrator reads the real input
    # to the posted bytes rather than trusting a receipt.
    _write_sidecar(visible)

    if no_expand:
        # Full mode: everything visible, no collapsed node.
        tail = [{"type": "rule"}] + blocks(details)
    else:
        tail = [
            {"type": "rule"},
            {
                "type": "expand",
                "attrs": {"title": expand_title},
                "content": blocks(details),
            },
        ]

    doc = {"version": 1, "type": "doc", "content": blocks(visible) + tail}
    json.dump(doc, sys.stdout, ensure_ascii=False, indent=2)


if __name__ == "__main__":
    main()
