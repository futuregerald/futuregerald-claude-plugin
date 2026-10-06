#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
from urllib.parse import urlparse

PLACEHOLDER = re.compile(r"^\[.*\]$")
ELLIPSIS = "…"
CARD_TYPES = ("inlineCard", "blockCard", "embedCard")
BARE_URL = re.compile(r"https?://[^\s<>\"']+")
URL_TRAILING = ").,;:!?]"
HIGH_PRIORITY_RANK = {"p0": 0, "blocker": 0, "highest": 0, "p1": 1, "critical": 1}
MENTION_EXCERPT_LIMIT = 100
PERSON_GROUPS = ("completed", "in_progress", "stuck", "to_do")
ROSTER_COLUMNS = {
    "name": "name",
    "github handle": "github",
    "role": "role",
    "jira account id": "jira_account_id",
}


def _cells(line):
    return [cell.strip() for cell in line.strip().strip("|").split("|")]


def _is_separator(line):
    return bool(re.fullmatch(r"\|?[\s:|-]+\|?", line.strip())) and "-" in line


def _handle(value):
    return value.strip("`").lstrip("@").strip()


def parse_roster(markdown: str) -> list[dict]:
    lines = markdown.splitlines()
    for index, line in enumerate(lines[:-1]):
        if not line.strip().startswith("|") or not _is_separator(lines[index + 1]):
            continue
        header = [cell.lower() for cell in _cells(line)]
        if "name" not in header or "github handle" not in header:
            continue
        columns = {ROSTER_COLUMNS[cell]: position for position, cell in enumerate(header)
                   if cell in ROSTER_COLUMNS}
        roster = []
        for row in lines[index + 2:]:
            if not row.strip().startswith("|"):
                break
            cells = _cells(row)
            values = {field: (cells[position] if position < len(cells) else "")
                      for field, position in columns.items()}
            if any(PLACEHOLDER.match(values.get(field, "")) for field in ("name", "github", "role")):
                raise ValueError("team config not filled in")
            roster.append({
                "name": values["name"],
                "github": _handle(values["github"]),
                "role": values.get("role", ""),
                "jira_account_id": values.get("jira_account_id") or None,
            })
        return roster
    return []


def match_assignee(assignee: dict | None, roster: list[dict]) -> dict | None:
    if not assignee:
        return None
    account_id = assignee.get("accountId")
    if account_id:
        for person in roster:
            if person.get("jira_account_id") == account_id:
                return person
    display = (assignee.get("displayName") or "").casefold()
    if display:
        for person in roster:
            if person["name"].casefold() == display:
                return person
    return None


def _adf_pieces(node, pieces):
    if not isinstance(node, dict):
        return
    kind = node.get("type")
    if kind == "text":
        pieces.append(node.get("text") or "")
    elif kind == "mention":
        pieces.append((node.get("attrs") or {}).get("text") or "")
    elif kind == "hardBreak":
        pieces.append(" ")
    for child in node.get("content") or []:
        _adf_pieces(child, pieces)
    if kind not in ("text", "mention", "hardBreak"):
        pieces.append(" ")


def _truncate(text, limit):
    if len(text) <= limit:
        return text
    return text[:max(limit - 1, 0)].rstrip() + ELLIPSIS


def adf_text(node: dict | None, limit: int) -> str:
    pieces = []
    _adf_pieces(node, pieces)
    return _truncate(" ".join("".join(pieces).split()), limit)


def _adf_link_walk(node, found):
    if not isinstance(node, dict):
        return
    if node.get("type") in CARD_TYPES:
        url = (node.get("attrs") or {}).get("url")
        if url:
            found.append(url)
    if node.get("type") == "text":
        found.extend(match.rstrip(URL_TRAILING) for match in BARE_URL.findall(node.get("text") or ""))
    for mark in node.get("marks") or []:
        if mark.get("type") == "link":
            href = (mark.get("attrs") or {}).get("href")
            if href:
                found.append(href)
    for child in node.get("content") or []:
        _adf_link_walk(child, found)


def adf_links(node: dict | None) -> list[str]:
    found = []
    _adf_link_walk(node, found)
    return list(dict.fromkeys(found))


def classify_doc_link(url: str) -> str | None:
    parsed = urlparse(url)
    host = (parsed.hostname or "").lower()
    path = parsed.path
    if host.endswith(".atlassian.net") and path.startswith("/wiki"):
        return "wiki"
    if host == "docs.google.com":
        if path.startswith("/document"):
            return "doc"
        if path.startswith("/spreadsheets"):
            return "sheet"
    if host == "figma.com" or host.endswith(".figma.com"):
        return "design"
    return None


def _category(issue):
    return (((issue.get("fields") or {}).get("status") or {}).get("statusCategory") or {}).get("key")


def _status_name(issue):
    return ((issue.get("fields") or {}).get("status") or {}).get("name") or ""


def epic_progress(children: list[dict]) -> dict:
    total = len(children)
    done = sum(1 for child in children if _category(child) == "done")
    in_progress = sum(1 for child in children if _category(child) == "indeterminate")
    pct = round(done * 100 / total) if total else None
    return {"total": total, "done": done, "in_progress": in_progress, "pct": pct}


