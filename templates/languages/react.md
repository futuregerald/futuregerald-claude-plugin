## React-Specific Rules

### Component Structure

- Use functional components with hooks (no class components)
- Define explicit `Props` interface for all components
- Prefer named exports over default exports (except where framework requires default, e.g., Next.js pages)
- Keep components small and focused (< 100 lines)

```tsx
interface UserCardProps {
  user: User | null
  onAction: (id: string) => void
}

export function UserCard({ user, onAction }: UserCardProps) {
  const [isLoading, setIsLoading] = useState(false)

  if (!user) return <span>No user</span>
  if (isLoading) return <span>Loading...</span>

  return (
    <div>
      <h2>{user.name}</h2>
      <button onClick={() => onAction(user.id)}>Action</button>
    </div>
  )
}
```

### Hooks Best Practices

- Use `useMemo` and `useCallback` only when necessary (measure first)
- Extract custom hooks for reusable logic
- Prefer controlled components over uncontrolled

### Error Handling

ALL API calls MUST handle errors gracefully with user-friendly messages.

```typescript
// Assumes showError is provided by your app's toast/notification system
async function apiCall() {
  try {
    const response = await fetch('/api/endpoint')
    if (!response.ok) throw new Error('Request failed')
    return await response.json()
  } catch (error) {
    showError('Something went wrong. Please try again.')
  }
}
```
