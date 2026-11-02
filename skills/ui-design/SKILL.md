---
name: ui-design
description: Generate 3 distinct, high-fidelity UI prototypes of the same screen, page, or app using huashu-design's 3-direction method and open a single interactive HTML comparison page (design-demos/index.html) in the browser so the user can compare, interact with, and choose a direction before implementation. Mandatory in Phase 0½ of the Development Lifecycle for net-new pages, new apps, and non-trivial/major changes to existing UI pages, or whenever the user invokes /ui-design.
tags: [ui, design, prototype, huashu-design, frontend, workflow]
---

# UI Design — 3-Prototype Interactive Showcase (`/ui-design`)

Generate **3 genuinely different, high-fidelity HTML prototypes of the exact same screen, page, or app** using [`../huashu-design/SKILL.md`](../huashu-design/SKILL.md), bundle them into a **single interactive HTML comparison page (`design-demos/index.html`)**, open that page in the user's browser, and **stop for the user's selection** before writing production UI code.

---

## When This Runs

1. **Automatically in Phase 0½ (`UI Design`) of `/lifecycle`** whenever the work involves:
   - A **net-new page** or route
   - A **new app** or product surface
   - **Major or non-trivial UI/UX changes** to an existing page or component flow
2. **Anytime the user invokes `/ui-design`** (or asks to see UI design options / prototypes).

**Only exemptions** (must be recorded in `direction-approved.md` if skipped during `/lifecycle`):
- Backend-only, CLI-only, or pure logic/data changes with no UI surface
- Trivial UI fixes (e.g., fixing a typo, adjusting a single spacing token, or fixing a broken click handler without changing layout/visual design)
- Follow-up iterations after the user has already picked a direction in `direction-approved.md` for this work

---

## Step-by-Step Workflow

### Step 1: Load Point-of-View Brief, Existing Design Context, and Real Assets
1. Read the confirmed Phase 0 brief (`docs/point-of-view/<slug>.md` or `~/.claude/point-of-view/<slug>.md`), `PRODUCT.md`, `DESIGN.md`, and any existing design tokens/components in the repo.
2. Read [`../huashu-design/SKILL.md`](../huashu-design/SKILL.md) and [`../no-slop-ui/SKILL.md`](../no-slop-ui/SKILL.md).
3. If real brands or products appear in the UI, fetch their real logos/assets first per `huashu-design` §1.a (`references/brand-asset-protocol.md`). Never use fake SVG approximations or `Lorem ipsum`.
4. Write a concise **shared design spec** covering:
   - Target user and primary job-to-be-done (from the Phase 0 brief)
   - Exact real content, data rows, labels, and interactive states all 3 prototypes must include
   - Target viewport (`1440×900` for desktop web, `390×844` for mobile/iOS app)

### Step 2: Generate 3 Distinct Prototypes of the Same Thing (`huashu-design` 3-Direction Gate)
Spawn 3 parallel sub-agents (or generate sequentially with strict context separation if sub-agents are unavailable). All 3 must render the **same real product content and user flow**, but each uses a **different layout skeleton, typography pairing, and visual philosophy**:

1. **Prototype 1 — Style Roulette (`design-demos/direction-1-roulette.html`):**
   - Run `date +%S`, compute `sec % 20 + 1`, and select that numbered style from [`../huashu-design/references/design-styles.md`](../huashu-design/references/design-styles.md) (Web 1–20, Deck 1–20, or Infographic 1–20).
   - Purpose: Break the model's deterministic "safe minimal SaaS" default.
2. **Prototype 2 — Benchmark Reference Migration (`design-demos/direction-2-benchmark.html`):**
   - Pick a specific, world-class real-world product, site, or award-winning interface (`Awwwards`, `Apple Design Award`, `Linear`, `Stripe`, `Read.cv`, `Raycast`, etc.) that fits the problem domain.
   - Translate its layout grammar, density, and interaction feel onto the user's real content.
3. **Prototype 3 — Dream Studio Bespoke (`design-demos/direction-3-studio.html`):**
   - Ask: *"If budget were unlimited, which world-class design studio or designer (e.g., Pentagram, Collins, IDEO, Kenya Hara, Jony Ive, Teenage Engineering, Work & Co) would we hire for this exact user and problem?"*
   - Design from first principles using that studio's signature philosophy and craft details.

**Hard quality rules across all 3 prototypes:**
- **Structural divergence:** The 3 prototypes MUST have visibly different layout skeletons (e.g., split-pane workbench vs. editorial single-column canvas vs. spatial command deck)—never the same wireframe with 3 color themes.
- **Readability floor:** Body copy `>= 14px`, labels `>= 12px`, contrast `>= 4.5:1`, and zero `no-slop-ui` tells (no purple-gradient everything, no emoji icons, no left-border fingernail cards, no default Inter/Roboto display headers).
- **Interactive:** Tabs switch, modals/drawers open, hover and active states respond.

### Step 3: Build & Open the Single 3-Prototype Comparison HTML Page (`design-demos/index.html`)
Combine all 3 prototypes into a single self-contained comparison page at `design-demos/index.html` based on [`assets/prototype-stage.html`](assets/prototype-stage.html):
- Embeds all 3 prototypes in isolated `<iframe srcdoc="...">` frames (or links `direction-1-roulette.html`, `direction-2-benchmark.html`, `direction-3-studio.html`) so their CSS/JS never collide and `file://` works directly without a server.
- Provides **two viewing modes** in one page:
  - **Side-by-Side 3-Up Grid (`G`):** See all 3 prototypes scaled proportionally side by side, each with its Direction title, studio/style anchor, palette swatches, and 1-sentence design thesis.
  - **Full-Viewport Interactive Mode (`1`, `2`, `3`):** Click any prototype card or top tab (`1`, `2`, `3`) to interact with that prototype at 100% viewport scale.
- Includes a **"Pick / Mix Direction"** bar that lets the user copy a one-line decision (e.g., *"Use Direction 2's layout with Direction 3's typography and palette"*) to paste back into chat.
- **Open the page immediately in the user's browser:**
  - macOS: `open design-demos/index.html`
  - Linux: `xdg-open design-demos/index.html`

### Step 4: Stop and Record the User's Choice
- **Stop your turn** after opening `design-demos/index.html` and presenting a 3-bullet summary of the 3 directions. Never pick a direction on the user's behalf.
- Once the user selects a direction (or a mix of elements from the 3 prototypes), record the choice verbatim in `direction-approved.md` (and update the Phase 3 implementation plan) before writing production UI code.
