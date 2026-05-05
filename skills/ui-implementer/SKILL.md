---
name: ui-implementer
description: Implements UI components from scratch based on design references (Figma, screenshots, mockups) with intelligent validation and adaptive agent switching. Supports React, Vue, and Svelte 5 with auto-detection. Use when user provides a design and wants pixel-perfect UI implementation with design fidelity validation. Triggers automatically when user mentions Figma links, design screenshots, or wants to implement UI from designs.
allowed-tools: Task, AskUserQuestion, Bash, Read, Write, TodoWrite, Glob, Grep
author: tianzecn (github.com/tianzecn/myclaudecode)
contributors: Gerald Onyango (github.com/futuregerald) -- multi-framework support (React/Vue/Svelte 5), Puppeteer-first screenshot capture, validation loop hardening, framework-specific conventions
---

# UI Implementer

This Skill implements UI components from scratch based on design references using specialized UI development agents with intelligent validation and adaptive agent switching for optimal results.

## When to use this Skill

Claude should invoke this Skill when:

**Design References Provided:**
- User shares a Figma URL (e.g., "Here's the Figma design: https://figma.com/...")
- User provides a screenshot/mockup path (e.g., "I have a design at /path/to/design.png")
- User mentions a design URL they want to implement

**Intent to Implement UI:**
- "Implement this UI design"
- "Create components from this Figma file"
- "Build this interface from the mockup"
- "Make this screen match the design"

**Pixel-Perfect Requirements:**
- "Make it look exactly like the design"
- "Implement pixel-perfect from Figma"
- "Match the design specifications exactly"

**Examples of User Messages:**
- "Here's a Figma link, can you implement the UserProfile component?"
- "I have a design screenshot, please create the dashboard layout"
- "Implement this navbar from the mockup at designs/navbar.png"
- "Build the product card to match this Figma: https://figma.com/..."

## DO NOT use this Skill when:

- User just wants to validate existing UI (use browser-debugger or /validate-ui instead)
- User wants to fix existing components (use regular developer agent)
- User wants to implement features without design reference (use regular implementation flow)

## Instructions

**ALL PHASES (0-4) ARE MANDATORY. NEVER SKIP A PHASE OR STEP. NO EXCEPTIONS.**

A "simple" or "small" component follows the exact same lifecycle as a complex one. Do not shortcut Phase 3 validation because the implementation "looks right." Do not skip Phase 0 because the task "seems straightforward." Every phase exists because skipping it has caused failures.

**Phase gates:** Do NOT proceed to the next phase until the current phase is fully complete. If a step requires user input, wait for it. If a step requires reading a file, read it. If a step requires capturing screenshots, capture them.

This Skill implements the same workflow as the `/implement-ui` command. Follow these phases:

### PHASE 0: Initialize Workflow (REQUIRED FIRST)

**You MUST create this todo list before doing anything else.** This is not optional. Do it now.

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

**Complete every step in order. Do not skip steps even if you think you can infer the answer.** Steps that require user input MUST ask the user -- do not guess.

**Step 1: Extract Design Reference**

Check if user already provided design reference in their message:
- Scan for Figma URLs: `https://figma.com/design/...` or `https://figma.com/file/...`
- Scan for file paths: `/path/to/design.png`, `~/designs/mockup.jpg`
- Scan for remote URLs: `http://example.com/design.png`

If design reference found in user's message:
- Extract and store as `design_reference`
- Log: "Design reference detected: [design_reference]"

If NOT found, ask:
```
I'd like to implement UI from your design reference.

Please provide the design reference:
1. Figma URL (e.g., https://figma.com/design/abc123.../node-id=136-5051)
2. Screenshot file path (local file on your machine)
3. Remote URL (live design reference)

What is your design reference?
```

**Step 2: Detect Framework**

Auto-detect the project's UI framework, then confirm with the user.

1. Read `package.json` in the project root (use Read tool, not `cat`)
2. If not found at the project root, also check `package.json` at `target_location` and intermediate directories (for monorepos/workspaces)
3. Check `dependencies` and `devDependencies` for:
   - `react` or `react-dom` -> **React**
   - `vue` -> **Vue**
   - `svelte` -> **Svelte**
4. Handle edge cases:
   - **No `package.json` found at any level:** Skip detection, ask user directly
   - **Multiple frameworks found:** List all detected frameworks, ask user to choose
   - **None found** (monorepo, workspace root): Ask user directly
4. If exactly one framework detected, confirm:
   ```
   Detected **[Framework]** in package.json. Is that correct?
   ```
5. If user corrects or no detection, ask:
   ```
   Which UI framework is this project using?
   1. React
   2. Vue
   3. Svelte

   Framework?
   ```

