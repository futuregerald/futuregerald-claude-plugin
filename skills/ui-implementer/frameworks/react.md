# React Implementation Instructions

**Version:** React 19 with TypeScript

## Component Patterns

- **Functional components only** -- no class components
- **Named exports** -- `export function UserCard() {}`, not `export default`
- **File extension:** `.tsx` for components, `.ts` for utilities/types
- **Co-locate types** with the component file. Only extract to a shared `types.ts` if used by 3+ files.

```tsx
interface UserCardProps {
  name: string;
  email: string;
  avatarUrl?: string;
  onEdit?: (id: string) => void;
}

export function UserCard({ name, email, avatarUrl, onEdit }: UserCardProps) {
  return (
    <article className="...">
      {avatarUrl && <img src={avatarUrl} alt={`${name}'s avatar`} />}
      <h3>{name}</h3>
      <p>{email}</p>
      {onEdit && <button onClick={() => onEdit(name)}>Edit</button>}
    </article>
  );
}
```

## React 19 APIs

Use these when appropriate:

- **`use()`** -- read promises and context directly in render. Replaces `useContext()` for context reading and enables Suspense-based data fetching.
- **`useActionState()`** -- manage form action state (pending, result, error). Replaces manual `useState` + `useTransition` for form handling.
- **`useFormStatus()`** -- read pending state of a parent `<form>` from within a child component.
- **`useOptimistic()`** -- optimistic UI updates during async operations.
- **`ref` as prop** -- React 19 passes `ref` as a regular prop. Do NOT use `forwardRef()` -- it is deprecated in React 19.
- **`<form action={fn}>`** -- forms can accept async functions as the `action` prop.

## Server Components (Next.js / RSC)

If the project uses Next.js App Router or another RSC framework:

- Components are **Server Components by default** -- they run on the server and cannot use hooks, event handlers, or browser APIs
- Add `"use client"` at the top of the file for components that need interactivity (onClick, useState, useEffect, etc.)
- Keep Server Components for data fetching and static rendering
- Move interactive parts to small `"use client"` leaf components

If the project does **not** use RSC, ignore this section.

## State & Data Flow

- **Props** for parent-to-child data
- **`useState`** for local component state
- **`useReducer`** for complex state logic
- **`useContext`** (or `use(Context)` in React 19) for cross-tree shared state
- **Check for existing state management** (Redux, Zustand, Jotai, etc.) in the project and use it if present

## Event Handling

React uses JSX callback props for events -- no special event system:

```tsx
<button onClick={(e: React.MouseEvent<HTMLButtonElement>) => handleClick(e)}>
  Click
</button>

<input onChange={(e: React.ChangeEvent<HTMLInputElement>) => setValue(e.target.value)} />
```

**React 19 form actions:** Use `<form action={fn}>` for async form submissions instead of `onSubmit` + `preventDefault`:

```tsx
async function saveUser(formData: FormData) {
  await api.save(Object.fromEntries(formData));
}

<form action={saveUser}>
  <input name="email" />
  <button type="submit">Save</button>
</form>
```

Common event types: `React.MouseEvent`, `React.ChangeEvent`, `React.FormEvent`, `React.KeyboardEvent`, `React.FocusEvent`.

## Content Composition

React uses `children` as a prop for content composition:

```tsx
interface CardProps {
  children: React.ReactNode;
  header?: React.ReactNode;
}

export function Card({ children, header }: CardProps) {
  return (
    <div>
      {header && <div className="card-header">{header}</div>}
      <div className="card-body">{children}</div>
    </div>
  );
}

// Usage
<Card header={<h2>Title</h2>}>
  <p>Body content</p>
</Card>
```

- Use `React.ReactNode` for any renderable content (strings, elements, arrays, fragments)
- Use `React.ReactElement` when you specifically need a JSX element (rare)
- Named content areas are just additional `ReactNode` props (e.g., `header`, `footer`, `sidebar`)

## Hooks Rules

- Only call hooks at the top level (not inside conditions, loops, or nested functions)
- Custom hooks start with `use` prefix
- Extract repeated logic into custom hooks

## TypeScript

- Define `Props` interfaces for every component
- Use `React.ReactNode` for children, `React.CSSProperties` for inline styles
- Use discriminated unions for variant props:
  ```tsx
  type ButtonProps =
    | { variant: "link"; href: string }
    | { variant: "button"; onClick: () => void };
  ```

## Typecheck Command

```bash
npx tsc --noEmit
```

## Testing

- **vitest** + **@testing-library/react**
- Test user interactions, not implementation details
- Use `screen.getByRole()` over `getByTestId()` for accessible queries
