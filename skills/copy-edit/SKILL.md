---
name: copy-edit
description: Optional on-demand slash command (/copy-edit) that audits a page, UI, flow, or document using both the copywriting and marketing-psychology skills and generates a structured report with concrete copy rewrites and behavioral-science suggestions. Use when the user invokes /copy-edit or asks for a combined copy edit and marketing psychology review report.
tags: [marketing, copywriting, psychology, review, report]
---

# Copy Edit & Marketing Psychology Report (`/copy-edit`)

Run a combined **Conversion Copywriting + Marketing Psychology** audit on a page, UI, onboarding flow, pricing table, feature, or document, and output a structured **suggestions report** (never mutate the source files automatically unless the user asks you to apply specific items afterward).

This skill is **optional** (not a mandatory lifecycle gate).

## Steps

1. **Load context and the target work:**
   - Read the target files, UI components, copy deck, or URL/screenshots provided by the user.
   - Check for an existing point-of-view brief at `docs/point-of-view/<slug>.md` or `~/.claude/point-of-view/<slug>.md` (plus `PRODUCT.md` / `DESIGN.md` if present) so every suggestion stays anchored to who the work is for, what they care about, and the generic default to avoid.
   - Load both underlying skills:
     - [`../copywriting/SKILL.md`](../copywriting/SKILL.md) (and its [`references/page-sections.md`](../copywriting/references/page-sections.md) / [`references/page-specific-guidance.md`](../copywriting/references/page-specific-guidance.md) when reviewing a full page)
     - [`../marketing-psychology/SKILL.md`](../marketing-psychology/SKILL.md) (and its reference files [`references/buyer-psychology.md`](../marketing-psychology/references/buyer-psychology.md), [`references/persuasion-and-pricing.md`](../marketing-psychology/references/persuasion-and-pricing.md), and [`references/design-and-growth.md`](../marketing-psychology/references/design-and-growth.md) as applicable)

2. **Audit through Lens 1 — Copy Editing & Conversion Copywriting (`copywriting`):**
   - **Clarity over cleverness:** Identify sentences trying to do too much, passive voice, hedging qualifiers ("almost", "very", "really"), and empty buzzwords ("streamline", "optimize", "innovative").
   - **Specificity & benefits over features:** Flag vague claims or raw feature labels that fail to state the concrete customer outcome.
   - **Customer language vs. internal jargon:** Flag leaked implementation terms or company-speak where voice-of-customer phrasing belongs.
   - **Above-the-fold & CTA strength:** Evaluate the Headline, Subheadline, and Primary/Secondary CTAs (`[Action Verb] + [What They Get]`). Propose 2–3 concrete alternative options with rationale.

3. **Audit through Lens 2 — Marketing Psychology & Behavioral Science (`marketing-psychology`):**
   - Select the specific mental models that bear on this page or flow (typically 4–7 models; never dump a generic list):
     - *Conversion & friction:* Hick's Law, Activation Energy, BJ Fogg (`B = MAP`), EAST, Curse of Knowledge, Paradox of Choice, Default Effect
     - *Value & pricing:* Jobs to Be Done, Anchoring, Framing, Loss Aversion, Regret Aversion, Mental Accounting, Good-Better-Best / Decoy Effect
     - *Trust & momentum:* Social Proof, Authority, Pratfall Effect, Reciprocity, Goal-Gradient Effect, IKEA Effect, Endowment Effect, Switching Costs, Status-Quo Bias
   - Diagnose where cognitive load, unaddressed risk, or missing behavioral triggers slow down or block the user's decision, and propose ethical, concrete interventions.

4. **Generate the Report:**
   - Present the report in chat (or as an Artifact in Antigravity when lengthy) using the structure below.

---

## Report Template

```markdown
# Copy Edit & Marketing Psychology Report: <Target Name>

## 1. Executive Summary & Top 3 High-Impact Changes
- **Target audience & primary action:** <1-2 lines from brief or inferred>
- **Biggest copy weakness:** <1 line>
- **Biggest psychological/friction bottleneck:** <1 line>
- **Top 3 changes to make first:**
  1. **[Location]** — <change> (<why it moves the needle>)
  2. **[Location]** — <change> (<why it moves the needle>)
  3. **[Location]** — <change> (<why it moves the needle>)

## 2. Copy Editing & Conversion Suggestions (`copywriting`)

### Above-the-Fold & CTA Options
- **Current Headline / Subheadline / CTA:** `<exact current text>`
- **Option A:** `<headline>` / `<subheadline>` / CTA: `<button text>` — *<rationale>*
- **Option B:** `<headline>` / `<subheadline>` / CTA: `<button text>` — *<rationale>*
- **Option C:** `<headline>` / `<subheadline>` / CTA: `<button text>` — *<rationale>*

### Line-by-Line Copy Edits
| # | Location (`file:line` or section) | Current Copy | Suggested Rewrite | Copywriting Principle |
|---|---|---|---|---|
| C1 | ... | "..." | "..." | Clarity / Specificity / Benefit over feature / Active voice |

## 3. Marketing Psychology Suggestions (`marketing-psychology`)

| # | Location / Flow Step | Mental Model | Current Friction or Missed Opportunity | Suggested Change & Expected Effect |
|---|---|---|---|---|
| P1 | ... | e.g. Activation Energy / Curse of Knowledge / Loss Aversion | ... | ... |
```

After presenting the report, ask which numbered items (`C1`, `P1`, etc. or Headline Option `A`/`B`/`C`) the user wants applied.
