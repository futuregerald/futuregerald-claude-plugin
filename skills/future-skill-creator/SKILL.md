---
name: future-skill-creator
description: Create or update a skill so it encodes judgment, not just knowledge — decision trees, do/don't pairs, on-demand references, a check script with tests, and a stated picture of good output. Preferred over skill-creator in this plugin. Use when the user wants to create a new skill, fork or rework an existing one, or turn a method, talk, checklist or workflow into a skill.
author: Gerald Onyango
tags: [authoring]
license: Complete terms in LICENSE.txt
---

# Future Skill Creator

Skills are modular packages that extend Claude with specialized knowledge, workflows and
tools — onboarding guides that turn a general agent into a specialist. This skill builds
them so the agent can **act on** the knowledge: a skill that only describes a domain
produces the most probable output for that domain, which is the generic one.

## Core Principles

### Concise is Key

The context window is a public good. Skills share it with the system prompt, the
conversation, other skills' metadata and the user's request.

**Default assumption: Claude is already very smart.** Only add context Claude doesn't
already have. Challenge each paragraph: "Does Claude really need this?" and "Does it
justify its token cost?" Prefer concise examples over verbose explanations.

### Set Appropriate Degrees of Freedom

Match specificity to the task's fragility and variability:

- **High freedom (text instructions)** — many approaches are valid; heuristics guide.
- **Medium freedom (pseudocode, parameterised scripts)** — a preferred pattern exists,
  some variation is fine.
- **Low freedom (specific scripts, few parameters)** — operations are fragile, consistency
  is critical, or a sequence must be followed.

A narrow bridge with cliffs needs guardrails; an open field allows many routes.

### Encode Judgment, Not Just Knowledge

Knowledge says what exists. Judgment says which one to use, when, and how to tell good
from merely finished. Agents follow judgment only when it is written in a form they can
execute. For each decision the skill covers, pick the form that fits:

| The skill needs to… | Encode it as |
|---|---|
| Choose between options | a **decision tree** ("single short action → Dialog; multi-step → wizard") |
| Show the line between right and almost-right | a **do/don't pair** with concrete examples |
| Cover many items the agent needs one at a time | **per-item reference files**, loaded on demand |
| Hold a standard that a machine can test | a **check script** with tests |
| Produce a whole artifact, not parts | a **template or full flow**, not atomic rules |

"Use dialogs appropriately" is knowledge. "Single short action → Dialog" is judgment. See
[references/judgment-patterns.md](references/judgment-patterns.md) for each pattern with
an example.

Rules set a floor, not a ceiling: a skill can satisfy every rule and still produce dead
output. That is why the last step is editing the output, not the skill.

### Say What Good Looks Like

Every skill that produces output states, near the top:

- who uses the output and what good means for them
- the generic default the skill exists to prevent
- one detail a careless version would miss

Without this, the agent fills the gap with the most probable answer.

### Anatomy of a Skill

```
skill-name/
├── SKILL.md (required)
│   ├── YAML frontmatter: name, description (required); author, tags (this plugin)
│   └── Markdown instructions (required)
└── Bundled Resources (optional)
    ├── scripts/          - Executable code, plus test_*.py
    ├── references/       - Docs loaded into context only when needed
    └── assets/           - Files used in output (templates, icons, fonts)
```

- **SKILL.md** — the frontmatter `name` and `description` are the only fields Claude reads
  when deciding to load the skill, so the description carries every "when to use" trigger.
  The body loads only after triggering.
