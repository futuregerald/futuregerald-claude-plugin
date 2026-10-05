#!/usr/bin/env python3
"""Pull PRs from GitHub and reconcile them against tracker ticket keys.

Emits prs.json and prs.md describing which PRs carry a ticket key from the
configured project scope, which carry someone else's key, and which carry none
at all. The last group is work the tracker cannot see.
"""

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
from datetime import datetime, timezone

KEY_PATTERN = re.compile(r"\b[A-Z][A-Z0-9]{1,9}-\d+\b")
NO_TICKET_PATTERN = re.compile(r"(?i)(?:^|[^A-Z0-9])no-ticket(?:[^A-Z0-9]|$)")
SEARCH_API_CAP = 1000

NOT_TICKET_PREFIXES = frozenset({
    "CVE", "GHSA", "RFC", "PEP", "ADR", "RCE", "XSS", "CSRF", "SSRF",
    "UTF", "ASCII", "AES", "RSA", "SHA", "MD", "HMAC", "TLS", "SSL", "SSH",
    "HTTP", "HTTPS", "TCP", "UDP", "IP", "IPV", "DNS", "URL", "URI",
    "ISO", "SOC", "PCI", "DSS", "GDPR", "HIPAA", "NIST", "OWASP", "CWE", "CVSS",
    "JSON", "XML", "YAML", "CSV", "SQL", "API", "UI", "UX", "CSS", "HTML",
    "AMD", "ARM", "X", "V", "PR", "GB", "MB", "KB", "PX", "EM",
})


def is_ticket_key(candidate, scope=()):
    prefix = candidate.split("-", 1)[0]
    if prefix in scope:
        return True
    return prefix not in NOT_TICKET_PREFIXES

JSON_FIELDS = (
    "number,title,headRefName,body,author,createdAt,mergedAt,closedAt,"
    "isDraft,reviewDecision,url,additions,deletions"
)

REVIEW_LABELS = {
    "APPROVED": "approved, not merged",
    "CHANGES_REQUESTED": "changes requested",
    "REVIEW_REQUIRED": "review required",
}


class TruncatedError(RuntimeError):
    pass


class GhError(RuntimeError):
    pass


def extract_keys(title, branch, body, scope=()):
    haystack = " ".join(part or "" for part in (title, branch, body))
    found = {key for key in KEY_PATTERN.findall(haystack) if is_ticket_key(key, scope)}
    return sorted(found)


def classify(keys, branch, scope, title="", body=""):
    if keys:
        prefixes = {key.split("-", 1)[0] for key in keys}
        if prefixes & set(scope):
            return "linked_in_scope"
        return "linked_out_of_scope"
    declared = " ".join(part or "" for part in (title, branch, body))
    if NO_TICKET_PATTERN.search(declared):
        return "declared_no_ticket"
    return "no_ticket"


def is_stale(state, is_draft, age_days, review, limit, author_is_bot=False):
    if state != "open" or is_draft or author_is_bot:
        return False
    return age_days > limit


def is_stale_unreviewed(stale, review):
    return stale and review in ("", "REVIEW_REQUIRED")


def is_team_pr(pr, roster):
    if pr.get("author", "").lower() in roster:
        return True
    return pr.get("bucket") == "linked_in_scope"


def effective_limit(state, limit):
    if state in ("merged", "closed"):
        return min(limit, SEARCH_API_CAP)
    return limit


def check_not_truncated(items, limit, repo, state):
    cap = effective_limit(state, limit)
    if len(items) >= cap:
        raise TruncatedError(
            f"TRUNCATED: {repo} {state} returned {len(items)} items, at or above the "
            f"effective cap of {cap}. Results are incomplete. Narrow the window with a "
            f"later --since, or scan fewer repos per run. Raising --limit will NOT help "
            f"for merged or closed PRs: GitHub's search API stops at {SEARCH_API_CAP} regardless."
        )


def _repo_in(repo, names):
    lowered = {name.lower() for name in names}
    repo = repo.lower()
    return repo in lowered or repo.split("/", 1)[-1] in lowered


def classify_stack(repo: str, frontend: set[str], backend: set[str]) -> str:
    if _repo_in(repo, frontend):
        return "frontend"
    if _repo_in(repo, backend):
        return "backend"
    return "other"


def load_jira_map(path: str) -> dict[str, str]:
    with open(path) as handle:
        data = json.load(handle)
    if not isinstance(data, dict):
        raise ValueError(f"{path} is not a JSON object")
    return dict(data.get("child_to_epic") or {})


