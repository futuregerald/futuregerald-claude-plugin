---
name: craftsmans-wisdom
description: Challenge the user, one question at a time, to think a piece of work through as a craftsman would — what good means here, the generic default to avoid, what is written down for agents, whether done is actually good, and who maintains it — then write a confirmed point-of-view brief that steers the build and that editor-pass reviews against, and offer to hand off to grilling for the details. Use before writing any plan, PRD or spec, and when the user says "craftsman's wisdom", "craftsmans wisdom", "am I thinking this through", or "what does good mean here".
author: Gerald Onyango
tags: [planning, design, quality, interview]
---

# Craftsman's Wisdom

AI makes building cheap, so the hard part moves to knowing what good means, writing it
down where agents will follow it, and refusing to call built work done. This skill asks
the questions that force those answers, before and during the work, and leaves a brief
that every later step reads.

- Read [references/questions.md](references/questions.md) before the first question.
- Read [references/methodology.md](references/methodology.md) only when a question needs
  its reasoning, or the user asks what the method says.

## What good output looks like

- **For the user:** questions that make them commit to specifics they hadn't written
  down, and a short brief, confirmed by them, that the builder and `editor-pass` both
  hold the work against.
- **The default to prevent:** a checklist read aloud, leading questions with the answer
  baked in, and accepting "clean", "modern" or "intuitive" as an answer.
- **The detail a careless session misses:** the answers the user has already written
  down elsewhere.

## When it runs

Before any plan, PRD or spec is written — the user should not have to invoke it.

- **A confirmed brief already covers this work:** read it back, ask whether it still
  holds, update it if not, and go straight to the next step.
- **Internal technical work** (a refactor, a dependency bump, a small fix): ask once
  whether to skip it. On "skip", go straight to planning.

## Before the first question

Read what already exists, so the session sharpens it instead of starting blank:

- an earlier brief: `docs/point-of-view/<slug>.md` in the repo, or
  `~/.claude/point-of-view/<slug>.md`
- `PRODUCT.md` and `DESIGN.md` (written by `impeccable`), `product-facts.md` (written by
  `huashu-design`)
- the ticket, PRD, plan, draft and the repo itself

**The slug** is the kebab-case name of the work (the feature, page or product, e.g.
`checkout-redesign`). When several briefs exist and none clearly matches, list them and
ask which applies.

**Facts versus judgments.** Look up facts — what is being built, the stack, the deadline,
what exists — and never ask for them. Always ask for judgments — who specifically it is
for, what good means, the generic default, the detail one level deeper — and never fill
one from a document someone else wrote. The exception is anything the user wrote
themselves (an earlier brief, their PRODUCT.md or PRD): read it back and ask what they
would change.

## Session rules

1. **One question at a time.** Wait for the answer before the next one.
2. **Never offer an answer first.** No examples, no menus of options. The point is their
   answer.
3. **Push once on a generic answer.** If it could describe any project, push with the
   matching line from questions.md, once, then record what they said and move on.
4. **Move on from a specific answer** without commentary or praise.
5. **The user steers:** "skip", "next stage" and "enough" are honoured immediately.
6. **Budget about eight questions.** Spend them on the required fields first (below),
   then on the stages that apply.
7. **Don't start building** until the user has confirmed the brief.

## Required in every session

Unless something the user wrote already answers them — and they confirm it still holds —
ask these three, whatever the stage:

| Brief field | Question |
|---|---|
| For | Who is this for, specifically? |
| Good means | What does good mean here? |
| Generic default | What would the generic version of this look like? |

## Which stages

Run each stage whose condition holds, in order:

```
Nothing built yet                         → 1. Point of view
Others or agents will build parts of it   → 2. Encode it
A draft or build exists                   → 3. Done ≠ good
Always                                    → 4. Ambition and ownership
```

1. **Point of view** — whose need this serves and what would make it theirs.
2. **Encode it** — what gets decided without the user in the room, and what is written
   down for it.
3. **Done ≠ good** — whether the built thing is actually good, not just finished.
4. **Ambition and ownership** — what would make it distinctive, and who keeps it alive.

Each brief field below names the stage whose questions fill it.

## The brief

When the questions end, write the brief from the user's answers — their words,
tightened, nothing invented — and **show it before saving**. Save only after the user
confirms or corrects it.

```markdown
# Point of view: <the work>

- **For:** <who, specifically>                                   (required)
- **They care about:** <what they need, not only what they say>  (stage 1)
- **Good means:** <a sentence the work could fail>               (required)
- **The generic default to avoid:** <the most probable version>  (required)
- **One level deeper:** <the detail a careless version misses>   (stage 1)
- **Written down for agents:** <trees, pairs, templates, checks> (stage 2)
- **Editor:** <who judges it good before it ships>               (stage 4)
- **Owner in six months:** <who maintains it>                    (stage 4)
- **Open:** <questions the user chose not to answer yet>
```

**Where it lives:** `docs/point-of-view/<slug>.md` in the repo, committed with the work
it describes, so a later session's `editor-pass` can find it. The brief can name people
and plans, so in a public repo ask before committing it, and on a no save it to
`~/.claude/point-of-view/<slug>.md` instead — the same path used outside a repo. When a
brief already exists, update it in place.

## Next step

- **Nothing built yet:** once the brief is saved, ask: "Do you want to be grilled on the
  details?"
  - **Yes:** run `grilling` with the brief's path as context. Tell it the brief settles
    who it's for and what good means, so it spends its questions on how — and checks
    each answer against the brief.
  - **No:** go straight to planning.
  - Either way, hand the brief to whatever builds next — `writing-plans`, `write-a-prd`,
    `impeccable` (fold it into PRODUCT.md), `huashu-design` or `brainstorming` — so it
    steers the build, not just judges it.
- **A draft exists:** offer `editor-pass` against the brief.
- **Stage 2 found a standard worth encoding:** offer `future-skill-creator`.

---

Method adapted from Katie Dill (Stripe), "How to scale intent, quality, and artistry with
AI", Lenny & Friends Summit, 2026 — https://www.youtube.com/watch?v=GLvFTMtw4Jk
