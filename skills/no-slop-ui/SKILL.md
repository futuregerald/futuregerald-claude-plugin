---
name: no-slop-ui
description: "Eleven visual tells that make an interface read as AI slop — gradient-everything, rainbow palettes, pulsing status badges, fingernail cards, emoji icons, misaligned glyphs, default Inter/JetBrains Mono, leaked stack text, reflex glassmorphism, hype taglines, subject-blind templates — and what to do instead. Use when designing, building, restyling or reviewing any UI, landing page, dashboard, README banner, or generated frontend code."
author: Gerald Onyango
tags: [ui, design, frontend, review, anti-patterns]
---

# No Slop UI

Slop is generated UI with no vision behind it: decoration where nothing needed
decorating, labels for states that cannot vary, and copy written for a landing page
that isn't one. None of the tells below is a banned technique — each is a *default to
resist*. Use one when there is a reason, and be able to say the reason.

Slop is cumulative. One gradient is a choice; gradient plus purple plus a pulsing
badge plus a hype H1 is a tell.

## The eleven tells

1. **Gradients everywhere** — on buttons, cards, backgrounds, headings, and above all
   purple-to-blue. Default to flat fills. At most one gradient per screen, where it
   carries meaning (a scale, a fade over an image).
2. **Rainbow palette** — a different unrelated hue per card or section. Work to roughly
   70/30/10: one dominant neutral, one secondary, one accent. Hue encodes meaning
   (state, category), never variety.
3. **Pulsing badges** — animated "Active", "Live", "Verified" dots. Show a status only
   if the user can also see its opposite; name that opposite state and when it appears.
   A "Verified" badge backed by no verification is a lie. Animate only what changes.
4. **Fingernail cards** — everything in a heavy-radius, soft-shadow card, cards nested
   in cards. Pick one radius scale and one elevation rule; group with whitespace and
   alignment instead of more boxes.
5. **Emoji as icons** — in headings, bullets, buttons, section titles. Use the icon set
   or nothing. Emoji render differently per platform and set a tone most products
   don't want.
6. **Misalignment** — SVGs, icons, list markers and ASCII art sitting a few pixels off
   the text baseline. Align to a grid and the baseline, not by eye; check optical
   centring for any glyph inside a circle.
7. **Default type** — Inter for everything, JetBrains Mono the instant the subject is
   technical, plus `//` or `>` sprinkled on as decoration. Choose the typeface
   deliberately and drop the punctuation cosplay.
8. **Leaked context** — "Built with Hugo", "Written in Neovim", "Modern C++20
   features", framework names in a product hero or README banner. Say what the thing
   does for the reader; the stack belongs in docs.
9. **Reflex glassmorphism** — frosted translucent panels over a blurred blob, and its
   twin the identical-looking "brutalist" theme. Either is fine as a chosen language
   and poor as a default; whichever you pick, don't apply it to every surface.
10. **Hype copy** — "Elevate", "Seamless", "Next-generation", "Supercharge", "Unleash",
    "Empower"; an H1 trailed by a grey subtitle promising a paradigm; the two-beat
    tagline ("Ship faster. Sleep better.", "Wellness that fits real life."); "Welcome to
    your dashboard, [Name] ✨". Write what the screen does. A functional tool is not a
    landing page.
11. **One template for every subject** — the same hero, card grid, dashboard mock and
    gradient whether the page sells software, wellness or Korean barbecue. Every element
    can pass the ten checks above and the page is still slop, because it could belong to
    anyone. Run the swap test: replace the business name with a different kind of
    business — if nothing else would need to change, it's a template, not a design.
    Start from the subject's own world: its materials, its vocabulary, its photographs,
    and what a visitor actually comes to do (see the menu, book a table, find the
    location).

## Two checks that catch most of it

- **Point at each decorative element and name the job it does.** No job, delete it.
- **Read the copy back as a sentence a user would say.** If it sounds like the prompt
  that produced it, or like a press release, rewrite it plainly.

## When reviewing someone else's UI

Name the specific element and the fix — "the `Active` pill on the ID card has no
inactive state, drop it" — not the category. Flag a tell once per pattern, not once per
instance.

Adapted from "10 tells of a slop ui", hereticpleb, 2026-09-27.
Tell 11 and the two-beat tagline adapted from Katie Dill (Stripe), "How to scale intent,
quality, and artistry with AI", Lenny & Friends Summit, 2026 —
https://www.youtube.com/watch?v=GLvFTMtw4Jk
