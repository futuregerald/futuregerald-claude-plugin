# Investigation Phase

Ground every ticket in actual codebase and project state. No hallucinating, no guessing.

## Depth by Ticket Type

### Initiative (Shallow)

Understand the strategic landscape. Don't trace code paths.

**Codebase:**
- `get_architecture` or `query_graph` for high-level structure
- Check if relevant packages/domains exist

**Context (always):**
- Read parent initiative or program-level tickets
- Search Jira for related epics and their status
- Search Confluence for related docs, RFCs, ADRs
- Search GitHub for related PRs (merged and open)

**Sub-agent dispatch:**
```
Agent(sonnet): "Search Jira for all epics under initiative {KEY}.
Also search for tickets mentioning {keywords}. Return: ticket keys,
summaries, statuses, and any blocked/blocking relationships."
```

### Epic (Medium)

Understand the domain and key technical boundaries.

**Codebase:**
- `search_graph` and `search_code` for key entities mentioned in the epic
- `trace_call_path` for the primary flow being modified
- Read key files (models, interactors, controllers) -- not exhaustively, just the ones central to the epic
- Check test coverage in the area

**Context (always):**
- Read the parent initiative
- Read all linked tickets (blocked by, blocks, relates to)
- Search Jira for tickets in the same area (by label, component, or keyword)
- Search Confluence for related docs
- Search GitHub for recent PRs touching the same files/area
- Read CONTRIBUTING.md for repo-specific conventions

**Sub-agent dispatch (parallel):**
```
Agent(sonnet): "Investigate the codebase for {domain}. Find:
relevant models and their associations, key interactors/services,
controller endpoints, test files. Return file paths and brief
descriptions of what each does."

Agent(sonnet): "Search Jira for tickets related to {keywords} in
projects {projects}. Also search GitHub PRs in {repos} mentioning
{keywords}. Search Confluence for docs about {topic}. Return:
ticket keys with summaries, PR numbers with titles, doc titles
with links."
```

### Story (Deep)

Understand exactly what code will change and how.

**Codebase:**
- Everything from Epic, plus:
- Read the actual source files that will be modified
- Trace the full call path end-to-end
- Check for callbacks, hooks, observers, and side effects
- Map database schema for relevant tables
- Read existing tests to understand expected behavior
- Check for feature flags, environment-specific behavior

**Context (always):**
- Everything from Epic, plus:
- Read the parent epic's acceptance criteria
- Check if sibling stories have been completed (what's already built)
- Search git log for recent changes to the files in scope

**Sub-agent dispatch (parallel):**
```
Agent(sonnet): "Deep investigation of {feature area}. Read these
files: {file_list}. For each: document the class/module purpose,
public methods, associations/dependencies, callbacks, and any
feature flags. Check for existing tests. Return structured findings."

Agent(sonnet): "Search git log for recent changes to {file_paths}.
Search GitHub PRs that modified these files. Check Jira for related
tickets. Search Confluence for docs. Return: recent commits with
messages, PR numbers with descriptions, related tickets, docs."

Agent(opus): "Analyze the architectural implications of {proposed change}.
Consider: data model impacts, API contract changes, cross-package
dependencies (Packwerk), authorization (Pundit), and potential
side effects. Return: risks, dependencies, and recommendations."
```

### Spike (Medium)

Understand enough to frame what's unknown.

**Codebase:**
- Same as Epic depth
- Focus on identifying boundaries of current knowledge

**Context (always):**
- Same as Epic
- Specifically look for prior attempts or discussions about the unknown

## Verification Rules

1. **Every file path mentioned in the ticket must be verified** -- check it exists via Glob or Read
2. **Every model relationship must be verified** -- read the model file, confirm the association
3. **Every method/scope referenced must be verified** -- grep for it, confirm it exists and does what you say
4. **Every status/state mentioned must be verified** -- check the enum, state machine, or constant definition
5. **Follow threads to verify** -- if investigation reveals a callback or observer, read it. Don't assume what it does.

## Observability (Datadog)

When Datadog MCP is available and the ticket touches request paths, background jobs, or performance-sensitive code, check production metrics to ground the ticket in reality.

**When to check:**
- Stories/epics modifying API endpoints -- check latency, error rates, throughput
- Stories touching background jobs -- check job duration, failure rates, queue depth
- Any ticket mentioning performance concerns -- get baseline metrics
- Incidents or bug fixes -- check logs, traces, error patterns

**What to pull:**
- `search_datadog_logs` -- recent errors or patterns in the area being modified
- `search_datadog_monitors` -- existing monitors/alerts for affected services
- `get_datadog_metric` -- baseline metrics (latency p50/p95/p99, error rate, throughput)
- `search_datadog_spans` / `get_datadog_trace` -- trace data for the request path
- `search_datadog_services` / `search_datadog_service_dependencies` -- service topology

**How to use findings:**
- Include baseline metrics in the ticket when relevant ("current p95 latency: 120ms")
- Reference existing monitors that will be affected
- Flag error patterns that inform the design
- Note if Datadog was unavailable and metrics were skipped

**Sub-agent dispatch:**
```
Agent(sonnet): "Check Datadog for the {service/endpoint} area.
Search logs for errors mentioning {keywords}. Get latency metrics
for {endpoint}. Check for existing monitors. Return: error patterns,
baseline metrics, and any active alerts."
```

## When Investigation Finds Surprises

If investigation reveals something that contradicts the user's assumptions or the linked ticket's description:
- **Interactive mode:** Surface it during the grill phase. "Investigation found X, which contradicts Y. How should we handle this?"
- **Autonomous mode:** Log it as an assumption with context. "Investigation found X. Assuming Y because Z. Confidence: Medium."