def child_to_epic(children_by_epic: dict[str, list[dict]]) -> dict[str, str]:
    return {child["key"]: epic for epic, children in children_by_epic.items()
            for child in children}


def is_blocked(issue):
    return "blocked" in _status_name(issue).casefold()


def _person_group(issue):
    category = _category(issue)
    if category == "done":
        return "completed"
    if is_blocked(issue) or issue.get("stalled"):
        return "stuck"
    if category == "indeterminate":
        return "in_progress"
    return "to_do"


def group_by_person(issues: list[dict], roster: list[dict]) -> tuple[dict[str, dict], list[str]]:
    people = {person["name"]: {group: [] for group in PERSON_GROUPS} for person in roster}
    unmatched = []
    for issue in issues:
        fields = issue.get("fields") or {}
        assignee = fields.get("assignee")
        if not assignee:
            continue
        person = match_assignee(assignee, roster)
        if person is None:
            name = assignee.get("displayName") or assignee.get("accountId") or "unknown"
            if name not in unmatched:
                unmatched.append(name)
            continue
        people[person["name"]][_person_group(issue)].append({
            "key": issue["key"],
            "summary": fields.get("summary") or "",
            "status": _status_name(issue),
        })
    return people, unmatched


def _key_order(key):
    prefix, _, number = key.rpartition("-")
    return (prefix, int(number) if number.isdigit() else -1, key)


def _issuetype_name(issue):
    return ((issue.get("fields") or {}).get("issuetype") or {}).get("name") or ""


def unassigned_issues(issues: list[dict]) -> list[dict]:
    found, seen = [], set()
    for issue in issues:
        fields = issue.get("fields") or {}
        issuetype = _issuetype_name(issue)
        category = _category(issue)
        if (issue["key"] in seen or fields.get("assignee") or category == "done"
                or issuetype.casefold() == "epic"):
            continue
        if issuetype.casefold() != "bug" and category != "indeterminate":
            continue
        seen.add(issue["key"])
        found.append({"key": issue["key"], "type": issuetype, "status": _status_name(issue),
                      "priority": (fields.get("priority") or {}).get("name"),
                      "summary": fields.get("summary") or ""})
    return found


def high_priority_outside_epics(issues: list[dict], child_map: dict[str, str],
                                has_parent: set[str]) -> list[dict]:
    found, seen = [], set()
    for issue in issues:
        fields = issue.get("fields") or {}
        priority = (fields.get("priority") or {}).get("name") or ""
        if (issue["key"] in seen or issue["key"] in child_map or issue["key"] in has_parent
                or priority.casefold() not in HIGH_PRIORITY_RANK or _category(issue) == "done"
                or _issuetype_name(issue).casefold() == "epic"):
            continue
        seen.add(issue["key"])
        found.append({"key": issue["key"], "type": _issuetype_name(issue),
                      "status": _status_name(issue), "priority": priority,
                      "assignee": (fields.get("assignee") or {}).get("displayName"),
                      "summary": fields.get("summary") or ""})
    return sorted(found, key=lambda item: (HIGH_PRIORITY_RANK[item["priority"].casefold()],
                                           _key_order(item["key"])))


def _open_rank(child):
    if "review" in (child.get("status") or "").casefold():
        return 0
    if child.get("category") == "indeterminate":
        return 1
    return 2


def open_children(children: list[dict]) -> list[dict]:
    return sorted((child for child in children if child.get("category") != "done"),
                  key=lambda child: (_open_rank(child), _key_order(child["key"])))


def parse_jira_time(value):
    text = re.sub(r"([+-]\d{2})(\d{2})$", r"\1:\2", value.replace("Z", "+00:00"))
    return datetime.fromisoformat(text)


def _comment_text(body, limit):
    if isinstance(body, str):
        return _truncate(" ".join(body.split()), limit)
    return adf_text(body, limit)


def _mentions(node, found):
    if not isinstance(node, dict):
        return
    if node.get("type") == "mention":
        attrs = node.get("attrs") or {}
        found.append((attrs.get("id"), (attrs.get("text") or "").lstrip("@").strip()))
    for child in node.get("content") or []:
        _mentions(child, found)


def _addressees(comment):
    author = comment.get("author") or {}
    mentions = []
    _mentions(comment.get("body"), mentions)
    names = []
    for account_id, name in mentions:
        if account_id:
            if account_id == author.get("accountId"):
                continue
        elif name.casefold() == (author.get("displayName") or "").casefold():
            continue
        if name and name not in names:
            names.append(name)
    return names


def question_candidates(comments: list[dict], limit: int) -> list[dict]:
    dated = [comment for comment in comments if comment.get("created")]
    if not dated:
        return []
    newest = max(dated, key=lambda comment: parse_jira_time(comment["created"]))
    full = _comment_text(newest.get("body"), 10 ** 9)
    asks = "?" in full
    addressees = _addressees(newest)
    if not asks and not addressees:
        return []
    created = parse_jira_time(newest["created"])
    return [{
        "author": (newest.get("author") or {}).get("displayName") or "unknown",
        "created": newest["created"],
        "age_days": (datetime.now(timezone.utc) - created).days,
        "addressees": addressees,
        "excerpt": _truncate(full, limit if asks else min(limit, MENTION_EXCERPT_LIMIT)),
    }]


