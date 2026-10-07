# Team Configuration

**This file ships empty. Fill it in on first run — the skill will offer to.**

This file is tracked in a public repo. **Copy it to `references/team.local.md` (git-ignored) and
fill that copy in — do not edit this tracked file with your real roster.** Everything below is a
template: delete the bracketed placeholders and replace them with your own team in your local
copy. Nothing in `references/team.local.md` is shared or uploaded; it stays in your local plugin
checkout.

If a section does not apply to you, delete it rather than leaving placeholders — the skill
treats a bracketed value as "not configured" and will ask again.

## Tracker

- **Type:** [jira | github | linear]
- **Project key:** [e.g. ABC]
- **Cloud ID:** [Jira only — from your Atlassian admin, or ask Claude to look it up]
- **Site:** [e.g. yourcompany.atlassian.net]
- **Excluded resolutions:** [Won't Do, Declined, Duplicate]

Excluded resolutions are left out of every epic's done and total counts. Every value must exist on
your Jira site, or every query fails: resolution names differ per site (`Cancelled` exists on some
and not others).

## GitHub

- **Org:** [your-org]
- **Primary repos:** [repo-one, repo-two, repo-three]
- **Frontend repos:** [repo-one]
- **Backend repos:** [repo-two, repo-three]

## Meetings (optional)

Which tool holds your meeting notes or transcripts, by its MCP server name. Leave this section out
and the skill uses whatever meeting or transcript tool is connected. Name one and it uses only
that one, so a document or wiki tool is never mistaken for meeting notes.

To name one, add a line here such as `- **Meeting sources:** <mcp-server-name>`, using the
server's name as it appears in your MCP configuration.

## Standing documents (optional)

Docs worth checking every run even when no epic links them: a roadmap sheet, a decision log, the
team's wiki space. Links only; Agent G checks their modified date and reads only what changed.

- [Roadmap sheet URL]
- [Decision log URL]

## Roster

One row per person. The **Notes** column is free-form and genuinely useful — put standing
instructions there ("assess by tickets groomed, not PRs" for a PM, "always include my own
activity in a dedicated section" for yourself).

The **Jira Account ID** column is optional. When it is empty, Jira assignees are matched to the
roster by display name instead, which misses anyone whose Jira name differs from the Name column.

| Name | GitHub Handle | Role | Jira Account ID | Notes |
|------|--------------|------|-----------------|-------|
| [Your Name] | [your-handle] | [Engineering Manager] | [account ID] | [Report author — include your own activity in its own section] |
| [Teammate] | [handle] | [Software Engineer] | | |

## Key Active Initiatives

Checked on every pulse report. Keep this short — three to five things that actually matter
this quarter.

| Initiative | Ticket Key | Status |
|-----------|-----------|--------|
| [Initiative name] | [KEY-123] | [Planning \| Active \| Blocked] |

## Leadership

Optional. Used for framing and escalation context.

| Name | Role |
|------|------|
| [Name] | [VP of Engineering] |

## Adjacent Teams

Optional. Used for cross-team context when work spans boundaries.

| Team | EM | PM | Ticket Key |
|------|----|----|-----------|
| [Team name] | [Name] | [Name] | [KEY] |
