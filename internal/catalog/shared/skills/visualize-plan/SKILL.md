---
name: visualize-plan
description: Turns an implementation plan into a visual HTML document, in addition to the markdown plan, which stays the source of truth. Shows how data flows between parts and teams, the data model as an ER diagram, a worked example, each job's inputs, the choices made and why, the API, the work, and open decisions, in plain language. Publishes it as an Artifact and saves a standalone local copy. Use when asked to "visualize the plan", "make the plan pretty", "make a page for the plan", "HTML version of the plan", "show me the plan visually", or "make it pretty" while a markdown plan is the subject; also to update or republish an existing plan page after the plan changes. Not for styling an app's UI.
author: Gerald Onyango
tags: [planning, documentation, design, html]
---

# Visualize Plan

A plan in markdown is for the person building it. This page is for everyone else who has
to understand it: reviewers, the other team whose code it touches, the PM, the future
reader asking "why did we do it this way?". The markdown plan stays; this is an extra view
of it.

`<skill-dir>` below means this skill's base directory (the directory containing this
`SKILL.md`).

## What good output looks like

- **For a reader who hasn't seen the code:** in five minutes they can say what changes, how
  data moves and who owns each piece, what was decided and why, and what is still open.
- **The default to prevent:** the markdown pasted into styled HTML: a wall of headings and
  bullets, decorative hero, jargon, diagrams that don't match the plan.
- **The detail a careless version misses:** the page goes stale. Every plan change must
  reach the markdown, the page, the published link and the local copy, with removed names
  swept out of all of them.

## What you produce

Next to the plan file (`plan.md` → same folder):

| File | What it is |
|---|---|
| `plan.md` | Unchanged. The source of truth |
| `<stem>.html` | The page as an Artifact fragment. Publish this |
| `<stem>.standalone.html` | A complete page that opens from disk, built by script. Diagrams and fonts load from the network |

Same commit rule as the plan: if the plan is never committed, neither are these.

## Process

1. **Read the whole plan.** Every fact on the page comes from it, or from code it cites.
   If a section needs something the plan doesn't say, leave the section out. Ask only when
   the plan is too thin for the header and summary points. Never invent a number, name or
   behaviour. Example figures say they are examples.
2. **Choose the sections** with the tree below.
3. **Build from `<skill-dir>/assets/base.html`.** Copy it to `<stem>.html` and fill the
   placeholders. It already meets the Artifact page rules (tokens, both themes, phone
   width, fonts); add to it, don't restyle it. For each section, read only its entry in
   `<skill-dir>/references/components.md`. Before drawing, read
   `<skill-dir>/references/diagrams.md`.
4. **Write it plainly** (see Writing).
5. **Check it:** `python3 <skill-dir>/scripts/check_plan_page.py <stem>.html`. Fix every
   line it prints, until it exits 0.
6. **Publish**, if an Artifact tool is available: load `artifact-design` first if it is
   installed, then publish `<stem>.html` with a short generic icon word and a one-sentence
   description. Look at the rendered page once; fix what it shows. No Artifact tool → skip
   this step; the standalone copy is the deliverable.
7. **Save the local copy:** `python3 <skill-dir>/scripts/build_standalone.py <stem>.html`.
   It writes `<stem>.standalone.html` and loads Mermaid only when the page has a diagram,
   because the Artifact draws Mermaid itself and a local file can't.
8. **Reply** with the link (when published), the local path, and one line on anything left
   out because the plan didn't cover it.

## Choosing sections

Ask these in order; include every section that gets a yes. Page order follows this list.

```
Always                                              → header · summary points
Data moves between systems, jobs or teams?          → flow diagram
Input records turn into an output record?           → worked example
Work passes between jobs or services?               → handoff (what each step is sent)
Calls go back and forth between parts?              → sequence diagram, right after the handoff
An ordered process, maybe with branches?            → steps, branch table inside the branching step
A record moves through named states?                → state diagram, right after the steps
A choice between options was made?                  → choice cards
A choice a reviewer will question?                  → problem → answer table
Tables are created or changed?                      → schema cards + ER diagram
The UI shows computed numbers?                      → tiles, each saying how it's worked out
There is an API?                                    → endpoint cards
Existing code is reused or copied?                  → patterns table
Stories, affected code or rollback steps?           → work table + affected code + rollback
Known gaps are accepted?                            → accepted gaps
Always                                              → decisions: settled and open
```

A section the plan has no material for is left out, not filled with generalities.

## Writing

Plain language, the reader's words, short sentences. Real names (tables, jobs, flags,
files) in code font, each explained in plain words the first time it appears. Number
things only when order matters. No emoji, no hero banner, no "Overview" headings.

Headings state the point:

```
Do:    "Their rows can change; alerts can't"
Don't: "Data model considerations"
```

Say what happens, not the category it belongs to:

```
Do:    "A rebuild deletes and rewrites their rows, so a past alert could change"
Don't: "Mitigates data volatility in the upstream source"
```

Captions add what the picture can't show, usually the edge a reader would wrongly assume:

```
Do:    "Rebuilds call the service directly, not the job, so they never reach our tables"
Don't: "The diagram above shows the overall flow"
```

## Keeping it in sync

When the plan changes, in this order:

1. Edit the markdown plan.
2. Edit the same facts in `<stem>.html`: text, diagrams, ER diagram, tables, decisions.
3. Sweep for what was removed: grep both files for every name, flag or table that no
   longer exists, and fix each hit.
4. Run the check script.
5. Republish the same file path, which keeps the same link (skip without an Artifact tool).
6. Rebuild the standalone copy.

When the user deletes or rewords something on the page, make the same change in the
markdown if it says it there too.
