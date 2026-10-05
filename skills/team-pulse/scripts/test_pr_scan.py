import json
import re
import pytest

import pr_scan


@pytest.mark.parametrize(
    "title,branch,body,expected",
    [
        ("[ABC-2706] Add ignored state", "feature/x/y", "", ["ABC-2706"]),
        ("Add ignored state", "feature/ABC-2706/mark-unreviewed", "", ["ABC-2706"]),
        ("Add ignored state", "feature/x/y", "Closes ABC-2706", ["ABC-2706"]),
        ("[ABC-2706] see also XYZ-195", "feature/ABC-2706/x", "", ["ABC-2706", "XYZ-195"]),
        ("fix abc-2706 lowercase", "feature/x/y", "", []),
        ("Bump undici to 5.29.0", "chore/bump-undici", "", []),
        ("Support HTTP-2 and UTF-8", "chore/protocols", "", []),
        ("", "", "", []),
    ],
)
def test_extract_keys(title, branch, body, expected):
    assert pr_scan.extract_keys(title, branch, body) == expected


@pytest.mark.parametrize(
    "keys,branch,scope,expected",
    [
        (["ABC-2706"], "feature/ABC-2706/x", ["ABC"], "linked_in_scope"),
        (["XYZ-195"], "feature/XYZ-195/x", ["ABC", "XYZ"], "linked_in_scope"),
        (["QRS-921"], "feature/QRS-921/x", ["ABC"], "linked_out_of_scope"),
        ([], "chore/bump-undici", ["ABC"], "no_ticket"),
        ([], "docs/NO-TICKET/fix-typo", ["ABC"], "declared_no_ticket"),
        ([], "docs/no-ticket/fix-typo", ["ABC"], "declared_no_ticket"),
    ],
)
def test_classify(keys, branch, scope, expected):
    assert pr_scan.classify(keys, branch, scope) == expected


def test_truncation_raises_when_result_equals_limit():
    with pytest.raises(pr_scan.TruncatedError) as exc:
        pr_scan.check_not_truncated([{"number": n} for n in range(30)], 30, "acme/repo", "merged")
    assert "narrow" in str(exc.value).lower()


def test_truncation_detected_at_search_api_cap_even_when_limit_is_higher():
    items = [{"number": n} for n in range(pr_scan.SEARCH_API_CAP)]
    with pytest.raises(pr_scan.TruncatedError):
        pr_scan.check_not_truncated(items, 5000, "acme/repo", "merged")


def test_open_state_is_not_capped_by_search_api():
    items = [{"number": n} for n in range(pr_scan.SEARCH_API_CAP)]
    pr_scan.check_not_truncated(items, 5000, "acme/repo", "open")


def test_no_truncation_below_limit():
    pr_scan.check_not_truncated([{"number": 1}], 30, "acme/repo", "merged")


def test_empty_result_is_not_truncation():
    pr_scan.check_not_truncated([], 30, "acme/repo", "merged")


def test_gh_args_always_bound_the_result_set():
    for state in ("merged", "open"):
        args = pr_scan.build_gh_args("acme/repo", state, "2026-09-09", 1000)
        assert "-L" in args, f"{state} must bound results or gh silently caps at 30"
        assert args[args.index("-L") + 1] == "1000"


def test_gh_args_clamp_merged_limit_to_search_cap():
    args = pr_scan.build_gh_args("acme/repo", "merged", "2026-09-09", 5000)
    assert args[args.index("-L") + 1] == str(pr_scan.SEARCH_API_CAP)


@pytest.mark.parametrize(
    "review,expected",
    [
        ("REVIEW_REQUIRED", True),
        ("APPROVED", True),
        ("", True),
        ("CHANGES_REQUESTED", True),
    ],
)
def test_stale_ignores_review_state(review, expected):
    assert pr_scan.is_stale(state="open", is_draft=False, age_days=9,
                            review=review, limit=3) is expected


def test_stale_requires_open_nondraft_and_age():
    assert pr_scan.is_stale(state="open", is_draft=False, age_days=5, review="REVIEW_REQUIRED", limit=3)
    assert not pr_scan.is_stale(state="open", is_draft=True, age_days=5, review="REVIEW_REQUIRED", limit=3)
    assert not pr_scan.is_stale(state="open", is_draft=False, age_days=1, review="REVIEW_REQUIRED", limit=3)
    assert not pr_scan.is_stale(state="merged", is_draft=False, age_days=9, review="APPROVED", limit=3)
    assert pr_scan.is_stale(state="open", is_draft=False, age_days=9, review="APPROVED", limit=3)


@pytest.mark.parametrize(
    "text",
    [
        "Bump nokogiri to fix CVE-2024-1234",
        "Use AES-256-GCM for token encryption",
        "Follow RFC-9110 semantics",
        "Normalise everything to UTF-8",
        "Gather SOC-2 evidence",
        "Support HTTP-2 and TLS-1",
        "See ADR-0014 for rationale",
    ],
)
def test_standards_identifiers_are_not_ticket_keys(text):
    assert pr_scan.extract_keys(text, "chore/x", "") == []


def test_real_ticket_keys_still_extracted():
    assert pr_scan.extract_keys("[ABC-12] fix", "feature/ABC-12/x", "") == ["ABC-12"]


def test_dependabot_pr_citing_a_cve_stays_untracked():
    pr = pr_scan.build_pr({"number": 1, "title": "Bump nokogiri from 1.15.2 to 1.16.5",
                           "headRefName": "dependabot/bundler/nokogiri-1.16.5",
                           "body": "Fixes CVE-2024-1234 and CVE-2024-5678.",
                           "author": {"login": "app/dependabot", "is_bot": True},
                           "createdAt": "2026-09-01T00:00:00Z", "mergedAt": None,
                           "isDraft": False, "reviewDecision": "", "url": "u",
                           "additions": 2, "deletions": 2},
                          repo="r", state="open", scope=["ABC"], stale_days=3,
                          now="2026-09-16T00:00:00Z")
    assert pr["keys"] == []
    assert pr["bucket"] == "no_ticket"
    assert pr["author_is_bot"] is True