def epic_for(pr: dict, jira_map: dict[str, str]) -> str | None:
    keys = pr.get("keys") or []
    for key in keys:
        if key in jira_map:
            return jira_map[key]
    epics = set(jira_map.values())
    for key in keys:
        if key in epics:
            return key
    return None


def build_epic_rollup(prs: list[dict], frontend: set[str], backend: set[str],
                      jira_map: dict[str, str]) -> dict[str, dict]:
    rollup = {}
    for pr in prs:
        if jira_map:
            epic = epic_for(pr, jira_map)
            targets = [epic] if epic else []
        elif pr.get("bucket") == "linked_in_scope":
            targets = pr["keys"]
        else:
            targets = []
        stack = classify_stack(pr["repo"], frontend, backend)
        state = "merged" if pr.get("merged_at") else "open"
        for target in targets:
            entry = rollup.setdefault(target, {"frontend": {"merged": 0, "open": 0},
                                               "backend": {"merged": 0, "open": 0}})
            entry.setdefault(stack, {"merged": 0, "open": 0})[state] += 1
    return rollup


def build_stack_counts(prs: list[dict], frontend: set[str], backend: set[str],
                       roster: set[str]) -> dict[str, dict]:
    roster = {name.lower() for name in roster}
    stacks = {stack: {"merged": 0, "open": 0, "additions": 0, "deletions": 0}
              for stack in ("frontend", "backend", "other")}
    for pr in prs:
        if not is_team_pr(pr, roster):
            continue
        entry = stacks[classify_stack(pr["repo"], frontend, backend)]
        entry["merged" if pr.get("merged_at") else "open"] += 1
        entry["additions"] += pr.get("additions") or 0
        entry["deletions"] += pr.get("deletions") or 0
    return stacks


def _parse_ts(value):
    if not value:
        return None
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def build_pr(raw, repo, state, scope, stale_days, now):
    title = raw.get("title") or ""
    branch = raw.get("headRefName") or ""
    body = raw.get("body") or ""
    keys = extract_keys(title, branch, body, scope)
    created = _parse_ts(raw.get("createdAt"))
    age_days = (_parse_ts(now) - created).days if created else 0
    review = raw.get("reviewDecision") or ""
    is_draft = bool(raw.get("isDraft"))
    author = raw.get("author") or {}
    author_is_bot = bool(author.get("is_bot"))
    stale = is_stale(state, is_draft, age_days, review, stale_days, author_is_bot)
    return {
        "repo": repo,
        "number": raw.get("number"),
        "url": raw.get("url"),
        "title": title,
        "author": author.get("login") or "unknown",
        "author_is_bot": author_is_bot,
        "branch": branch,
        "created_at": raw.get("createdAt"),
        "merged_at": raw.get("mergedAt"),
        "closed_at": raw.get("closedAt"),
        "is_draft": is_draft,
        "review_decision": review,
        "age_days": age_days,
        "additions": raw.get("additions") or 0,
        "deletions": raw.get("deletions") or 0,
        "keys": keys,
        "bucket": classify(keys, branch, scope, title=title, body=body),
        "stale": stale,
        "stale_unreviewed": is_stale_unreviewed(stale, review),
    }


def build_report(merged, open_prs, window, config, now):
    buckets = ("linked_in_scope", "linked_out_of_scope", "no_ticket", "declared_no_ticket")
    everything = merged + open_prs
    counts = {"merged": len(merged), "open": len(open_prs)}
    for bucket in buckets:
        counts[bucket] = sum(1 for pr in everything if pr["bucket"] == bucket)
    orphans = [pr for pr in everything if pr["bucket"] == "no_ticket"]
    counts["no_ticket_human"] = sum(1 for pr in orphans if not pr["author_is_bot"])
    counts["no_ticket_bot"] = sum(1 for pr in orphans if pr["author_is_bot"])
    counts["stale"] = sum(1 for pr in everything if pr["stale"])
    counts["stale_unreviewed"] = sum(1 for pr in everything if pr["stale_unreviewed"])

    roster = {name.lower() for name in (config.get("roster") or [])}
    counts["merged_team"] = sum(1 for pr in merged if is_team_pr(pr, roster))
    counts["open_team"] = sum(1 for pr in open_prs if is_team_pr(pr, roster))
    counts["stale_team"] = sum(1 for pr in everything if pr["stale"] and is_team_pr(pr, roster))
    counts["stale_unreviewed_team"] = sum(
        1 for pr in everything if pr["stale_unreviewed"] and is_team_pr(pr, roster))
    counts["no_ticket_team"] = sum(
        1 for pr in orphans if not pr["author_is_bot"] and is_team_pr(pr, roster))
    counts["opened_in_window_team"] = sum(
        1 for pr in everything
        if is_team_pr(pr, roster)
        and window["since"] <= (pr["created_at"] or "")[:10] <= window["until"])
    return {
        "generated_at": now,
        "window": window,
        "config": config,
        "counts": counts,
        "merged": merged,
        "open": open_prs,
    }


