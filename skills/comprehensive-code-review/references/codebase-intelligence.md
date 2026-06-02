# Codebase Intelligence Gathering

**Always populate `{CODEBASE_CONTEXT}` with actual search output — never with instructions to search.**

Delegate search to cheaper models. The orchestrator should NOT run raw grep/glob searches itself.

## Dispatch a Sonnet exploration agent

For multi-step codebase context gathering:

```
Agent({
  model: "sonnet",
  subagent_type: "Explore",
  prompt: "I'm reviewing a PR that changes these files: {FILE_LIST}.
  I need codebase context for a code review. For each changed file, find:
  1. Similar patterns in the codebase (interactors, policies, serializers, controllers)
  2. Existing callbacks on changed models
  3. Related factories in spec/factories/
  4. If codebase-memory-mcp is available, use get_architecture and trace_call_path on affected areas.
  5. Otherwise, run 3-5 targeted grep searches.

  Report findings as labeled sections with file paths and line numbers:
  ### Interactor pattern examples
  ### Existing policies
  ### Callbacks on ChangedModel
  ### Related factories"
})
```

## Dispatch a Haiku agent for simple lookups

```
Agent({
  model: "haiku",
  subagent_type: "Explore",
  prompt: "Find all files matching **/policies/*_policy.rb and list their paths"
})
```

## Parallelize independent searches

Dispatch multiple agents in a single message when gathering different categories of context.

## Output

Set `{CODEBASE_CONTEXT}` to the **combined output** from agents, labelled by section:

```
### Interactor pattern examples
<agent output>

### Existing policies
<agent output>

### Callbacks on ChangedModel
<agent output>
```
