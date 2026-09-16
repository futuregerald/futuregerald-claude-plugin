# ADF Posting Reference

How to post triaging notes to Jira. One procedure, two modes, no shortcuts.

## Hard Rules

- **NEVER post triaging notes to Jira as Markdown, in either mode.** Jira's Markdown converter
  silently strips the ``[`code`](url)`` link form — which is what nearly every permalink in the
  details block is — and it cannot express an `expand` node at all. Both modes go through
  `scripts/md2adf.py` and post ADF via `acli --body-file`.
- **NEVER use HTML `<details>` or `<summary>` tags.** Jira renders them as raw text.
- **NEVER omit `contentFormat` on an MCP call**, if you are posting something other than these
  notes. Omitting it defaults to ADF and renders markdown as broken plain text.

## The two modes

Both take the same two-block `notes.md`. They differ by one flag:

| Mode | Command | Result |
|---|---|---|
| Short (default) | `md2adf.py notes.md --title "Full Investigation Details"` | Details collapsed behind an expand node |
| Full (`--full`) | `md2adf.py notes.md --no-expand` | Details inline after a rule; no expand node |

GitHub is the exception: it renders Markdown correctly, including the link form Jira strips, so
`gh issue comment` posts `notes.md` as-is.

## The expand node

Short mode requires an ADF `expand` node. ADF is JSON — there is no markdown equivalent.

### Procedure

**Prefer the generator.** `scripts/md2adf.py` converts the sub-agent's `notes.md` into a valid
document, including the `expand` node and the `code`+`link` mark combination below. Hand-building
ADF is for the cases the converter does not cover — reach for the skeleton then, not by default.

```
python3 {SKILL_DIR}/scripts/md2adf.py {SCRATCHPAD}/{TICKET_KEY}/notes.md --title "Full Investigation Details"
```

Validate the result before posting: assert it is a doc at version 1 containing exactly one `expand`
node. `jq -e .` alone only proves the file is JSON, not that it is ADF.

---

The manual route, when you need it:

**Step 1:** Build the ADF JSON document following the skeleton below. Replace placeholders with actual content.

**Step 2:** Write the ADF JSON to `{SCRATCHPAD}/{TICKET_KEY}/notes.adf.json`, where `{SCRATCHPAD}`
is the session scratchpad directory. Never `/tmp` — it is shared, unscoped, and not cleaned up
with the session.

**Step 3:** Post via acli:
```bash
acli jira workitem comment create --key {TICKET_KEY} --body-file {SCRATCHPAD}/{TICKET_KEY}/notes.adf.json
```

**If acli is unavailable**, post via MCP with `contentFormat: "adf"` and pass the ADF JSON string as `commentBody`.

### ADF Skeleton

This is the exact structure to follow. The visible summary uses bold labels (no headers) to stay compact. The investigation goes inside a single `expand` node at the end.

**IMPORTANT:** JSON does not support comments. The `// --` lines below are for documentation only — remove them when constructing the actual JSON.

