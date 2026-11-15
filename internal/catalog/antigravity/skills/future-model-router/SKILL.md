---
name: future-model-router
description: >-
  Measurements behind delegation and model routing in Antigravity and Gemini —
  whether to run work inline, in an invoke_subagent helper, or in a Peer-Pipeline
  Swarm, and at what reasoning budget. Load it to settle a routing question: is
  a cheaper model actually cheaper, when to use Argon Medium vs Carbon High vs
  Hydrogen Low, and how restricted tool schemas cut subagent dispatch cost.
---

# Future Model Router (Antigravity & Gemini)

Empirical measurements and routing rules for deciding whether work runs **inline** in the Coordinator, in a harness-enforced **`invoke_subagent`** helper, or in Antigravity's **`peer-pipeline-swarm`** (`swarm`), and which model tier runs each role.

The per-turn habits live in `~/.gemini/GEMINI.md` (§0.9) and project `GEMINI.md` (generated from `templates/GEMINI-BASE.md`). Load this skill and [`references/model-map.md`](references/model-map.md) when settling a routing or token-budget question.

## Core Findings

1. **Carbon Medium (`2×`) vs. Argon Medium (`1×`) on End-to-End `/start` + `peer-pipeline-swarm`:**
   - Across our controlled 24-assertion zero-knowledge browser-extension M2 benchmark, **`gemini-4-carbon-medium`** (Coordinator + `impl`) beat **`gemini-4-argon-medium`** `24/24 (100%)` vs `20/24 (83.3%)` and won `2–0` blind (`84 vs 66` Opus; `94 vs 54` Carbon), while `Carbon Implementer` finished `2.8m` faster (`53.1m` vs `55.9m`) in `13.5%` fewer turns and cut downstream reviewer rework by `-24.1%` (`-40.9%` less weekly Opus quota). Meanwhile, `Argon Medium` remains the minimum `1×` coding floor (`96.4%` vs `82.1%` and `5.9×` fewer input tokens than `Model: "flash"`).
   - **Rule:** Keep **`Carbon Medium`** as the default Coordinator and `/start` `impl` (`gemini-4-carbon-medium`) for feature work, architecture, and complex debugging; invoke **`/start-small`** (`impl: gemini-4-argon-medium` + `gemini-4-carbon-high` reviewers) for smaller, well-bounded changes without switching the UI dropdown; use **`Hydrogen`** (`0.2×`) for non-code EM/Jira/meeting/ops tasks; and reserve `Model: "flash"` strictly for `<50` LOC single-file mechanical boilerplate.

2. **Never pass `Model: "pro"` on Argon or Carbon sessions:**
   - `invoke_subagent` only accepts `Model: "inherit" | "pro" | "flash" | "flash_lite"`. Passing `"pro"` routes to the legacy Gemini 3.1 Pro slot (`20/28` pass rate and `2.1×` more tokens), whereas `"inherit"` copies the active UI model (**Carbon** or **Argon**).

3. **Preserve `Carbon High` + Cross-Family `Opus 5.5 High` on Adversarial Reviews:**
   - Under `/start`, pin `code-rev` to **`claude-opus-5-5-high`** and `plan-rev` / `sec-rev` / `sql-rev` / `editor-rev` to **`gemini-4-carbon-high`**. On a `Carbon Medium` UI session, spawn one-shot `gemini-4-carbon-high` `swarm` workers for non-trivial `plan-review` and standalone pre-PR reviews (`Model: "inherit"` on `Carbon Medium` downgrades reviewers to `Medium`). Under `/start-small`, pin `code-rev` + `sec-rev` to `gemini-4-carbon-high` on 5-hour Google quota to preserve 7-day Anthropic Opus quota.

4. **Restricted `tools:` Grants & Lean Coordinator Rule:**
   - Dedicated subagents in `~/.gemini/config/plugins/antigravity-subagents/agents/*.md` with `tools: [read_file]` load ~31k tokens (vs ~57k–64k in `swarm`). Never let a `Carbon Medium` Coordinator code multi-file (`>1` file / `>15` LOC) post-review or Phase 12 `editor-pass` fixes inline on its warm context—delegate them to `implementer` + `fix-check`.

5. **Warm Subagent Context Ceiling (~80k–100k tokens):**
   - Reusing the same `implementer` across multiple roadmap phases re-bills a `150k–200k` history across `30+` tool calls (`5–8×` more expensive than spawning a fresh implementer per phase). Cap warm reuse at **1 roadmap phase (+ its fix round)**.

## Decision Tree

```text
Is this a code change?
├── No (chat, EM report, Jira triage, meeting debrief, tracker lookup, 1:1 prep)
│   ├── Mechanical MCP extraction on pre-fed IDs (<=3 turns)?
│   │   └── invoke_subagent(Model: "flash_lite", max 3 turns)
│   ├── Bounded read-only repo/doc search (<=15 turns)?
│   │   └── invoke_subagent(Model: "flash", max 15 turns)
│   └── Direct synthesis / non-code Q&A?
│       └── Run inline (Hydrogen High/Low 0.2x recommended)
└── Yes (12-phase Development Lifecycle)
    ├── Phases 0–3 (Shape, Trace, Plan) & Phases 10–12 (Blast-Radius, Commit, Handoff)
    │   └── Run inline in Coordinator (Carbon Medium 2x default; Argon Medium 1x optional for simple tasks)
    ├── Phase 3½ (Plan Review)
    │   ├── /start + UI not on Carbon High -> one-shot swarm worker (gemini-4-carbon-high, immediate kill)
    │   └── /start-small, Carbon High UI, small plan, or Gemini CLI -> invoke_subagent(TypeName: "plan-reviewer", Model: "inherit")
    └── Phases 4–9 (Implement, Test, Lint, Simplify, Code/Security/SQL Review)
        ├── Trivial (<5 turns, <=2 files already in context)?
        │   └── Implement inline + invoke_subagent(TypeName: "code-quality-reviewer" & "security-reviewer", Model: "inherit")
        ├── Small/bounded change (/start-small, >=3 files)?
        │   └── Peer-Pipeline Swarm (small): impl (gemini-4-argon-medium, share) + code-rev & sec-rev (gemini-4-carbon-high) + monitor (gemini-4-hydrogen-low)
        ├── Standard feature / debugging (/start, /begin, /peer-pipeline-swarm)?
        │   └── Peer-Pipeline Swarm (default): impl (gemini-4-carbon-medium, share) + code-rev (claude-opus-5-5-high) + sec-rev/sql-rev (gemini-4-carbon-high) + monitor (gemini-4-hydrogen-low)
        └── Step-by-step Subagent-Driven (or Gemini CLI)?
            └── invoke_subagent(TypeName: "implementer", Model: "inherit") + parallel code-quality-reviewer & security-reviewer
```

See [`references/model-map.md`](references/model-map.md) for the complete model table, dispatch templates, and `antigravity-subagents` tool grants.