@pytest.mark.parametrize(
    "title,branch,body",
    [
        ("chore: bump deps", "docs/NO-TICKET/typo", ""),
        ("[NO-TICKET] chore: bump deps", "chore/bump", ""),
        ("chore: bump deps", "chore/bump", "NO-TICKET: trivial dependency bump"),
    ],
)
def test_declared_no_ticket_read_from_title_branch_or_body(title, branch, body):
    keys = pr_scan.extract_keys(title, branch, body)
    assert pr_scan.classify(keys, branch, ["ABC"], title=title, body=body) == "declared_no_ticket"


def test_not_on_board_table_lists_humans_and_excludes_bots():
    def make(login, is_bot, number):
        return pr_scan.build_pr({"number": number, "title": f"work by {login}",
                                 "headRefName": "chore/x", "body": "",
                                 "author": {"login": login, "is_bot": is_bot},
                                 "createdAt": "2026-09-01T00:00:00Z", "mergedAt": None,
                                 "isDraft": False, "reviewDecision": "", "url": "u",
                                 "additions": 1, "deletions": 1},
                                repo="r", state="open", scope=["ABC"], stale_days=3,
                                now="2026-09-16T00:00:00Z")

    report = pr_scan.build_report(
        [], [make("app/dependabot", True, 1), make("a-real-person", False, 2)],
        window={"since": "2026-09-09", "until": "2026-09-16"},
        config={"org": "o", "repos": ["r"], "keys": ["ABC"], "stale_days": 3, "limit": 1000},
        now="2026-09-16T00:00:00Z")

    board = pr_scan.render_markdown(report).split("## Not on the board")[1].split("##")[0]
    table = "\n".join(line for line in board.splitlines() if line.startswith("|"))
    assert "a-real-person" in table
    assert "app/dependabot" not in table


def test_pipe_in_title_does_not_break_the_table():
    pr = pr_scan.build_pr({"number": 1, "title": "fix a | b parsing", "headRefName": "chore/x",
                           "body": "", "author": {"login": "person", "is_bot": False},
                           "createdAt": "2026-09-01T00:00:00Z", "mergedAt": None,
                           "isDraft": False, "reviewDecision": "", "url": "u",
                           "additions": 1, "deletions": 1},
                          repo="r", state="open", scope=["ABC"], stale_days=3,
                          now="2026-09-16T00:00:00Z")
    report = pr_scan.build_report([], [pr],
                                  window={"since": "2026-09-09", "until": "2026-09-16"},
                                  config={"org": "o", "repos": ["r"], "keys": ["ABC"],
                                          "stale_days": 3, "limit": 1000},
                                  now="2026-09-16T00:00:00Z")
    row = [ln for ln in pr_scan.render_markdown(report).splitlines() if "parsing" in ln][0]
    assert "\\|" in row
    assert len(re.findall(r"(?<!\\)\|", row)) == 8


def test_bot_authorship_is_recorded():
    bot = pr_scan.build_pr({"number": 1, "title": "bump rubyzip", "headRefName": "dependabot/x",
                            "body": "", "author": {"login": "app/dependabot", "is_bot": True},
                            "createdAt": "2026-09-01T00:00:00Z", "mergedAt": None,
                            "isDraft": False, "reviewDecision": "", "url": "u",
                            "additions": 2, "deletions": 2},
                           repo="r", state="open", scope=["ABC"], stale_days=3,
                           now="2026-09-16T00:00:00Z")
    assert bot["author_is_bot"] is True
    assert bot["bucket"] == "no_ticket"

    human = pr_scan.build_pr({"number": 2, "title": "owasp checklist", "headRefName": "chore/owasp",
                              "body": "", "author": {"login": "someone", "is_bot": False},
                              "createdAt": "2026-09-01T00:00:00Z", "mergedAt": None,
                              "isDraft": False, "reviewDecision": "", "url": "u",
                              "additions": 546, "deletions": 790},
                             repo="r", state="open", scope=["ABC"], stale_days=3,
                             now="2026-09-16T00:00:00Z")
    assert human["author_is_bot"] is False


def test_no_ticket_human_count_excludes_bots():
    def make(login, is_bot, number):
        return pr_scan.build_pr({"number": number, "title": "bump x", "headRefName": "chore/x",
                                 "body": "", "author": {"login": login, "is_bot": is_bot},
                                 "createdAt": "2026-09-01T00:00:00Z", "mergedAt": None,
                                 "isDraft": False, "reviewDecision": "", "url": "u",
                                 "additions": 1, "deletions": 1},
                                repo="r", state="open", scope=["ABC"], stale_days=3,
                                now="2026-09-16T00:00:00Z")

    report = pr_scan.build_report(
        [], [make("app/dependabot", True, 1), make("app/dependabot", True, 2), make("person", False, 3)],
        window={"since": "2026-09-09", "until": "2026-09-16"},
        config={"org": "o", "repos": ["r"], "keys": ["ABC"], "stale_days": 3, "limit": 1000},
        now="2026-09-16T00:00:00Z")

    assert report["counts"]["no_ticket"] == 3
    assert report["counts"]["no_ticket_human"] == 1
    assert report["counts"]["no_ticket_bot"] == 2


