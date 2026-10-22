# Organization, Tracker and Repo Configuration (example)

Copy this file to `~/.gemini/config/rules/<org>.local.md` (git-ignored) and fill in your organization's team, tracker and repository rules.

## 1. Team Roster
- List team members, GitHub handles, tracker account IDs and roles.

## 2. Issue Tracker & Repositories
- **Tracker:** site `[your-site.atlassian.net]`, cloud ID `[uuid]`, default project `[KEY]`.
- **GitHub org:** `[your-org]` and primary repositories.
- **Branch format:** `<type>/<TICKET>/<summary>` (`feature`, `fix`, `chore`, `maintenance` + ticket key, or `docs/NO-TICKET/...`).
- **PR conventions:** include `Closes <KEY>-XXXX` on work repos; auto-push and open PR after commit on personal repos (`[your-personal-org]/*`).
- **Observability & logging:** structured logger convention and Datadog service names.
