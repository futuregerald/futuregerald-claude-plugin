# Go investigation rules

Loaded by the investigation sub-agent after Phase 0 detects the framework.
See [README.md](README.md) for the detection table.

- Go is statically typed. Integer values CANNOT contain SQL injection when used with `fmt.Sprintf("%d", val)` or direct interpolation.
- String values from user input CAN be dangerous in `fmt.Sprintf` SQL construction — flag these.
- `database/sql` with `?` placeholders is parameterized.

## This file is a stub

These notes are thin compared with `rails.md`. Treat that as "rules not yet written", not as
"nothing else matters" — follow the discipline in [README.md](README.md) for a framework without a
rules file: verify every framework-dependent claim against the framework's own source before
asserting it, and say in the notes that you did.

If you learn something durable about this framework, add it here.