def test_stale_ignores_bot_prs():
    bot = pr_scan.build_pr({"number": 1, "title": "bump x", "headRefName": "dependabot/x",
                            "body": "", "author": {"login": "app/dependabot", "is_bot": True},
                            "createdAt": "2026-09-01T00:00:00Z", "mergedAt": None,
                            "isDraft": False, "reviewDecision": "REVIEW_REQUIRED", "url": "u",
                            "additions": 1, "deletions": 1},
                           repo="r", state="open", scope=["ABC"], stale_days=3,
                           now="2026-09-16T00:00:00Z")
    assert bot["stale"] is False


def test_build_report_shape_and_counts():
    merged = [
        pr_scan.build_pr({"number": 1, "title": "[ABC-1] a", "headRefName": "feature/ABC-1/a",
                          "body": "", "author": {"login": "x"}, "createdAt": "2026-09-01T00:00:00Z",
                          "mergedAt": "2026-09-02T00:00:00Z", "isDraft": False,
                          "reviewDecision": "APPROVED", "url": "u", "additions": 1, "deletions": 0},
                         repo="r", state="merged", scope=["ABC"], stale_days=3, now="2026-09-16T00:00:00Z"),
    ]
    open_prs = [
        pr_scan.build_pr({"number": 2, "title": "bump dep", "headRefName": "chore/bump",
                          "body": "", "author": {"login": "y"}, "createdAt": "2026-09-01T00:00:00Z",
                          "mergedAt": None, "isDraft": False,
                          "reviewDecision": "REVIEW_REQUIRED", "url": "u", "additions": 1, "deletions": 0},
                         repo="r", state="open", scope=["ABC"], stale_days=3, now="2026-09-16T00:00:00Z"),
    ]
    report = pr_scan.build_report(merged, open_prs, window={"since": "2026-09-09", "until": "2026-09-16"},
                                  config={"org": "o", "repos": ["r"], "keys": ["ABC"],
                                          "stale_days": 3, "limit": 1000},
                                  now="2026-09-16T00:00:00Z")

    assert set(report) == {"generated_at", "window", "config", "counts", "merged", "open"}
    assert report["counts"]["merged"] == 1
    assert report["counts"]["open"] == 1
    assert report["counts"]["linked_in_scope"] == 1
    assert report["counts"]["no_ticket"] == 1
    assert report["counts"]["stale"] == 1
    json.dumps(report)


def test_is_team_pr_matches_roster_author_or_scope_key():
    roster = {"alice", "bob"}
    assert pr_scan.is_team_pr({"author": "alice", "bucket": "no_ticket"}, roster)
    assert pr_scan.is_team_pr({"author": "carol", "bucket": "linked_in_scope"}, roster)
    assert not pr_scan.is_team_pr({"author": "carol", "bucket": "linked_out_of_scope"}, roster)
    assert not pr_scan.is_team_pr({"author": "carol", "bucket": "no_ticket"}, roster)


def test_roster_match_is_case_insensitive():
    assert pr_scan.is_team_pr({"author": "Alice", "bucket": "no_ticket"}, {"alice"})


def test_no_roster_means_scope_key_decides():
    assert pr_scan.is_team_pr({"author": "anyone", "bucket": "linked_in_scope"}, set())
    assert not pr_scan.is_team_pr({"author": "anyone", "bucket": "no_ticket"}, set())


def test_counts_separate_team_prs_from_repo_wide():
    def make(login, title, branch, number):
        return pr_scan.build_pr({"number": number, "title": title, "headRefName": branch,
                                 "body": "", "author": {"login": login, "is_bot": False},
                                 "createdAt": "2026-09-01T00:00:00Z",
                                 "mergedAt": "2026-09-02T00:00:00Z", "isDraft": False,
                                 "reviewDecision": "APPROVED", "url": "u",
                                 "additions": 1, "deletions": 1},
                                repo="r", state="merged", scope=["ABC"], stale_days=3,
                                now="2026-09-16T00:00:00Z")

    merged = [
        make("alice", "[ABC-1] ours", "feature/ABC-1/x", 1),
        make("carol", "[QRS-9] theirs", "feature/QRS-9/x", 2),
        make("dave", "[QRS-8] theirs too", "feature/QRS-8/x", 3),
    ]
    report = pr_scan.build_report(merged, [],
                                  window={"since": "2026-09-09", "until": "2026-09-16"},
                                  config={"org": "o", "repos": ["r"], "keys": ["ABC"],
                                          "stale_days": 3, "limit": 1000,
                                          "roster": ["alice", "bob"]},
                                  now="2026-09-16T00:00:00Z")
    assert report["counts"]["merged"] == 3, "repo-wide total still available"
    assert report["counts"]["merged_team"] == 1, "team total must exclude other teams"
    assert report["counts"]["open_team"] == 0


def test_window_bounds_merged_only_not_open():
    args = pr_scan.build_gh_args("acme/repo", "merged", "2026-09-09", 1000)
    assert "--search" in args
    assert "merged:>=2026-09-09" in args

    args = pr_scan.build_gh_args("acme/repo", "open", "2026-09-09", 1000)
    assert "--search" not in args


def test_empty_report_is_valid():
    report = pr_scan.build_report([], [], window={"since": "2099-01-01", "until": "2099-01-08"},
                                  config={"org": "o", "repos": ["r"], "keys": ["ABC"],
                                          "stale_days": 3, "limit": 1000},
                                  now="2099-01-08T00:00:00Z")
    assert report["merged"] == []
    assert report["open"] == []
    assert report["counts"]["merged"] == 0
    assert report["counts"]["no_ticket"] == 0
    assert pr_scan.render_markdown(report).strip() != ""


def test_gh_args_bound_merged_search_by_until_when_given():
    args = pr_scan.build_gh_args("acme/repo", "merged", "2026-09-09", 1000, until="2026-09-16")
    assert "merged:2026-09-09..2026-09-16" in args


def test_gh_args_merged_search_is_open_ended_without_until():
    args = pr_scan.build_gh_args("acme/repo", "merged", "2026-09-09", 1000)
    assert "merged:>=2026-09-09" in args


