# Design Skills Guide: Impeccable vs Huashu Design

## Overview

This plugin includes two complementary design skills. This guide explains when to use each.

## Comparison

| | **Impeccable** | **Huashu Design** |
|---|---|---|
| **Focus** | Design *vocabulary* — teaches the AI better design language so existing code output improves | Design *workflow* — turns the AI into a full design tool that produces deliverables |
| **Output** | Better-designed code in your existing stack (React, HTML, CSS, etc.) | Standalone HTML prototypes, slide decks (PPTX), animations (MP4/GIF), infographics (PDF/SVG) |
| **How it works** | Injects 7 domain-specific reference files (typography, color, spatial, motion, interaction, responsive, UX writing) + 20 slash commands | 20 design philosophies in 5 schools + auto-selects 3 and generates parallel demos when requests are vague |
| **Anti-slop** | Bans generic AI patterns via design vocabulary precision | Explicit rules banning purple gradients, emoji-as-icons, excessive border-radius, unmotivated serif fonts; uses OKLCH color space |
| **Best for** | Improving the design quality of production code you're already building | Creating standalone design artifacts — prototypes, presentations, motion graphics |
| **Agent support** | 54 agents (Claude Code, Cursor, Codex, Gemini CLI, etc.) | Claude Code, Cursor, Codex |

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

## Using Them Together

They're complementary, not competing. A typical workflow:
1. Use **Huashu Design** to prototype a concept and explore design directions
2. Use **Impeccable** to guide the production implementation with proper design vocabulary
