---
name: future-grilling
description: Grill the user relentlessly about a plan, design or decision — an interview with the user, not a review of a written plan file (that is plan-review) — one question at a time, each with a recommended answer, walking every branch of the decision tree in dependency order until there is shared understanding. Preferred over grilling in this plugin. Runs after craftsmans-wisdom's brief, checking each answer against it, or when the user asks to be grilled.
author: Gerald Onyango
tags: [planning, interview]
license: MIT, see LICENSE.txt
---

# Future Grilling

Interview the user relentlessly about every aspect of the work until you reach a shared
understanding. Walk down each branch of the decision tree, resolving dependencies between
decisions one by one.

## What good output looks like

- **For the user:** every decision the work depends on, settled in dependency order, with
  the reasoning recorded.
- **The default to prevent:** a questionnaire in random order, and questions the
  codebase, docs or tools could have answered.
- **The detail a careless session misses:** a decision that quietly contradicts what the
  user said good means.

## With a brief

Use the brief path you were given. Otherwise look for a `craftsmans-wisdom` brief at
`docs/point-of-view/<slug>.md` in the repo or `~/.claude/point-of-view/<slug>.md`, where
the slug is the kebab-case name of the work; when several exist and none clearly matches,
ask which applies.

- Don't re-ask what it settles — who it's for, what good means, the generic default.
  Spend the questions on how.
- Check each answer against the brief's **For** and **Good means**. When one conflicts,
  say so and ask which gives way. If the brief changes, offer to update it.
- Leave the brief's **Open** items alone unless a decision depends on one; the user left
  them open on purpose.

## Rules

1. **One question at a time.** Wait for the answer before the next one; several at once
   is bewildering. Exception: with 4 or more decisions left that don't depend on each
   other's answers, or when the user asks for a page, use `grilling-page`.
2. **Give your recommended answer** with each question, and why.
3. **Facts are looked up, decisions are asked.** If the filesystem, code, docs or tools
   can answer it, look it up instead of asking. Every decision goes to the user.
4. **Order by dependency.** Settle the decisions others depend on first (data model
   before UI, contract before callers).
5. **Don't act** on the work until the user confirms you share an understanding.

## Ending

List the decisions made, each with its reasoning. If a plan or PRD comes next, hand the
list and the brief to `writing-plans` or `write-a-prd` as settled, and tell it not to
re-ask what they cover. Otherwise stop at the list.

---

Forked by Gerald Onyango from `grilling` in Matt Pocock's skills (MIT, see LICENSE.txt) —
https://github.com/mattpocock/skills/blob/170ad48655825783d0193e850e31a9aac957bb95/skills/productivity/grilling/SKILL.md