def test_gh_args_until_does_not_affect_open_state():
    args = pr_scan.build_gh_args("acme/repo", "open", "2026-09-09", 1000, until="2026-09-16")
    assert "--search" not in args


@pytest.mark.parametrize(
    "review,expected",
    [
        ("", True),
        ("REVIEW_REQUIRED", True),
        ("APPROVED", False),
        ("CHANGES_REQUESTED", False),
    ],
)
def test_stale_unreviewed_only_when_nobody_reviewed(review, expected):
    assert pr_scan.is_stale_unreviewed(True, review) is expected


def test_stale_unreviewed_requires_stale():
    assert pr_scan.is_stale_unreviewed(False, "") is False


def test_build_pr_records_stale_unreviewed_field():
    def make(review):
        return pr_scan.build_pr({"number": 1, "title": "x", "headRefName": "chore/x", "body": "",
                                 "author": {"login": "person", "is_bot": False},
                                 "createdAt": "2026-09-01T00:00:00Z", "mergedAt": None,
                                 "isDraft": False, "reviewDecision": review, "url": "u",
                                 "additions": 1, "deletions": 1},
                                repo="r", state="open", scope=["ABC"], stale_days=3,
                                now="2026-09-16T00:00:00Z")

    unreviewed = make("REVIEW_REQUIRED")
    assert unreviewed["stale"] is True
    assert unreviewed["stale_unreviewed"] is True

    approved = make("APPROVED")
    assert approved["stale"] is True
    assert approved["stale_unreviewed"] is False


def test_counts_separate_stale_unreviewed_from_approved_but_unmerged():
    def make(login, review, number):
        return pr_scan.build_pr({"number": number, "title": "x", "headRefName": "chore/x",
                                 "body": "", "author": {"login": login, "is_bot": False},
                                 "createdAt": "2026-09-01T00:00:00Z", "mergedAt": None,
                                 "isDraft": False, "reviewDecision": review, "url": "u",
                                 "additions": 1, "deletions": 1},
                                repo="r", state="open", scope=["ABC"], stale_days=3,
                                now="2026-09-16T00:00:00Z")

    open_prs = [make("alice", "REVIEW_REQUIRED", 1), make("carol", "APPROVED", 2)]
    report = pr_scan.build_report(
        [], open_prs,
        window={"since": "2026-09-09", "until": "2026-09-16"},
        config={"org": "o", "repos": ["r"], "keys": ["ABC"], "stale_days": 3, "limit": 1000,
                "roster": ["alice"]},
        now="2026-09-16T00:00:00Z")

    assert report["counts"]["stale_unreviewed"] == 1
    assert report["counts"]["stale_unreviewed_team"] == 1
    assert report["counts"]["stale"] == 2


def test_scope_key_exempts_matching_not_ticket_prefix():
    assert pr_scan.extract_keys("[PR-42] fix parser", "feature/PR-42/x", "", scope=["PR"]) == ["PR-42"]


def test_without_scope_a_not_ticket_prefix_is_still_excluded():
    assert pr_scan.extract_keys("[PR-42] fix parser", "feature/PR-42/x", "") == []


def test_render_markdown_no_ticket_table_is_team_only_when_roster_given():
    def make(login, number):
        return pr_scan.build_pr({"number": number, "title": "chore bump", "headRefName": "chore/x",
                                 "body": "", "author": {"login": login, "is_bot": False},
                                 "createdAt": "2026-09-01T00:00:00Z", "mergedAt": None,
                                 "isDraft": False, "reviewDecision": "", "url": "u",
                                 "additions": 1, "deletions": 1},
                                repo="r", state="open", scope=["ABC"], stale_days=3,
                                now="2026-09-16T00:00:00Z")

    report = pr_scan.build_report(
        [], [make("alice", 1), make("carol", 2)],
        window={"since": "2026-09-09", "until": "2026-09-16"},
        config={"org": "o", "repos": ["r"], "keys": ["ABC"], "stale_days": 3, "limit": 1000,
                "roster": ["alice"]},
        now="2026-09-16T00:00:00Z")

    board = pr_scan.render_markdown(report).split("## Not on the board")[1].split("##")[0]
    table = "\n".join(line for line in board.splitlines() if line.startswith("|"))
    assert "alice" in table
    assert "carol" not in table


def test_render_markdown_no_ticket_table_lists_everyone_without_a_roster():
    def make(login, number):
        return pr_scan.build_pr({"number": number, "title": "chore bump", "headRefName": "chore/x",
                                 "body": "", "author": {"login": login, "is_bot": False},
                                 "createdAt": "2026-09-01T00:00:00Z", "mergedAt": None,
                                 "isDraft": False, "reviewDecision": "", "url": "u",
                                 "additions": 1, "deletions": 1},
                                repo="r", state="open", scope=["ABC"], stale_days=3,
                                now="2026-09-16T00:00:00Z")

    report = pr_scan.build_report(
        [], [make("alice", 1), make("carol", 2)],
        window={"since": "2026-09-09", "until": "2026-09-16"},
        config={"org": "o", "repos": ["r"], "keys": ["ABC"], "stale_days": 3, "limit": 1000},
        now="2026-09-16T00:00:00Z")

    board = pr_scan.render_markdown(report).split("## Not on the board")[1].split("##")[0]
    table = "\n".join(line for line in board.splitlines() if line.startswith("|"))
    assert "alice" in table
    assert "carol" in table


def test_main_propagates_truncated_error(monkeypatch, tmp_path):
    def fake_fetch(repo, state, since, limit, until=None):
        raise pr_scan.TruncatedError("TRUNCATED: acme/repo merged returned too many items")

    monkeypatch.setattr(pr_scan, "fetch", fake_fetch)
    with pytest.raises(pr_scan.TruncatedError):
        pr_scan.main(["--org", "acme", "--repos", "repo", "--since", "2026-09-09",
                      "--out", str(tmp_path)])


