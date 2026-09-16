# Config resolution

Resolve configuration **once**, in Pre-Flight step 0. Without this the orchestrator rediscovers the
ticket system, the repo list, every local path and every GitHub slug on each run — several tool
calls before any investigation starts.

## Precedence

First match wins. A bracketed placeholder (`[your-org]`) counts as **not configured** — it is a
template that was never filled in, so skip that tier rather than passing `[your-org]` into a URL.

| # | Source | Scope |
|---|---|---|
| 1 | `.ticket-grooming.json` in the working directory | This repo. Gitignored, machine-local |
| 2 | A `### Ticket Grooming` block in `CLAUDE.md` (project, then user-level) | Project or user |
| 3 | `references/config.md` shipped alongside this skill | Whoever installed the skill |
| 4 | Ask the user | — |

**Tier 3 is where a private plugin puts company specifics**, so the skill itself stays generic and
publishable. The public copy ships [config.example.md](config.example.md) instead.

## When nothing resolves

Ask for what you need — and then offer to write tier 1 so the question is asked once, not forever:

```json
{
  "ticket_system": "jira",
  "jira_site": "example.atlassian.net",
  "jira_cloud_id": "...",
  "github_org": "example",
  "default_project": "ABC",
  "grooming_mode": "short",
  "auto_apply_fields": false,
  "repos": {
    "api": {"path": "~/dev/api", "slug": "example/api", "kind": "backend",
            "datadog_service": "api"},
    "web": {"path": "~/dev/web", "slug": "example/web", "kind": "frontend"}
  },
  "observability": {"retention_days": 14}
}
```

Write it only with the user's agreement, and add it to `.gitignore` — paths and site names are
machine- and company-specific, and it is config, not source.

## What is resolved per run regardless

- **HEAD SHA per repo** — `git rev-parse HEAD`, plus a check that the commit is pushed. Permalinks
  built on an unpushed SHA 404 for everyone else.
- **Index availability** — whether `codebase-memory-mcp` actually connected this session.
- **Issue type** of the ticket being groomed, when a field-level decision depends on it.

## What is NOT stored

Do not cache a project's field metadata. Jira field availability is per **issue type**, not per
project — a project commonly has a dozen, and a field present on Story is absent on Bug. Caching
one type's answer under the project key gives a wrong answer for the rest. If a specific field
write is ever needed, resolve it for that ticket's issue type at the time, passing
`requiredFieldsOnly: false` — the default is `true` and returns only creation-required fields,
which makes every optional field look absent.

Today the skill writes no custom fields, so this does not arise. Story points are printed in the
note text only.
