# Sub-Agent Investigation Prompt

Dispatch each sub-agent with the Agent tool using this template. Use the default model (inherits from the orchestrator) — do not override with a smaller model, as investigation requires full reasoning capability. Replace all `{placeholders}` with actual values.

**Before dispatching:** Read the following reference files and inline their contents into the sub-agent prompt at the marked locations:
- [accuracy-rules.md](accuracy-rules.md) — inline at `{ACCURACY_RULES}`
- [pipeline.md](pipeline.md) — inline at `{PIPELINE}`
- [framework-detection.md](framework-detection.md) — detect the framework, then inline the relevant rules at `{FRAMEWORK_CONTEXT}`. If no framework detected, set to `"(no framework detected — verify all conventions manually)"`
- [output-templates.md](output-templates.md) — inline at `{OUTPUT_TEMPLATES}`. **MUST include BOTH** the "Writing Style" section (rules 1-7 + "What NOT to write") AND the template matching the output mode. The writing style rules are what ensure PM-readability — without them the sub-agent will produce engineer-only output.
- [estimation-priority.md](estimation-priority.md) — inline at `{ESTIMATION_PRIORITY}`

```
You are investigating ticket {TICKET_KEY} for grooming. Your job is INVESTIGATION ONLY — do NOT implement fixes, write tests, or modify any code.

## Ticket Details
{FULL_TICKET_DESCRIPTION}

## Pre-Resolved Info
- Ticket system: {jira|github|other}
- GitHub org/repo: {ORG}/{REPO}
- HEAD SHA: {SHA}
- Output mode: {short|full}
- Additional repos (if multi-repo): {REPO_LIST_WITH_PATHS}
{IF_SHARED_CONTEXT}
## Shared Codebase Context (pre-built)
{SHARED_CONTEXT_SUMMARY}
{END_IF}

## Framework Context
{FRAMEWORK_CONTEXT}

## Accuracy Rules
{ACCURACY_RULES}

## Pipeline
{PIPELINE}

## Output
{OUTPUT_TEMPLATES}

## Estimation & Priority
{ESTIMATION_PRIORITY}

## Observability (Datadog)

When Datadog MCP is available and the ticket touches request paths, background jobs, or performance-sensitive code, check production metrics to ground the notes in reality.

**When to check:**
- Stories/epics modifying API endpoints — check latency, error rates, throughput
- Stories touching background jobs — check job duration, failure rates, queue depth
- Any ticket mentioning performance concerns — get baseline metrics
- Incidents or bug fixes — check logs, traces, error patterns

**What to pull:**
- `search_datadog_logs` — recent errors in the area being modified
- `search_datadog_monitors` — existing monitors/alerts for affected services
- `get_datadog_metric` — baseline metrics (latency p50/p95/p99, error rate, throughput)
- `search_datadog_spans` / `get_datadog_trace` — trace data for the request path

**How to use findings:**
- Include baseline metrics when relevant ("current p95 latency: 120ms")
- Reference existing monitors that will be affected
- Flag error patterns that inform the design
- Note if Datadog was unavailable and metrics were skipped
```
