---
name: grilling-page
description: Puts any series of three or more questions for the user on one HTML page instead of asking them one at a time in chat — decisions left in a grilling session, open questions in a handoff or plan, review questions, choices needed before work starts. Each question shows the context, what waits on it, its source with links, what each option means and costs, and a recommendation with its reason. The answers come back in one paste and are recorded. Use whenever three or more questions are ready to ask, or when the user says "visualize the questions", "visualize the open questions", "a page", "a form", "put the questions on a page", "let me answer them all at once" or "batch the questions". One or two questions stay in chat. Not for open questions with no options, a chain where each answer changes the next question, or skills that forbid offering answers (craftsmans-wisdom, brainstorming).
author: Gerald Onyango
tags: [planning, interview, html]
---

# Grilling Page

Asking questions one at a time in chat turns into a long back-and-forth for answers the
user could give in one sitting. This skill puts them on one page, with the reasoning
written out, and takes the answers back in one paste. It serves any source of questions:
a grilling session, a handoff's or plan's open questions, a review, or the choices needed
before work starts.

`<skill-dir>` below means this skill's base directory (the directory containing this
`SKILL.md`).

## What good output looks like

- **For the user:** they can make every decision on the page without asking a follow-up
  or opening another file, then paste all their answers back at once.
- **The default to prevent:** a questionnaire: terse titles, bare option labels, jargon, a
  recommendation with no reason. The user comes back with questions instead of answers.
- **The detail a careless version misses:** an option that names how it works instead of
  what it means for users. "Use a background job" tells them nothing; "Imports finish in
  a few minutes and the page shows progress" lets them choose.

## Page or one at a time

Ask these in order; the first yes decides, except the no-options row, which takes that
question out and carries on with the rest.

```
User asked for one at a time?                                    → chat
The running skill forbids offering answers                       → chat, as that skill says
  (craftsmans-wisdom, brainstorming)?
A question has no real set of options                            → ask it in chat; count the rest
  ("which tenant hit this?")?
User asked for a page, or to visualize the questions?            → page
1 or 2 questions?                                                → chat
The next answer changes which options later questions even have? → settle it in chat, then count again
3 or more left?                                                  → page
```

A chain of dependent decisions stays in chat, one at a time, as `future-grilling` asks.
A page can hold a question that builds on an earlier one on the same page only when its
options make sense whichever way the earlier one goes.

## Process

1. **Gather the questions.** Look up anything the code, docs or tools can answer; only
   questions that need the user's decision go on the page. Note where each came from (a
   file and line, a ticket, a PR, an earlier answer), because each question names its
   source. Don't re-ask what earlier decisions or the `craftsmans-wisdom` brief settle.
   Give each an id that carries on from the last one asked this session (D1, D2… if none
   were numbered), so the user can refer to them.
2. **Order them by dependency**, the ones others build on first.
3. **Copy the template.** Copy `<skill-dir>/assets/question-page.html` to:

   | Questions from | Page | `{{STORE_KEY}}` |
   |---|---|---|
   | a grilling session | `docs/grilling/round-<N>.html` | `<project>-grilling-round-<N>` |
   | a document (handoff, plan, review) | beside it, `<source-stem>-questions-<N>.html` | `<project>-<path-slug>-questions-<N>` |
   | anything else | `docs/questions/<slug>-<N>.html` | `<project>-<slug>-questions-<N>` |

   `N` is the next round for that source, and `<path-slug>` is the document's
   repo-relative path, slugged, so no two pages share saved answers. Fill `{{TITLE}}` (a
   two-to-four-word name), `{{LABEL}}` (the header, e.g. "Recipe app · round 2"),
   `{{HEADING}}` (e.g. "Six decisions before the plan"), `{{INTRO}}` (one sentence on
   what is already settled and where), and in the `page-data` JSON block `{{STORE_KEY}}`
   and `{{ANSWER_HEADING}}` (the first line of the pasted answers). These six are plain
   text, except `{{INTRO}}` may use `<code>`, `<em>` and `<strong>`; the check script
   doesn't read them. Replace the example questions entirely. Change nothing else; the
   page already meets the Artifact rules.
   Its look is modelled on the GOV.UK Design System: a numbered list of the questions
   down the left showing each pick, the recommendation and its reason above large radio
   buttons or checkboxes, and a "Check your answers" list, with a Change link per row,
   above the plain text to copy.
4. **Write each question** to the standards below.
5. **Check it:** `python3 <skill-dir>/scripts/check_question_page.py <page>`.
   Fix every line it prints until it exits 0. Then read every question against all eleven
   standards: the script checks that fields exist and fit, not that they say the right
   thing.
6. **Commit the page**, so the record shows the options that were weighed, unless it sits
   beside a document the project keeps out of git (a local plan or handoff); then leave it
   uncommitted with that document.
7. **Publish**, if an Artifact tool is available: load `artifact-design` first if it is
   installed (the Artifact tool requires it), then publish the local file with a short
   generic icon word and a one-sentence description. Look at it once; fix what you see.
   Without an Artifact tool, give the local path to open in a browser.
