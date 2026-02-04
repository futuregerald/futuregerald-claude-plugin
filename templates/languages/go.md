## Go Rules

### Style

- Follow `gofmt` and `go vet` conventions
- Use short variable names for short scopes
- Return early to reduce nesting
- Handle errors explicitly, don't ignore them

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

### Best Practices

- Use `defer` for cleanup
- Keep interfaces small (1-3 methods)
- Accept interfaces, return concrete types
- Use table-driven tests
- Prefer composition over inheritance (embedding)

### Testing

```bash
go test ./...             # Run all tests
go test -v ./...          # Verbose
go test -cover ./...      # With coverage
go vet ./...              # Static analysis
```
