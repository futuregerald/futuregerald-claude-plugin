# Validation Loop Logic

This file contains the loop state management, agent switching, and stagnation detection logic for Phase 3.

## Loop Variables (initialize before entering loop)

```pseudocode
iteration_count = 0
max_iterations = 10
previous_issues_count = Infinity   // Treat first iteration as progress baseline
previous_critical_count = 0
current_issues_count = null
current_critical_count = 0
last_agent_used = "ui-developer"   // Phase 2 used UI Developer
ui_developer_consecutive_failures = 0
codex_consecutive_failures = 0
design_fidelity_achieved = false
issue_count_history = []           // Track last N counts for stagnation detection
```

## Smart Agent Switching

```pseudocode
function determineFixingAgent() {
  // If Codex not enabled, always use UI Developer
  if (!codex_enabled) return "ui-developer"

  // Smart switching based on consecutive failures
  if (ui_developer_consecutive_failures >= 2) return "ui-developer-codex"
  if (codex_consecutive_failures >= 2) return "ui-developer"

  // Default: continue with last agent used
  return last_agent_used
}
```

Set `last_agent_used` to the chosen agent **before** launching it.

## Metrics Update (after each fixing iteration)

```pseudocode
const progress_made = (current_issues_count < previous_issues_count)

if (progress_made) {
  // Only reset the counter for the agent that made progress
  if (last_agent_used === "ui-developer") ui_developer_consecutive_failures = 0
  else if (last_agent_used === "ui-developer-codex") codex_consecutive_failures = 0
} else {
  // No progress - increment failure counter for the agent that tried
  if (last_agent_used === "ui-developer") ui_developer_consecutive_failures++
  else if (last_agent_used === "ui-developer-codex") codex_consecutive_failures++
}

// Update for next iteration
previous_issues_count = current_issues_count
previous_critical_count = current_critical_count
iteration_count++
```

## Stagnation Detection

After updating metrics, check for stagnation:

```pseudocode
issue_count_history.push(current_issues_count)

if (issue_count_history.length >= 4) {
  const last4 = issue_count_history.slice(-4)
  const minRecent = Math.min(...last4)
  if (minRecent >= last4[0]) {
    // No net progress across 4 iterations -- ask the user
    ASK USER: "The validation loop has not made net progress across 4 iterations
    (issue counts: [last4]). Options:
    A) Continue iterating (I'll try different approaches)
    B) Accept the current state and move to Phase 4
    C) Stop and let me fix issues manually"
    issue_count_history = []  // Reset after user decision
  }
}
```

## CRITICAL Regression Check

In Step 3.3, after extracting `current_critical_count` from the designer report:

If `current_critical_count > previous_critical_count`, a fixing agent introduced NEW critical issues. Immediately escalate to the user: "The last fix introduced new CRITICAL issues. Please review before continuing." Wait for user guidance before proceeding.
