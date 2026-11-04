---
name: shape
description: Run Phase 0 (Shape) of the Development Lifecycle before planning — interviews the user with craftsmans-wisdom to produce a confirmed point-of-view brief, offers future-grilling to walk the decision tree in dependency order, and hands the brief to write-a-prd, brainstorming, or writing-plans. Use when the user invokes /shape or before planning new or user-facing work.
tags: [planning, design, quality, interview, workflow]
---

# Phase 0: Shape (`/shape`)

Run **Phase 0 (Shape)** of the Development Lifecycle before writing an implementation plan (`writing-plans`).

## Steps

1. **Check for an existing brief:**
   - Look for `docs/point-of-view/<slug>.md` in the repo or `~/.claude/point-of-view/<slug>.md`.
   - Look up facts from `PRODUCT.md`, `DESIGN.md`, `product-facts.md`, the ticket, PRD, and codebase so you never ask the user for facts the workspace already answers.
   - If a confirmed brief from an earlier session already covers this work, read it back, ask whether it still holds, update it if needed, and proceed to Step 3.

2. **Point-of-view brief (`craftsmans-wisdom`):**
   - Read and follow [`../craftsmans-wisdom/SKILL.md`](../craftsmans-wisdom/SKILL.md) (and its [`references/questions.md`](../craftsmans-wisdom/references/questions.md)).
   - Ask **one question at a time** in chat, with **no suggested answers or menus**, covering the required judgments:
     - **For:** who specifically it is for
     - **Good means:** what good means for this work
     - **The generic default to avoid:** what the careless/generic version looks like
     - Plus the applicable stage questions (Point of view, Encode it, Done isn't good, Ambition and ownership).
   - Push once on vague answers ("clean", "modern", "intuitive"). Honour `"skip"`, `"next stage"`, and `"enough"` immediately.
   - Present the tightened brief in the user's words and get confirmation before saving it to `docs/point-of-view/<slug>.md` (or `~/.claude/point-of-view/<slug>.md` on public repos when preferred).

3. **Decision-tree grilling (`future-grilling`):**
   - Once the brief is confirmed, offer to grill the details using [`../future-grilling/SKILL.md`](../future-grilling/SKILL.md).
   - On a yes: walk every branch of the decision tree in dependency order (data model before UI, contract before callers), asking one question at a time **with a recommended answer and why** (or using [`../grilling-page/SKILL.md`](../grilling-page/SKILL.md) if ≥3 remaining decisions are independent). Check every answer against the brief's **For** and **Good means**.
   - If the decisions won't fit one session (they need research, prototypes or several sittings), run [`../future-wayfinder/SKILL.md`](../future-wayfinder/SKILL.md) with the brief instead; it hands off to step 4 when its map is clear.

4. **Hand off to planning:**
   - Pass the confirmed brief and settled decision list into `write-a-prd`, `brainstorming`, or Phase 3 (`writing-plans`), instructing it not to re-ask what the brief and grilling already settled.