SCOPES = ("team", "person", "epic")
GROUP_LABELS = {"completed": "Completed", "in_progress": "In progress", "stuck": "Stuck",
                "to_do": "To do"}
MAX_COMMENTS = 3
FLAGGED_WITH_EXCERPTS = 15
EXCERPTS_PER_FLAGGED = 1
OPEN_CHILDREN_SHOWN = 4
SUMMARY_LIMIT = 80
NOT_STARTED_KEPT = 10


def _clean(text):
    return " ".join(str(text or "").split()).replace('{"', '{ "')


def _same_person(left, right):
    return bool(left) and bool(right) and left.casefold() == right.casefold()


def _section(heading, lines):
    return "\n".join([heading, ""] + (lines or ["None."]))


def _render_open_children(children):
    shown = open_children(children)
    if not shown:
        return ["- Open children: none"]
    lines = ["- Open children:"] + [
        f"  - {child['key']} {_clean(child.get('status'))} · "
        f"{_clean(child.get('assignee')) or 'unassigned'} · "
        f"{_truncate(_clean(child.get('summary')), SUMMARY_LIMIT)}"
        for child in shown[:OPEN_CHILDREN_SHOWN]
    ]
    if len(shown) > OPEN_CHILDREN_SHOWN:
        lines.append(f"  - +{len(shown) - OPEN_CHILDREN_SHOWN} more open")
    return lines


def _render_epic(epic, with_open_children=False):
    parent = epic.get("parent")
    assignee = epic.get("assignee")
    if assignee:
        owner = _clean(assignee.get("name"))
        if assignee.get("active") is False:
            owner += " (inactive)"
    else:
        owner = "unassigned"
    progress = epic.get("progress") or {}
    pct = progress.get("pct")
    details = [
        f"Status: {_clean(epic.get('status'))}",
        f"Priority: {_clean(epic.get('priority')) or 'none'}",
        f"Parent: {_clean(parent['key'] + ' ' + (parent.get('summary') or '')) if parent else 'none'}",
        f"Assignee: {owner}",
        f"Created: {str(epic.get('created') or '')[:10] or 'unknown'}",
    ]
    description = _clean(epic.get("description")) or "no description on the ticket"
    lines = [
        f"### {epic['key']} {_clean(epic.get('summary'))}",
        "- " + " · ".join(details),
        f"- Progress: {progress.get('done', 0)}/{progress.get('total', 0)} done, "
        f"{progress.get('in_progress', 0)} in progress, "
        f"{'n/a' if pct is None else str(pct) + '%'}",
        f"- Description: {description}",
    ]
    if with_open_children:
        lines += _render_open_children(epic.get("children") or [])
    return "\n".join(lines)


def _render_epics(epics, count_jql, with_open_children=False):
    lines = []
    if count_jql:
        lines.append(
            f"Count JQL template; substitute the epic key: total `{count_jql}`. "
            "Done adds `AND statusCategory = Done`; in progress adds "
            "`AND statusCategory = \"In Progress\"`."
        )
        lines.append("")
    blocks = [_render_epic(epic, with_open_children) for epic in epics]
    return "\n".join(["## Active epics", ""] + lines + ["\n\n".join(blocks) if blocks else "None."])


def _render_flagged(flagged, child_map):
    lines = []
    cut = 0
    for index, item in enumerate(flagged):
        parts = [_clean(item.get("status")), _clean(item.get("assignee")) or "unassigned",
                 _clean(item.get("reason"))]
        if item.get("age_days") is not None:
            parts.append(f"{item['age_days']} days in status category")
        if item["key"] in child_map:
            parts.append(f"epic {child_map[item['key']]}")
        lines.append(f"- {item['key']} {_clean(item.get('summary'))} — " + " · ".join(parts))
        comments = item.get("comments") or []
        shown = comments[-EXCERPTS_PER_FLAGGED:] if index < FLAGGED_WITH_EXCERPTS else []
        cut += len(comments) - len(shown)
        for comment in shown:
            lines.append(f"  - {_clean(comment.get('author'))} {str(comment.get('created') or '')[:10]}: "
                         f"{_clean(comment.get('excerpt'))}")
    return _section("## Flagged", lines), cut


def _render_unassigned(items):
    return _section("## Unassigned", [
        f"- {item['key']} · {_clean(item.get('type'))} · {_clean(item.get('status'))} · "
        f"{_clean(item.get('priority')) or 'no priority'} · "
        f"{_truncate(_clean(item.get('summary')), SUMMARY_LIMIT)}"
        for item in items
    ])


def _render_high_priority(items):
    return _section("## High priority outside epics", [
        f"- {item['key']} · {_clean(item.get('type'))} · {_clean(item.get('status'))} · "
        f"{_clean(item.get('priority'))} · {_clean(item.get('assignee')) or 'unassigned'} · "
        f"{_truncate(_clean(item.get('summary')), SUMMARY_LIMIT)}"
        for item in items
    ])


def _render_questions(questions):
    return _section("## Question candidates", [
        f"- {item['key']} · {_clean(item.get('author'))}"
        f"{' → ' + ', '.join(_clean(name) for name in item['addressees']) if item.get('addressees') else ''}"
        f" · {str(item.get('created') or '')[:10]} · "
        f"{item.get('age_days')} days: {_clean(item.get('excerpt'))}"
        for item in questions
    ])


