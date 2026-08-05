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

## Gem / Library Source Constraint

**Do NOT search for installed gem source files on the CI runner.** Common gems
(interactor, pundit, cancancan, active_model_serializers, devise, sidekiq,
dry-rb, etc.) are installed via bundler but their source paths are not
reliably locatable in the CI environment. Attempting to find them wastes
many turns on fruitless `find`, `bundle show`, and `gem which` calls.

Instead:
- **Describe gem behavior from your training knowledge** — you know how these
  gems work. State your understanding and flag it as "based on standard gem
  behavior" if relevant to a finding.
- **Check the Gemfile / Gemfile.lock** for version constraints if version-specific
  behavior matters.
- **Read the app's own wrapper/base classes** (e.g., `ApplicationInteractor`,
  `ApplicationPolicy`) to understand how the app customizes gem behavior.

**Never:** run `find / -iname`, `bundle show`, `gem which`, or fetch gem source
from GitHub to understand how a standard gem works. This is a hard constraint.
