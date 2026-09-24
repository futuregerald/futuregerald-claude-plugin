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