def _render_doc_links(links):
    return _section("## Doc links", [
        f"- {link['url']} ({link['epic']}, {link['kind']})" for link in links
    ])


def _person_detail(name, groups):
    lines = [f"### {_clean(name)}"]
    entries = [(group, entry) for group in PERSON_GROUPS for entry in groups.get(group, [])]
    if not entries:
        lines.append("- No activity in the window.")
    for group, entry in entries:
        lines.append(f"- {GROUP_LABELS[group]}: {entry['key']} {_clean(entry.get('summary'))} "
                     f"({_clean(entry.get('status'))})")
    return "\n".join(lines)


def _person_counts(name, groups):
    counts = ", ".join(f"{len(groups.get(group, []))} {GROUP_LABELS[group].lower()}"
                       for group in PERSON_GROUPS)
    keys = [entry["key"] for group in PERSON_GROUPS for entry in groups.get(group, [])]
    return f"- {_clean(name)}: {counts}. Keys: {', '.join(keys) or 'none'}"


def _render_people(heading, people, collapsed):
    if collapsed:
        lines = [_person_counts(name, groups) for name, groups in people.items()]
        return _section(heading, lines)
    blocks = [_person_detail(name, groups) for name, groups in people.items()]
    return "\n".join([heading, ""] + ["\n\n".join(blocks) if blocks else "None."])


def _render_children(epics, collapsed):
    blocks = []
    for epic in epics:
        children = epic.get("children") or []
        if collapsed:
            keys = ", ".join(child["key"] for child in children) or "none"
            blocks.append(f"### {epic['key']}\n- {len(children)} children. Keys: {keys}")
            continue
        lines = [f"### {epic['key']}"] + [
            f"- {child['key']} {_clean(child.get('summary'))} ({_clean(child.get('status'))}) · "
            f"{_clean(child.get('assignee')) or 'unassigned'}"
            for child in children
        ]
        blocks.append("\n".join(lines))
    return "\n".join(["## Children", ""] + ["\n\n".join(blocks) if blocks else "None."])


def _render_not_started(epics, collapsed):
    ordered = sorted(epics, key=lambda epic: -(epic.get("age_days") or 0))
    kept = ordered[:NOT_STARTED_KEPT] if collapsed else ordered
    lines = [f"- {epic['key']} {_clean(epic.get('summary'))} · created "
             f"{str(epic.get('created') or '')[:10]} · {epic.get('age_days')} days old"
             for epic in kept]
    dropped = len(ordered) - len(kept)
    if dropped:
        lines.append(f"- and {dropped} more not-started epics")
    return _section("## Not started", lines), dropped


def _scoped(data, scope, person):
    epics = list(data.get("epics") or [])
    flagged = list(data.get("flagged") or [])
    questions = list(data.get("questions") or [])
    links = list(data.get("doc_links") or [])
    not_started = list(data.get("not_started") or [])
    unassigned = list(data.get("unassigned") or [])
    high_priority = list(data.get("high_priority") or [])
    people = dict(data.get("people") or {})
    child_map = data.get("child_to_epic") or {}
    if scope == "person":
        people = {name: groups for name, groups in people.items() if _same_person(name, person)}
        own_keys = {entry["key"] for groups in people.values()
                    for group in PERSON_GROUPS for entry in groups.get(group, [])}
        own_flagged = [item for item in flagged if _same_person(item.get("assignee"), person)]
        touched = {child_map.get(key, key) for key in own_keys}
        epics = [epic for epic in epics
                 if epic["key"] in touched
                 or _same_person((epic.get("assignee") or {}).get("name"), person)
                 or any(_same_person(child.get("assignee"), person)
                        for child in epic.get("children") or [])]
        keys = own_keys | {item["key"] for item in own_flagged}
        flagged = own_flagged
        questions = [item for item in questions if item["key"] in keys]
        not_started = [epic for epic in not_started if _same_person(epic.get("assignee"), person)]
        high_priority = [item for item in high_priority if _same_person(item.get("assignee"), person)]
    elif scope == "epic":
        high_priority = []
        wanted = data.get("epic")
        if wanted:
            epics = [epic for epic in epics if epic["key"] == wanted]
        in_epic = {epic["key"] for epic in epics}
        flagged = [item for item in flagged
                   if item["key"] in in_epic or child_map.get(item["key"]) in in_epic]
        questions = [item for item in questions
                     if item["key"] in in_epic or child_map.get(item["key"]) in in_epic]
    epic_keys = {epic["key"] for epic in epics}
    if scope != "team":
        links = [link for link in links if link["epic"] in epic_keys]
        unassigned = [item for item in unassigned if child_map.get(item["key"]) in epic_keys]
    return epics, flagged, unassigned, high_priority, questions, links, not_started, people


def _word_count(parts):
    return sum(len(part.split()) for part in parts)


def _detail_section(scope, epics, people, collapsed):
    if scope == "epic":
        cut = sum(len(epic.get("children") or []) for epic in epics) if collapsed else 0
        return _render_children(epics, collapsed), cut
    heading = "## By person" if scope == "team" else "## Issues"
    cut = sum(len(groups.get(group, [])) for groups in people.values()
              for group in PERSON_GROUPS) if collapsed else 0
    return _render_people(heading, people, collapsed), cut