8. **Hand it over:** the link, the local path, and one line: fill it in, press Copy
   answers, paste here. Then wait.

If a question changes before the user answers, edit the local file, re-run the check and
republish the same path so the link stays the same.

## Writing each question

Each question in the `questions` array has `id`, `title`, `context`, `why`, `blocks`,
`source`, `type` (`radio` for one answer, `checkbox` for "pick all that apply") and
`options`, each with `key`, `title`, `desc`, `cost` and `rec: true` on the recommended
ones. Ids and keys use only letters, digits, `-` and `_`. Text may use `<code>`, `<em>`
and `<strong>`; `source` may also hold links, `<a href="https://…">`, with no other
attribute.

The standards. The script checks those marked *(checked)*; the rest are yours to read for.

1. **The title is the question as the user would ask it**, ending in "?", at most 16
   words. *(checked)*
2. **The context gives everything needed to decide:** what is at stake, what users would
   notice, and any number, limit or earlier decision the choice rests on. At most 130
   words. *(length checked)* Before moving on, name the follow-up the user would most
   likely ask. If the context doesn't answer it, add the answer.
3. **Every technical term is replaced with plain words, or followed by a short clause
   saying what it is.**

   ```
   Do:    "...turns each recipe into a short list of numbers that captures what it's about
           (an 'embedding'), so similar recipes get similar numbers."
   Don't: "...generates embeddings for semantic retrieval."
   ```
4. **Every option says what picking it means for users**, in at most 50 words. No option
   without one, including "decide later", which says what happens meanwhile. *(checked:
   present and length)*

   ```
   Do:    "The household loses every recipe that person saved, with no warning."
   Don't: "Cascade delete on membership removal."
   ```
5. **The recommended option is marked and listed first.** A radio question has exactly
   one; a checkbox question has at least one. *(checked)*
6. **The why says why this pick beats the next-best one**, naming the trade-off, in at
   most 50 words. *(checked: present and length)*

   ```
   Do:    "Starting private means nobody shares a family recipe by accident. Sharing is one
           tap, so the cost is small."
   Don't: "This is the most flexible and scalable option."
   ```
7. **The options are real alternatives**, each one a person might sensibly pick. No
   strawman listed to make the recommendation look good. A checkbox question where
   "none of these" is a fair answer gets a "None" option, because an empty question
   reads as unanswered and "Use recommendations for the rest" would fill it.
8. **Blocks names the work waiting on the answer**, in at most 40 words, so the user can
   tell an urgent question from one that can wait. When nothing waits, say that.
   *(checked: present and length)*

   ```
   Do:    "Ticking two acceptance criteria and the wording of the evidence comment. The
           code doesn't wait on it."
   Don't: "Implementation."
   ```
9. **Source says where the question came from**, in at most 40 words, with a link or a
   `file:line` the user can open: the handoff section, the plan line, the review comment,
   the earlier answer. *(checked: present, length and https-only links)*
10. **Every option names its cost**: the work, risk or loss that comes with picking it,
    in at most 40 words. The recommended option has one too; a pick with no downside
    isn't a decision. *(checked: present and length)*

    ```
    Do:    "One more small PR and a production migration."
    Don't: "Some effort."
    ```
11. **Recommendations respect the brief.** None conflicts with the brief's **For** or
    **Good means**; where the honest recommendation does, say so in the context.

## When the answers come back

The paste lists each question with `Answer:`, `(recommended)` when the pick matches the
recommendation, and `Note:` when the user wrote one.

1. **Read every note first.** Call out each one that adds to or changes the picked
   option, and say how you will record it. A note that is ambiguous, or that asks a
   question, is held back and settled in chat, one question at a time.
2. **Check each answer against the brief** (**For**, **Good means**), as `future-grilling`
   does. A conflict is held back too: say which part of the brief it contradicts and ask
   which gives way.
3. **Record the rest immediately**, before discussing anything else. Questions that came
   from a document (handoff, plan, review notes) are answered there: write each answer
   into that document beside its question, and commit only if the document is committed.
   Any other answers go where the work they shape is tracked: its plan, or its ticket,
   after asking before any ticket edit. Grilling decisions are appended to the project's
   decisions file, `docs/decisions/<slug>.md`, where the slug is the brief's slug or the
   kebab-case name of the work, unless the project keeps decisions elsewhere. If the file
   doesn't exist yet, create it and record the decisions already settled in chat this
   session first, so the record is complete and numbered in order.
   Each entry gets a heading that states the decision with its number and question id,
   the date, the reason (the why when the user took the recommendation, otherwise their
   note, or ask), and what the note changed. Commit right after. Record each held-back
   decision, and commit, as soon as it is settled.
4. **Unanswered questions** go back through *Page or one at a time*, with any new ones
   the notes raised, as the next round.

When nothing is left, finish as `future-grilling`'s *Ending* says.
