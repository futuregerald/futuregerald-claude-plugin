# ADF Posting Reference

How to post triaging notes to Jira. Two modes, two procedures. No alternatives, no shortcuts.

## Hard Rules

- **NEVER use HTML `<details>` or `<summary>` tags.** Jira does not render them. They show up as raw text.
- **NEVER post short-mode comments with `contentFormat: "markdown"`.** Markdown cannot produce an expand node. The investigation section will render flat — not collapsed.
- **NEVER omit `contentFormat` on MCP calls.** Omitting it defaults to ADF and renders markdown as broken plain text.

## Full Mode (no expand)

One step. Post via MCP with markdown:

```
addCommentToJiraIssue(
  cloudId: "{CLOUD_ID}",
  issueIdOrKey: "{TICKET_KEY}",
  commentBody: "{FULL_MARKDOWN_CONTENT}",
  contentFormat: "markdown"
)
```

Done.

## Short Mode (with expand)

Short mode requires an ADF `expand` node. ADF is JSON — there is no markdown equivalent.

### Procedure

**Step 1:** Build the ADF JSON document following the skeleton below. Replace placeholders with actual content.

**Step 2:** Write the ADF JSON to `/tmp/triaging-notes-{TICKET_KEY}.json`.

**Step 3:** Post via acli:
```bash
acli jira workitem comment create --key {TICKET_KEY} --body-file /tmp/triaging-notes-{TICKET_KEY}.json
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
        { "type": "text", "text": "Groomed: {ISO_DATE} (iteration {N})", "marks": [{ "type": "em" }] }
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

    // -- @mention / open questions (omit if no questions) --
    {
      "type": "paragraph",
      "content": [{ "type": "text", "text": "@{PM} — {open questions}" }]
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

- **Full mode:** Re-post with MCP `contentFormat: "markdown"`.
- **Short mode:** Re-post with acli `--body-file` using corrected ADF JSON. Do NOT fall back to markdown — it will not have the expand node.