def _omitted_line(omitted, issues_cut, not_started_cut, comments_cut, word_limit):
    return (f"Omitted: {(omitted.get('issues') or 0) + issues_cut} issues, "
            f"{not_started_cut} not-started epics, "
            f"{(omitted.get('comments') or 0) + comments_cut} comments (word limit {word_limit})")


def _header(data, scope, person):
    window = data.get("window") or {}
    scope_line = f"Scope: {scope}" + (f" ({_clean(person)})" if person else "")
    return (f"# Jira digest: {', '.join(window.get('keys') or [])}, "
            f"{window.get('since', '')} to {window.get('until', '')}\n\n{scope_line}")


def render_digest(data: dict, scope: str, word_limit: int, person: str | None = None) -> str:
    if scope not in SCOPES:
        raise ValueError(f"unknown scope: {scope}")
    if scope == "person" and not person:
        raise ValueError("person scope needs a person")
    epics, flagged, unassigned, high_priority, questions, links, not_started, people = _scoped(
        data, scope, person)
    flagged_text, comments_cut = _render_flagged(flagged, data.get("child_to_epic") or {})
    never = [_header(data, scope, person)]
    if data.get("failures"):
        never.append(_section("## Not measured", [f"- {key}" for key in data["failures"]]))
    never += [
        _render_epics(epics, data.get("count_jql"), with_open_children=scope == "team"),
        flagged_text,
        _render_unassigned(unassigned),
        _render_high_priority(high_priority),
        _render_questions(questions),
        _section("## Unmatched assignees", [f"- {_clean(name)}" for name in data.get("unmatched") or []]),
        _render_doc_links(links),
    ]
    omitted = data.get("omitted") or {}
    for collapse_detail, collapse_not_started in ((False, False), (True, False), (True, True)):
        detail_text, issues_cut = _detail_section(scope, epics, people, collapse_detail)
        parts = never + [detail_text]
        not_started_cut = 0
        if scope != "epic":
            not_started_text, not_started_cut = _render_not_started(not_started, collapse_not_started)
            parts.append(not_started_text)
        parts.append(_omitted_line(omitted, issues_cut, not_started_cut, comments_cut, word_limit))
        if _word_count(parts) <= word_limit:
            break
    return "\n\n".join(parts) + "\n"


BANNER = re.compile(r"^\W*(?:You[’']re using an outdated version|Follow this link:)")
JSON_START = re.compile(r"^[ \t]*[\[{]", re.MULTILINE)
RATE_LIMITED = re.compile(r"\b429\b")
COUNT_LINE = re.compile(r"(\d+)\s*$")


class AcliError(Exception):
    pass


ACLI_ATTEMPTS = 2


def _acli_text(args, runner):
    command = ["acli"] + list(args)
    for attempt in range(1, ACLI_ATTEMPTS + 1):
        try:
            result = runner(command, capture_output=True, text=True)
        except FileNotFoundError as error:
            raise AcliError(f"acli not found on PATH: {error}") from error
        stderr = (result.stderr or "").strip()
        if RATE_LIMITED.search(stderr):
            raise AcliError(f"Jira rate limit hit (HTTP 429) on: {' '.join(command)}\n{stderr}")
        if result.returncode == 0:
            return result.stdout or ""
    raise AcliError(f"acli failed: {' '.join(command)}\n{stderr}")


def _without_banner(text):
    return "\n".join(line for line in text.splitlines() if not BANNER.match(line))


def run_acli(args: list[str], runner=subprocess.run) -> list | dict:
    text = _acli_text(args, runner)
    start = JSON_START.search(text)
    try:
        if start:
            return json.JSONDecoder().raw_decode(text, start.end() - 1)[0]
        return json.loads(_without_banner(text))
    except json.JSONDecodeError as error:
        raise AcliError(f"acli returned non-JSON output for {' '.join(args)}: {error}") from error


def search(jql: str, fields: str, runner=subprocess.run) -> list[dict]:
    result = run_acli(["jira", "workitem", "search", "--jql", jql, "--fields", fields,
                       "--paginate", "--json"], runner=runner)
    return [item for item in result or [] if item]


def view(key: str, fields: str, runner=subprocess.run) -> dict:
    return run_acli(["jira", "workitem", "view", key, "--fields", fields, "--json"], runner=runner)


def count(jql: str, runner=subprocess.run) -> int:
    text = _without_banner(_acli_text(["jira", "workitem", "search", "--jql", jql, "--count"],
                                      runner)).strip()
    match = COUNT_LINE.search(text)
    if not match:
        raise AcliError(f"acli count gave no number for {jql}: {text}")
    return int(match.group(1))


ITEM_FIELDS = "key,summary,status,assignee,issuetype,priority"
EPIC_FIELDS = "key,summary,status,assignee,description,priority"
EPIC_VIEW_FIELDS = "summary,description,priority,parent,assignee,status,created,updated,comment"
NOT_STARTED_VIEW_FIELDS = "created,summary,description,assignee,priority"
FLAGGED_VIEW_FIELDS = "comment,statuscategorychangedate"
DESCRIPTION_LIMIT = 200
COMMENT_LIMIT = 300
QUESTION_EXCERPT_LIMIT = 200