Store as `framework` (one of: `react`, `vue`, `svelte`).

**Step 3: Extract Component Description**

Check if user mentioned what to implement:
- Look for component names: "UserProfile", "navbar", "dashboard", "ProductCard"
- Look for descriptions: "implement the header", "create the sidebar", "build the form"

If found:
- Extract and store as `component_description`

If NOT found, ask:
```
What UI component(s) should I implement from this design?

Examples:
- "User profile card component"
- "Navigation header with mobile menu"
- "Product listing grid with filters"
- "Dashboard layout with widgets"

What component(s) should I implement?
```

**Step 4: Ask for Target Location**

Ask:
```
Where should I create this component?

Options:
1. Provide a specific directory path (e.g., "src/components/profile/")
2. Let me suggest based on component type
3. I'll tell you after seeing the component structure

Where should I create the component files?
```

Store as `target_location`.

**Step 5: Ask for Application URL**

Ask:
```
What is the URL where I can preview the implementation?

Examples:
- http://localhost:5173 (Vite default)
- http://localhost:3000 (Next.js/CRA default)
- https://staging.yourapp.com

Preview URL?
```

Store as `app_url`.

**Step 6: Ask for UI Developer Codex Preference**

Use AskUserQuestion:
```
Enable intelligent agent switching with UI Developer Codex?

When enabled:
- If UI Developer struggles (2 consecutive failures), switches to UI Developer Codex
- If UI Developer Codex struggles (2 consecutive failures), switches back
- Provides adaptive fixing with both agents for best results

Enable intelligent agent switching?
```

Options:
- "Yes - Enable intelligent agent switching"
- "No - Use only UI Developer"

Store as `codex_enabled` (boolean).

**Step 7: Load Framework Instructions (CRITICAL -- do not skip)**

You MUST read the framework-specific instruction files using the Read tool. These MUST be inlined into sub-agent prompts -- sub-agents cannot read skill files from the filesystem. If you skip this step, the sub-agent will have no framework guidance and will produce incorrect code.

1. Read `frameworks/shared.md` from this skill's directory
2. Read `frameworks/[framework].md` (e.g., `frameworks/react.md`) from this skill's directory
3. Store the file contents as `shared_conventions` and `framework_conventions`

The skill's base directory is provided at the top of this file. Use it to construct absolute paths:
- `[skill_base_dir]/frameworks/shared.md`
- `[skill_base_dir]/frameworks/[framework].md`

**Step 8: Validate Inputs**

Validate all inputs using the same logic as /implement-ui command:
- Design reference format (Figma/Remote/Local)
- Component description not empty
- Target location valid
- Application URL valid
- Framework is one of: `react`, `vue`, `svelte`

### PHASE 2: Initial Implementation from Scratch

**STOP.** Before proceeding, verify you have completed ALL of Phase 1:
- [ ] `design_reference` is set
- [ ] `framework` is set and confirmed by user
- [ ] `component_description` is set
- [ ] `target_location` is set
- [ ] `app_url` is set
- [ ] `codex_enabled` is set
- [ ] `shared_conventions` and `framework_conventions` are loaded (file contents read into memory)

If any variable is missing, go back to the relevant Phase 1 step. Do NOT proceed with empty variables.

Launch UI Developer agent using Task tool with `subagent_type: frontend:ui-developer`.

**You MUST inline `shared_conventions` and `framework_conventions`** (loaded in Step 7) directly into the agent prompt. Do not tell the sub-agent to read files -- paste the actual file contents:

```
Implement the following UI component(s) from scratch based on the design reference.

**Design Reference**: [design_reference]
**Component Description**: [component_description]
**Target Location**: [target_location]
**Application URL**: [app_url]
**Framework**: [framework]

**Your Task:**

1. **Analyze the design reference:**
   - If Figma: Use Figma MCP to fetch design screenshot and specs. If Figma MCP is unavailable, ask the user to export the design as PNG and provide the file path.
   - If Remote URL: Use Puppeteer/Playwright to navigate and capture a screenshot. If neither is available, try Chrome DevTools MCP. As a last resort, ask the user for a screenshot.
   - If Local file: Read the file to view design

2. **Plan component structure:**
   - Determine component hierarchy
   - Identify reusable sub-components
   - Plan file structure following the framework conventions below

3. **Implement UI components from scratch.** Follow these conventions exactly:

--- SHARED CONVENTIONS ---
[inline shared_conventions here]
--- END SHARED CONVENTIONS ---

--- FRAMEWORK-SPECIFIC INSTRUCTIONS ---
[inline framework_conventions here]
--- END FRAMEWORK-SPECIFIC INSTRUCTIONS ---

4. **Create component files in target location:**
   - Use Write tool to create files
   - Follow project conventions and the framework instructions above
   - Include TypeScript types per the framework instructions

5. **Ensure code quality:**
   - Run the typecheck command specified in the framework instructions
   - Run linter: `npm run lint`
   - Run build: `npm run build`
   - Fix any errors

6. **Provide implementation summary:**
   - Files created
   - Components implemented
   - Key decisions
   - Any assumptions

Return detailed implementation summary when complete.
```