- **scripts/** — code that would otherwise be rewritten each time, or that must be
  deterministic. Runs without being read into context.
- **references/** — material Claude reads while working: schemas, policies, detailed
  guides, example sets. Keeps SKILL.md lean. For files over 10k words, give grep patterns
  in SKILL.md. Information lives in SKILL.md or a reference, never both.
- **assets/** — files copied into the output, never read into context: templates,
  boilerplate projects, fonts, images.

Do not add README.md, CHANGELOG.md, INSTALLATION_GUIDE.md or other files about the skill
rather than for the agent. A `LICENSE` file required by the source's licence stays.

### Progressive Disclosure

Three levels load at different times: metadata (always), SKILL.md body (on trigger, keep
under 300 lines), bundled resources (as needed). When a skill supports several variants,
keep the selection logic in SKILL.md and move each variant to its own reference file, one
level deep, each linked from SKILL.md with when to read it. Patterns and examples:
[references/progressive-disclosure.md](references/progressive-disclosure.md).

## Skill Creation Process

1. Understand the skill with concrete examples and a point of view
2. Plan the reusable contents and the judgment patterns
3. Initialize the skill (`init_skill.py`)
4. Edit the skill
5. Validate (`quick_validate.py`)
6. Edit the output

Follow the steps in order, skipping one only for a clear reason.

### Step 1: Understand the Skill

Get concrete examples of use, from the user or generated and confirmed by them:

- "What should this skill support? Give me examples of how it would be used."
- "What would a user say that should trigger it?"

Then the point of view — the questions that stop the skill producing generic output:

- "Who uses what this skill produces, and what does good mean to them?"
- "What does the generic, most-probable version look like — the thing this skill must
  prevent?"
- "Which decisions does the skill make, and how would an expert make each one?"
- "Is there a source — a talk, a guide, a team convention — this is adapted from?" Its
  credit goes in the skill.

Ask a few questions at a time, most important first. Finish when the functionality and
the standard for good output are both clear.

### Step 2: Plan the Contents

For each example, work out how to execute it from scratch, then what would make doing it
repeatedly reliable:

- Code rewritten every time → `scripts/`
- Schemas or domain detail rediscovered every time → `references/`
- Boilerplate copied every time → `assets/`

Then plan the judgment: list each decision from Step 1 and choose its form from the
*Encode Judgment* table. Any standard you can test mechanically gets a check script.

### Step 3: Initialize the Skill

For a new skill, run:

```bash
scripts/init_skill.py <skill-name> --path <output-directory>
```

It creates the directory, a SKILL.md template with `author`, `tags` and a "What good
output looks like" block, and example `scripts/`, `references/` and `assets/` files.
Delete the examples you don't need. Skip this step when editing an existing skill.

### Step 4: Edit the Skill

Write for another instance of Claude: include what is non-obvious to it. Patterns:

- **Judgment** — [references/judgment-patterns.md](references/judgment-patterns.md)
- **Multi-step processes** — [references/workflows.md](references/workflows.md)
- **Output formats and quality standards** — [references/output-patterns.md](references/output-patterns.md)

Start with the bundled resources. Run every script you add; test check scripts with
pytest (judgment-patterns.md has the conventions the futuregerald plugin repo's CI
expects).

**Frontmatter:**

- `name` — kebab-case, matches the directory.
- `description` — what the skill does and every trigger for it. "When to use" belongs
  here, not in the body. No angle brackets.
- `author` — the person who wrote the skill.
- `tags` — an inline list, `tags: [review, quality]`; the installer filters on it.

Keep any other field an existing skill already uses — `model`, `effort`, `languages`
(the installer filters on it), `argument-hint`, `trigger`, `version`, `user-invocable`,
`license`, `allowed-tools`, `metadata`. Don't invent new ones; the validator rejects them.

**Body:** imperative form. Put "What good output looks like" near the top. When content
is adapted from a source, end with a credit line: `Adapted from <author>, "<title>",
<venue>, <year> — <link>`.

### Step 5: Validate

```bash
python3 scripts/quick_validate.py <path/to/skill-folder>
# without PyYAML installed:
uv run --with pyyaml python3 scripts/quick_validate.py <path/to/skill-folder>
```

It checks the frontmatter fields, that `name` matches the directory, that `tags` is an
inline list, the description, and that no `TODO` placeholder from `init_skill.py`
remains. In the futuregerald plugin repo, skills ship through the installer, so don't
build `.skill` packages there; `scripts/package_skill.py` is for distributing a skill
on its own.

Then review the skill with `skill-reviewer`.

### Step 6: Edit the Output

A skill is done when its output is good, not when its files exist.

1. Run the skill on a real task.
2. Review what it produced with `editor-pass`, not the skill text.
3. Trace each problem in the output back to the skill — a missing decision tree, a vague
   instruction, an absent example — and fix it there.
4. Repeat. Expect several rounds; accepting the first result is how generic skills ship.

---

Forked by Gerald Onyango from Anthropic's `skill-creator` (Apache-2.0, see LICENSE.txt).
Modified: SKILL.md, scripts/init_skill.py, scripts/quick_validate.py; added
references/judgment-patterns.md, references/progressive-disclosure.md,
scripts/test_quick_validate.py.

Judgment principles adapted from Katie Dill (Stripe), "How to scale intent, quality, and
artistry with AI", Lenny & Friends Summit, 2026 — https://www.youtube.com/watch?v=GLvFTMtw4Jk
