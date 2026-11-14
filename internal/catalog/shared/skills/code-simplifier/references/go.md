# Go Best Practices

- Follow `gofmt` and `go vet` conventions
- Use short variable names for short scopes, descriptive for longer
- Return early to reduce nesting
- Use named return values only when they add clarity
- Prefer composition over inheritance (embedding)
- Handle errors explicitly, don't ignore them
- Use `defer` for cleanup
- Keep interfaces small (1-3 methods)
- Accept interfaces, return concrete types
- Use table-driven tests

```go
// Before
func processItems(items []Item) ([]Result, error) {
    results := []Result{}
    for i := 0; i < len(items); i++ {
        item := items[i]
        if item.Valid {
            result, err := process(item)
            if err != nil {
                return nil, err
            }
            results = append(results, result)
        }
    }
    return results, nil
}

// After
func processItems(items []Item) ([]Result, error) {
    var results []Result
    for _, item := range items {
        if !item.Valid {
            continue
        }
        result, err := process(item)
        if err != nil {
            return nil, err
        }
        results = append(results, result)
    }
    return results, nil
}
```
