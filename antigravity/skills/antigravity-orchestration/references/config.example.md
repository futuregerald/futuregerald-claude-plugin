# Antigravity Orchestration Local Configuration (example)

Copy this file to `references/config.md` (git-ignored) and fill in your local paths and organization names.

```yaml
review_lens:
  orgs:
    - [your-work-org]
  exclude_orgs:
    - [your-personal-org]
  analyzer_dir: [~/path/to/review-analyzer]
  binary: review-lens-darwin-arm64
  db: reviews.db
  venv_bin: [~/.venvs/review-lens/bin]
```