def test_main_returns_2_and_reports_incomplete_repos_on_gh_failure(monkeypatch, tmp_path, capsys):
    def fake_fetch(repo, state, since, limit, until=None):
        if state == "merged":
            raise pr_scan.GhError("gh failed: rate limited")
        return []

    monkeypatch.setattr(pr_scan, "fetch", fake_fetch)
    code = pr_scan.main(["--org", "acme", "--repos", "repo", "--since", "2026-09-09",
                        "--out", str(tmp_path)])
    assert code == 2
    captured = capsys.readouterr()
    assert "INCOMPLETE: acme/repo" in captured.err


NOW = "2026-10-05T00:00:00Z"
WINDOW = {"since": "2026-09-28", "until": "2026-10-05"}


def _pr(number, login="alice", title=None, branch="chore/x", repo="acme/web",
        created="2026-09-30T00:00:00Z", merged=None, closed=None, review="",
        draft=False, bot=False, state=None, additions=None, deletions=None):
    raw = {"number": number, "title": title if title is not None else f"work {number}",
           "headRefName": branch, "body": "", "author": {"login": login, "is_bot": bot},
           "createdAt": created, "mergedAt": merged, "closedAt": closed, "isDraft": draft,
           "reviewDecision": review, "url": f"https://example.test/pr/{number}",
           "additions": number * 10 if additions is None else additions,
           "deletions": number if deletions is None else deletions}
    if state is None:
        state = "merged" if merged else ("closed" if closed else "open")
    return pr_scan.build_pr(raw, repo=repo, state=state, scope=["DL"], stale_days=3, now=NOW)


def _report(merged=(), open_prs=(), roster=("alice", "bob")):
    return pr_scan.build_report(
        list(merged), list(open_prs), window=dict(WINDOW),
        config={"org": "acme", "repos": ["web"], "keys": ["DL"], "stale_days": 3,
                "limit": 1000, "roster": list(roster)},
        now=NOW)


def _section(markdown, heading):
    body = markdown.split(heading + "\n", 1)[1]
    return body.split("\n## ", 1)[0]


@pytest.mark.parametrize(
    "repo,frontend,backend,expected",
    [
        ("org/web", {"web"}, {"api"}, "frontend"),
        ("org/api", {"web"}, {"api"}, "backend"),
        ("org/docs", {"web"}, {"api"}, "other"),
        ("org/web", {"org/web"}, set(), "frontend"),
        ("web", {"web"}, {"api"}, "frontend"),
        ("org/api", set(), {"org/api"}, "backend"),
    ],
)
def test_classify_stack(repo, frontend, backend, expected):
    assert pr_scan.classify_stack(repo, frontend, backend) == expected


def test_load_jira_map_reads_child_to_epic(tmp_path):
    path = tmp_path / "jira.json"
    path.write_text(json.dumps({"window": [], "child_to_epic": {"DL-2986": "DL-2047"}}))
    assert pr_scan.load_jira_map(str(path)) == {"DL-2986": "DL-2047"}


def test_load_jira_map_missing_file_raises(tmp_path):
    with pytest.raises(FileNotFoundError):
        pr_scan.load_jira_map(str(tmp_path / "absent.json"))


def test_load_jira_map_without_the_key_is_empty(tmp_path):
    path = tmp_path / "jira.json"
    path.write_text(json.dumps({"window": []}))
    assert pr_scan.load_jira_map(str(path)) == {}


def test_epic_for_returns_first_mapped_key():
    jira_map = {"DL-11": "DL-1", "DL-12": "DL-2"}
    assert pr_scan.epic_for({"keys": ["DL-10", "DL-11", "DL-12"]}, jira_map) == "DL-1"


def test_epic_for_returns_a_key_that_is_itself_an_epic():
    jira_map = {"DL-11": "DL-1"}
    assert pr_scan.epic_for({"keys": ["DL-1", "DL-99"]}, jira_map) == "DL-1"


def test_epic_for_returns_none_when_nothing_maps():
    assert pr_scan.epic_for({"keys": ["DL-99"]}, {"DL-11": "DL-1"}) is None
    assert pr_scan.epic_for({"keys": []}, {"DL-11": "DL-1"}) is None


def test_json_fields_request_closed_at():
    assert "closedAt" in pr_scan.JSON_FIELDS.split(",")


def test_build_pr_carries_closed_at():
    pr = _pr(7, closed="2026-10-01T12:00:00Z")
    assert pr["closed_at"] == "2026-10-01T12:00:00Z"
    assert pr["merged_at"] is None
    assert pr["stale"] is False


def test_gh_args_closed_state_searches_closed_window():
    args = pr_scan.build_gh_args("o/r", "closed", "2026-09-28", 100, "2026-10-05")
    assert args[args.index("--state") + 1] == "closed"
    assert "closed:2026-09-28..2026-10-05" in args


def test_gh_args_closed_state_is_open_ended_without_until():
    args = pr_scan.build_gh_args("o/r", "closed", "2026-09-28", 100)
    assert "closed:>=2026-09-28" in args


def test_closed_state_is_capped_by_search_api():
    args = pr_scan.build_gh_args("o/r", "closed", "2026-09-28", 5000, "2026-10-05")
    assert args[args.index("-L") + 1] == str(pr_scan.SEARCH_API_CAP)
    with pytest.raises(pr_scan.TruncatedError):
        pr_scan.check_not_truncated([{"number": n} for n in range(pr_scan.SEARCH_API_CAP)],
                                    5000, "o/r", "closed")


