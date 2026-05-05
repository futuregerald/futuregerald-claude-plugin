---
name: ui-implementer
description: Implements UI components from scratch based on design references (Figma, screenshots, mockups) with intelligent validation and adaptive agent switching. Supports React, Vue, and Svelte 5 with auto-detection. Use when user provides a design and wants pixel-perfect UI implementation with design fidelity validation. Triggers automatically when user mentions Figma links, design screenshots, or wants to implement UI from designs.
allowed-tools: Task, AskUserQuestion, Bash, Read, Write, TodoWrite, Glob, Grep
author: tianzecn (github.com/tianzecn/myclaudecode)
contributors: Gerald Onyango (github.com/futuregerald) -- multi-framework support (React/Vue/Svelte 5), Puppeteer-first screenshot capture, validation loop hardening, framework-specific conventions
---

# UI Implementer

Implements UI components from scratch based on design references (Figma, screenshots, mockups) using specialized sub-agents with a validation loop and adaptive agent switching.

## When to use this Skill

- User provides a **design reference** (Figma URL, screenshot path, remote URL) and wants UI implemented from it
- User says "implement this design", "build from this Figma", "match the mockup"
- User wants **pixel-perfect** implementation from a design

**Do NOT use** when the user wants to validate existing UI, fix existing components, or implement without a design reference.

## Instructions

**ALL PHASES (0-4) ARE MANDATORY. NEVER SKIP A PHASE OR STEP. NO EXCEPTIONS.**

A "simple" component follows the exact same lifecycle as a complex one. Do not shortcut Phase 3 because it "looks right." Every phase exists because skipping it has caused failures.

**Phase gates:** Do NOT proceed to the next phase until the current phase is fully complete.

### PHASE 0: Initialize Workflow (REQUIRED FIRST)

**You MUST create this todo list before doing anything else.**

```
TodoWrite with:
- PHASE 1: Gather inputs (design reference, framework detection, component description, preferences)
- PHASE 1: Load framework instructions and validate inputs
- PHASE 2: Launch UI Developer for initial implementation
- PHASE 3: Start validation and iterative fixing loop
- PHASE 3: Quality gate - ensure design fidelity achieved
- PHASE 4: Generate final implementation report
- PHASE 4: Present results and complete handoff
```

### PHASE 1: Gather User Inputs (Steps 1-8, ALL REQUIRED)

**Complete every step in order. Do not skip steps even if you think you can infer the answer.**

**Step 1: Extract Design Reference**
- Scan user's message for Figma URLs, file paths, or remote URLs
- If found, store as `design_reference`
- If NOT found, ask user for their design reference (Figma URL, file path, or remote URL)

**Step 2: Detect Framework**
1. Read `package.json` (use Read tool). Also check `target_location` directory for monorepos.
2. Check deps for: `react`/`react-dom` -> React, `vue` -> Vue, `svelte` -> Svelte
3. Edge cases: no package.json, multiple frameworks, none found -> ask user directly
4. Confirm with user. Store as `framework` (one of: `react`, `vue`, `svelte`).

**Step 3: Extract Component Description**
- Scan user's message for component names or descriptions
- If NOT found, ask what component(s) to implement. Store as `component_description`.

**Step 4: Ask for Target Location**
- Ask where to create the component files (specific path, or let you suggest). Store as `target_location`.

**Step 5: Ask for Application URL**
- Ask for the preview URL (e.g., `http://localhost:5173`). Store as `app_url`.

**Step 6: Ask for UI Developer Codex Preference**
- Ask if the user wants intelligent agent switching (UI Developer + Codex). Store as `codex_enabled` (boolean).

**Step 7: Load Framework Instructions (CRITICAL -- do not skip)**

You MUST read the framework-specific instruction files using the Read tool. Sub-agents cannot read skill files from the filesystem -- you must inline the contents into their prompts.

1. Read `frameworks/shared.md` from this skill's directory
2. Read `frameworks/[framework].md` from this skill's directory
3. Store the file contents as `shared_conventions` and `framework_conventions`

**Step 8: Validate Inputs**
- Design reference format valid (Figma/Remote/Local)
- Component description not empty
- Target location valid
- Application URL valid
- Framework is one of: `react`, `vue`, `svelte`

### PHASE 2: Initial Implementation from Scratch

**STOP.** Verify ALL Phase 1 variables are set: `design_reference`, `framework`, `component_description`, `target_location`, `app_url`, `codex_enabled`, `shared_conventions`, `framework_conventions`. If any missing, go back.

1. Read `prompts/implement.md` from this skill's directory
2. Inline `shared_conventions` and `framework_conventions` into the marked sections
3. Launch UI Developer agent (Task with `subagent_type: frontend:ui-developer`) using the completed prompt
4. Wait for completion

### PHASE 3: Validation and Adaptive Fixing Loop (MANDATORY -- do not skip)

**Phase 3 is NOT optional.** "It looks right to me" is not validation -- only a designer agent comparing screenshots counts.

1. Read `validation-loop.md` from this skill's directory and initialize the loop variables
2. Read `prompts/validate.md` from this skill's directory for the designer prompt template

**Loop: While iteration_count < max_iterations AND NOT design_fidelity_achieved**

**Step 3.1: Capture Screenshots**
Read `screenshot-capture.md` and follow its instructions to capture desktop + mobile screenshots from `app_url` and the design reference. Store paths as `impl_screenshot`, `impl_screenshot_mobile`, `design_screenshot`.

**Step 3.2: Launch Designer for Validation**
Use the prompt template from `prompts/validate.md` with Task (`subagent_type: frontend:designer`).

**Step 3.3: Check Results**
Extract from designer report: assessment, total issue count, CRITICAL issue count, fidelity score. Run the CRITICAL regression check from `validation-loop.md`. If PASS, set `design_fidelity_achieved = true` and exit loop.

**Step 3.4: Determine Fixing Agent**
Use the smart switching logic from `validation-loop.md` to pick the agent.

**Step 3.5: Launch Fixing Agent**
Set `last_agent_used`, then launch the chosen agent with designer feedback.

**Step 3.6: Update Metrics**
Follow the metrics update and stagnation detection logic from `validation-loop.md`.

Continue loop until design fidelity achieved or max iterations reached.

### PHASE 4: Final Report & Completion

Generate a report covering:
- **Component info**: description, design reference, location, preview URL
- **Implementation**: files created, components list
- **Validation**: iterations, final status (PASS/NEEDS IMPROVEMENT/FAIL), fidelity score, remaining issues
- **Agent performance**: iterations per agent, switches
- **Quality**: design fidelity, accessibility, responsive, code quality (typecheck/lint/build)
- **Usage**: how to preview, component location, example usage
- **Outstanding items**: remaining issues or recommendations

Present results to user and offer next actions.

## Quality Gates

- Design fidelity score >= 54/60 for PASS
- All CRITICAL issues must be resolved
- Accessibility compliance required (WCAG 2.1 AA)
- Maximum 10 iterations; if no net progress across 4, escalate to user
- If a fix introduces new CRITICAL issues, immediately escalate to user
