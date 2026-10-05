#!/usr/bin/env python3

import re
from datetime import datetime, timezone
from urllib.parse import urlparse

PLACEHOLDER = re.compile(r"^\[.*\]$")
ELLIPSIS = "…"
CARD_TYPES = ("inlineCard", "blockCard", "embedCard")
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


def parse_jira_time(value):
    text = re.sub(r"([+-]\d{2})(\d{2})$", r"\1:\2", value.replace("Z", "+00:00"))
    return datetime.fromisoformat(text)


def _comment_text(body, limit):
    if isinstance(body, str):
        return _truncate(" ".join(body.split()), limit)
    return adf_text(body, limit)


def question_candidates(comments: list[dict], limit: int) -> list[dict]:
    dated = [comment for comment in comments if comment.get("created")]
    if not dated:
        return []
    newest = max(dated, key=lambda comment: parse_jira_time(comment["created"]))
    full = _comment_text(newest.get("body"), 10 ** 9)
    if "?" not in full:
        return []
    created = parse_jira_time(newest["created"])
    return [{
        "author": (newest.get("author") or {}).get("displayName") or "unknown",
        "created": newest["created"],
        "age_days": (datetime.now(timezone.utc) - created).days,
        "excerpt": _truncate(full, limit),
    }]


SCOPES = ("team", "person", "epic")
GROUP_LABELS = {"completed": "Completed", "in_progress": "In progress", "stuck": "Stuck",
                "to_do": "To do"}
MAX_COMMENTS = 3
NOT_STARTED_KEPT = 10


def _clean(text):
    return " ".join(str(text or "").split()).replace('{"', '{ "')


def _same_person(left, right):
    return bool(left) and bool(right) and left.casefold() == right.casefold()


def _section(heading, lines):
    return "\n".join([heading, ""] + (lines or ["None."]))


def _render_epic(epic):
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
    ]
    description = _clean(epic.get("description")) or "no description on the ticket"
    return "\n".join([
        f"### {epic['key']} {_clean(epic.get('summary'))}",
        "- " + " · ".join(details),
        f"- Progress: {progress.get('done', 0)}/{progress.get('total', 0)} done, "
        f"{progress.get('in_progress', 0)} in progress, "
        f"{'n/a' if pct is None else str(pct) + '%'}",
        f"- Description: {description}",
    ])


def _render_epics(epics, count_jql):
    lines = []
    if count_jql:
        lines.append(
            f"Count JQL template; substitute the epic key: total `{count_jql}`. "
            "Done adds `AND statusCategory = Done`; in progress adds "
            "`AND statusCategory = \"In Progress\"`."
        )
        lines.append("")
    blocks = [_render_epic(epic) for epic in epics]
    return "\n".join(["## Active epics", ""] + lines + ["\n\n".join(blocks) if blocks else "None."])


def _render_flagged(flagged, child_map):
    lines = []
    cut = 0
    for item in flagged:
        parts = [_clean(item.get("status")), _clean(item.get("assignee")) or "unassigned",
                 _clean(item.get("reason"))]
        if item["key"] in child_map:
            parts.append(f"epic {child_map[item['key']]}")
        lines.append(f"- {item['key']} {_clean(item.get('summary'))} — " + " · ".join(parts))
        comments = item.get("comments") or []
        cut += max(len(comments) - MAX_COMMENTS, 0)
        for comment in comments[:MAX_COMMENTS]:
            lines.append(f"  - {_clean(comment.get('author'))} {str(comment.get('created') or '')[:10]}: "
                         f"{_clean(comment.get('excerpt'))}")
    return _section("## Flagged", lines), cut


def _render_questions(questions):
    return _section("## Question candidates", [
        f"- {item['key']} · {_clean(item.get('author'))} · {str(item.get('created') or '')[:10]} · "
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
    elif scope == "epic":
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
    return epics, flagged, questions, links, not_started, people


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
    epics, flagged, questions, links, not_started, people = _scoped(data, scope, person)
    flagged_text, comments_cut = _render_flagged(flagged, data.get("child_to_epic") or {})
    never = [_header(data, scope, person)]
    if data.get("failures"):
        never.append(_section("## Not measured", [f"- {key}" for key in data["failures"]]))
    never += [
        _render_epics(epics, data.get("count_jql")),
        flagged_text,
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
