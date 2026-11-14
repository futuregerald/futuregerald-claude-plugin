# Ticket Grooming Configuration (example)

Copy this to `references/config.md` and replace every bracketed value, or let
`/ticket-grooming init` write a `.ticket-grooming.json` cache for you. See
[config-resolution.md](config-resolution.md) for precedence.

A bracketed placeholder counts as "not configured" — the skill asks rather than guessing.

**This repo is public.** `references/config.md` is gitignored here precisely because it holds
real site names, org slugs and local paths. Fill in the copy, never this template, and check
`git status` before committing if you are unsure.

```yaml
ticket_system: jira          # jira | github | linear
jira_site: [yoursite.atlassian.net]
jira_cloud_id: [uuid]
github_org: [your-org]
default_project: [KEY]
dry_run: false
grooming_mode: short         # short | full
auto_apply_fields: false     # true = write Jira labels/priority without asking

repos:
  [repo-one]:
    path: [~/path/to/repo-one]
    slug: [your-org/repo-one]
    kind: backend            # backend | frontend
    datadog_service: [service-name]
  [repo-two]:
    path: [~/path/to/repo-two]
    slug: [your-org/repo-two]
    kind: frontend
```

## Repos is a list, and the whole list gets searched

Add every repo a ticket might touch. Grooming that searches only the first reports "no code found"
for work that plainly exists.

If one repo is being migrated into another, say so here — the skill reads the note, searches both,
and proposes fixes only in the destination:

```yaml
  [repo-two]:
    migrating_into: [repo-one]
    migration_path: [components/two/]
```

## Observability

If your services have a log/metric retention window, state it so the skill does not claim history
it cannot see:

```yaml
observability:
  retention_days: [14]
```