WINDOW_JQL = ('project in ({keys}) AND updated >= "{since}" AND updated <= "{until} 23:59" '
              'ORDER BY updated DESC')
EPICS_JQL = ('project in ({keys}) AND issuetype = Epic AND (statusCategory = "In Progress" '
             'OR updated >= "{since}")')
CHILDREN_JQL = "parent = {epic} AND (resolution is EMPTY OR resolution not in ({excluded}))"
STALLED_JQL = ('project in ({keys}) AND statusCategory = "In Progress" AND NOT status CHANGED '
               'AFTER -5d ORDER BY statusCategoryChangedDate ASC')
NOT_STARTED_JQL = 'project in ({keys}) AND issuetype = Epic AND statusCategory = "To Do"'
NOT_STARTED_DONE_JQL = ("parent = {epic} AND statusCategory = Done AND "
                        "(resolution is EMPTY OR resolution not in ({excluded}))")


def _parallel(tasks, workers):
    results, errors = {}, {}
    if not tasks:
        return results, errors
    with ThreadPoolExecutor(max_workers=max(1, min(workers, len(tasks)))) as pool:
        futures = {name: pool.submit(call) for name, call in tasks.items()}
        for name, future in futures.items():
            try:
                results[name] = future.result()
            except AcliError as error:
                errors[name] = str(error)
    return results, errors


def _quote_jql(value):
    return '"' + value.replace("\\", "\\\\").replace('"', '\\"') + '"'


def _fields(issue):
    return issue.get("fields") or {}


def _assignee_name(assignee, roster):
    if not assignee:
        return None
    person = match_assignee(assignee, roster)
    if person:
        return person["name"]
    return assignee.get("displayName") or assignee.get("accountId")


def _is_epic_type(issuetype):
    return ((issuetype or {}).get("name") or "").casefold() == "epic"


def _child_entry(issue, roster):
    fields = _fields(issue)
    return {"key": issue["key"], "summary": fields.get("summary") or "",
            "status": _status_name(issue),
            "assignee": _assignee_name(fields.get("assignee"), roster),
            "category": _category(issue)}


def _epic_entry(key, fields, children, children_jql, roster):
    parent = fields.get("parent")
    assignee = fields.get("assignee")
    owner = None
    if assignee:
        owner = {"name": _assignee_name(assignee, roster), "active": assignee.get("active")}
    return {
        "key": key,
        "summary": fields.get("summary") or "",
        "created": fields.get("created") or "",
        "status": ((fields.get("status") or {}).get("name")) or "",
        "priority": (fields.get("priority") or {}).get("name"),
        "parent": ({"key": parent["key"],
                    "summary": (parent.get("fields") or {}).get("summary") or ""}
                   if parent else None),
        "assignee": owner,
        "description": adf_text(fields.get("description"), DESCRIPTION_LIMIT),
        "progress": epic_progress(children),
        "jql": {"total": children_jql,
                "done": children_jql + " AND statusCategory = Done",
                "in_progress": children_jql + ' AND statusCategory = "In Progress"'},
        "children": [_child_entry(child, roster) for child in children],
    }


def _comments(view_result):
    return list((_fields(view_result).get("comment") or {}).get("comments") or [])


def _by_created(comments):
    return sorted((comment for comment in comments if comment.get("created")),
                  key=lambda comment: parse_jira_time(comment["created"]))


def _comment_entry(comment):
    return {"author": (comment.get("author") or {}).get("displayName") or "unknown",
            "created": comment.get("created") or "",
            "excerpt": _comment_text(comment.get("body"), COMMENT_LIMIT)}


