---
name: baton-pass
description: Pass the baton with a deterministic ACTIVE_CONTEXT.md snapshot so fresh context picks up the work. MANDATORY in Antigravity and Gemini at turn 40-45 or the ~160k-250k token limit, before auto-compaction hits. In Claude Code, use it at turn 40-45 of a long orchestration to hand the next pending step to a fresh sub-agent (with subagent-driven-development), or when asked to 'pass the baton', 'baton pass', 'save active context', 'snapshot context', or 'continue from active context'. Not a replacement for regular handoffs.
author: Gerald Onyango
tags: [workflow, context, tokens, handoff, architecture]
---

# Baton Pass

The Baton Pass is a deterministic state snapshot that transfers execution context to fresh context: a new session in Antigravity and Gemini, a new sub-agent in Claude Code. It prevents auto-compaction amnesia and keeps the working context small.

## Not a Handoff

`ACTIVE_CONTEXT.md` is a brief for the next agent, used only by this skill. It is not a handoff, and it never replaces one.

- Regular handoffs keep working unchanged: Prism `session_save_handoff`, the SessionEnd handoff in `~/.claude/handoffs/`, and the 4-section phase handoffs (Changes Made, Flawed Assumptions, Carried Risks, Verification Evidence). They hold more than the active context and are saved persistently.
- Never write `ACTIVE_CONTEXT.md` into `~/.claude/handoffs/`, and never save it as the Prism handoff. Save the regular handoff as you otherwise would.

## What Good Looks Like

- **Target consumer:** The fresh agent waking up on Turn 1 of Session B, or an autonomous worker subagent.
- **Good output:** A self-contained markdown snapshot that enables the incoming agent to execute the next pending step immediately without re-reading files, grepping for symbols, or re-running exploratory traces.
- **Generic default to prevent:** Vague narrative summaries ("Updated the auth service and made progress on tokens; next need to wire up the controller") that lose line numbers, schema shapes, exact test commands, and failure modes.
- **The detail a careless version misses:** Strict negative constraints and failed dead ends. Without documented negative constraints, Session B repeats the exact flawed approaches Session A already spent turns debugging.

## When to Trigger

```
Antigravity / Gemini:
  Turn 40-45 or token usage near ~160k-250k?                   -> Execute Baton Pass immediately
  User said 'pass the baton' or 'snapshot context'?            -> Execute Baton Pass immediately
  Under 30 turns with low token usage?                         -> Continue in-session

Claude Code:
  Turn 40-45 while orchestrating multi-step work?              -> Snapshot, hand the next step to a fresh sub-agent
  User said 'pass the baton' or 'snapshot context'?            -> Snapshot, hand the next step to a fresh sub-agent
  Conversational work (decisions, synthesis, review with user)? -> Continue in-session; compaction hooks cover it
  Token count alone?                                           -> Not a trigger (see below)
```

In Antigravity and Gemini, never let a session run into auto-compaction: it compacts early, drops edge-case constraints mid-generation, and burns context re-reading files.

