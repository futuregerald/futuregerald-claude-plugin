# Crit Sheet

Read this when writing or checking a crit sheet. It covers:

- Format
- Do/don't pairs
- Worked example

The format comes from a design crit at Stripe that Katie Dill showed in her talk: one ad, an overall note, and 17 lettered fixes, each pinned to a spot on the image.

## Format

```
**Overall:** <one paragraph: what matters most, what is working well enough to keep>

A. [<pin>] <what is off> → <what it should be or feel like>
B. [<pin>] <what is off> → <direction>, same treatment as A
```

| Part | Rule | `check_crit.py` enforces |
|---|---|---|
| Overall | One paragraph, before the items | present; no praise or score |
| Letters | A, B, C… in order, no gaps, no repeats | yes |
| Pin | `[where]`: a place a reader can find | non-empty |
| Body | what is off `→` what it should be or feel like | has `→` (or `->`) |
| References | "same as C", "similar to H", "see B", "match D", "matching E" | target letter exists |
| Wording | no praise ("looks great"), no scores ("8/10", "score: 7") | yes; a fraction like "step 3/10" also reads as a score, so write "step 3 of 10" |

An item can wrap onto following lines, and its text can start on the line after the pin. A blank line or the next letter ends it. A sheet holds at most 26 items (A–Z); past that, merge related items.

## Do/don't pairs

**Pin it.**
- Do: `C. [Pricing table, mobile] The third column clips at 390px → stack the plans vertically below 600px`
- Don't: `C. [Layout] Responsiveness could be improved → fix it`

**Say what it should feel like.**
- Do: `A. [Ice cube edges] Too sharp and glass-like → cloudier, with fine cracks, so it reads as ice, not plastic`
- Don't: `A. [Ice cube edges] Edges are wrong → adjust the edges`

**Point at the pattern.**
- Do: `E. [Lower-left cube] Same note as A → add cloudiness and fine crackle`
- Don't: repeat item A's full text five times. Name the pattern once in the Overall note and reference it.

## Worked example

A restaurant landing page built from a prompt. Brief: *for people deciding where to eat tonight; they care about the food, the price and whether they can walk in; good means they know all three within one screen.*

<!-- example:start -->
**Overall:** The page could sell any SaaS product, and the visitor's three questions (what's the food, what does it cost, can I walk in) are answered only below the fold. A, B and D matter most. The location finder and the clear opening hours are worth keeping.

A. [Hero headline] "Utah's #1 All-You-Can-Eat" reads like a software tagline
   → name what's on the grill and the price per person, so the first line answers "what and how much"
B. [Hero background] Dark gradient with no food in it → lead with a photo of the grill at the table; the food is the product
C. [Nav pill "View Menu"] Red pill styled like a SaaS "Start trial" button → a plain text link, same treatment as F
D. [Below the fold, "Walk-ins welcome"] The walk-in answer is buried → move it into the hero next to the price (see A)
E. [Location cards] Six identical cards with no distances → sort by distance from the visitor and show today's wait time
F. [Footer links] Four link styles across the page → one link style throughout, matching C
G. [Mobile, hero] The two CTAs wrap into a stack that hides the price → keep a single primary action: find the nearest location
<!-- example:end -->

The example passes `check_crit.py`. Keep it passing when you edit it.
