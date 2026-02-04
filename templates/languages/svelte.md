## Svelte 5 Specific Rules

### Run Svelte Autofixer on .svelte Changes

After ANY `.svelte` file change, run `mcp__svelte__svelte-autofixer`.

### Use Svelte 5 Runes

```svelte
<script lang="ts">
  interface Props {
    user: User | null
    onAction: (id: string) => void
  }

  let { user, onAction }: Props = $props()

  let isLoading = $state(false)
  let displayName = $derived(user?.name.toUpperCase() ?? 'Unknown')

  function handleAction() {
    isLoading = true
    onAction(user!.id)
  }
</script>

{#if !user}
  <span>No user</span>
{:else if isLoading}
  <span>Loading...</span>
{:else}
  <div>
    <h2>{displayName}</h2>
    <button onclick={handleAction}>Action</button>
  </div>
{/if}
```

### Svelte 5 Runes Quick Reference

| Rune | Purpose | Replaces |
|------|---------|----------|
| `$state(value)` | Reactive state | `let x = value` |
| `$derived(expr)` | Computed value | `$: x = expr` |
| `$effect(() => {})` | Side effects | `$: { ... }` |
| `$props()` | Component props | `export let` |
| `onclick` | Event handler | `on:click` |

### Prefer $derived over $effect

Use `$effect` sparingly - only for side effects. For computed values, always use `$derived`.

**Valid $effect uses:** DOM measurements after render, external library initialization, logging/analytics, subscriptions to external stores.
