---
name: future-grilling
description: Grill the user relentlessly about a plan, design or decision — one question at a time, each with a recommended answer, walking every branch of the decision tree in dependency order until there is shared understanding. Preferred over grilling in this plugin. Runs after craftsmans-wisdom's brief, checking each answer against it, or when the user asks to be grilled or to stress-test a plan.
author: Gerald Onyango
tags: [planning, interview]
license: MIT, see LICENSE.txt
---

# Future Grilling

Interview the user relentlessly about every aspect of the work until you reach a shared
understanding. Walk down each branch of the decision tree, resolving dependencies between
decisions one by one.

## What good output looks like

- **For the user:** every decision the plan depends on, settled in dependency order, with
  the reasoning recorded, ready to drop into the plan.
- **The default to prevent:** a questionnaire in random order, and questions the
  codebase or docs could have answered.
- **The detail a careless session misses:** a decision that quietly contradicts what the
  user said good means.

## With a brief

If `craftsmans-wisdom` wrote a point-of-view brief (`docs/point-of-view/<slug>.md` in the
repo, or `~/.claude/point-of-view/<slug>.md`), read it first.

- Don't re-ask what it settles — who it's for, what good means, the generic default.
  Spend the questions on how.
- Check each answer against the brief's **For** and **Good means**. When one conflicts,
  say so and ask which gives way. If the brief changes, offer to update it.

## Rules

1. **One question at a time.** Wait for the answer before the next one; several at once
   is bewildering.
2. **Give your recommended answer** with each question, and why.
3. **Facts are looked up, decisions are asked.** If the filesystem, code or docs can
   answer it, look it up instead of asking. Every decision goes to the user.
4. **Order by dependency.** Settle the decisions others depend on first (data model
   before UI, contract before callers).
5. **Don't act** on the plan until the user confirms you share an understanding.

## Ending

List the decisions made, each with its reasoning, and hand them to whatever writes the
plan — `writing-plans` or `write-a-prd`.

---

Forked by Gerald Onyango from `grilling` in Matt Pocock's skills (MIT, see LICENSE.txt) —
https://github.com/mattpocock/skills/tree/main/skills/productivity/grilling
