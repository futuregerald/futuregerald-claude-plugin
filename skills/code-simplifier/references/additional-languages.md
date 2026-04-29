# Additional Language & Framework Best Practices

Reference material for the code-simplifier skill. These sections cover languages and frameworks beyond Cobalt's primary stack (Go, Ruby/Rails, JS/TS, React).

---

## Java

- Follow Java naming conventions (camelCase methods, PascalCase classes)
- Use meaningful names over comments
- Prefer composition over inheritance
- Use `Optional` instead of null for return types
- Leverage streams for collection operations (when readable)
- Use `var` for local variables when type is obvious
- Keep methods short (< 20 lines ideally)
- Use builder pattern for complex object construction
- Prefer immutability (`final` fields, unmodifiable collections)
- Use dependency injection

```java
// Before
public String processUser(User user) {
    if (user != null) {
        if (user.isActive()) {
            return user.getName().toUpperCase();
        } else {
            return "inactive";
        }
    } else {
        return "unknown";
    }
}

// After
public String processUser(User user) {
    if (user == null) return "unknown";
    if (!user.isActive()) return "inactive";
    return user.getName().toUpperCase();
}

// Or with Optional
public String processUser(Optional<User> user) {
    return user
        .filter(User::isActive)
        .map(u -> u.getName().toUpperCase())
        .orElse(user.isPresent() ? "inactive" : "unknown");
}
```

---

## PHP

- Follow PSR-12 coding standard
- Use type declarations for parameters and return types
- Prefer `declare(strict_types=1)` at file top
- Use null coalescing (`??`) and null safe operator (`?->`)
- Prefer early returns to reduce nesting
- Use constructor property promotion (PHP 8+)
- Prefer named arguments for clarity when many parameters
- Use match expressions over switch when appropriate
- Leverage enums instead of class constants (PHP 8.1+)
- Use attributes instead of docblock annotations where possible

```php
// Before
class UserService {
    private $repository;
    private $logger;

    public function __construct($repository, $logger) {
        $this->repository = $repository;
        $this->logger = $logger;
    }

    public function getUser($id) {
        if ($id !== null) {
            $user = $this->repository->find($id);
            if ($user !== null) {
                if ($user->isActive()) {
                    return $user;
                } else {
                    return null;
                }
            } else {
                return null;
            }
        } else {
            return null;
        }
    }
}

// After
declare(strict_types=1);

class UserService {
    public function __construct(
        private readonly UserRepository $repository,
        private readonly LoggerInterface $logger,
    ) {}

    public function getUser(?int $id): ?User {
        if ($id === null) return null;

        $user = $this->repository->find($id);

        if (!$user?->isActive()) return null;

        return $user;
    }
}
```

### Laravel-specific

- Use Eloquent scopes for reusable query logic
- Prefer `firstOrFail()` over `find()` + null check in controllers
- Use form requests for validation
- Leverage Laravel collections instead of array functions
- Use dependency injection over facades in classes
- Keep controllers thin - use actions/services for business logic

```php
// Before (Laravel)
public function show($id) {
    $user = User::find($id);
    if ($user == null) {
        abort(404);
    }
    $posts = Post::where('user_id', $user->id)
        ->where('published', true)
        ->orderBy('created_at', 'desc')
        ->get();
    return view('user.show', ['user' => $user, 'posts' => $posts]);
}

// After (Laravel)
public function show(int $id): View {
    $user = User::with(['posts' => fn($q) => $q->published()->latest()])
        ->findOrFail($id);

    return view('user.show', compact('user'));
}
```

---

## Python

- Follow PEP 8 style guide
- Use type hints for function signatures
- Prefer list/dict/set comprehensions when readable
- Use `f-strings` for string formatting
- Use context managers (`with`) for resource management
- Leverage `dataclasses` or `pydantic` for data structures
- Use `pathlib` over `os.path`
- Prefer `raise` over returning error codes
- Use `enumerate()` when you need index and value

```python
# Before
def process_users(users):
    results = []
    for i in range(len(users)):
        user = users[i]
        if user is not None:
            if user.active == True:
                results.append(user.name.upper())
    return results

# After
def process_users(users: list[User]) -> list[str]:
    return [
        user.name.upper()
        for user in users
        if user and user.active
    ]
```

---

## Svelte (5)

- Use Svelte 5 runes (`$state`, `$derived`, `$effect`, `$props`)
- Define explicit `Props` interface with `$props()`
- Prefer `$derived` over `$effect` for computed values
- Use `$effect` sparingly - only for side effects
- Keep components small and focused
- Extract reusable logic into `.svelte.ts` files
- Use `{#snippet}` for reusable template fragments
- Prefer `bind:` for two-way binding when appropriate
- Use `use:` actions for DOM manipulation
- Avoid `$effect` for things that can be `$derived`

```svelte
<!-- Before (Svelte 4 style) -->
<script lang="ts">
  export let user: User | null = null;
  export let onAction: (id: string) => void;

  let isLoading = false;
  let displayName: string;

  $: displayName = user ? user.name.toUpperCase() : 'Unknown';
  $: if (user) {
    console.log('User changed:', user.id);
  }
</script>

{#if user}
  {#if isLoading}
    <span>Loading...</span>
  {:else}
    <div>
      <h2>{displayName}</h2>
      <button on:click={() => { isLoading = true; onAction(user.id); }}>
        Action
      </button>
    </div>
  {/if}
{:else}
  <span>No user</span>
{/if}

<!-- After (Svelte 5 style) -->
<script lang="ts">
  interface Props {
    user: User | null
    onAction: (id: string) => void
  }

  let { user, onAction }: Props = $props()

  let isLoading = $state(false)
  let displayName = $derived(user?.name.toUpperCase() ?? 'Unknown')

  $effect(() => {
    if (user) console.log('User changed:', user.id)
  })

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

**Svelte 5 Runes Quick Reference:**

- `$state(value)` - Reactive state (replaces `let x = value`)
- `$derived(expr)` - Computed value (replaces `$: x = expr`)
- `$effect(() => {})` - Side effects (replaces `$: { ... }`)
- `$props()` - Component props (replaces `export let`)
- `$bindable()` - Two-way bindable props
- `onclick` not `on:click` - New event syntax
