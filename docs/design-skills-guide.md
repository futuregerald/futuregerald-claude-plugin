# Design Skills Guide: Impeccable, Huashu Design, and UI Implementer

## Overview

This plugin includes three complementary design skills. This guide explains when to use each.

## Comparison

| | **Impeccable** | **Huashu Design** | **UI Implementer** |
|---|---|---|---|
| **Focus** | Design *vocabulary* — teaches the AI better design language so existing code output improves | Design *workflow* — turns the AI into a full design tool that produces deliverables | Design *implementation* — translates a visual reference into pixel-perfect production code |
| **Output** | Better-designed code in your existing stack (React, HTML, CSS, etc.) | Standalone HTML prototypes, slide decks (PPTX), animations (MP4/GIF), infographics (PDF/SVG) | Production components in React 19, Vue 3.5+, or Svelte 5 matching a design reference |
| **How it works** | Injects 7 domain-specific reference files (typography, color, spatial, motion, interaction, responsive, UX writing) + 20 slash commands | 20 design philosophies in 5 schools + auto-selects 3 and generates parallel demos when requests are vague | Auto-detects framework from package.json, dispatches a UI Developer agent, then validates output via screenshot comparison against the design reference |
| **Anti-slop** | Bans generic AI patterns via design vocabulary precision | Explicit rules banning purple gradients, emoji-as-icons, excessive border-radius, unmotivated serif fonts; uses OKLCH color space | Stagnation detection halts repeated identical changes; CRITICAL regression checks prevent visual regressions |
| **Best for** | Improving the design quality of production code you're already building | Creating standalone design artifacts — prototypes, presentations, motion graphics | Implementing a specific design (Figma URL, screenshot, mockup) as production code with verified fidelity |
| **Agent support** | 54 agents (Claude Code, Cursor, Codex, Gemini CLI, etc.) | Claude Code, Cursor, Codex | Claude Code (adaptive switching between UI Developer and UI Developer Codex) |
| **Requires design reference** | No | No | Yes (Figma URL, screenshot, or mockup) |

## Invocation

**Neither skill is invoked automatically.** Both require explicit invocation:

- **Impeccable**: Use `/impeccable` slash commands (e.g., `/impeccable craft`, `/impeccable shape`, `/impeccable teach`). The skill description tells the AI *when* to suggest it, but it won't activate without a slash command or explicit user request.
- **Huashu Design**: Use trigger words like "prototype", "design demo", "UI mockup", "make slides", "animation demo", "design exploration", or explicitly invoke `/huashu-design`. It activates when the AI detects design-related intent in your request.

## When to Use Which

**Use Impeccable when:**
- You're building a feature and want the UI code to come out well-designed from the start
- You want to audit, polish, or critique existing UI code
- You're working in your production codebase and want design guidance inline
- You need to iterate on live UI elements in the browser

**Use Huashu Design when:**
- You need standalone design deliverables (prototypes, decks, animations)
- You want to explore multiple design directions in parallel before committing
- You're producing visual assets like infographics or motion graphics
- You want to prototype in HTML without touching your production codebase
- You need to export to PPTX, MP4/GIF, or PDF

## When to Use UI Implementer

**Use UI Implementer when:**
- You have a specific design reference (Figma URL, screenshot, or mockup) and want to implement it as production code
- You need pixel-perfect fidelity between the design and the implementation
- You want automated visual validation — the skill captures screenshots of the implementation and compares them to the reference
- You're working in React 19, Vue 3.5+, or Svelte 5 and want framework-idiomatic code generated automatically
- You need adaptive agent switching — the skill picks between UI Developer and UI Developer Codex based on model availability

**Don't use UI Implementer when:**
- You're designing from scratch without a reference — use Huashu Design or Impeccable instead
- You need standalone design artifacts (slides, animations, PDFs) — use Huashu Design
- You want to improve design quality of code you're already writing — use Impeccable

## Key Differences: UI Implementer vs Others

1. **Design reference required**: UI Implementer always starts from a visual reference. Impeccable and Huashu Design can work from text descriptions alone.
2. **Validation loop**: UI Implementer includes a built-in designer agent that captures screenshots (via Puppeteer) and compares them against the design reference, iterating until fidelity is achieved. The other skills don't have automated visual validation.
3. **Multi-framework support**: UI Implementer auto-detects your framework from `package.json` and generates idiomatic code for React 19, Vue 3.5+, or Svelte 5. It loads framework-specific conventions from separate reference files.
4. **Stagnation detection**: If the implementation isn't converging (same changes repeated), the skill detects this and halts with a report rather than looping endlessly.
5. **CRITICAL regression checks**: Before completing, the skill verifies no visual regressions were introduced in already-approved areas.

## Using Them Together

They're complementary, not competing. A typical workflow:
1. Use **Huashu Design** to prototype a concept and explore design directions
2. Use **Impeccable** to guide the production implementation with proper design vocabulary
3. Use **UI Implementer** when you have a finalized design (from Figma, Huashu, or any other source) and want to implement it as verified, pixel-perfect production code
