---
name: write-a-prd
description: Create a PRD through user interview, codebase exploration, and module design, then submit as an issue. Carries the product experience for user-facing work and is coupled to the delivery roadmap. Use when user wants to write a PRD, create a product requirements document, or plan a new feature. Supports both GitHub Issues and Jira.
tags: [workflow, planning, project-management]
---

This skill will be invoked when the user wants to create a PRD. You may skip steps if you don't consider them necessary.

1. Ask the user for a long, detailed description of the problem they want to solve and any potential ideas for solutions.

2. Explore the repo to verify their assertions and understand the current state of the codebase.

3. Interview the user relentlessly about every aspect of this plan until you reach a shared understanding. Walk down each branch of the design tree, resolving dependencies between decisions one-by-one. For user-facing work, cover the product experience too — feel, look, screens, copy tone — and collect links to any designs.

4. Sketch out the major modules you will need to build or modify to complete the implementation. Actively look for opportunities to extract deep modules that can be tested in isolation.

A deep module (as opposed to a shallow module) is one which encapsulates a lot of functionality in a simple, testable interface which rarely changes.

Check with the user that these modules match their expectations. Check with the user which modules they want tests written for.

5. Once you have a complete understanding of the problem and solution, use the template below to write the PRD. The PRD should be submitted as an issue in the project's configured issue tracker.

**Issue tracker detection:**
- Check CLAUDE.md for ticket system configuration (e.g., `### Ticket Grooming` section or Jira project key)
- If Jira is configured: create via `createJiraIssue` (Atlassian MCP) with issue type "Story" and the `claude-code` label
- If GitHub Issues is used: create via `gh issue create`
- If neither is configured: ask the user where to submit
- When posting to Jira, always set `contentFormat: "markdown"` for the description

6. **The PRD and the roadmap change together.** When the stories ship over more than one plan, `writing-plans` keeps the phases in `docs/roadmap.md`, and the PRD's Delivery section links it. A new or changed story gets a phase in the roadmap; a change of strategy in the roadmap is reflected in the PRD. When either changes, update the other in the same pass.

<prd-template>

## Problem Statement

The problem that the user is facing, from the user's perspective.

## Solution

The solution to the problem, from the user's perspective.

## User Stories

A LONG, numbered list of user stories. Each user story should be in the format of:

1. As an <actor>, I want a <feature>, so that <benefit>

<user-story-example>
1. As a mobile bank customer, I want to see balance on my accounts, so that I can make better informed decisions about my spending
</user-story-example>

This list of user stories should be extremely extensive and cover all aspects of the feature.

## Product Experience

For user-facing work. For work no user sees, replace this section with one line saying so.

- **Feel** — the product's personality in a few words, and what it must never feel like
- **Look** — visual direction: palette, type, density, reference products
- **Screens** — each screen, what it is for, and what the user does there
- **Copy tone** — how the product talks, with one example line
- **Designs** — links to mockups and prototypes, and to the design history: the directions tried and why each was dropped

<feel-example>
Not "clean and modern", which fits every product. A recipe app: "a well-thumbed cookbook — warm, unhurried, type large enough to read from the stove; never gamified, no streaks."
</feel-example>

## Implementation Decisions

A list of implementation decisions that were made. This can include:

- The modules that will be built/modified
- The interfaces of those modules that will be modified
- Technical clarifications from the developer
- Architectural decisions
- Schema changes
- API contracts
- Specific interactions

Do NOT include specific file paths or code snippets. They may end up being outdated very quickly.

## Testing Decisions

A list of testing decisions that were made. Include:

- A description of what makes a good test (only test external behavior, not implementation details)
- Which modules will be tested
- Prior art for the tests (i.e. similar types of tests in the codebase)

## Delivery

A link to the roadmap by its full repository URL (a relative `docs/roadmap.md` does not resolve inside an issue). Which phase delivers which story lives only in the roadmap, so it cannot drift here. If the roadmap does not exist yet, say so; `writing-plans` writes it before the first phase plan. For work that fits in one plan, say so in one line.

## Out of Scope

A description of the things that are out of scope for this PRD.

## Further Notes

Any further notes about the feature.

</prd-template>
