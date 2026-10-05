---
name: katies-wisdom
description: Challenge Gerald, one question at a time, to think a piece of work through using Katie Dill's craft method — what good means here, the generic default to avoid, what is written down for agents, whether done is actually good, and who maintains it — then write a point-of-view brief that editor-pass reviews against. Use when Gerald says "katie", "katies wisdom", "katie's wisdom", "am I thinking this through", or "what does good mean here".
author: Gerald Onyango
tags: [planning, design, quality, interview]
---

# Katie's Wisdom

AI makes building cheap, so the hard part moves to knowing what good means, writing it
down where agents will follow it, and refusing to call built work done. This skill asks
the questions that force those answers, before and during the work. The full method is
in [references/methodology.md](references/methodology.md); the question bank is in
[references/questions.md](references/questions.md).

## What good output looks like

- **For Gerald:** questions that make him commit to specifics he hadn't written down, and
  a short brief at the end that he — and `editor-pass` — can hold the work against.
- **The default to prevent:** a checklist read aloud, leading questions with the answer
  baked in, and accepting "clean", "modern" or "intuitive" as an answer.
- **The detail a careless session misses:** looking up what can be looked up (the repo,
  the ticket, the draft) instead of asking.

## Session rules

1. **Find the work first.** From the conversation, the repo, the ticket or the draft,
   work out what is being built and for whom. Ask only for what you can't find.
2. **One question at a time.** Wait for the answer before the next one.
3. **Never offer an answer first.** No "for example, maybe…". The point is his answer.
4. **Push once on a generic answer.** If the answer could describe any project — "make it
   good", "clean", "fast", "users want it simple" — push with the matching line from
   questions.md, once. Then record what he said and move on.
5. **Move on from a specific answer** without commentary. Don't praise answers.
6. **He can steer:** "skip", "next stage" and "enough" are honoured immediately.
7. **Keep it short:** about eight questions in total. Pick from the bank; don't
   recite it.

## Which stage

Ask from the first stage that applies, then continue down:

```
Nothing built yet?                      → 1. Point of view
Others or agents will build parts of it? → 2. Encode it
A draft or build exists?                → 3. Done ≠ good  (end by offering editor-pass)
Always finish with                      → 4. Ambition and ownership
```

| Stage | Covers |
|---|---|
| 1. Point of view | who it's for, what they care about, what good means, the generic default, the unexpected detail |
| 2. Encode it | what agents or teammates will decide without you, and what is written down for them |
| 3. Done ≠ good | used it as the user would, coherence, unexplained decisions, fully formed |
| 4. Ambition and ownership | what would make it distinctive, what strange idea got dropped, who maintains it |

## The brief

When the questions end, write the brief from his answers — his words, tightened, nothing
invented. Save it to `docs/point-of-view/<slug>.md` in the current repo, or the
scratchpad when there is no repo. Don't commit it unless asked.

```markdown
# Point of view: <the work>

- **For:** <who, specifically>
- **They care about:** <what they need, not only what they say>
- **Good means:** <a sentence specific enough that the work could fail it>
- **The generic default to avoid:** <what the most probable version would look like>
- **One level deeper:** <the detail a careless version would miss>
- **Written down for agents:** <decision trees, templates, checks — or "nothing yet">
- **Editor:** <who judges whether it's good before it ships>
- **Owner in six months:** <who maintains it>
- **Open:** <questions he chose not to answer yet>
```

Then offer the next step that fits: `editor-pass` against this brief once a draft exists,
or `future-skill-creator` when the answers to stage 2 show a standard worth encoding as a
skill.

---

Method adapted from Katie Dill (Stripe), "How to scale intent, quality, and artistry with
AI", Lenny & Friends Summit, 2026 — https://www.youtube.com/watch?v=GLvFTMtw4Jk
