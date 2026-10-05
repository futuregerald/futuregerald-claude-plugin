# Dispatch Prompt

Read this when dispatching the editor. Use a read-only agent, such as `reviewer` (Read/Grep/Glob, no shell), so it can't change the work under review.

- **Pass paths and identifiers, not pasted content.** The sub-agent reads the files itself.
- **For UI,** capture phone-width and desktop-width screenshots before dispatching and pass their paths. The sub-agent can't open a browser.
- **Fill every placeholder,** and leave the rest of the prompt as written. Don't add your own suspicions about the work. An editor pointed at your worries inherits your blind spots.

```
Agent tool:
  subagent_type: reviewer
  description: "Editor pass"
  prompt: |
    You are the editor for a finished piece of work. Decide whether it is good,
    not whether it is done. Do not edit anything; return a crit sheet.

    Method: read <ABSOLUTE PATH TO editor-pass/SKILL.md>, sections "The pass"
    and "The crit sheet", and <ABSOLUTE PATH TO editor-pass/references/crit-sheet.md>.

    The work (<OUTPUT TYPE: UI / report / PRD / tickets / skill / code>):
    <paths, URL, screenshot paths>

    Point of view: <path to the brief, PRODUCT.md, ticket or PR description>
    — or "none: write the three-line brief above the Overall note".

    Return only the crit sheet, in the exact format from crit-sheet.md.
```

After it returns, validate the sheet as described in SKILL.md ("Validate, then present").
