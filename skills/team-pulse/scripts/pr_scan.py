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


def is_ticket_key(candidate):
    return candidate.split("-", 1)[0] not in NOT_TICKET_PREFIXES

JSON_FIELDS = (
    "number,title,headRefName,body,author,createdAt,mergedAt,"
    "isDraft,reviewDecision,url,additions,deletions"
)


class TruncatedError(RuntimeError):
    pass


class GhError(RuntimeError):
    pass


def extract_keys(title, branch, body):
    haystack = " ".join(part or "" for part in (title, branch, body))
    found = {key for key in KEY_PATTERN.findall(haystack) if is_ticket_key(key)}
    return sorted(found - {"NO-TICKET"})


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


def is_team_pr(pr, roster):
    if pr.get("author", "").lower() in roster:
        return True
    return pr.get("bucket") == "linked_in_scope"


def effective_limit(state, limit):
    if state == "merged":
        return min(limit, SEARCH_API_CAP)
    return limit


def check_not_truncated(items, limit, repo, state):
    cap = effective_limit(state, limit)
    if len(items) >= cap:
        raise TruncatedError(
            f"TRUNCATED: {repo} {state} returned {len(items)} items, at or above the "
            f"effective cap of {cap}. Results are incomplete. Narrow the window with a "
            f"later --since, or scan fewer repos per run. Raising --limit will NOT help "
            f"for merged PRs: GitHub's search API stops at {SEARCH_API_CAP} regardless."
        )


def _parse_ts(value):
    if not value:
        return None
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def build_pr(raw, repo, state, scope, stale_days, now):
    title = raw.get("title") or ""
    branch = raw.get("headRefName") or ""
    body = raw.get("body") or ""
    keys = extract_keys(title, branch, body)
    created = _parse_ts(raw.get("createdAt"))
    age_days = (_parse_ts(now) - created).days if created else 0
    review = raw.get("reviewDecision") or ""
    is_draft = bool(raw.get("isDraft"))
    author = raw.get("author") or {}
    author_is_bot = bool(author.get("is_bot"))
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
        "is_draft": is_draft,
        "review_decision": review,
        "age_days": age_days,
        "additions": raw.get("additions") or 0,
        "deletions": raw.get("deletions") or 0,
        "keys": keys,
        "bucket": classify(keys, branch, scope, title=title, body=body),
        "stale": is_stale(state, is_draft, age_days, review, stale_days, author_is_bot),
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

    roster = {name.lower() for name in (config.get("roster") or [])}
    counts["merged_team"] = sum(1 for pr in merged if is_team_pr(pr, roster))
    counts["open_team"] = sum(1 for pr in open_prs if is_team_pr(pr, roster))
    counts["stale_team"] = sum(1 for pr in everything if pr["stale"] and is_team_pr(pr, roster))
    counts["no_ticket_team"] = sum(
        1 for pr in orphans if not pr["author_is_bot"] and is_team_pr(pr, roster))
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


def render_markdown(report):
    counts = report["counts"]
    window = report["window"]
    failures = report.get("failures") or []
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
        f"Stale {counts['stale']}",
        "",
        "## Not on the board",
        "",
        "Human-authored PRs carrying no ticket key anywhere in title, branch, or body. "
        "This is work the tracker cannot see.",
        "",
    ]
    everything = report["merged"] + report["open"]
    orphans = [pr for pr in everything if pr["bucket"] == "no_ticket"]
    human_orphans = [pr for pr in orphans if not pr["author_is_bot"]]
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
              "In your repos, but filed under a key outside your scope.", ""]
    others = [pr for pr in everything if pr["bucket"] == "linked_out_of_scope"]
    lines.append(_table(
        [(f"{pr['repo']}#{pr['number']}", _cell(pr["title"]), pr["author"],
          ", ".join(pr["keys"]), "merged" if pr["merged_at"] else "open") for pr in others],
        ["PR", "Title", "Author", "Keys", "State"]))

    lines += ["", "## Stale open PRs", ""]
    stale = [pr for pr in report["open"] if pr["stale"]]
    lines.append(_table(
        [(f"{pr['repo']}#{pr['number']}", _cell(pr["title"]), pr["author"],
          f"{pr['age_days']}d", pr["review_decision"], pr["url"]) for pr in stale],
        ["PR", "Title", "Author", "Age", "Review", "URL"]))

    lines += ["", "## Merged in window, in scope", ""]
    shipped = [pr for pr in report["merged"] if pr["bucket"] == "linked_in_scope"]
    lines.append(_table(
        [(f"{pr['repo']}#{pr['number']}", _cell(pr["title"]), pr["author"],
          ", ".join(pr["keys"])) for pr in shipped],
        ["PR", "Title", "Author", "Keys"]))
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


def build_gh_args(repo, state, since, limit):
    args = ["pr", "list", "--repo", repo, "--state", state,
            "-L", str(effective_limit(state, limit)), "--json", JSON_FIELDS]
    if state == "merged":
        args += ["--search", f"merged:>={since}"]
    return args


def fetch(repo, state, since, limit):
    items = _gh(build_gh_args(repo, state, since, limit))
    check_not_truncated(items, limit, repo, state)
    return items


def main(argv=None):
    parser = argparse.ArgumentParser(description="Reconcile GitHub PRs against tracker keys.")
    parser.add_argument("--org", required=True)
    parser.add_argument("--repos", required=True, help="comma-separated repo names")
    parser.add_argument("--since", required=True, help="YYYY-MM-DD")
    parser.add_argument("--keys", default="", help="comma-separated project key prefixes")
    parser.add_argument("--roster", default="",
                        help="comma-separated GitHub handles of the team; a PR counts as the "
                             "team's when its author is on this list OR its ticket key is in "
                             "--keys. Without it, shared-repo totals include other teams.")
    parser.add_argument("--stale-days", type=int, default=3)
    parser.add_argument("--limit", type=int, default=1000)
    parser.add_argument("--out", default=".")
    args = parser.parse_args(argv)

    repos = [r.strip() for r in args.repos.split(",") if r.strip()]
    scope = [k.strip().upper() for k in args.keys.split(",") if k.strip()]
    roster = [h.strip() for h in args.roster.split(",") if h.strip()]
    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    merged, open_prs, failures = [], [], []
    for name in repos:
        repo = name if "/" in name else f"{args.org}/{name}"
        for state, sink in (("merged", merged), ("open", open_prs)):
            try:
                for raw in fetch(repo, state, args.since, args.limit):
                    sink.append(build_pr(raw, repo, state, scope, args.stale_days, now))
            except GhError as error:
                failures.append({"repo": repo, "state": state, "error": str(error)})

    report = build_report(
        merged, open_prs,
        window={"since": args.since, "until": now[:10]},
        config={"org": args.org, "repos": repos, "keys": scope,
                "stale_days": args.stale_days, "limit": args.limit, "roster": roster},
        now=now,
    )
    report["failures"] = failures

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
          f"stale={counts['stale_team']} no_ticket={counts['no_ticket_team']}")
    print(f"repos  merged={counts['merged']} open={counts['open']} "
          f"stale={counts['stale']} no_ticket={counts['no_ticket']} "
          f"other_teams={counts['linked_out_of_scope']}")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (TruncatedError, GhError) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