def test_build_epic_rollup_splits_by_stack_through_the_jira_map():
    fe = _pr(1, title="[DL-2986] form", repo="acme/web", merged="2026-10-01T00:00:00Z")
    be = _pr(2, title="[DL-2047] endpoint", repo="acme/api")
    rollup = pr_scan.build_epic_rollup([fe, be], {"web"}, {"api"}, {"DL-2986": "DL-2047"})
    assert rollup == {"DL-2047": {"frontend": {"merged": 1, "open": 0},
                                  "backend": {"merged": 0, "open": 1}}}


def test_build_epic_rollup_without_a_map_uses_the_in_scope_key():
    fe = _pr(1, title="[DL-2986] form", repo="acme/web", merged="2026-10-01T00:00:00Z")
    be = _pr(2, title="[DL-2986] endpoint", repo="acme/api")
    untracked = _pr(3, title="chore", repo="acme/api")
    rollup = pr_scan.build_epic_rollup([fe, be, untracked], {"web"}, {"api"}, {})
    assert rollup == {"DL-2986": {"frontend": {"merged": 1, "open": 0},
                                  "backend": {"merged": 0, "open": 1}}}


def test_build_epic_rollup_ignores_out_of_scope_keys_without_a_map():
    other = _pr(1, title="[QRS-9] theirs", repo="acme/web")
    assert pr_scan.build_epic_rollup([other], {"web"}, {"api"}, {}) == {}


def test_build_epic_rollup_drops_unmapped_prs_when_a_map_exists():
    loose = _pr(1, title="[DL-5] loose task", repo="acme/web")
    assert pr_scan.build_epic_rollup([loose], {"web"}, {"api"}, {"DL-2986": "DL-2047"}) == {}


def test_build_epic_rollup_counts_other_repos_only_when_present():
    docs = _pr(1, title="[DL-2986] docs", repo="acme/docs", merged="2026-10-01T00:00:00Z")
    rollup = pr_scan.build_epic_rollup([docs], {"web"}, {"api"}, {"DL-2986": "DL-2047"})
    assert rollup["DL-2047"]["other"] == {"merged": 1, "open": 0}
    assert rollup["DL-2047"]["frontend"] == {"merged": 0, "open": 0}


def test_build_stack_counts_counts_team_prs_and_sums_size():
    prs = [
        _pr(1, login="alice", repo="acme/web", merged="2026-10-01T00:00:00Z",
            additions=100, deletions=10),
        _pr(2, login="alice", repo="acme/web", additions=5, deletions=1),
        _pr(3, login="bob", repo="acme/api", merged="2026-10-02T00:00:00Z",
            additions=40, deletions=4),
        _pr(4, login="carol", repo="acme/api", merged="2026-10-02T00:00:00Z",
            additions=999, deletions=999),
        _pr(5, login="carol", title="[DL-7] ours by key", repo="acme/docs",
            additions=3, deletions=2),
    ]
    stacks = pr_scan.build_stack_counts(prs, {"web"}, {"api"}, {"alice", "bob"})
    assert stacks["frontend"] == {"merged": 1, "open": 1, "additions": 105, "deletions": 11}
    assert stacks["backend"] == {"merged": 1, "open": 0, "additions": 40, "deletions": 4}
    assert stacks["other"] == {"merged": 0, "open": 1, "additions": 3, "deletions": 2}


def test_opened_in_window_team_counts_team_prs_created_in_window():
    inside = _pr(1, login="alice", created="2026-09-29T09:00:00Z",
                 merged="2026-10-01T00:00:00Z")
    before = _pr(2, login="alice", created="2026-09-20T09:00:00Z")
    last_day = _pr(3, login="bob", created="2026-10-05T23:00:00Z")
    not_team = _pr(4, login="carol", created="2026-09-30T09:00:00Z")
    report = _report(merged=[inside], open_prs=[before, last_day, not_team])
    assert report["counts"]["opened_in_window_team"] == 2


def test_merged_in_scope_table_has_opened_merged_and_size():
    shipped = _pr(1, title="[DL-1] ship it", created="2026-09-25T08:00:00Z",
                  merged="2026-09-30T17:00:00Z", additions=120, deletions=7)
    table = _section(pr_scan.render_markdown(_report(merged=[shipped])),
                     "## Merged in window, in scope")
    rows = [line for line in table.splitlines() if line.startswith("|")]
    assert rows[0] == "| PR | Title | Author | Keys | Opened | Merged | Size |"
    assert rows[2] == ("| [acme/web#1](https://example.test/pr/1) | [DL-1] ship it | alice | "
                       "DL-1 | 2026-09-25 | 2026-09-30 | +120/-7 |")


def test_render_open_prs_lists_every_team_open_pr_oldest_first():
    fresh = _pr(1, login="alice", title="[DL-3] fresh", created="2026-10-04T00:00:00Z",
                review="REVIEW_REQUIRED")
    draft = _pr(2, login="bob", title="spike", created="2026-08-01T00:00:00Z", draft=True)
    approved = _pr(3, login="alice", title="[DL-4] ready", created="2026-09-15T00:00:00Z",
                   review="APPROVED")
    stranger = _pr(4, login="carol", title="not ours", created="2026-09-01T00:00:00Z")
    section = pr_scan.render_open_prs([fresh, draft, approved, stranger], {"alice", "bob"})
    lines = section.splitlines()
    assert lines[0] == "## Open PRs (team)"
    rows = [line for line in lines if line.startswith("|")]
    assert rows[0] == "| PR | Title | Author | Opened | Age | Draft | Review | Keys |"
    body = rows[2:]
    assert [row.split(" | ")[0] for row in body] == [
        "| [acme/web#2](https://example.test/pr/2)",
        "| [acme/web#3](https://example.test/pr/3)",
        "| [acme/web#1](https://example.test/pr/1)",
    ]
    assert "draft" in body[0]
    assert "approved, not merged" in body[1]
    assert "2026-10-04" in body[2] and "DL-3" in body[2]
    assert "carol" not in section