Wait for UI Developer to complete.

### PHASE 3: Validation and Adaptive Fixing Loop (MANDATORY -- do not skip)

**Phase 3 is NOT optional.** You MUST validate every implementation against the design reference, even if you believe the implementation is correct. "It looks right to me" is not validation -- only a designer agent comparing screenshots counts. Do NOT skip to Phase 4 without at least one full validation cycle (Steps 3.1 through 3.3).

Initialize loop variables:
```pseudocode
iteration_count = 0
max_iterations = 10
previous_issues_count = Infinity   // Treat first iteration as progress baseline
previous_critical_count = 0
current_issues_count = null
current_critical_count = 0
last_agent_used = "ui-developer"   // Phase 2 used UI Developer
ui_developer_consecutive_failures = 0
codex_consecutive_failures = 0
design_fidelity_achieved = false
issue_count_history = []           // Track last N counts for stagnation detection
```

**Loop: While iteration_count < max_iterations AND NOT design_fidelity_achieved**

**Step 3.1: Capture Screenshots**

Read `screenshot-capture.md` from this skill's directory (`[skill_base_dir]/screenshot-capture.md`) and follow the instructions to capture:
- Implementation screenshots (desktop + mobile) from `[app_url]`
- Design reference screenshot from `[design_reference]`

Priority: Puppeteer first, then Playwright, then Chrome DevTools MCP, then ask user.

Store paths as `impl_screenshot`, `impl_screenshot_mobile`, and `design_screenshot`.

**Step 3.2: Launch Designer for Validation**

Use Task tool with `subagent_type: frontend:designer`:

```
Review the implemented UI component against the design reference.

**Iteration**: [iteration_count + 1] / 10
**Design Reference**: [design_reference]
**Component Description**: [component_description]
**Implementation Files**: [List of files]
**Application URL**: [app_url]

**Screenshots captured for you:**
- Design reference: [design_screenshot path]
- Implementation (desktop 1280px): [impl_screenshot path]
- Implementation (mobile 375px): [impl_screenshot_mobile path]

Read these image files to perform your visual comparison.

**Your Task:**
1. Compare the design reference screenshot against the implementation screenshots
2. Perform comprehensive design review:
   - Colors & theming
   - Typography
   - Spacing & layout
   - Visual elements
   - Responsive design (compare desktop vs mobile screenshots)
   - Accessibility (WCAG 2.1 AA)
   - Interactive states

3. Document ALL discrepancies between design and implementation
4. Categorize by severity (CRITICAL/MEDIUM/LOW)
5. Provide actionable fixes with code snippets
6. Calculate design fidelity score (X/60)

7. **Overall assessment:**
   - PASS (score >= 54/60)
   - NEEDS IMPROVEMENT (score 40-53/60)
   - FAIL (score < 40/60)

Return detailed design review report.
```

**Step 3.3: Check if Design Fidelity Achieved**

Extract from designer report:
- Overall assessment
- Total issue count (store as `current_issues_count`)
- Count of CRITICAL-severity issues (store as `current_critical_count`)
- Design fidelity score

**CRITICAL regression check:** If `current_critical_count > previous_critical_count`, a fixing agent introduced NEW critical issues. Immediately escalate to the user: "The last fix introduced new CRITICAL issues. Please review before continuing." Wait for user guidance before proceeding.

If assessment is "PASS":
- Set `design_fidelity_achieved = true`
- Exit loop (success)

**Step 3.4: Determine Fixing Agent (Smart Switching Logic)**

```pseudocode
function determineFixingAgent() {
  // If Codex not enabled, always use UI Developer
  if (!codex_enabled) return "ui-developer"

  // Smart switching based on consecutive failures
  if (ui_developer_consecutive_failures >= 2) {
    // UI Developer struggling - switch to Codex
    return "ui-developer-codex"
  }

  if (codex_consecutive_failures >= 2) {
    // Codex struggling - switch to UI Developer
    return "ui-developer"
  }

  // Default: continue with last agent used
  return last_agent_used
}
```

**Step 3.5: Launch Fixing Agent**

