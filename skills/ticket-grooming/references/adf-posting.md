# ADF Posting Reference

Details for posting triaging notes to Jira, including ADF expand node usage for short-mode comments.

## Short Mode (with expand section)

The short format includes a collapsible "Full Investigation Details" section. Jira does NOT support HTML `<details>` tags — you MUST use an ADF `expand` node. This requires posting the comment as raw ADF JSON, not markdown.

### How to post short-mode comments with expand

1. Convert the **short sections** (Block 1 from the sub-agent) from markdown to ADF nodes. Use the MCP `addCommentToJiraIssue` with `contentFormat: "markdown"` to test the short sections render correctly first, OR construct the ADF manually.

2. Construct the full ADF document that combines visible short sections + expand node wrapping the full investigation. Write this ADF JSON to a temp file and post via acli:

```json
{
  "version": 1,
  "type": "doc",
  "content": [
    // ... ADF nodes for the short sections (TLDR, Key Findings, Risks, etc.) ...
    {
      "type": "expand",
      "attrs": {
        "title": "Full Investigation Details (click to expand)"
      },
      "content": [
        // ... ADF nodes for the full investigation sections ...
      ]
    }
  ]
}
```

3. Post using acli with the ADF JSON file:
```bash
acli jira workitem comment create --key {TICKET_KEY} --body-file /tmp/triaging-notes-adf.json
```

### Practical shortcut — Two-part posting

If constructing full ADF is too complex, post in two steps:
1. Post the short sections via MCP with `contentFormat: "markdown"` (renders correctly).
2. Then edit the comment to add the expand section via ADF, OR accept that the full investigation is available in the conversation but not on the ticket.

## Full Mode (no expand)

Use MCP with `contentFormat: "markdown"` — simplest path, handles conversion automatically.

## General Posting Rules

- **MCP** (`addCommentToJiraIssue`): Pass `contentFormat: "markdown"` for plain markdown, or `contentFormat: "adf"` with raw ADF JSON for expand sections. Omitting `contentFormat` defaults to ADF and renders markdown as broken plain text.
- **acli** (`workitem comment create/update`): `--body-file` accepts ADF JSON. Do NOT pass markdown text via `--body` or `--body-file` — it renders as unformatted plain text (acli does not convert markdown to ADF).
- **Preferred for updating comments:** Delete via acli + re-post, or write ADF JSON and use `acli --body-file`.

## ADF `expand` Node Reference

```json
{
  "type": "expand",
  "attrs": {
    "title": "Section title shown when collapsed"
  },
  "content": [
    {
      "type": "paragraph",
      "content": [
        { "type": "text", "text": "Content inside the expand..." }
      ]
    }
  ]
}
```

ADF heading, paragraph, bulletList, listItem, codeBlock, and other standard nodes can be nested inside the expand's `content` array. See [Atlassian ADF documentation](https://developer.atlassian.com/cloud/jira/platform/apis/document/nodes/expand/) for the full spec.