```json
{
  "version": 1,
  "type": "doc",
  "content": [
    {
      "type": "heading",
      "attrs": { "level": 1 },
      "content": [{ "type": "text", "text": "Triaging Notes" }]
    },
    {
      "type": "paragraph",
      "content": [
        { "type": "text", "text": "Groomed: {ISO_TIMESTAMP} (iteration {N})", "marks": [{ "type": "em" }] }
      ]
    },

    // -- What's happening --
    {
      "type": "paragraph",
      "content": [
        { "type": "text", "text": "What's happening: ", "marks": [{ "type": "strong" }] },
        { "type": "text", "text": "{1-2 sentences. Plain English. What's broken and who it affects.}" }
      ]
    },

    // -- Root cause --
    {
      "type": "paragraph",
      "content": [
        { "type": "text", "text": "Root cause: ", "marks": [{ "type": "strong" }] },
        { "type": "text", "text": "{1-2 sentences. WHY it happens. Confidence: high/medium/low.}" }
      ]
    },

    // -- Fix --
    {
      "type": "paragraph",
      "content": [
        { "type": "text", "text": "Fix: ", "marks": [{ "type": "strong" }] },
        { "type": "text", "text": "{What to do, which repo, which area. 1-3 short sentences.}" }
      ]
    },

    // -- Estimate --
    {
      "type": "paragraph",
      "content": [
        { "type": "text", "text": "Estimate: ", "marks": [{ "type": "strong" }] },
        { "type": "text", "text": "{S/M/L/XL} · {days} · {N} SP · Confidence: {level}" }
      ]
    },

    // -- Risks (omit entire paragraph if no high/critical risks) --
    {
      "type": "paragraph",
      "content": [
        { "type": "text", "text": "Risks: ", "marks": [{ "type": "strong" }] },
        { "type": "text", "text": "{High/critical risks only. One line each.}" }
      ]
    },

    // -- Priority --
    {
      "type": "paragraph",
      "content": [
        { "type": "text", "text": "Priority: ", "marks": [{ "type": "strong" }] },
        { "type": "text", "text": "P{N} — {one sentence justification}" }
      ]
    },

    // -- @mention / open questions (omit entire paragraph if no questions) --
    {
      "type": "paragraph",
      "content": [{ "type": "text", "text": "@{PM or reporter} — {open questions}" }]
    },

    // -- Divider before expand --
    { "type": "rule" },

    // -- Collapsed investigation (THE EXPAND NODE) --
    {
      "type": "expand",
      "attrs": { "title": "Full Investigation Details" },
      "content": [
        {
          "type": "heading",
          "attrs": { "level": 3 },
          "content": [{ "type": "text", "text": "Codebase findings" }]
        },
        {
          "type": "paragraph",
          "content": [{ "type": "text", "text": "{files, models, call paths with GitHub permalinks...}" }]
        },
        {
          "type": "heading",
          "attrs": { "level": 3 },
          "content": [{ "type": "text", "text": "History" }]
        },
        {
          "type": "paragraph",
          "content": [{ "type": "text", "text": "{related tickets and PRs with links...}" }]
        },
        {
          "type": "heading",
          "attrs": { "level": 3 },
          "content": [{ "type": "text", "text": "Root cause analysis (full)" }]
        },
        {
          "type": "paragraph",
          "content": [{ "type": "text", "text": "{hypotheses with evidence, counterarguments...}" }]
        },
        {
          "type": "heading",
          "attrs": { "level": 3 },
          "content": [{ "type": "text", "text": "Risk details" }]
        },
        {
          "type": "paragraph",
          "content": [{ "type": "text", "text": "{full risk analysis, blast radius, security/performance...}" }]
        },
        {
          "type": "heading",
          "attrs": { "level": 3 },
          "content": [{ "type": "text", "text": "Priority" }]
        },
        {
          "type": "paragraph",
          "content": [{ "type": "text", "text": "{Severity · Urgency · P{N} with justification}" }]
        },
        {
          "type": "heading",
          "attrs": { "level": 3 },
          "content": [{ "type": "text", "text": "Breadcrumbs" }]
        },
        {
          "type": "paragraph",
          "content": [{ "type": "text", "text": "{key files and call paths to start from...}" }]
        }
      ]
    }
  ]
}
```

### ADF Node Quick Reference

Use these nodes inside paragraphs and lists:

**Plain text:**
```json
{ "type": "text", "text": "plain text" }
```

**Bold:**
```json
{ "type": "text", "text": "bold text", "marks": [{ "type": "strong" }] }
```

**Italic:**
```json
{ "type": "text", "text": "italic text", "marks": [{ "type": "em" }] }
```

**Inline code:**
```json
{ "type": "text", "text": "code_text", "marks": [{ "type": "code" }] }
```

**Link:**
```json
{ "type": "text", "text": "link text", "marks": [{ "type": "link", "attrs": { "href": "https://..." } }] }
```

**Code + link (for GitHub permalinks):**
```json
{ "type": "text", "text": "ClassName#method", "marks": [{ "type": "code" }, { "type": "link", "attrs": { "href": "https://github.com/..." } }] }
```

**Bullet list:**
```json
{
  "type": "bulletList",
  "content": [
    {
      "type": "listItem",
      "content": [{
        "type": "paragraph",
        "content": [{ "type": "text", "text": "item" }]
      }]
    }
  ]
}
```

**Ordered list:**
```json
{
  "type": "orderedList",
  "content": [
    {
      "type": "listItem",
      "content": [{
        "type": "paragraph",
        "content": [{ "type": "text", "text": "step" }]
      }]
    }
  ]
}
```

**Horizontal rule:**
```json
{ "type": "rule" }
```

**Multiple marks** can be combined in a single array:
```json
{ "type": "text", "text": "bold code", "marks": [{ "type": "strong" }, { "type": "code" }] }
```

## Deleting a Comment

When a comment renders incorrectly and needs to be reposted:

```bash
acli jira workitem comment delete --key {TICKET_KEY} --id {COMMENT_ID}
```

Get the comment ID from the MCP post response (`id` field) or from `acli jira workitem comment list --key {TICKET_KEY}`.

## Reposting After Deletion

Correct `notes.md`, regenerate, and re-post with `acli --body-file` — the same path as the
original post, in both modes. Do NOT fall back to Markdown: short mode loses the expand node, and
both modes lose every ``[`code`](url)`` permalink.
