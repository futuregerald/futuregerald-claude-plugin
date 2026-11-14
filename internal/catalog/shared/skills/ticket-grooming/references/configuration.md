# Configuration

> **See [config-resolution.md](config-resolution.md) for the full precedence order.** This file
> documents the CLAUDE.md block, which is tier 2 of four.
>
> **If no configuration exists at all** — no cache, no CLAUDE.md block, no shipped `config.md` —
> that is the common case on a first run, not an error. Ask the user for what the ticket actually
> needs (usually: ticket system, site, and which repos to search), then offer to write a
> `.ticket-grooming.json` so the question is asked once. Do not guess a repo list from the
> directories you happen to see, and do not treat a bracketed placeholder as a value.

Add to CLAUDE.md to customize behavior. Replace every bracketed value — the skill treats a
bracketed placeholder as "not configured" and will ask before running.

```markdown
### Ticket Grooming
- Default ticket system: jira        # jira | github | linear
- Jira site: [yourcompany.atlassian.net]
- GitHub org: [your-org]
- dry-run: false
- grooming-mode: short  # short (default) | full (--full flag overrides this)
- Repos:
  - [repo-one]: ~/path/to/repo-one
  - [repo-two]: ~/path/to/repo-two
```

**Repos is a list, and the whole list gets searched.** Add every repo a ticket might touch —
grooming that searches only the first one reports "no code found" for work that plainly exists.

If one repo is being migrated into another, say so here (e.g. `# repo-two is migrating into
repo-one under components/`). Grooming reads that note and searches both while proposing fixes
only in the destination.