def collect(cfg: dict, runner=subprocess.run, workers: int = 8) -> dict:
    keys = ", ".join(cfg["keys"])
    since, until = cfg["since"], cfg["until"]
    roster = cfg.get("roster") or []
    excluded = ", ".join(_quote_jql(value) for value in cfg["excluded"])
    count_jql = CHILDREN_JQL.format(epic="{EPIC}", excluded=excluded)
    failures, errors = [], {}

    def children_jql(epic):
        return CHILDREN_JQL.format(epic=epic, excluded=excluded)

    def record(failed):
        for name, message in failed.items():
            key = name[1]
            if key not in failures:
                failures.append(key)
            errors.setdefault(key, message)

    first, fatal = _parallel({
        "window": lambda: search(WINDOW_JQL.format(keys=keys, since=since, until=until),
                                 ITEM_FIELDS, runner),
        "epics": lambda: search(EPICS_JQL.format(keys=keys, since=since), EPIC_FIELDS, runner),
        "stalled": lambda: search(STALLED_JQL.format(keys=keys), ITEM_FIELDS, runner),
        "not_started": lambda: search(NOT_STARTED_JQL.format(keys=keys), EPIC_FIELDS, runner),
    }, workers)
    if fatal:
        raise AcliError("; ".join(f"{name}: {message}" for name, message in fatal.items()))
    window, stalled = first["window"], first["stalled"]
    epic_records = {epic["key"]: epic for epic in first["epics"]}
    epic_order = list(epic_records)
    if cfg.get("epic") and cfg["epic"] not in epic_records:
        epic_order.append(cfg["epic"])
    not_started = first["not_started"]

    def epic_tasks(epic_keys):
        tasks = {}
        for key in epic_keys:
            tasks[("children", key)] = lambda key=key: search(children_jql(key), ITEM_FIELDS, runner)
            tasks[("epic", key)] = lambda key=key: view(key, EPIC_VIEW_FIELDS, runner)
        return tasks

    tasks = epic_tasks(epic_order)
    for epic in not_started:
        tasks[("done", epic["key"])] = lambda key=epic["key"]: count(
            NOT_STARTED_DONE_JQL.format(epic=key, excluded=excluded), runner)
    second, failed = _parallel(tasks, workers)
    record(failed)

    children_by_epic = {key: second[("children", key)] for key in epic_order
                        if ("children", key) in second}
    claimed = child_to_epic(children_by_epic)
    unclaimed = [issue["key"] for issue in window
                 if issue["key"] not in claimed and issue["key"] not in epic_order
                 and not _is_epic_type(_fields(issue).get("issuetype"))]
    parents, failed = _parallel({("parent", key): lambda key=key: view(key, "parent", runner)
                                 for key in unclaimed}, workers)
    record(failed)
    extra = []
    for key in unclaimed:
        parent = _fields(parents.get(("parent", key)) or {}).get("parent")
        if (parent and _is_epic_type((parent.get("fields") or {}).get("issuetype"))
                and parent["key"] not in epic_order and parent["key"] not in extra):
            extra.append(parent["key"])
    epic_order += extra
    has_parent = {key for key in unclaimed
                  if _fields(parents.get(("parent", key)) or {}).get("parent")}

    active = set(epic_order)
    zero_done = [epic["key"] for epic in not_started
                 if second.get(("done", epic["key"])) == 0 and epic["key"] not in active]
    tasks = epic_tasks(extra)
    for key in zero_done:
        tasks[("not_started", key)] = lambda key=key: view(key, NOT_STARTED_VIEW_FIELDS, runner)
    third, failed = _parallel(tasks, workers)
    record(failed)
    second.update(third)

    children_by_epic = {key: second[("children", key)] for key in epic_order
                        if ("children", key) in second}
    child_map = child_to_epic(children_by_epic)
    stalled_keys = {issue["key"] for issue in stalled}

    def in_scope(issue):
        return (match_assignee(_fields(issue).get("assignee"), roster) is not None
                or issue["key"] in child_map)

    candidates, seen = [], set()
    pool = window + [child for children in children_by_epic.values() for child in children]
    for issue in pool:
        if issue["key"] not in seen and is_blocked(issue) and in_scope(issue):
            candidates.append((issue, "blocked"))
            seen.add(issue["key"])
    for issue in stalled:
        if issue["key"] not in seen and in_scope(issue):
            candidates.append((issue, "stalled"))
            seen.add(issue["key"])
    max_flagged = cfg.get("max_flagged", 60)
    kept = candidates[:max_flagged]
    comment_views, failed = _parallel({("comment", issue["key"]): lambda key=issue["key"]: view(
        key, FLAGGED_VIEW_FIELDS, runner) for issue, _ in kept}, workers)
    record(failed)

    now = datetime.now(timezone.utc)
    flagged, questions, comments_cut = [], [], 0
    for issue, reason in kept:
        flagged_view = comment_views.get(("comment", issue["key"])) or {}
        changed = _fields(flagged_view).get("statuscategorychangedate")
        comments = _by_created(_comments(flagged_view))
        comments_cut += max(len(comments) - MAX_COMMENTS, 0)
        fields = _fields(issue)
        flagged.append({"key": issue["key"], "summary": fields.get("summary") or "",
                        "status": _status_name(issue),
                        "assignee": _assignee_name(fields.get("assignee"), roster),
                        "reason": reason,
                        "age_days": (now - parse_jira_time(changed)).days if changed else None,
                        "comments": [_comment_entry(comment)
                                     for comment in comments[-MAX_COMMENTS:]]})
        questions += [{"key": issue["key"], **candidate}
                      for candidate in question_candidates(comments, QUESTION_EXCERPT_LIMIT)]

    epics, doc_links = [], []
    for key in epic_order:
        if ("children", key) not in second:
            continue
        fields = dict(_fields(epic_records.get(key) or {}))
        fields.update(_fields(second.get(("epic", key)) or {}))
        epics.append(_epic_entry(key, fields, second[("children", key)], children_jql(key), roster))
        for url in adf_links(fields.get("description")):
            kind = classify_doc_link(url)
            if kind:
                doc_links.append({"url": url, "epic": key, "kind": kind})
        questions += [{"key": key, **candidate} for candidate in
                      question_candidates(_comments(second.get(("epic", key)) or {}),
                                          QUESTION_EXCERPT_LIMIT)]

    questions = list({item["key"]: item for item in reversed(questions)}.values())[::-1]

    not_started_entries = []
    for key in zero_done:
        fields = dict(_fields(next(epic for epic in not_started if epic["key"] == key)))
        fields.update(_fields(second.get(("not_started", key)) or {}))
        created = fields.get("created") or ""
        not_started_entries.append({
            "key": key, "summary": fields.get("summary") or "", "created": created,
            "age_days": (now - parse_jira_time(created)).days if created else None,
            "assignee": _assignee_name(fields.get("assignee"), roster),
        })

    stalled_window = [dict(issue, stalled=issue["key"] in stalled_keys) for issue in window
                      if not _is_epic_type(_fields(issue).get("issuetype"))]
    people, unmatched = group_by_person(stalled_window, roster)
    unassigned = unassigned_issues(pool)
    high_priority = high_priority_outside_epics(
        [issue for issue in window if not _is_epic_type(_fields(issue).get("issuetype"))],
        child_map, has_parent)
    window_by_key = {issue["key"]: issue for issue in window}
    for item in high_priority:
        item["assignee"] = _assignee_name(_fields(window_by_key[item["key"]]).get("assignee"), roster)
    data = {
        "window": {"since": since, "until": until, "keys": list(cfg["keys"])},
        "count_jql": count_jql,
        "epics": epics,
        "child_to_epic": child_map,
        "people": people,
        "unmatched": unmatched,
        "unassigned": unassigned,
        "high_priority": high_priority,
        "flagged": flagged,
        "questions": questions,
        "doc_links": doc_links,
        "not_started": not_started_entries,
        "failures": failures,
        "errors": errors,
        "omitted": {"issues": len(candidates) - len(kept), "comments": comments_cut},
        "counts": {"window": len(window), "stalled": len(stalled),
                   "blocked": sum(1 for _, reason in candidates if reason == "blocked"),
                   "flagged_candidates": len(candidates)},
    }
    if cfg.get("scope") == "epic" and cfg.get("epic"):
        data["epic"] = cfg["epic"]
    return data


