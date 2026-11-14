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

1. **Argon Medium (`1×`) vs. Flash on Phase 4 TDD Implementation:**
   - Across our 3-arm TypeScript/Go benchmark on identical `PLAN.md` files, **Gemini 4 Argon Medium** (`Model: "inherit"` on an Argon UI session, or `gemini-4-argon-medium` in `swarm`) achieved **96.4% (`27/28`)** adversarial pass rate vs. **82.1% (`23/28`)** for `Model: "flash"`, while using **5.0× fewer turns** (`15` vs `75`), **5.93× fewer input tokens** (`1.31M` vs `7.77M`), and **19.5% less wall-clock time**.
   - **Rule:** Use **Argon Medium** (`Model: "inherit"` or `gemini-4-argon-medium`) for all standard Phase 4 implementation. Reserve `Model: "flash"` strictly for `<50` LOC single-file mechanical boilerplate/renames with zero error-handling, concurrency, or security judgment.

2. **Never pass `Model: "pro"` on Argon or Carbon sessions:**
   - `invoke_subagent` only accepts `Model: "inherit" | "pro" | "flash" | "flash_lite"`. Passing `"pro"` routes to the legacy Gemini 3.1 Pro slot (`20/28` pass rate and `2.1×` more tokens), whereas `"inherit"` copies the active UI model (**Argon** or **Carbon**).

3. **Surgical Carbon High (`2×`) + Cross-Family Opus 5.5 High for Adversarial Reviews:**
   - Because ~80–85% of stream tokens are spent in Lead triage/gates and Phase 4 TDD implementation, keeping the Coordinator and `impl` on **Argon Medium** (`1×`) while pinning adversarial reviewers to **`claude-opus-5-5-high`** (`code-rev`) and **`gemini-4-carbon-high`** (`plan-rev`, `sec-rev`, `sql-rev`, `editor-rev`) delivers maximum adversarial rigor at `~1.26×` blended stream cost vs. `2.0×` all-Carbon.

4. **Restricted `tools:` Grants Cut Subagent Schema Overhead by ~45%:**
   - An unrestricted subagent or `swarm` worker loads ~57k–64k tokens of tool/MCP schemas on turn 1.
   - Dedicated subagents in `~/.gemini/config/plugins/antigravity-subagents/agents/*.md` with `tools: [read_file]` load ~31k tokens and cannot execute shell commands or mutate the working tree.

5. **Warm Subagent Context Ceiling (~80k–100k tokens):**
   - Reusing the same `implementer` across multiple roadmap phases re-bills a `150k–200k` history across `30+` tool calls (`5–8×` more expensive than spawning a fresh implementer per phase). Cap warm reuse at **1 roadmap phase (+ its fix round)**.

## Decision Tree

```text
Is this a code change?
├── No (chat, triage, report, tracker lookup, 1:1 prep)
│   ├── Mechanical MCP extraction on pre-fed IDs (<=3 turns)?
│   │   └── invoke_subagent(Model: "flash_lite", max 3 turns)
│   ├── Bounded read-only repo/doc search (<=15 turns)?
│   │   └── invoke_subagent(Model: "flash", max 15 turns)
│   └── Direct synthesis / non-code Q&A?
│       └── Run inline (Argon Medium or Hydrogen Low/High 0.2x)
└── Yes (12-phase Development Lifecycle)
    ├── Phases 0–3 (Shape, Trace, Plan) & Phases 10–12 (Blast-Radius, Commit, Handoff)
    │   └── Run inline in Coordinator (Argon Medium 1x default; switch UI to Carbon Medium/High 2x only for ~10% complex architecture/auth/crypto/concurrency/migrations)
    ├── Phase 3½ (Plan Review)
    │   ├── Argon UI + non-trivial plan -> one-shot swarm worker (gemini-4-carbon-high, immediate kill)
    │   └── Carbon UI, small plan, or Gemini CLI -> invoke_subagent(TypeName: "plan-reviewer", Model: "inherit")
    └── Phases 4–9 (Implement, Test, Lint, Simplify, Code/Security/SQL Review)
        ├── Trivial (<5 turns, <=2 files already in context)?
        │   └── Implement inline + invoke_subagent(TypeName: "code-quality-reviewer" & "security-reviewer", Model: "inherit")
        ├── Antigravity hands-off pipeline (/start, /begin, /peer-pipeline-swarm)?
        │   └── Peer-Pipeline Swarm: impl (gemini-4-argon-medium, share) + code-rev (claude-opus-5-5-high) + sec-rev/sql-rev (gemini-4-carbon-high) + monitor (gemini-4-hydrogen-low)
        └── Step-by-step Subagent-Driven (or Gemini CLI)?
            └── invoke_subagent(TypeName: "implementer", Model: "inherit") + parallel code-quality-reviewer & security-reviewer
```

See [`references/model-map.md`](references/model-map.md) for the complete model table, dispatch templates, and `antigravity-subagents` tool grants.
