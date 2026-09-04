## PHP/Laravel Rules

### Style

- Follow PSR-12 coding standards
- Use type declarations for parameters and return types
- Use `camelCase` for methods, `snake_case` for database columns
- Keep controllers thin, use service classes for business logic

```php
// Before
class UserController extends Controller
{
    public function getUser($id)
    {
        $user = User::find($id);
        if ($user == null) {
            return response()->json(['error' => 'Not found'], 404);
        } else {
            return response()->json($user);
        }
    }
}

// After
class UserController extends Controller
{
    public function show(int $id): JsonResponse
    {
        $user = User::find($id);

        if (!$user) {
            return response()->json(['error' => 'Not found'], 404);
        }

        return response()->json($user);
    }
}
```

### Best Practices

- Use dependency injection over facades when testing is important
- Use Form Requests for validation
- Use Resource classes for API responses
- Use Eloquent scopes for reusable query logic
- Queue long-running tasks

### Laravel Conventions

```php
// Form Request for validation
class StoreUserRequest extends FormRequest
{
    public function rules(): array
    {
        return [
            'email' => ['required', 'email', 'unique:users'],
            'name' => ['required', 'string', 'max:255'],
        ];
    }
}

// API Resource for response transformation
class UserResource extends JsonResource
{
    public function toArray(Request $request): array
    {
        return [
            'id' => $this->id,
            'name' => $this->name,
            'email' => $this->email,
            'created_at' => $this->created_at->toIso8601String(),
        ];
    }
}

// Controller using both
class UserController extends Controller
{
    public function store(StoreUserRequest $request): UserResource
    {
        $user = User::create($request->validated());
        return new UserResource($user);
    }
}
```

### Testing

```bash
php artisan test           # Run all tests
php artisan test --filter=UserTest  # Run specific test
./vendor/bin/phpstan analyse  # Static analysis
./vendor/bin/pint          # Code formatting (Laravel Pint)
```
