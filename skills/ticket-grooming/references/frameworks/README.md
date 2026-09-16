# Framework detection

Detect the framework **before** reading application code. The framework determines what the
runtime guarantees; findings that ignore it produce misdiagnoses that waste engineers' time and
erode trust in the notes.

## Detection table

| Signal in the repo | Framework | Read next |
|---|---|---|
| `Gemfile` containing `rails` | Ruby on Rails | [rails.md](rails.md) |
| `go.mod` | Go | [go.md](go.md) |
| `package.json` with `next`, `react`, `express`, or a TS config | JavaScript / TypeScript | [javascript.md](javascript.md) |
| `requirements.txt` / `pyproject.toml` with `django` | Django | *no rules file yet — see below* |
| anything else | unknown | *see below* |

**Read only the file for the framework you detected.** In a multi-repo ticket, read one per repo —
a Rails backend plus a React frontend means `rails.md` and `javascript.md`, not all three.

## When the framework has no rules file

Django is in the table above, and other frameworks will turn up, with no rules written for them
yet. That is a real state, not an error — and it must not silently become "no framework detected".

In that case:

1. Verify every framework-dependent claim against the framework's own source or documentation
   before asserting it. The general principle in every rules file applies: **understand what the
   language runtime and framework guarantee before claiming a bug or vulnerability exists.**
2. Say so explicitly in the notes: "Framework detected: Django. No rules file exists for it, so
   framework conventions were verified manually against the source."
3. If you learn something durable, it belongs in a new file here.

The one thing you must not do is proceed on habits borrowed from another framework. Rails
assumptions applied to Django produce confident nonsense.

## The rule every framework shares

**If you did not read the model, concern, or config, you do not know what it does.** "HIGH
confidence" without having read the source is a false claim, whatever the language.
