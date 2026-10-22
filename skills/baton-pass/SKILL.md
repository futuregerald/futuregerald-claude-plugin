---
name: baton-pass
description: Pass the baton with a deterministic ACTIVE_CONTEXT.md snapshot so fresh context picks up the work. MANDATORY in Antigravity and Gemini when the breaker hook warns that the context estimate has reached ~105k tokens (it blocks at ~130k), before auto-compaction hits, or when asked to 'pass the baton', 'baton pass', 'save active context', 'snapshot context', or 'continue from active context'. Antigravity and Gemini only: never used in Claude Code. Not a replacement for regular handoffs.
author: Gerald Onyango
tags: [workflow, context, tokens, handoff, architecture]
---

# Baton Pass

The Baton Pass is a deterministic state snapshot that transfers execution context to a fresh session in Antigravity and Gemini. It prevents auto-compaction amnesia and keeps the working context small.

## Not a Handoff

`ACTIVE_CONTEXT.md` is a brief for the next agent, used only by this skill. It is not a handoff, and it never replaces one.

- Regular handoffs keep working unchanged: Prism `session_save_handoff`, any SessionEnd handoff store you use (such as `~/.claude/handoffs/`), and the 4-section phase handoffs (Changes Made, Flawed Assumptions, Carried Risks, Verification Evidence). They hold more than the active context and are saved persistently.
- Never write `ACTIVE_CONTEXT.md` into `~/.claude/handoffs/`, and never save it as the Prism handoff. Save the regular handoff as you otherwise would.

## What Good Looks Like

- **Target consumer:** The fresh agent waking up on Turn 1 of Session B, or an autonomous worker subagent.
- **Good output:** A self-contained markdown snapshot that enables the incoming agent to execute the next pending step immediately without re-reading files, grepping for symbols, or re-running exploratory traces.
- **Generic default to prevent:** Vague narrative summaries ("Updated the auth service and made progress on tokens; next need to wire up the controller") that lose line numbers, schema shapes, exact test commands, and failure modes.
- **The detail a careless version misses:** Strict negative constraints and failed dead ends. Without documented negative constraints, Session B repeats the exact flawed approaches Session A already spent turns debugging.

## When to Trigger

```
Antigravity / Gemini:
  Breaker block (context estimate ~130k)?                      -> Execute Baton Pass immediately
  Breaker warn (context estimate ~105k)?                       -> Execute Baton Pass at the next natural break
  User said 'pass the baton' or 'snapshot context'?            -> Execute Baton Pass immediately
  No warn yet?                                                 -> Continue in-session

Claude Code:
  Any trigger                                                  -> Not used; continue in-session
```

In Antigravity and Gemini, never let a session run into auto-compaction: it compacts early, drops edge-case constraints mid-generation, and burns context re-reading files. Measured over 70 real Antigravity compactions, the context at compaction (transcript bytes since the previous compaction ÷ 4) had a p10 of ~145k tokens, so the warn at ~105k leaves room to finish the current step and hand off. The estimate is per segment, so a turn count says little: replayed over 150 transcripts, the warn would have fired as early as step 54 and at a median of step 163.

When the breaker's baton rule is enforced (it starts in log mode), the block level allows only the snapshot write, Prism calls and reading the snapshot. It denies `invoke_subagent`, because a helper's report comes back into the parent and grows its context. Writing the snapshot unlocks this conversation until the estimate grows another ~50k.

Claude Code does not use this skill. Its context window is much larger (up to 1M), compaction comes late, and the model cannot see its own token count, so a snapshot-and-restart interrupts work while most of the window is still usable.

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
git rev-parse --path-format=absolute --git-common-dir | sed 's|/\.git$||' | xargs basename | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9' '-' | sed 's/^-//; s/-$//'
```

1. Write snapshot file to collision-safe path:
   `~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<branch-or-topic-slug>-ACTIVE_CONTEXT.md`
2. In Antigravity IDE, also mirror to:
   `<appDataDir>/brain/<conversation-id>/ACTIVE_CONTEXT.md`
3. If Prism MCP is active, save the regular Prism handoff as usual (`session_save_handoff` with the current `expected_version`). Do not save the snapshot as the handoff; it would replace the richer one (see Not a Handoff).
4. Output ready-to-paste prompt for the user:
   `Ingest ~/.gemini/handoffs/<project-key>/<YYYY-MM-DD-HHMMSS>-<slug>-ACTIVE_CONTEXT.md and continue from Pending Steps.`

### 2. Claude Code: compaction checkpoint only

Claude Code never writes an `ACTIVE_CONTEXT.md` and never hands work off through this skill. The one piece it keeps is the compaction checkpoint script below, which runs from hooks, not from the model.

#### Compaction checkpoint (hooks)

`scripts/compact_checkpoint.py` runs on two Claude Code hook events:

- **PreCompact:** writes `~/.claude/baton-pass/<project-key>/<timestamp>-<session-id>-precompact.md` with the branch, HEAD, uncommitted changes, the files edited this session (sub-agents included) and the path of the full transcript. It never blocks compaction.
- **SessionStart, matcher `compact`:** tells Claude the summary is lossy, and points it at this session's checkpoint and at the full transcript to grep for dropped detail before re-reading files. It does not point at any `ACTIVE_CONTEXT.md`.

Register both in `~/.claude/settings.json`, appending these entries to any existing `PreCompact` and `SessionStart` arrays rather than replacing them. The path assumes the skills directory is symlinked to `~/.claude/skills`; for any other install, point `f` at wherever `compact_checkpoint.py` lives. The command must never exit 2, because exit 2 on PreCompact blocks compaction, and `python3` exits 2 when the script is missing (for example after the plugin checkout switches branch):

The first entry goes in `PreCompact`, the second in `SessionStart`:

```json
{"hooks": [{"type": "command", "timeout": 30, "command": "f=\"$HOME/.claude/skills/baton-pass/scripts/compact_checkpoint.py\"; [ -f \"$f\" ] && python3 \"$f\"; exit 0"}]}
{"matcher": "compact", "hooks": [{"type": "command", "timeout": 30, "command": "f=\"$HOME/.claude/skills/baton-pass/scripts/compact_checkpoint.py\"; [ -f \"$f\" ] && python3 \"$f\"; exit 0"}]}
```

### 3. Subagent Delegation in Antigravity and Gemini (Brief In, Handoff Out)

A worker sub-agent gets its brief from Sections 1, 3 and 4 and returns a handoff with Changes Made, Flawed Assumptions, Carried Risks and Verification Evidence. The orchestrator verifies its gate, then updates the snapshot in place: corrects Section 1 from Changes Made and Flawed Assumptions, records results in Section 2, advances Section 3, adds new traps to Section 4, and sets `Commit:` to the new HEAD.

## Turn 1 Rehydration Protocol

When starting Session B with `Ingest <path> and continue from Pending Steps`:

1. **Read Snapshot:** Load the file directly via filesystem read.
2. **Absorb Contracts:** Ingest Section 1 schemas into memory. Before trusting them, run two cheap checks: `git diff --stat <snapshot Commit>` and `git status --short`. Tracked files in this repo that neither lists are as documented, so do not re-read or grep them. Re-read the ones that changed, untracked files, and anything outside this repo, which git cannot vouch for.
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