def test_render_open_prs_without_open_prs_says_none():
    section = pr_scan.render_open_prs([], {"alice"})
    assert section.splitlines()[0] == "## Open PRs (team)"
    assert "_none_" in section


def test_open_prs_section_always_renders():
    report = _report(open_prs=[_pr(1, login="alice", created="2026-09-01T00:00:00Z")])
    markdown = pr_scan.render_markdown(report)
    assert "## Open PRs (team)" in markdown
    assert markdown.index("## Merged in window, in scope") < markdown.index("## Open PRs (team)")
    assert "## Open PRs (team)" in pr_scan.render_markdown(_report())


def _report_with_extras():
    fe = _pr(1, title="[DL-2986] form", repo="acme/web", merged="2026-10-01T00:00:00Z")
    be = _pr(2, title="[DL-2047] endpoint", repo="acme/api")
    closed = _pr(3, title="[DL-5] abandoned", created="2026-09-20T00:00:00Z",
                 closed="2026-10-02T10:00:00Z")
    report = _report(merged=[fe], open_prs=[be])
    report["stacks"] = pr_scan.build_stack_counts([fe, be], {"web"}, {"api"}, {"alice"})
    report["epics"] = pr_scan.build_epic_rollup([fe, be], {"web"}, {"api"},
                                                {"DL-2986": "DL-2047"})
    report["closed_unmerged"] = [closed]
    return report


def test_new_sections_render_after_merged_in_scope():
    markdown = pr_scan.render_markdown(_report_with_extras())
    anchor = markdown.index("## Merged in window, in scope")
    for heading in ("## Frontend vs backend", "## By epic", "## Closed without merging"):
        assert heading + "\n" in markdown
        assert markdown.index(heading) > anchor


def test_stack_and_epic_sections_show_the_split():
    markdown = pr_scan.render_markdown(_report_with_extras())
    stacks = _section(markdown, "## Frontend vs backend")
    assert "| Frontend | 1 | 0 | +10/-1 |" in stacks
    assert "| Backend | 0 | 1 | +20/-2 |" in stacks
    epics = _section(markdown, "## By epic")
    assert "| DL-2047 | 1 merged, 0 open | 0 merged, 1 open |" in epics


def test_closed_without_merging_renders_the_closed_date():
    closed = _section(pr_scan.render_markdown(_report_with_extras()), "## Closed without merging")
    rows = [line for line in closed.splitlines() if line.startswith("|")]
    assert rows[0] == "| PR | Title | Author | Keys | Opened | Merged |"
    assert "closed unmerged 2026-10-02" in rows[2]
    assert "2026-09-20" in rows[2]


def test_extra_sections_absent_without_their_keys():
    markdown = pr_scan.render_markdown(_report(merged=[_pr(1, title="[DL-1] x",
                                                           merged="2026-10-01T00:00:00Z")]))
    for heading in ("## Frontend vs backend", "## By epic", "## Closed without merging"):
        assert heading not in markdown


GUARDED_SECTIONS = (
    "## Not on the board\n\nHuman-authored PRs carrying no ticket key anywhere in title, branch, "
    "or body. This is work the tracker cannot see. Repo-wide total above; table below is the "
    "team's PRs only, when a roster is configured.\n\n"
    "| PR | Title | Author | State | Age since opened | Size | URL |\n|---|---|---|---|---|---|---|\n"
    "| acme/web#2 | chore no key | alice | merged | 9d | +20/-2 | https://example.test/pr/2 |\n\n\n"
    "Plus 1 bot PRs with no ticket (app/dependabot) — dependency bumps, not team work. Not shown."
    "\n\n\n## Another team's ticket\n\nIn your repos, but filed under a key outside your scope. "
    "Repo-wide total above; table below is the team's PRs only, when a roster is configured.\n\n"
    "| PR | Title | Author | Keys | State |\n|---|---|---|---|---|\n"
    "| acme/web#3 | [QRS-9] theirs | bob | QRS-9 | open |\n\n\n"
    "## Stale, unreviewed open PRs\n\nStale, and nobody is reviewing it. An approved-but-unmerged "
    "PR is a different problem and is not listed here. Repo-wide total above; table below is the "
    "team's PRs only, when a roster is configured.\n\n"
    "| PR | Title | Author | Age | Review | URL |\n|---|---|---|---|---|---|\n"
    "| acme/web#3 | [QRS-9] theirs | bob | 15d | REVIEW_REQUIRED | https://example.test/pr/3 |\n"
    "| acme/web#4 | [DL-2] waiting | alice | 14d | REVIEW_REQUIRED | https://example.test/pr/4 |"
    "\n\n\n"
)


def test_existing_sections_render_exactly_as_before():
    merged = [
        _pr(1, title="[DL-1] ship it", branch="feature/DL-1/x", created="2026-09-25T00:00:00Z",
            merged="2026-09-30T00:00:00Z", review="APPROVED"),
        _pr(2, title="chore no key", created="2026-09-26T00:00:00Z",
            merged="2026-09-29T00:00:00Z"),
    ]
    open_prs = [
        _pr(3, login="bob", title="[QRS-9] theirs", branch="feature/QRS-9/x",
            created="2026-09-20T00:00:00Z", review="REVIEW_REQUIRED"),
        _pr(4, title="[DL-2] waiting", branch="feature/DL-2/x",
            created="2026-09-21T00:00:00Z", review="REVIEW_REQUIRED"),
        _pr(5, login="app/dependabot", title="bump x", branch="dependabot/x",
            created="2026-09-22T00:00:00Z", bot=True),
        _pr(6, login="carol", title="not ours", branch="chore/y", created="2026-09-22T00:00:00Z"),
    ]
    markdown = pr_scan.render_markdown(_report(merged=merged, open_prs=open_prs))
    start = markdown.index("## Not on the board")
    end = markdown.index("## Merged in window, in scope")
    assert markdown[start:end] == GUARDED_SECTIONS


