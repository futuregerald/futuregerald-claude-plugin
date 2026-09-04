# Ruby/Rails Best Practices

- Follow Ruby style guide (2 spaces, snake_case)
- Use guard clauses for early returns
- Prefer `&.` (safe navigation) over explicit nil checks
- Use symbols over strings for hash keys
- Leverage Ruby's expressiveness without being cryptic
- Use `%w[]` and `%i[]` for word/symbol arrays
- Prefer `each` over `for`
- Use `present?`, `blank?`, `presence` appropriately
- Keep controllers thin, models reasonable, use service objects
- Avoid N+1 queries - use `includes`, `preload`, `eager_load`

```ruby
# Before
def process_user(user)
  if user != nil
    if user.active == true
      return user.name.upcase
    else
      return "inactive"
    end
  else
    return "unknown"
  end
end

# After
def process_user(user)
  return "unknown" unless user
  return "inactive" unless user.active?

  user.name.upcase
end
```