def _split(value):
    return [part.strip().strip("\"'").strip() for part in value.split(",") if part.strip()]


def build_parser():
    parser = argparse.ArgumentParser(
        description="Collect Jira epics, issues and flagged items through acli into "
                    "jira.json and a size-bounded jira.md digest.")
    parser.add_argument("--keys", required=True, help="comma-separated Jira project keys")
    parser.add_argument("--since", required=True, help="YYYY-MM-DD")
    parser.add_argument("--until", default=None, help="YYYY-MM-DD (default: today, UTC)")
    parser.add_argument("--config", required=True, metavar="PATH",
                        help="team config markdown holding the roster table")
    parser.add_argument("--scope", choices=SCOPES, default="team")
    parser.add_argument("--person", default=None, help="roster name; required with --scope person")
    parser.add_argument("--epic", default=None, help="epic key; required with --scope epic")
    parser.add_argument("--excluded-resolutions", required=True,
                        help="comma-separated resolutions left out of epic counts, from the "
                             "team config; every value must exist on the Jira site")
    parser.add_argument("--word-limit", type=int, required=True,
                        help="digest word target, 50 x people x days")
    parser.add_argument("--max-flagged", type=int, default=60,
                        help="most blocked or stalled items whose comments are fetched")
    parser.add_argument("--workers", type=int, default=8, help="parallel acli calls")
    parser.add_argument("--out", default=".", help="directory for jira.json and jira.md")
    return parser


def main(argv: list[str] | None = None, runner=subprocess.run) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    if args.scope == "person" and not args.person:
        parser.error("--scope person needs --person")
    if args.scope == "epic" and not args.epic:
        parser.error("--scope epic needs --epic")
    keys = [key.upper() for key in _split(args.keys)]
    excluded = _split(args.excluded_resolutions)
    if not keys or not excluded:
        parser.error("--keys and --excluded-resolutions need at least one value")
    try:
        with open(args.config) as handle:
            roster = parse_roster(handle.read())
    except (OSError, ValueError) as error:
        print(f"cannot read --config {args.config}: {error}", file=sys.stderr)
        return 1
    try:
        _acli_text(["jira", "auth", "status"], runner)
    except AcliError as error:
        print(f"acli unavailable: {error}", file=sys.stderr)
        return 1
    cfg = {"keys": keys, "since": args.since,
           "until": args.until or datetime.now(timezone.utc).strftime("%Y-%m-%d"),
           "excluded": excluded, "roster": roster, "max_flagged": args.max_flagged,
           "scope": args.scope, "person": args.person, "epic": args.epic}
    try:
        data = collect(cfg, runner=runner, workers=args.workers)
    except AcliError as error:
        print(f"acli unavailable: {error}", file=sys.stderr)
        return 1

    os.makedirs(args.out, exist_ok=True)
    json_path = os.path.join(args.out, "jira.json")
    md_path = os.path.join(args.out, "jira.md")
    with open(json_path, "w") as handle:
        json.dump(data, handle, indent=2, ensure_ascii=False)
    with open(md_path, "w") as handle:
        handle.write(render_digest(data, args.scope, args.word_limit, args.person))

    print(f"{json_path}\n{md_path}")
    print(f"window={data['counts']['window']} stalled={data['counts']['stalled']} "
          f"epics={len(data['epics'])} children={len(data['child_to_epic'])} "
          f"flagged={len(data['flagged'])} not_started={len(data['not_started'])} "
          f"unmatched={len(data['unmatched'])}")
    if data["failures"]:
        for key in data["failures"]:
            print(f"{key}: {data['errors'].get(key, '')}", file=sys.stderr)
        print(f"INCOMPLETE: {', '.join(data['failures'])}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
