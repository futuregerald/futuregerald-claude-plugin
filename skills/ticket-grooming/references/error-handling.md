# Error Handling

| Failure | What to do |
|---------|------------|
| Ticket not found | Fail fast, tell the user |
| Sub-agent exceeds context | Summarize what it has, note "investigation truncated" |
| Comment post fails | Show notes in conversation so nothing is lost |
| Wrong comment format posted | Delete via `acli jira workitem comment delete --key {KEY} --id {ID}`. Re-post: full mode uses `contentFormat: "markdown"` via MCP; short mode MUST use ADF JSON via `acli --body-file` (markdown cannot produce the expand node). See [adf-posting.md](adf-posting.md). |
| MCP tools unavailable | Tell user which tools are needed; fall back to `acli` for unsupported ops |
| Repo not cloned locally | Skip that repo, note in findings, ask user for path |
| Sub-agent fails in a batch | Others continue; report failure with partial findings |
| GitHub remote detection fails | Use relative paths instead of permalinks |
| Staff review fails | Post unreviewed notes with disclaimer |
