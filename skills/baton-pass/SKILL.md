---
name: baton-pass
description: Pass the baton between sessions with a deterministic ACTIVE_CONTEXT.md snapshot. Use when approaching turn 40-45 in any coding session, when approaching the ~200k-250k token context limit before auto-compaction hits, when switching between sessions, when preparing a handoff snapshot, or when asked to 'pass the baton', 'baton pass', 'save active context', 'snapshot context', 'clear context', 'handoff snapshot', or 'continue from active context'.
author: Gerald Onyango
tags: [workflow, context, tokens, handoff, architecture]
---

# Baton Pass

The Baton Pass is a deterministic state snapshot that transfers execution context across session boundaries. It prevents auto-compaction amnesia, eliminates token bloat, stabilizes prompt caches, and guarantees zero token debt when transitioning between sessions or subagents.

## What Good Looks Like

- **Target consumer:** The fresh agent waking up on Turn 1 of Session B, or an autonomous worker subagent.
- **Good output:** A self-contained markdown snapshot that enables the incoming agent to execute the next pending step immediately without re-reading files, grepping for symbols, or re-running exploratory traces.
- **Generic default to prevent:** Vague narrative summaries ("Updated the auth service and made progress on tokens; next need to wire up the controller") that lose line numbers, schema shapes, exact test commands, and failure modes.
- **The detail a careless version misses:** Strict negative constraints and failed dead ends. Without documented negative constraints, Session B repeats the exact flawed approaches Session A already spent turns debugging.

## When to Trigger

Evaluate turn count and token accumulation continuously:

```
Is conversation at Turn 40-45 or token usage near ~200k-250k? -> Execute Baton Pass immediately
Did user say 'pass the baton', 'snapshot context', or '/clear'?   -> Execute Baton Pass immediately
Is a multi-phase implementation subagent concluding its phase?   -> Write Handoff snapshot
Under 30 turns with low token usage and active work ongoing?     -> Continue in-session
```

Never let a session run into auto-compaction. In-flight compaction causes silent amnesia, drops edge-case constraints, and burns context re-reading previously examined files.

## The Canonical 4-Section Schema

Every `ACTIVE_CONTEXT.md` file contains four required sections. A template is available in [references/template.md](references/template.md).

### 1. Active File Schemas & Contracts
List every active file, its exact path, exported interface, data shape, and wire protocol. The incoming session must never have to open these files just to inspect their interface.

### 2. Solved Issues, Architectural Decisions & Test Results
Document every settled decision, resolved defect root cause, exact test command, and gate output. State verified facts with commands and exit codes.

### 3. Ordered Pending Steps
List numbered, concrete tasks to execute next. Each task must specify target files, input/output requirements, and the exact verification command.

### 4. Strict Negative Constraints & Traps to Avoid
Enumerate forbidden patterns, rejected alternatives, known dead ends, and system invariants.

## Platform Execution Workflows

### 1. Antigravity & Gemini CLI

Determine the repository project key:
```bash
git rev-parse --path-format=absolute --git-common-dir | sed 's|/\.git$||' | xargs basename | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g'
```

1. Write snapshot file to collision-safe path:
   `~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<branch-or-topic-slug>-ACTIVE_CONTEXT.md`
2. In Antigravity IDE, also mirror to:
   `<appDataDir>/brain/<conversation-id>/ACTIVE_CONTEXT.md`
3. If Prism MCP is active, call `session_save_handoff` with the snapshot content and current `expected_version`.
4. Output ready-to-paste prompt for the user:
   `Ingest ~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<slug>-ACTIVE_CONTEXT.md and continue from Pending Steps.`

### 2. Claude Code

1. Write snapshot file to canonical path:
   `~/.gemini/handoffs/<project-key>/ACTIVE_CONTEXT.md`
2. Instruct the user to clear session state:
   Run `/clear`
3. Provide the resumption prompt:
   `Ingest ~/.gemini/handoffs/<project-key>/ACTIVE_CONTEXT.md and continue from Pending Steps.`

#### Prompt Caching Stability
Claude Code relies on prompt caching for fast, cost-effective responses. Unbounded sessions create token bloat that causes cache churn and write thrashing. Running `/clear` and ingesting a compact 4-section snapshot establishes a clean cache prefix, reducing latency and token costs on every subsequent turn.

### 3. Subagent Delegation (Brief In, Handoff Out)

In autonomous multi-phase workflows:
1. **Orchestrator to Worker (Brief In):** Orchestrator extracts Section 1 contracts and Section 3 next steps into the subagent's task brief.
2. **Worker Execution:** Subagent operates in bounded context without inheriting parent chat history.
3. **Worker to Orchestrator (Handoff Out):** Subagent writes a handoff detailing Changes Made, Flawed Assumptions, Carried Risks, and Verification Evidence.
4. **State Fold:** Orchestrator validates verification claims with real commands, updates `ACTIVE_CONTEXT.md`, and advances to the next step.

## Turn 1 Rehydration Protocol

When starting Session B with `Ingest <path> and continue from Pending Steps`:

1. **Read Snapshot:** Load the file directly via filesystem read.
2. **Absorb Contracts:** Ingest Section 1 schemas into memory. Treat documented interfaces as authoritative. Do not run `grep`, `find`, or file-reading tools on files documented in Section 1.
3. **Internalize Constraints:** Treat Section 4 rules as inviolable negative constraints.
4. **Execute Step 1:** Immediately begin working on Step 1 of Section 3. Run the designated gate command to verify the baseline before making modifications.

## Do / Don't Pairs

### File Schemas & Contracts

```markdown
Do:
### Active File Schemas & Contracts
- `app/services/token_verifier.rb`:
  `TokenVerifier.call(token: String, audience: String) -> Result[Claims, VerificationError]`
  `Claims = Struct.new(:user_id, :org_id, :scopes, :exp)`
- `config/jwt.yml`:
  `jwt.algorithms: ["ES256"]`
```

```markdown
Don't:
### Files
- Looked at token verifier and config files. They handle JWT tokens and algorithms.
```
The first provides callable signatures and data types. The second forces Session B to open and read both files again.

### Test Results & Verification

```markdown
Do:
### Solved Issues & Test Results
- Issue: Token expiration checked `< Time.now` instead of `< Time.now.to_i`.
  Fix: Converted expiration timestamp to integer before comparison in `TokenVerifier#valid_exp?`.
- Verification Gate:
  Command: `bundle exec rspec spec/services/token_verifier_spec.rb:45`
  Output: `1 example, 0 failures` (Exit Code 0)
```

```markdown
Don't:
### Status
- Fixed a bug with token timestamps.
- Tests are passing now.
```
The first specifies the exact failure mode, exact fix, target spec line, and exit code. The second provides no evidence.

### Negative Constraints

```markdown
Do:
### Strict Negative Constraints
- Do not use `JWT.decode` without passing `verify_iss: true`.
- Do not add fallback to RS256; system strictly requires ES256 per ADR-004.
- Do not introduce rescue block around `OpenSSL::PKey::ECError` in `TokenVerifier`; let caller handle key errors.
```

```markdown
Don't:
### Notes
- Be careful with JWT decoding and error handling.
```
The first defines unambiguous guardrails that prevent regressions and wasted work. The second is vacuous advice.
