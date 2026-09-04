# Accuracy Rules

Seven rules that prevent speculative, template-driven findings. Every phase of investigation must follow these.

## 1. Name it exactly or don't name it

Every class, method, file, and column you reference must match the codebase verbatim. If you didn't see it in this investigation (via `search_graph`, `Read`, or `trace_call_path`), don't include it. Cite evidence for every named entity — a GitHub permalink or graph query result.

Close-but-wrong names destroy trust. _Added after a false-positive where `DestroyResource` was cited but the actual class was `DestroyWithInvoiceUpdate`._

## 2. Show the mechanism, not just the name

Every hypothesis must cite three things:

1. **The symptom** — what the ticket describes or what the code produces
2. **The location** — `file:line` with a GitHub permalink
3. **The mechanism** — step-by-step trace from location to symptom, grounded in code you read

If you can't cite all three, mark it `[SPECULATION — not verified]` with LOW confidence. Don't present speculation alongside verified findings as if they're equal.

This applies especially to claims about callbacks, race conditions, N+1 queries, or framework behavior. Read the code — don't infer from names.

## 3. Challenge your own hypotheses

For every high/medium-confidence hypothesis, answer:

> "What is the strongest argument this hypothesis is wrong?"

Consider: does the code actually do what I claim? Is there a guard clause or upstream check that makes the failure path unreachable? Am I projecting from a similar ticket without verifying in THIS codebase?

If the counterargument holds, downgrade confidence or drop it. Include the counterargument in the final notes as `**Counterargument considered:**`.

_Added after a grooming run where three iterations investigated the wrong code path because no one challenged the initial hypothesis._

## 4. Label what's verified vs. speculative

Every claim falls into one of two categories:

- **Verified** — backed by code you read in this investigation, with a permalink
- **Speculative** — inferred from ticket text, names, or history; not confirmed by reading code

Label speculative claims clearly (`[speculative]` or LOW confidence). Readers must be able to tell at a glance which claims they can trust.

## 5. Stay on the reporter's actual problem

Before writing output, re-read the ticket title and description. Ask:

> "Do my root cause, findings, and suggested fix address the SPECIFIC problem the reporter described?"

Common mistakes:
- Investigating a **read** path when the ticket reports a **write** problem (or vice versa)
- Focusing on a **symptom** (how data displays) instead of the **cause** (how data is stored)
- Investigating an **adjacent** system that handles similar data but isn't the one described
- Letting the first interesting finding dominate even when it doesn't match the reported issue

If your findings don't address the ticket's stated problem, pivot before writing output — or clearly flag that your findings address a related-but-different issue.

_Added after a grooming run where investigation focused on the display filter (serializer) instead of the write path, wasting multiple iterations._

## 6. Respect framework behavior

Never assume what a framework method, convention, or mechanism does — read the source. Rails, Django, Express, and other frameworks have implicit behaviors (callbacks, association resolution, default scopes, middleware) that are invisible in the calling code. If your root cause or fix depends on how a framework feature works, you MUST verify it against the actual code.

**Mandatory checks before making framework-related claims:**

- **Column references:** Verify the column exists in `db/schema.rb` (or equivalent)
- **Association references:** Read the model's `belongs_to`/`has_many` definitions — association names differ between repos and resolve to FK columns implicitly
- **"Missing method" claims:** Check concerns, delegation, `method_missing`, and base classes before claiming a method doesn't exist
- **"N files affected" claims:** Verify EACH file individually. Never batch-count from grep results — what looks like the same pattern may behave differently due to different model definitions, concerns, or framework configuration.

See [framework-detection.md](framework-detection.md) for per-framework rules.

_Added after grooming flagged 8 files as buggy from grep results without reading the model's belongs_to definitions. Most files were correct in their own repo. A "removed column" root cause was fabricated — the actual issue was a wrong association name._

## 7. Leave a verification trail

Every HIGH or MEDIUM confidence claim must include a verification trail: what you checked, where you checked it, and what you found. The reader must be able to follow your trail and independently confirm each claim.

**Required for every finding:**
1. **What you checked** — "Read the Org model at `app/models/org.rb`"
2. **What you found** — "The model defines `belongs_to :owner, foreign_key: :assignee_id`"
3. **How it connects** — "Therefore `where(orgs: {owner: user})` resolves to `assignee_id`, which is correct"

**If you cannot provide all three, the finding is SPECULATIVE — label it LOW confidence.**

The verification trail is not optional polish. It is the evidence that distinguishes a verified finding from a guess. Claims without trails WILL be caught and rejected by the staff engineer review.

_Added after "HIGH confidence (verified)" was claimed without having read the model file that would have disproved the hypothesis._