**Set `last_agent_used` to the chosen agent before launching.**

If `fixing_agent == "ui-developer"`:
- Set `last_agent_used = "ui-developer"`
- Use Task with `subagent_type: frontend:ui-developer`
- Provide designer feedback
- Request fixes

If `fixing_agent == "ui-developer-codex"`:
- Set `last_agent_used = "ui-developer-codex"`
- Use Task with `subagent_type: frontend:ui-developer-codex`
- Prepare complete prompt with designer feedback + current code
- Request expert fix plan

**Step 3.6: Update Metrics and Loop**

```pseudocode
// Check if progress was made
const progress_made = (current_issues_count < previous_issues_count)

if (progress_made) {
  // Only reset the counter for the agent that made progress
  if (last_agent_used === "ui-developer") {
    ui_developer_consecutive_failures = 0
  } else if (last_agent_used === "ui-developer-codex") {
    codex_consecutive_failures = 0
  }
} else {
  // No progress - increment failure counter for the agent that tried
  if (last_agent_used === "ui-developer") {
    ui_developer_consecutive_failures++
  } else if (last_agent_used === "ui-developer-codex") {
    codex_consecutive_failures++
  }
}

// Track issue count history for stagnation detection
issue_count_history.push(current_issues_count)

// Stagnation check: if issue count has not decreased across the last 4 iterations,
// the loop is stuck. Escalate to the user.
if (issue_count_history.length >= 4) {
  const last4 = issue_count_history.slice(-4)
  const minRecent = Math.min(...last4)
  const first = last4[0]
  if (minRecent >= first) {
    // No net progress across 4 iterations -- ask the user
    ASK USER: "The validation loop has not made net progress across the last 4 iterations
    (issue counts: [last4]). Options:
    A) Continue iterating (I'll try different approaches)
    B) Accept the current state and move to Phase 4
    C) Stop and let me fix issues manually
    Which do you prefer?"
    // Reset history after user decision
    issue_count_history = []
  }
}

// Update for next iteration
previous_issues_count = current_issues_count
previous_critical_count = current_critical_count
iteration_count++
```

Continue loop until design fidelity achieved or max iterations reached.

### PHASE 4: Final Report & Completion

Generate comprehensive implementation report:

```markdown
# UI Implementation Report

## Component Information
- Component: [component_description]
- Design Reference: [design_reference]
- Location: [target_location]
- Preview: [app_url]

## Implementation Summary
- Files Created: [count]
- Components: [list]

## Validation Results
- Iterations: [count] / 10
- Final Status: [PASS/NEEDS IMPROVEMENT/FAIL]
- Design Fidelity Score: [score] / 60
- Issues: [count]

## Agent Performance
- UI Developer: [iterations, successes]
- UI Developer Codex: [iterations, successes] (if enabled)
- Agent Switches: [count] times

## Quality Metrics
- Design Fidelity: [Pass/Needs Improvement]
- Accessibility: [WCAG compliance]
- Responsive: [Mobile/Tablet/Desktop]
- Code Quality: [TypeScript/Lint/Build status]

## How to Use
[Preview instructions]
[Component location]
[Example usage]

## Outstanding Items
[List any remaining issues or recommendations]
```

Present results to user and offer next actions.

## Orchestration Rules

### Smart Agent Switching:
- Track consecutive failures independently for each agent
- Switch after 2 consecutive failures (no progress)
- On progress, only reset the counter for the agent that succeeded (not both)
- Log all switches with reasons
- Balance UI Developer (speed) with UI Developer Codex (expertise)

### Loop Prevention:
- Maximum 10 iterations before asking user
- Track issue count history; if no net progress across 4 iterations, escalate to user
- If a fix introduces new CRITICAL issues, immediately escalate to user
- Ask user for guidance if max iterations reached

### Quality Gates:
- Design fidelity score >= 54/60 for PASS
- All CRITICAL issues must be resolved
- Accessibility compliance required

## Success Criteria

Complete when:
1. [x] UI component implemented from scratch
2. [x] Designer validated against design reference
3. [x] Design fidelity score >= 54/60
4. [x] All CRITICAL issues resolved
5. [x] Accessibility compliant (WCAG 2.1 AA)
6. [x] Responsive (mobile/tablet/desktop)
7. [x] Code quality passed (typecheck/lint/build)
8. [x] Comprehensive report provided
9. [x] User acknowledges completion

## Notes

- This Skill wraps the `/implement-ui` command workflow
- Use proactively when user provides design references
- Implements from scratch (not for fixing existing UI)
- Smart switching maximizes success rate
- All work on unstaged changes until user approves
- Maximum 10 iterations with user escalation
