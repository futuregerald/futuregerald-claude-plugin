# React Best Practices

Also load [javascript-typescript.md](javascript-typescript.md) — React components follow JS/TS conventions plus these additional rules.

- Use functional components with hooks (no class components)
- Define explicit `Props` interface for all components
- Prefer named exports over default exports
- Use `useMemo` and `useCallback` only when necessary (measure first)
- Keep components small and focused (< 80 lines target, 120 hard limit)
- Extract custom hooks for reusable logic
- Use early returns for conditional rendering
- Avoid inline function definitions in JSX when possible
- Prefer controlled components over uncontrolled
- Use React.lazy() for code splitting large components

```tsx
// Before
const UserCard = (props: any) => {
  const [isLoading, setIsLoading] = useState(false)

  return (
    <div>
      {props.user ? (
        <div>
          {isLoading ? (
            <span>Loading...</span>
          ) : (
            <div>
              <h2>{props.user.name}</h2>
              <button
                onClick={() => {
                  setIsLoading(true)
                  props.onAction(props.user.id)
                }}
              >
                Action
              </button>
            </div>
          )}
        </div>
      ) : (
        <span>No user</span>
      )}
    </div>
  )
}

// After
interface UserCardProps {
  user: User | null
  onAction: (id: string) => void
}

export function UserCard({ user, onAction }: UserCardProps) {
  const [isLoading, setIsLoading] = useState(false)

  if (!user) return <span>No user</span>
  if (isLoading) return <span>Loading...</span>

  function handleAction() {
    setIsLoading(true)
    onAction(user.id)
  }

  return (
    <div>
      <h2>{user.name}</h2>
      <button onClick={handleAction}>Action</button>
    </div>
  )
}
```
