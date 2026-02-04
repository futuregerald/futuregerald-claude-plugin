## Python Rules

### Style

- Follow PEP 8 style guide
- Use type hints for function signatures
- Prefer list/dict/set comprehensions when readable
- Use `f-strings` for string formatting

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

### Best Practices

- Use context managers (`with`) for resource management
- Leverage `dataclasses` or `pydantic` for data structures
- Use `pathlib` over `os.path`
- Prefer `raise` over returning error codes
- Use `enumerate()` when you need index and value

### Testing

```bash
pytest                    # Run all tests
pytest -v                 # Verbose output
pytest --cov=src          # With coverage
mypy src/                 # Type checking
```
