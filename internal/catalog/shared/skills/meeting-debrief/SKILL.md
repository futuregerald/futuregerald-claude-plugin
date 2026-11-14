---
name: meeting-debrief
description: "Deep strategic analysis of meetings from Krisp transcripts. Acts as a second brain — surfaces insights, subtext, commitments, risks, and recommendations the user might miss. Use when: user asks to debrief a meeting, analyze a call, review a conversation, pull insights from a meeting, wants meeting notes or analysis, says 'debrief', 'second brain', 'what did I miss', 'meeting insights', or references a recent call they want analyzed. Works with any meeting type: 1:1s, group syncs, strategy sessions, standups, skip-levels."
---

# Meeting Debrief

Turn any Krisp meeting transcript into strategic analysis — not a summary, but a second brain that catches what you missed, reads between the lines, and tells you what to do next.

## Workflow

1. **Retrieve the meeting** from Krisp
2. **Pull meeting history** with the same person(s) for context
3. **Classify the meeting type** and adapt the analysis
4. **Analyze** using the full template
5. **Persist** key insights to session memory

## Step 1: Retrieve the Meeting

Use Krisp MCP tools to find and load the meeting:

```
# If user names a person or topic — search for it
search_meetings(search="<person or topic>", limit=5, isOwner=true,
  fields=["name","date","speakers","action_items","key_points"])

# Once identified — get the full document with transcript
get_multiple_documents(ids=["<meeting_id>"])
```

If multiple matches, show the user a numbered list with dates and let them pick. Default to the most recent.

## Step 2: Pull Meeting History

Search for past meetings with the same primary speaker(s) to enable commitment tracking and pattern detection:

```
search_meetings(search="<person name>", limit=10, isOwner=true,
  fields=["name","date","action_items","key_points"])
```

Extract prior action items and commitments. These feed into the Commitment Tracker and Unsaid Topics sections of the analysis.

## Step 3: Classify the Meeting Type

Detect from the transcript and adapt analysis depth:

| Type | Signals | Analysis Emphasis |
|------|---------|-------------------|
| **1:1 (skip-level or peer)** | 2 speakers, relationship-oriented | Dynamics, subtext, influence, career signals |
| **1:1 (direct report)** | 2 speakers, status/coaching | Blockers, growth signals, morale read |
| **Strategy / Planning** | Vision, roadmap, trade-offs discussed | Decisions, open questions, alignment gaps |
| **Group sync** | 3+ speakers, status updates | Who spoke vs. who didn't, action ownership gaps |
| **Standup / Scrum** | Short, structured, blockers | Blockers only, skip deep analysis |
| **External / Customer** | Non-org participants | Commitments made, expectations set, risks |

The template adapts — standups get a compressed format, 1:1s and strategy sessions get the full treatment.

## Step 4: Analyze

**Two-phase analysis process:**

### Phase A: Work Through the Analysis Questions (mandatory)

Read [references/analysis-questions.md](references/analysis-questions.md) and work through all 37 questions internally before writing any output. This is a thinking scaffold — the questions are not shown to the user, but the answers inform every section of the debrief.

- **For strong models (Opus, Sonnet):** Treat as a mental checklist. Skim each question, note the answer, move on.
- **For weaker models (Haiku, smaller):** Answer each question explicitly in your internal reasoning, step by step. Write out your answer before moving to the next question. This is the critical path to matching the quality of a stronger model — do not skip questions.

The questions are organized in 6 passes: Comprehension, Relationships, Commitments, Subtext, Risks, and Patterns. Complete all passes before writing the output.

### Phase B: Produce the Debrief

Read [references/analysis-template.md](references/analysis-template.md) and produce the full debrief using your analysis. Every section is required for 1:1s and strategy sessions. For standups and quick syncs, use only: TL;DR, Topics, Action Items, and one-line Recommendations.

### Analysis Principles

- **Be the advisor, not the stenographer.** Never restate what was said without adding interpretation. Every topic section must include a "My read" analysis paragraph.
- **Read the subtext.** What someone doesn't say matters as much as what they do. Note deflections, hedges, and non-answers.
- **Track power dynamics.** Who drove the agenda? Who conceded? Who deferred decisions? This reveals organizational reality.
- **Be concrete in recommendations.** "Follow up" is not a recommendation. "Send Deepak a Slack message Wednesday asking for the Volche access Slack channel name" is.
- **Flag risks honestly.** If a commitment sounds like it won't happen, say so and explain why.
- **Cross-reference history.** If a topic was discussed 3 meetings ago and nothing changed, that's a pattern worth naming.

## Step 5: Persist to Memory

After delivering the debrief, save key insights:

- **Prism `session_save_experience`** — save relationship dynamics, important decisions, or patterns worth remembering across sessions
- **Prism `session_save_ledger`** — log the debrief was done, which meeting, key outcomes

Only persist what's non-obvious and durable. Don't save meeting summaries — the transcript is in Krisp. Save the *interpretation* and *strategic takeaways* that won't be obvious from the raw notes.
