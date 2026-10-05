# Judgment Patterns

Use these forms to write a skill's decisions down in a way an agent can carry out. Pick one per decision; most skills need two or three.

1. Point of view block
2. Decision tree
3. Do/don't pair
4. On-demand lookup
5. Check script with tests
6. Templates and full flows

## 1. Point of view block

Put this near the top of SKILL.md for any skill that produces output.

```markdown
## What good output looks like

For the engineer fixing it: each finding names the file and line, what breaks,
under what input, and the fix — in that order.
The default to prevent: a list of categories ("error handling could be improved")
with no location and no consequence.
The detail a careless version misses: whether the failing path has a test at all.
```

**Don't** write "high-quality, professional output". That describes every skill, so it guides none of them.

## 2. Decision tree

Use one whenever the skill chooses between options. Write the questions in the order an expert would ask them. Each leaf names the exact choice.

```markdown
Is it a single decision or a short, focused action?   → Dialog
Is it a multi-step sequence the user completes once?  → Onboarding wizard
Is it an extended working session?                    → Full-screen overlay
None of these?                                        → Inline on the page
```

**Don't** write "Use the appropriate container for the interaction." An agent can't act on "appropriate".

## 3. Do/don't pair

Use one where right and almost-right look alike. Show both versions concretely, then add one line on what separates them.

```markdown
Do:    "A. [Hero headline] Reads as a SaaS tagline → name the dish and the city"
Don't: "Improve the hero copy"
The first says where, what is off, and what it should become.
```

Keep a skill to the two or three pairs that cover the most common mistakes. More than that is noise.

## 4. On-demand lookup

Use one when the skill covers many items (components, tables, endpoints, rules) but any task needs only a few of them. Give each item its own file, or a script that returns one item, and have SKILL.md say how to fetch it.

```markdown
For a component, read `references/components/<name>.md`. Don't read the folder.
```

Or, when the data is structured:

```bash
scripts/lookup.py date-range-picker --json   # name, import, props, do/don't
```

One item per read keeps unrelated material out of context. A single 2,000-line reference file defeats this.

## 5. Check script with tests

Use one when part of the standard can be tested mechanically: a format, a required section, banned phrases, cross-references. The agent writes the output, the script checks it, and the agent fixes what the script reports.

**Contract:**
- `check(text) -> list[str]` returns problems; an empty list means valid.
- The CLI takes `FILE` or `-` for stdin.
- Exit 0 when valid, 1 when there are problems (one per line), 2 on a usage error.

**Tests:**
- They live next to the script as `scripts/test_<name>.py`, using pytest.
- CI runs `pytest -q skills/*/scripts`, and pytest imports each test file by its basename, so **the basename must be unique across all skills**. Check with `ls skills/*/scripts/test_*.py`.
- Import the module directly (`import check_thing`); pytest puts the script's directory on the path.
- Stick to the standard library. CI installs only `pytest` and `pyyaml`.
- Cover a valid example, each rule failing alone, and the CLI's exit codes.

**Who runs it:** the agent that holds the output and has a shell. When a read-only sub-agent produces the output, the main agent pipes it to `check -`.

Check only what a script can actually decide. Judgment stays in SKILL.md: a check script that tries to score quality produces confident nonsense.

## 6. Templates and full flows

When the skill produces a whole artifact (a report, a page, a ticket set), ship a template or a worked example of the complete thing in `assets/` or `references/`, not just rules about its parts. Parts that each follow the rules can still add up to an incoherent whole. A full example shows how they fit together.

- **Template:** the skeleton with placeholders, used when structure must not vary.
- **Worked example:** one complete, realistic output, used when structure can flex but the standard can't.
- **Flow:** the ordered steps for a multi-stage job, each with its exit condition (see `workflows.md`).

Run the skill's own check script against the worked example. An example that fails its own skill's check teaches the wrong thing.
