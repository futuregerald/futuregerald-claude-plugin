# UI Developer Sub-Agent Prompt Template

Use this as the prompt when launching the UI Developer sub-agent (Task with `subagent_type: frontend:ui-developer`).

**You MUST inline `shared_conventions` and `framework_conventions`** (loaded in Phase 1 Step 7) into the marked sections below. Do not tell the sub-agent to read files -- paste the actual file contents.

---

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