def _cell(text, width=70):
    flat = str(text).replace("|", "\\|").replace("\n", " ").strip()
    if len(flat) <= width:
        return flat
    return flat[: width - 1].rstrip() + "…"


def _table(rows, headers):
    if not rows:
        return "_none_\n"
    out = ["| " + " | ".join(headers) + " |",
           "|" + "|".join("---" for _ in headers) + "|"]
    for row in rows:
        out.append("| " + " | ".join(str(cell) for cell in row) + " |")
    return "\n".join(out) + "\n"


def _team_only(prs, roster):
    if not roster:
        return prs
    return [pr for pr in prs if is_team_pr(pr, roster)]


def render_markdown(report):
    counts = report["counts"]
    window = report["window"]
    failures = report.get("failures") or []
    roster = {name.lower() for name in (report["config"].get("roster") or [])}
    lines = [
        f"# PR scan {window['since']} to {window['until']}",
        "",]
    if failures:
        lines += ["> **Incomplete scan.** These repos could not be read, so work in them is "
                  "missing from every section below:", ""]
        lines += [f"> - `{f['repo']}` ({f['state']})" for f in failures]
        lines += [""]
    lines += [
        f"Repos: {', '.join(report['config']['repos'])} · "
        f"Scope keys: {', '.join(report['config']['keys']) or 'none'}",
        "",
        f"Merged {counts['merged']} · Open {counts['open']} · "
        f"In scope {counts['linked_in_scope']} · "
        f"Other team {counts['linked_out_of_scope']} · "
        f"**No ticket {counts['no_ticket']}** "
        f"({counts['no_ticket_human']} human, {counts['no_ticket_bot']} bot) · "
        f"Declared no-ticket {counts['declared_no_ticket']} · "
        f"Stale, unreviewed {counts['stale_unreviewed']}",
        "",
        "## Not on the board",
        "",
        "Human-authored PRs carrying no ticket key anywhere in title, branch, or body. "
        "This is work the tracker cannot see. Repo-wide total above; table below is the team's "
        "PRs only, when a roster is configured.",
        "",
    ]
    everything = report["merged"] + report["open"]
    orphans = [pr for pr in everything if pr["bucket"] == "no_ticket"]
    human_orphans = _team_only([pr for pr in orphans if not pr["author_is_bot"]], roster)
    bot_orphans = [pr for pr in orphans if pr["author_is_bot"]]
    lines.append(_table(
        [(f"{pr['repo']}#{pr['number']}", _cell(pr["title"]), pr["author"],
          "merged" if pr["merged_at"] else "open", f"{pr['age_days']}d",
          f"+{pr['additions']}/-{pr['deletions']}", pr["url"]) for pr in human_orphans],
        ["PR", "Title", "Author", "State", "Age since opened", "Size", "URL"]))
    if bot_orphans:
        lines += ["", f"Plus {len(bot_orphans)} bot PRs with no ticket "
                      f"({', '.join(sorted({pr['author'] for pr in bot_orphans}))}) — "
                      f"dependency bumps, not team work. Not shown.", ""]

    lines += ["", "## Another team's ticket", "",
              "In your repos, but filed under a key outside your scope. Repo-wide total above; "
              "table below is the team's PRs only, when a roster is configured.", ""]
    others = _team_only([pr for pr in everything if pr["bucket"] == "linked_out_of_scope"], roster)
    lines.append(_table(
        [(f"{pr['repo']}#{pr['number']}", _cell(pr["title"]), pr["author"],
          ", ".join(pr["keys"]), "merged" if pr["merged_at"] else "open") for pr in others],
        ["PR", "Title", "Author", "Keys", "State"]))

    lines += ["", "## Stale, unreviewed open PRs", "",
              "Stale, and nobody is reviewing it. An approved-but-unmerged PR is a different "
              "problem and is not listed here. Repo-wide total above; table below is the team's "
              "PRs only, when a roster is configured.", ""]
    stale_unreviewed = _team_only([pr for pr in report["open"] if pr["stale_unreviewed"]], roster)
    lines.append(_table(
        [(f"{pr['repo']}#{pr['number']}", _cell(pr["title"]), pr["author"],
          f"{pr['age_days']}d", pr["review_decision"], pr["url"]) for pr in stale_unreviewed],
        ["PR", "Title", "Author", "Age", "Review", "URL"]))

    lines += ["", "## Merged in window, in scope", ""]
    shipped = [pr for pr in report["merged"] if pr["bucket"] == "linked_in_scope"]
    lines.append(_table(
        [(_link(pr), _cell(pr["title"]), pr["author"], ", ".join(pr["keys"]),
          _date(pr["created_at"]), _date(pr["merged_at"]), _size(pr)) for pr in shipped],
        ["PR", "Title", "Author", "Keys", "Opened", "Merged", "Size"]))

    lines += ["", render_open_prs(report["open"], roster)]
    if "stacks" in report:
        lines += ["", _render_stacks(report["stacks"])]
    if "epics" in report:
        lines += ["", _render_epics(report["epics"])]
    if "closed_unmerged" in report:
        lines += ["", _render_closed(report["closed_unmerged"], roster)]
    return "\n".join(lines)


