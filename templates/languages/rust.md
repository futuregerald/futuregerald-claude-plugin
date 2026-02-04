## Rust Rules

### Style

- Follow `rustfmt` conventions
- Use `clippy` for additional linting
- Prefer `snake_case` for functions/variables, `CamelCase` for types
- Keep functions small and focused

```rust
// Before
fn process_items(items: Vec<Item>) -> Result<Vec<Result>, Error> {
    let mut results: Vec<Result> = Vec::new();
    for i in 0..items.len() {
        let item = &items[i];
        if item.valid == true {
            match process(&item) {
                Ok(r) => results.push(r),
                Err(e) => return Err(e),
            }
        }
    }
    return Ok(results);
}

// After
fn process_items(items: &[Item]) -> Result<Vec<ProcessedItem>, Error> {
    items
        .iter()
        .filter(|item| item.valid)
        .map(process)
        .collect()
}
```

### Best Practices

- Prefer `&str` over `String` for function parameters when possible
- Use `?` operator for error propagation
- Prefer iterators over explicit loops
- Use `Option` and `Result` instead of null/exceptions
- Derive common traits: `Debug`, `Clone`, `PartialEq` when appropriate

### Error Handling

```rust
use thiserror::Error;

#[derive(Error, Debug)]
pub enum AppError {
    #[error("Not found: {0}")]
    NotFound(String),
    #[error("Database error: {0}")]
    Database(#[from] sqlx::Error),
    #[error("Invalid input: {0}")]
    Validation(String),
}

fn find_user(id: &str) -> Result<User, AppError> {
    users.get(id).ok_or_else(|| AppError::NotFound(id.to_string()))
}
```

### Testing

```bash
cargo test                # Run all tests
cargo test -- --nocapture # With output
cargo clippy              # Linting
cargo fmt --check         # Format check
```