def _raw(number, title, created, merged=None, closed=None, login="alice"):
    return {"number": number, "title": title, "headRefName": "chore/x", "body": "",
            "author": {"login": login, "is_bot": False}, "createdAt": created,
            "mergedAt": merged, "closedAt": closed, "isDraft": False,
            "reviewDecision": "", "url": f"https://example.test/pr/{number}",
            "additions": 1, "deletions": 1}


def _fake_github(calls):
    data = {
        ("acme/web", "merged"): [_raw(1, "[DL-2986] form", "2026-09-29T00:00:00Z",
                                      merged="2026-10-01T00:00:00Z",
                                      closed="2026-10-01T00:00:00Z")],
        ("acme/api", "open"): [_raw(2, "[DL-2047] endpoint", "2026-09-30T00:00:00Z")],
        ("acme/web", "closed"): [
            _raw(1, "[DL-2986] form", "2026-09-29T00:00:00Z", merged="2026-10-01T00:00:00Z",
                 closed="2026-10-01T00:00:00Z"),
            _raw(3, "[DL-5] abandoned", "2026-09-20T00:00:00Z", closed="2026-10-02T00:00:00Z"),
        ],
    }

    def fake_fetch(repo, state, since, limit, until=None):
        calls.append((repo, state, since, until))
        return data.get((repo, state), [])

    return fake_fetch


def test_main_writes_stacks_epics_and_closed_unmerged(monkeypatch, tmp_path):
    jira = tmp_path / "jira.json"
    jira.write_text(json.dumps({"child_to_epic": {"DL-2986": "DL-2047"}}))
    calls = []
    monkeypatch.setattr(pr_scan, "fetch", _fake_github(calls))
    code = pr_scan.main(["--org", "acme", "--repos", "web,api", "--since", "2026-09-28",
                         "--until", "2026-10-05", "--keys", "DL", "--roster", "alice",
                         "--frontend-repos", "web", "--backend-repos", "acme/api",
                         "--jira-map", str(jira), "--out", str(tmp_path)])
    assert code == 0
    assert ("acme/web", "closed", "2026-09-28", "2026-10-05") in calls
    report = json.loads((tmp_path / "prs.json").read_text())
    assert [pr["number"] for pr in report["closed_unmerged"]] == [3]
    assert report["closed_unmerged"][0]["closed_at"] == "2026-10-02T00:00:00Z"
    assert report["epics"] == {"DL-2047": {"frontend": {"merged": 1, "open": 0},
                                           "backend": {"merged": 0, "open": 1}}}
    assert report["stacks"]["frontend"]["merged"] == 1
    assert report["stacks"]["backend"]["open"] == 1
    assert report["counts"]["opened_in_window_team"] == 2
    markdown = (tmp_path / "prs.md").read_text()
    for heading in ("## Open PRs (team)", "## Frontend vs backend", "## By epic",
                    "## Closed without merging"):
        assert heading in markdown


def test_main_without_stack_or_map_flags_skips_those_sections(monkeypatch, tmp_path):
    monkeypatch.setattr(pr_scan, "fetch", _fake_github([]))
    code = pr_scan.main(["--org", "acme", "--repos", "web,api", "--since", "2026-09-28",
                         "--until", "2026-10-05", "--keys", "DL", "--out", str(tmp_path)])
    assert code == 0
    report = json.loads((tmp_path / "prs.json").read_text())
    assert "stacks" not in report
    assert "epics" not in report
    assert [pr["number"] for pr in report["closed_unmerged"]] == [3]


def test_main_rolls_up_by_key_without_a_jira_map(monkeypatch, tmp_path):
    monkeypatch.setattr(pr_scan, "fetch", _fake_github([]))
    pr_scan.main(["--org", "acme", "--repos", "web,api", "--since", "2026-09-28",
                  "--until", "2026-10-05", "--keys", "DL", "--frontend-repos", "web",
                  "--backend-repos", "api", "--out", str(tmp_path)])
    report = json.loads((tmp_path / "prs.json").read_text())
    assert set(report["epics"]) == {"DL-2986", "DL-2047"}


def test_main_reports_missing_jira_map_and_exits_1(monkeypatch, tmp_path, capsys):
    calls = []
    monkeypatch.setattr(pr_scan, "fetch", _fake_github(calls))
    missing = tmp_path / "absent.json"
    code = pr_scan.main(["--org", "acme", "--repos", "web", "--since", "2026-09-28",
                         "--jira-map", str(missing), "--out", str(tmp_path)])
    assert code == 1
    assert str(missing) in capsys.readouterr().err
    assert calls == []
    assert not (tmp_path / "prs.json").exists()


def test_main_reports_closed_fetch_failure_as_incomplete(monkeypatch, tmp_path, capsys):
    def fake_fetch(repo, state, since, limit, until=None):
        if state == "closed":
            raise pr_scan.GhError("gh failed: rate limited")
        return []

    monkeypatch.setattr(pr_scan, "fetch", fake_fetch)
    code = pr_scan.main(["--org", "acme", "--repos", "web", "--since", "2026-09-28",
                         "--out", str(tmp_path)])
    assert code == 2
    assert "INCOMPLETE: acme/web" in capsys.readouterr().err


def test_load_jira_map_rejects_a_file_that_is_not_an_object(tmp_path):
    path = tmp_path / "jira.json"
    path.write_text(json.dumps([{"child_to_epic": {}}]))
    with pytest.raises(ValueError):
        pr_scan.load_jira_map(str(path))
