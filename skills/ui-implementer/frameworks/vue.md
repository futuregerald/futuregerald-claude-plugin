# Vue Implementation Instructions

**Version:** Vue 3.5+ with TypeScript and `<script setup>`

## Component Structure

Use **Single-File Components (SFC)** with the Composition API exclusively. No Options API.

```vue
<script setup lang="ts">
import { ref, computed } from 'vue'

interface Props {
  name: string
  email: string
  avatarUrl?: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  edit: [id: string]
}>()

const initials = computed(() =>
  props.name.split(' ').map(n => n[0]).join('')
)
</script>

<template>
  <article class="...">
    <img v-if="props.avatarUrl" :src="props.avatarUrl" :alt="`${props.name}'s avatar`" />
    <h3>{{ props.name }}</h3>
    <p>{{ props.email }}</p>
    <button @click="emit('edit', props.name)">Edit</button>
  </article>
</template>

<style scoped>
/* Scoped styles here -- or use Tailwind classes in template */
</style>
```

**Section order:** `<script setup>` first, then `<template>`, then `<style scoped>`.

## File Conventions

- **File extension:** `.vue` for components, `.ts` for composables/utilities
- **PascalCase** file names for components: `UserCard.vue`
- **camelCase** with `use` prefix for composables: `useUserData.ts`
- **Named exports** for composables, default export is implicit for SFCs

## Vue 3.5+ APIs

Use these when appropriate:

- **`defineProps<T>()`** -- type-only props declaration. Supports defaults via `withDefaults()`:
  ```ts
  const props = withDefaults(defineProps<Props>(), {
    avatarUrl: '/default-avatar.png'
  })
  ```
- **`defineEmits<T>()`** -- typed event emissions with tuple syntax
- **`defineModel()`** -- two-way binding shorthand (Vue 3.4+). Replaces manual `modelValue` prop + `update:modelValue` emit:
  ```ts
  const modelValue = defineModel<string>()       // v-model
  const title = defineModel<string>('title')     // v-model:title
  ```
- **`useTemplateRef()`** -- typed template refs (Vue 3.5+). Replaces `ref<HTMLElement | null>(null)` with template `ref="name"`:
  ```ts
  const inputRef = useTemplateRef<HTMLInputElement>('input')
  ```
- **`defineExpose()`** -- explicitly expose properties to parent refs
- **`defineSlots<T>()`** -- typed slot definitions for template type-checking

## Auto-Imports

Many Vue projects use `unplugin-auto-import` and `unplugin-vue-components`:
- Check `vite.config.ts` or `nuxt.config.ts` for these plugins
- If present: `ref`, `computed`, `watch`, `onMounted`, etc. are auto-imported -- do **not** add explicit import statements
- If not present: add explicit imports from `'vue'`

## Event Handling

Vue uses `v-on` (shorthand `@`) for DOM event binding:

```vue
<button @click="handleClick">Click</button>
<button @click="(e: MouseEvent) => handleClick(e)">Click</button>
<input @input="(e: Event) => setValue((e.target as HTMLInputElement).value)" />
```

**Event modifiers** -- append to the event name with `.`:
- `.prevent` -- `event.preventDefault()`
- `.stop` -- `event.stopPropagation()`
- `.self` -- only trigger if event target is the element itself
- `.once` -- trigger at most once
- `.passive` -- passive event listener

```vue
<form @submit.prevent="onSubmit">...</form>
<div @click.self="onDivClick">...</div>
```

**Key modifiers** for keyboard events:
```vue
<input @keydown.enter="submit" />
<input @keydown.esc="cancel" />
<input @keydown.ctrl.s.prevent="save" />
```

**Component events** use `defineEmits`:
```vue
<button @click="emit('save', formData)">Save</button>
```

## Nuxt 3 (if applicable)

If the project uses Nuxt 3:

- **Auto-imports**: `useAsyncData`, `useFetch`, `useState`, `useRoute`, `useRouter`, `navigateTo` are auto-imported -- do not add explicit imports
- **`<ClientOnly>`**: Wrap components that use browser-only APIs (e.g., `window`, `document`, `localStorage`)
- **`definePageMeta()`**: Set page-level configuration (layout, middleware, auth)
- **Data fetching**: Use `useFetch()` for API calls (handles SSR + client), `useAsyncData()` for custom async logic
- **Server routes**: API endpoints in `server/api/` directory

If the project does **not** use Nuxt, ignore this section.

## Reactivity

- **`ref()`** for primitive values and single references
- **`reactive()`** for objects (use sparingly -- `ref()` is usually clearer)
- **`computed()`** for derived values
- **`watch()` / `watchEffect()`** for side effects
- **`shallowRef()`** for large objects where deep reactivity is unnecessary

## Slots

```vue
<!-- Default slot -->
<slot />

<!-- Named slot -->
<slot name="header" />

<!-- Scoped slot -->
<slot name="item" :item="item" :index="i" />
```

## TypeScript

- Props via `defineProps<T>()` -- no runtime declaration needed
- Emits via `defineEmits<T>()` with labeled tuple syntax
- Use `ComponentInstance` type for template ref typing when referencing child components
- Global component types: augment `@vue/runtime-core` module if needed

## Typecheck Command

```bash
vue-tsc --noEmit
```

## Testing

- **vitest** + **@vue/test-utils**
- Mount with `mount(Component, { props: { ... } })`
- Use `wrapper.find('[role="button"]')` for accessible queries
- Test emitted events: `wrapper.emitted('edit')`