def _link(pr):
    label = f"{pr['repo']}#{pr['number']}"
    return f"[{label}]({pr['url']})" if pr.get("url") else label


def _date(value):
    return (value or "")[:10]


def _size(pr):
    return f"+{pr['additions']}/-{pr['deletions']}"


def render_open_prs(open_prs: list[dict], roster: set[str]) -> str:
    team = sorted(_team_only(open_prs, roster), key=lambda pr: pr["created_at"] or "9999")
    lines = ["## Open PRs (team)", "",
             "Every PR the team has open right now, whatever its age, drafts included. "
             "Oldest first.", ""]
    lines.append(_table(
        [(_link(pr), _cell(pr["title"]), pr["author"], _date(pr["created_at"]),
          f"{pr['age_days']}d", "draft" if pr["is_draft"] else "no",
          REVIEW_LABELS.get(pr["review_decision"], "none"), ", ".join(pr["keys"]))
         for pr in team],
        ["PR", "Title", "Author", "Opened", "Age", "Draft", "Review", "Keys"]))
    return "\n".join(lines)


def _render_stacks(stacks):
    rows = [(stack.capitalize(), tally["merged"], tally["open"],
             f"+{tally['additions']}/-{tally['deletions']}")
            for stack, tally in stacks.items()
            if stack != "other" or tally["merged"] or tally["open"]]
    lines = ["## Frontend vs backend", "", "The team's PRs, merged in the window and open now.", ""]
    lines.append(_table(rows, ["Stack", "Merged", "Open", "Size"]))
    return "\n".join(lines)


def _render_epics(epics):
    def cell(tally):
        return f"{tally['merged']} merged, {tally['open']} open"

    empty = {"merged": 0, "open": 0}
    rows = [(epic, cell(sides["frontend"]), cell(sides["backend"]),
             cell(sides.get("other", empty))) for epic, sides in sorted(epics.items())]
    lines = ["## By epic", "",
             "PRs merged in the window and open now, by the epic their ticket sits under.", ""]
    lines.append(_table(rows, ["Epic", "Frontend", "Backend", "Other"]))
    return "\n".join(lines)


def _render_closed(closed, roster):
    team = _team_only(closed, roster)
    lines = ["## Closed without merging", ""]
    lines.append(_table(
        [(_link(pr), _cell(pr["title"]), pr["author"], ", ".join(pr["keys"]),
          _date(pr["created_at"]), f"closed unmerged {_date(pr['closed_at'])}") for pr in team],
        ["PR", "Title", "Author", "Keys", "Opened", "Merged"]))
    return "\n".join(lines)


def _gh(args):
    if not shutil.which("gh"):
        raise GhError("gh CLI not found on PATH. Install it from https://cli.github.com/")
    result = subprocess.run(["gh"] + args, capture_output=True, text=True)
    if result.returncode != 0:
        raise GhError(f"gh failed: {' '.join(args)}\n{result.stderr.strip()}")
    try:
        return json.loads(result.stdout or "[]")
    except json.JSONDecodeError as error:
        raise GhError(
            f"gh returned output that is not JSON: {' '.join(args)}\n"
            f"{result.stdout[:200]}"
        ) from error


def build_gh_args(repo, state, since, limit, until=None):
    args = ["pr", "list", "--repo", repo, "--state", state,
            "-L", str(effective_limit(state, limit)), "--json", JSON_FIELDS]
    if state in ("merged", "closed"):
        if until:
            args += ["--search", f"{state}:{since}..{until}"]
        else:
            args += ["--search", f"{state}:>={since}"]
    return args


