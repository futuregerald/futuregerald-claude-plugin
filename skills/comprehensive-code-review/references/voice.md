# Review Voice Guidelines

These rules apply to every finding, summary, and section you write.

## Principles

1. **Clarity over jargon.** Use plain language when it says the same thing. "Crashes if the array is empty" beats "exhibits undefined behavior when collection cardinality is zero."

2. **When jargon is required, explain it.** Some terms are precise and necessary (`NoMethodError`, `ActiveRecord::Rollback`, `context.fail!`). When you use one, add a short clause so the reader doesn't need to look it up: "`nil[:created_at]` raises a `NoMethodError` (Ruby's version of a null pointer crash)."

3. **Brevity.** Say it once, say it clearly, stop. Don't restate the same point in different words. Don't pad findings with filler like "It should be noted that..." or "It is worth mentioning that..."

4. **Details when they help.** Include code snippets, file paths, and step-by-step traces when they help the author understand the problem or fix it. Omit them when the point is already clear.

5. **Titles that say what's wrong.** "Missing logging when credit deduction rolls back" tells the author what to do. "CWE-778 — Insufficient observability on non-instrumented failure-recovery path" does not.

6. **Write for the author, not the auditor.** The person reading this is fixing the code. Tell them what's wrong, why it matters, and what to do — in that order.

## Anti-patterns

| Don't write | Write instead |
|------------|---------------|
| "exhibits undefined behavior when the receiver collection cardinality is zero" | "crashes if the array is empty" |
| "utilizes the non-bang variant of the interactor invocation API" | "`.call(context)` (non-bang) means failures are silently ignored" |
| "RSpec's execution model evaluates `before` hooks defined at the describe level prior to each example regardless of their lexical position" | "RSpec runs it for all examples regardless of placement" |
| "emit a structured log event with contextual metadata to enable downstream aggregation in APM tooling" | "log it so it shows up in Datadog" |
| "the absence of explicit documentation creates ambiguity for future maintainers regarding whether this represents a deliberate architectural decision" | "there's no comment saying this is intentional — future readers will wonder" |
