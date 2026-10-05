---
name: editor-pass
description: Post-build quality review that asks whether finished work is good — not just done, correct or free of slop — and returns a lettered crit sheet of pinned, specific fixes. Use as the last gate before a user-facing page, artifact, report, PRD, ticket set or skill goes out, or when asked "is this good?", "give it an editor pass", "edit this for quality", "ready to ship?". Runs after correctness and slop checks, in a fresh sub-agent.
author: Gerald Onyango
tags: [review, quality, design, writing]
---

# Editor Pass

AI makes work look finished long before it is good. When building was expensive, ideas
were filtered before anyone built them; now the filter has to run after the build, when
saying no is harder. The editor is that filter, and answers for every decision in the
work.

`<skill-dir>` below means this skill's base directory (the directory containing this
`SKILL.md`).

## What good output looks like

- **For the person fixing it:** a crit sheet — one **Overall** note, then lettered items,
  each pinned to a place, saying what is off → what it should be or feel like, and
  pointing at related items ("same treatment as C").
- **The default to prevent:** scores, "looks great", category labels ("improve
  consistency"), and reviewing what the author meant instead of what they shipped.
- **The detail a careless pass misses:** the need the user never stated but will have.

## Where it sits

1. Correctness review — `comprehensive-code-review`, `plan-review`. Editor-pass does not
   hunt bugs.
2. Slop check — `no-slop-ui` for any UI.
3. **Editor-pass.**

Run it in a **fresh sub-agent**; the author cannot edit their own work objectively.
Dispatch with [references/dispatch-prompt.md](references/dispatch-prompt.md).

## Inputs

The reviewer is read-only — it has no browser, shell or network — so the main agent
turns everything it needs into files first:

- **The work:** file paths. For a URL, save the page to a file. For UI, walk the user's
  main task yourself and capture **one screenshot per step**, at phone and desktop width.
- **The point of view**, first match wins:
  1. a brief from `craftsmans-wisdom` — `docs/point-of-view/<slug>.md` in the repo, or
     `~/.claude/point-of-view/<slug>.md`, where the slug is the kebab-case name of the
     work; when several exist and none clearly matches, ask which applies
  2. PRODUCT.md / DESIGN.md, the ticket, or the PR description
  3. none — the editor writes three lines (who it's for, what they care about, what good
     means) above the Overall note, and the author confirms them

## The pass

1. **Load the point of view.** Review against it, never against generic taste.
2. **Use it as the user would** — see the table below. Note every point where it fails
   the user's real task or doesn't match how they think about it.
3. **Swap test.** Replace the subject (the business, the product, the reader) with a
   different one. If nothing else would need to change, it's a template, not a design.
4. **Coherence.** Read the whole journey or document in order. Flag disconnects, tone
   shifts, and parts that work alone but clash together.
5. **Green-cup audit.** List the visible decisions — colour, wording, order, defaults,
   empty states, which sections exist. Name the reason for each. A decision with no
   reason ("the cup is green, but may as well have been blue") is a default: flag it.
6. **One level deeper.** Name at least one need the user didn't state but will have, and
   whether the work meets it (the calendar tab showing today's date; the checkout that
   asks for the passport before the agent needs it).
7. **Fully formed?** Flag anything stubbed, hand-waved, or left at "good enough", and say
   what would finish it.
8. **Ownership.** Who maintains this, and what breaks when nobody touches it for six
   months?

### Step 2 by output type

| Output | Use it by… |
|---|---|
| UI / page | walking the per-step screenshots in task order, phone then desktop; mark any step with no screenshot "not reviewed" rather than imagining it |
| Report / doc | reading it as the named reader, with the question they came with — can they act after the first screen? |
| PRD / tickets | trying to start the work from it alone — list every question you'd have to ask |
| Skill | reading it as the agent that loads it, on a real task — what would it get wrong? |
| Code / API | reading it as its consumer would call it — names, errors, defaults, the first-run path |

## The crit sheet

```
**Overall:** <what matters most, and what is already working well enough to keep>

A. [<where>] <what is off> → <what it should be or feel like>
B. [<where>] <what is off> → same treatment as A
```

- One problem per item. Order by importance; the Overall note says which items matter
  most.
- The pin is a place a reader can find: `[Hero headline]`, `[§2 Risks]`,
  `[report.md:42]`, `[Checkout, step 3]`.
- Say what it should **feel like**, not only what to change ("so it reads as ice, not
  plastic").
- No praise items, no scores. What works goes in one clause of the Overall note.
- The editor proposes; it does not edit the work.

Format rules, do/don't pairs and a worked example:
[references/crit-sheet.md](references/crit-sheet.md).

## Validate, then present

The main agent — not the read-only sub-agent — checks the returned sheet. Write it to a
scratch file with the Write tool, then pass the path; never paste returned text into a
shell command, where a crafted line could run as a command.

```bash
python3 <skill-dir>/scripts/check_crit.py <scratch-dir>/crit-sheet.md
```

- **Exit 0:** present the sheet.
- **Exit 1:** send the printed problems back to the same sub-agent once (SendMessage). If
  the second sheet still fails, fix the format inline; never drop an item to pass.
- The script checks the shape — Overall, letters, pins, arrows, references, no praise or
  scores. Whether a fix is right stays the editor's judgment.

## After

The author decides which fixes to apply. Once they are applied, verify each lettered item
against the change rather than running a second full pass.

---

Method adapted from Katie Dill (Stripe), "How to scale intent, quality, and artistry with
AI", Lenny & Friends Summit, 2026 — https://www.youtube.com/watch?v=GLvFTMtw4Jk
