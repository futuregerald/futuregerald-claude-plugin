# Report Payload

Step 4 writes one JSON payload and `pulsed` renders it. Start from an example in `assets/payloads/` and replace every value. Keep the keys the scope uses.

## 1. Which scope, which example

| Scope | `scope` | Start from | Notes |
|-------|---------|------------|-------|
| Team | `team` | `assets/payloads/team.json` | Renders with the `pulse` template |
| Epic | `epic` | `assets/payloads/team.json` | Same template as team; set `"scope": "epic"` and keep only the epic sections |
| Person | `person` | `assets/payloads/person.json` | Set `scope_label` to `1:1 Prep` or `Person Pulse`; renders with the `pulse-1on1` template |
| Initiative | `initiative` | `assets/payloads/initiative.json` | One initiative across its epics |

Forecast does not use a payload. It keeps its own script (see `references/forecast/method.md`, Phase 7).

## 2. Report section to payload field

| Report section (`references/report-format.md`) | Payload field |
|-----------------------------------------------|---------------|
| Headline | `headline` |
| Bottom line bullets | `bottom_line` |
| Executive Scorecard tiles | `tiles` (`label`, `value`, `context`, `severity`) |
| Strategic context | `strategic_context` |
| 02 epic cards | `epics` (`summary`, `why`, `activity`, `done`, `total`, `split`) |
| What's Left, per epic | `epics[].whats_left` |
| Epic groups and dependency notes | `epic_lists`, `dependencies` |
| 02b work breakdown | `work_breakdown` (`sides[].rows`, `still_to_do`, `closed_unmerged`) |
| 03 PR tables | `prs` |
| 04 person cards | `people`; the manager section is `manager` |
| 05 risk register | `risks` (`band`, `band_label`, `title`, `body`, `action`) |
| 1:1 scorecard | `one_on_one.prs_this_week`, `open_prs_now`, `tickets_done`, `reviews_given`, `overall` |
| 1:1 wins | `one_on_one.wins` |
| 1:1 talking points | `one_on_one.talking_points` |
| 1:1 questions | `one_on_one.questions` |
| Per-section notes | `section_notes` (see section 6) |

## 3. Prose fields

Prose fields hold Markdown only. No raw HTML. Links must be http, https or mailto. Put any `<...>` text inside backticks: PR titles are Prose, so `Support <Foo> tags` is written as `` Support `<Foo>` tags ``. Without the backticks, `<Foo>` is read as markup and the payload is rejected.

## 4. Plain-text fields

Every string field that is not Prose is plain text: `Epic.title`, `Epic.parent_title`, `Person.name`, tile labels and values, and the rest. Backticks do not help there. `pulsed` rejects any markup-looking `<` and any HTML entity.

- Decode entities: `&amp;` becomes `&`.
- Drop angle brackets: `Support <Foo> tags` becomes `Support Foo tags`.

This is the one content edit allowed under section 8.

## 5. Links

Run `pulsed config show --json` and read `links`. `pulsed` fills only empty URLs, and each kind needs its own base. Decide per field:

| URL | Leave it out only when |
|-----|------------------------|
| Ticket URLs (`url`, `ticket_url`, `parent_url`) | `tracker_browse` is set and the key's project is in `project_keys` |
| PR and repo URLs (`pr_url`, `repo_url`, `url` on a PR) | `code_host_org_url` is set |
| Person URLs (`profile_url`, `lead_url`, `author_url`, `person_url`) | `person_host` is set |

In every other case write the URL. `links` is absent when nothing is configured: write every URL. A missing URL does not fail validation; it renders unlinked.

## 6. Section notes

`section_notes[key]` takes one of `summary`, `epics`, `breakdown`, `prs`, `people`, `risks`. In the `pulse-1on1` template it replaces the section's static description. In the `pulse` templates it is appended after it.

## 7. Status and risk values

Status: `on_track`, `needs_attention`, `at_risk`, `blocked`, `not_rated`. They map to the Assessment Scale in `SKILL.md`: On Track, Needs Attention, At Risk, Blocked; `not_rated` when there is no evidence to rate.

Risk bands: `p0` (critical), `p1` (high), `p2` (watch). `band_label` is the text shown on the badge.

## 8. pulsed is the authority

`pulsed report submit` exits 2 when the payload is invalid and prints every error with its JSON path. Read every path, fix the payload, resubmit. Never strip content to get past validation, except the plain-text edit in section 4.

## 9. Meta fields

Leave `generated_at` out; `pulsed` fills it. Set `dashboard_path` only for the markdown render, to the HTML path `pulsed` printed.
