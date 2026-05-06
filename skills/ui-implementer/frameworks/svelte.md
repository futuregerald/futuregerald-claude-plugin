# Svelte 5 Implementation Instructions

**Version:** Svelte 5 with Runes syntax and TypeScript

**CRITICAL:** Svelte 5 uses **runes** -- a new reactivity model. Do NOT use Svelte 4 patterns (`$:` reactive declarations, `writable`/`readable` stores, `createEventDispatcher`, `<slot>`). These are legacy and will not work correctly in Svelte 5.

## Component Structure

```svelte
<script lang="ts">
  interface Props {
    name: string
    email: string
    avatarUrl?: string
    onEdit?: (id: string) => void
  }

  let { name, email, avatarUrl, onEdit }: Props = $props()

  let initials = $derived(
    name.split(' ').map(n => n[0]).join('')
  )
</script>

<article class="...">
  {#if avatarUrl}
    <img src={avatarUrl} alt={`${name}'s avatar`} />
  {/if}
  <h3>{name}</h3>
  <p>{email}</p>
  {#if onEdit}
    <button onclick={() => onEdit(name)}>Edit</button>
  {/if}
</article>

<style>
  /* Scoped by default -- or use Tailwind classes in markup */
</style>
```

## File Conventions

- **File extension:** `.svelte` for components, `.svelte.ts` for shared reactive logic (rune-aware modules)
- **PascalCase** file names for components: `UserCard.svelte`
- SvelteKit routes: `+page.svelte`, `+layout.svelte`, `+page.server.ts`
- Co-locate component-specific logic in `.svelte.ts` files alongside the component

## Svelte 5 Runes

These are the core reactivity primitives. Use them instead of Svelte 4 patterns:

| Rune | Purpose | Replaces (Svelte 4) |
|------|---------|---------------------|
| `$state(value)` | Reactive state | `let x = value` with `$:` |
| `$state.raw(value)` | Non-deep reactive state (for large objects) | N/A |
| `$derived(expr)` | Computed value | `$: x = expr` |
| `$derived.by(() => { ... })` | Computed with complex logic | `$: { ... }` |
| `$effect(() => { ... })` | Side effect (runs after DOM update) | `$: { sideEffect() }` |
| `$effect.pre(() => { ... })` | Side effect (runs before DOM update) | `beforeUpdate()` |
| `$props()` | Declare component props | `export let prop` |
| `$bindable()` | Props that support `bind:` | `export let prop` (all were bindable) |
| `$inspect(value)` | Debug logging (dev only, stripped in prod) | `$: console.log(value)` |

### State

```ts
let count = $state(0)                          // reactive primitive
let items = $state<string[]>([])               // reactive array
let user = $state<User>({ name: '', age: 0 })  // reactive object (deep)
let largeData = $state.raw(bigObject)           // shallow reactivity
```

### Derived

```ts
let doubled = $derived(count * 2)

let filtered = $derived.by(() => {
  // Complex logic with early returns, loops, etc.
  if (!query) return items
  return items.filter(i => i.includes(query))
})
```

### Effects

```ts
$effect(() => {
  // Runs when any referenced $state/$derived changes
  document.title = `Count: ${count}`

  // Optional cleanup (returned function runs before next effect or on destroy)
  return () => { /* cleanup */ }
})
```

### Props

```ts
// Basic typed props with defaults
let { name, count = 0, onUpdate }: Props = $props()

// Bindable prop (parent can use bind:value)
let { value = $bindable('') }: { value: string } = $props()

// Rest props (spread to element)
let { class: className, ...rest }: HTMLAttributes<'div'> & { class?: string } = $props()
```

## SvelteKit (if applicable)

If the project uses SvelteKit:

- **`load` functions**: Use `+page.ts` (client) or `+page.server.ts` (server-only) to fetch data before rendering. Return data as props to `+page.svelte`.
- **Form actions**: Use `+page.server.ts` `actions` for server-side form handling. Enhance with `use:enhance` for progressive enhancement.
- **`$app` modules**: Import from `$app/navigation` (`goto`, `invalidate`), `$app/stores` (`page`, `navigating`), `$app/environment` (`browser`, `dev`)
- **Layouts**: `+layout.svelte` for shared UI, `+layout.ts`/`+layout.server.ts` for shared data loading
- **Error handling**: `+error.svelte` for custom error pages, `error()` helper from `@sveltejs/kit`

If the project does **not** use SvelteKit, ignore this section.

## Event Handling

Svelte 5 uses **callback props** instead of `createEventDispatcher`:

```svelte
<!-- Child component -->
<script lang="ts">
  let { onClick }: { onClick?: (item: string) => void } = $props()
</script>
<button onclick={() => onClick?.('clicked')}>Click</button>

<!-- Parent -->
<Child onClick={(item) => console.log(item)} />
```

**DOM events** use lowercase `on` prefix directly: `onclick`, `onkeydown`, `onfocus` (not `on:click`).

## Content Composition (Snippets, not Slots)

Svelte 5 replaces `<slot>` with **snippets** and **`{@render}`**:

```svelte
<!-- Component with children (default content) -->
<script lang="ts">
  import type { Snippet } from 'svelte'

  let { children, header }: {
    children: Snippet
    header?: Snippet<[{ title: string }]>
  } = $props()
</script>

<div>
  {#if header}
    {@render header({ title: 'Hello' })}
  {/if}
  {@render children()}
</div>

<!-- Parent usage -->
<Card>
  {#snippet header({ title })}
    <h2>{title}</h2>
  {/snippet}
  <p>This is the default content (children).</p>
</Card>
```

## Styling

- `<style>` blocks are **scoped by default** -- styles only apply to the component's markup
- Use `:global()` to escape scoping when needed
- Tailwind classes work directly in markup if the project uses Tailwind
- Dynamic classes: `class:active={isActive}` or `class={isActive ? 'active' : ''}`

## TypeScript

- Use `lang="ts"` in `<script>` tags
- Props typed via interface + `$props()`
- Event callbacks typed as function props
- Snippet types imported from `'svelte'`: `Snippet`, `Snippet<[ArgType]>`
- Component types: `import type { Component } from 'svelte'`

## Typecheck Command

```bash
npx svelte-check --tsconfig ./tsconfig.json
```

## Testing

- **vitest** + **@testing-library/svelte**
- Render with `render(Component, { props: { ... } })`
- Use `screen.getByRole()` for accessible queries
- Test callback props: pass a `vi.fn()` and assert it was called