def fetch(repo, state, since, limit, until=None):
    items = _gh(build_gh_args(repo, state, since, limit, until))
    check_not_truncated(items, limit, repo, state)
    return items


def main(argv=None):
    parser = argparse.ArgumentParser(description="Reconcile GitHub PRs against tracker keys.")
    parser.add_argument("--org", required=True)
    parser.add_argument("--repos", required=True, help="comma-separated repo names")
    parser.add_argument("--since", required=True, help="YYYY-MM-DD")
    parser.add_argument("--until", default=None, help="YYYY-MM-DD (default: today)")
    parser.add_argument("--keys", default="", help="comma-separated project key prefixes")
    parser.add_argument("--roster", default="",
                        help="comma-separated GitHub handles of the team; a PR counts as the "
                             "team's when its author is on this list OR its ticket key is in "
                             "--keys. Without it, shared-repo totals include other teams.")
    parser.add_argument("--frontend-repos", default="",
                        help="comma-separated repos counted as frontend; bare names match "
                             "with or without the org/ prefix")
    parser.add_argument("--backend-repos", default="",
                        help="comma-separated repos counted as backend; bare names match "
                             "with or without the org/ prefix")
    parser.add_argument("--jira-map", default=None, metavar="PATH",
                        help="jira.json from jira_scan.py; its child_to_epic map rolls PRs "
                             "up to their epic")
    parser.add_argument("--stale-days", type=int, default=3)
    parser.add_argument("--limit", type=int, default=1000)
    parser.add_argument("--out", default=".")
    args = parser.parse_args(argv)

    repos = [r.strip() for r in args.repos.split(",") if r.strip()]
    scope = [k.strip().upper() for k in args.keys.split(",") if k.strip()]
    roster = [h.strip() for h in args.roster.split(",") if h.strip()]
    frontend = {r.strip() for r in args.frontend_repos.split(",") if r.strip()}
    backend = {r.strip() for r in args.backend_repos.split(",") if r.strip()}
    jira_map = {}
    if args.jira_map:
        try:
            jira_map = load_jira_map(args.jira_map)
        except (OSError, ValueError) as error:
            print(f"cannot read --jira-map {args.jira_map}: {error}", file=sys.stderr)
            return 1
    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    until = args.until or now[:10]

    merged, open_prs, closed, failures = [], [], [], []
    for name in repos:
        repo = name if "/" in name else f"{args.org}/{name}"
        for state, sink in (("merged", merged), ("open", open_prs), ("closed", closed)):
            try:
                for raw in fetch(repo, state, args.since, args.limit, until):
                    sink.append(build_pr(raw, repo, state, scope, args.stale_days, now))
            except GhError as error:
                failures.append({"repo": repo, "state": state, "error": str(error)})

    report = build_report(
        merged, open_prs,
        window={"since": args.since, "until": until},
        config={"org": args.org, "repos": repos, "keys": scope,
                "stale_days": args.stale_days, "limit": args.limit, "roster": roster},
        now=now,
    )
    report["failures"] = failures
    report["closed_unmerged"] = [pr for pr in closed if not pr["merged_at"]]
    if frontend or backend:
        report["stacks"] = build_stack_counts(merged + open_prs, frontend, backend, set(roster))
    if frontend or backend or args.jira_map:
        report["epics"] = build_epic_rollup(merged + open_prs, frontend, backend, jira_map)

    os.makedirs(args.out, exist_ok=True)
    json_path = os.path.join(args.out, "prs.json")
    md_path = os.path.join(args.out, "prs.md")
    with open(json_path, "w") as handle:
        json.dump(report, handle, indent=2)
    with open(md_path, "w") as handle:
        handle.write(render_markdown(report))

    counts = report["counts"]
    print(f"{json_path}\n{md_path}")
    print(f"TEAM   merged={counts['merged_team']} open={counts['open_team']} "
          f"stale_unreviewed={counts['stale_unreviewed_team']} no_ticket={counts['no_ticket_team']}")
    print(f"repos  merged={counts['merged']} open={counts['open']} "
          f"stale_unreviewed={counts['stale_unreviewed']} no_ticket={counts['no_ticket']} "
          f"other_teams={counts['linked_out_of_scope']}")
    if failures:
        failed_repos = ", ".join(sorted({f["repo"] for f in failures}))
        print(f"INCOMPLETE: {failed_repos}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (TruncatedError, GhError) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
