# Using an Index

An index makes the Impact Analysis fast. A **stale** index makes it fast and wrong, which is
worse than slow and right — it produces confident answers that feel verified, and confident
wrong answers are the exact failure the [[claim-ledger]] exists to remove.

## Which index

Use the hierarchy already defined in CLAUDE.md; do not invent a second one:

1. **graphify** — `graphify-out/graph.json`. Cross-module and architecture questions:
   `graphify query`, `graphify path "A" "B"`, `graphify explain`. AST-only, no API cost.
2. **codebase-memory-mcp** — `search_graph`, `query_graph`, `trace_call_path`,
   `get_architecture` for entity lookups and call paths.
3. **Semantic search** — `search_code` when you know the behaviour but not the name.
4. **Grep / Glob** — last, and for literal strings you already know.

Invisible edges (see [[impact-analysis]]) are grep territory no matter what. No index sees
`send`, string-named dispatch, or a CI YAML that greps source.

## Build once

If no index exists, build it **once, up front, and say so** before starting the analysis.
Never mid-plan: a real repo indexes at tens of thousands of nodes and edges, and a build in
the middle of tracing is a multi-minute stall that buys nothing you could not have grepped.

If a build would take longer than the analysis it serves, skip it and grep.

## Staleness

Check before trusting. An index built before the last three merges will confidently report a
caller set that no longer exists.

- Compare the index's build point against `git rev-parse HEAD`.
- If work happens in a worktree, the graph in the main checkout **does not contain it** —
  worktrees are the most common source of a silently stale answer.
- When in doubt, verify the specific claim with grep and record that in the ledger. One
  grepped fact beats a whole graph you are unsure of.

Never cite an index as evidence for a load-bearing claim without spot-checking at least one
result against the file.

## Repo scope

**Only ever index a project repository.**

```bash
root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0   # not a repo: do nothing
[ "$root" = "$HOME" ] && exit 0                                # never index $HOME
```

Refuse if `git rev-parse --show-toplevel` fails, if the result is `$HOME`, or if the
directory is a container of sibling repositories rather than a repo itself. A development
parent directory holding many checkouts must never be indexed — the result is enormous,
useless, and mixes unrelated codebases into one graph.

Index output must never reach a commit. `graphify-out/` is not in every repo's `.gitignore`,
so add it to `.git/info/exclude` before building, and stage files explicitly — never
`git add .` in a repo where the index lives in the tree.

## Graceful degradation

Index tooling being absent is normal, not an error.

- If the CLI is not installed, or the MCP server is not connected, or its tools do not
  resolve — **say so once, then continue with grep.** Do not retry, do not install anything,
  do not block the plan.
- State it in the plan so the reviewer knows how the analysis was built: "codebase-memory-mcp
  was configured but its tools did not resolve this session; call paths below were traced by
  grep."
- Degraded evidence is still evidence. A missing index lowers your confidence, and the honest
  place to record lowered confidence is the ledger's **Could Not Verify** section.

The failure to avoid is silently proceeding as though an index answered a question it never
saw.