Claude Code is different. Its context window is much larger (up to 1M), compaction comes late, and the model cannot see its own token count, so a token threshold would force a restart while most of the window is still usable. Long orchestrations still drift as stale tool output piles up, which is what the turn trigger catches. Work that cannot be delegated (decisions, synthesis, anything in the user's voice) stays in-session.

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

Determine the repository project key (not a git repo: `dev`):
```bash
git rev-parse --path-format=absolute --git-common-dir | sed 's|/\.git$||' | xargs basename | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-' | sed 's/-$//'
```

1. Write snapshot file to collision-safe path:
   `~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<branch-or-topic-slug>-ACTIVE_CONTEXT.md`
2. In Antigravity IDE, also mirror to:
   `<appDataDir>/brain/<conversation-id>/ACTIVE_CONTEXT.md`
3. If Prism MCP is active, save the regular Prism handoff as usual (`session_save_handoff` with the current `expected_version`). Do not save the snapshot as the handoff; it would replace the richer one (see Not a Handoff).
4. Output ready-to-paste prompt for the user:
   `Ingest ~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<slug>-ACTIVE_CONTEXT.md and continue from Pending Steps.`

### 2. Claude Code

Claude Code passes the baton to a fresh sub-agent, not a fresh session. The orchestrator keeps its conversation with the user; the heavy work moves out of its context.

1. Write the snapshot (project key as above):
   `~/.claude/baton-pass/<project-key>/<YYYY-MM-DD-HHMMSS>-<branch-or-topic-slug>-ACTIVE_CONTEXT.md`
   Never under `~/.claude/handoffs/`; that store belongs to regular handoffs.
2. Hand Section 3's next step to a fresh sub-agent using `subagent-driven-development`. The snapshot is its brief: derive the stream context file from Sections 1, 3 and 4, and include Section 4 verbatim.
3. Fold the sub-agent's 4-section handoff back in: verify its gate yourself, then update the snapshot (Sections 2 and 3, new Section 4 traps) before dispatching the next step.
4. Keep the snapshot current after every sub-agent return, so it never describes a finished state.
5. Run `/clear` only when the user asks. Then give the resume prompt:
   `Ingest ~/.claude/baton-pass/<project-key>/<file>-ACTIVE_CONTEXT.md and continue from Pending Steps.`

#### Compaction safety net (hooks)

`scripts/compact_checkpoint.py` runs on two Claude Code hook events:

- **PreCompact:** writes `~/.claude/baton-pass/<project-key>/<timestamp>-<session-id>-precompact.md` with the branch, HEAD, uncommitted changes, the files edited this session (sub-agents included) and the path of the full transcript. It never blocks compaction.
- **SessionStart, matcher `compact`:** tells Claude the summary is lossy, and points it at this session's checkpoint and at the full transcript to grep for dropped detail before re-reading files. It does not point at any `ACTIVE_CONTEXT.md`.

Register both in `~/.claude/settings.json`. The command must never exit 2, because exit 2 on PreCompact blocks compaction, and `python3` exits 2 when the script is missing (for example after the plugin checkout switches branch):

```json
"PreCompact": [{"hooks": [{"type": "command", "command": "f=\"$HOME/.claude/skills/baton-pass/scripts/compact_checkpoint.py\"; [ -f \"$f\" ] && python3 \"$f\"; exit 0"}]}],
"SessionStart": [{"matcher": "compact", "hooks": [{"type": "command", "command": "f=\"$HOME/.claude/skills/baton-pass/scripts/compact_checkpoint.py\"; [ -f \"$f\" ] && python3 \"$f\"; exit 0"}]}]
```

#### Cost
A long session does not rewrite the prompt cache; each turn writes only its new part. What grows is the cost of reading the cache: every turn re-reads the whole context at about a tenth of the input price. Moving heavy work into sub-agents keeps the orchestrator's context, and so that per-turn cost, small.

### 3. Subagent Delegation (Brief In, Handoff Out)

In autonomous multi-phase workflows:
1. **Orchestrator to Worker (Brief In):** Orchestrator extracts Section 1 contracts and Section 3 next steps into the subagent's task brief.
2. **Worker Execution:** Subagent operates in bounded context without inheriting parent chat history.
3. **Worker to Orchestrator (Handoff Out):** Subagent writes a handoff detailing Changes Made, Flawed Assumptions, Carried Risks, and Verification Evidence.
4. **State Fold:** Orchestrator validates verification claims with real commands, updates `ACTIVE_CONTEXT.md`, and advances to the next step.

## Turn 1 Rehydration Protocol

When starting Session B with `Ingest <path> and continue from Pending Steps`:

1. **Read Snapshot:** Load the file directly via filesystem read.
2. **Absorb Contracts:** Ingest Section 1 schemas into memory. Before trusting them, run one cheap check: `git diff --stat <snapshot Commit>`. Files it does not list are as documented, so do not re-read or grep them; re-read only the ones that changed.
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
