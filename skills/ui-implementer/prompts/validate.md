# Designer Validation Sub-Agent Prompt Template

Use this as the prompt when launching the Designer sub-agent (Task with `subagent_type: frontend:designer`).

---

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
