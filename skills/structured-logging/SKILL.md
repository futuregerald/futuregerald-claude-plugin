---
name: structured-logging
description: Use when writing or modifying Ruby code in a Rails app that uses SemanticLogger. Enforces structured logging via SemanticLogger + Datadog so logs are queryable, dashboardable, and debuggable. Triggers on new interactors, jobs, services, controllers, error handling, and any business logic that should be observable.
---

# Structured Logging

## Overview

This applies to Rails apps using `rails_semantic_logger` with Datadog. Keyword arguments passed to `Rails.logger` become `metadata` fields in Datadog — filterable, facetable, and alertable. String interpolation buries data in unstructured text that Datadog cannot index.

**Core rule:** Every log statement must use the two-argument form: a short event name + keyword arguments for all variable data.

## When to Use

- Adding error handling or rescue blocks
- Implementing business logic with outcomes worth observing (assignments, state transitions, skips, retries)
- Writing jobs or services where success/failure tracking aids debugging
- Modifying existing code that uses string-interpolated logging (upgrade it)

When writing or reviewing code in such an app: add structured logging where it aids debugging or observability — error rescues, business decisions, job outcomes, and external service calls. Don't add logging to every method by default; log when the information would help someone debugging a production issue. Flag any string-interpolated logs as needing upgrade. Verify error rescues include `error_class` and `error_message` fields.

## The Pattern

```ruby
Rails.logger.info(
  'event_name',
  key: value,
  another_key: another_value,
)
```

- **First argument**: Short, searchable event name (string) — use `domain.action` or `noun_verb_past` style
- **Remaining arguments**: Keyword args that become `metadata` in Datadog

### Examples

**Business event:**

```ruby
Rails.logger.info(
  'tpm_auto_assigned',
  assignment_source: 'org_default',
  selected_tpm: org.tpm.email,
  engagement_token: engagement.token,
  org_id: org.public_id.to_s,
)
```

**Error with structured context:**

```ruby
Rails.logger.error(
  'credential_validation.failed',
  credential_id: credential.public_id,
  error_class: e.class.name,
  error_message: e.message,
)
```

**Anti-pattern — string interpolation (unqueryable in Datadog):**

```ruby
# BAD
Rails.logger.info("Skipping schedule #{@schedule.public_id} - customer skipped next run")

# GOOD
Rails.logger.info('schedule.skipped', schedule_id: @schedule.public_id, reason: 'customer_skipped_next_run')
```

## Event Naming

| Pattern | Example | Use for |
|---------|---------|---------|
| `domain.action` | `sidekiq.health` | Metrics/health checks |
| `noun_verb_past` | `tpm_auto_assigned` | Business events |
| `LOG_TAG + description` | `[JobName] Succeeded` | Jobs with multiple log points |

## Log Levels

| Level | When |
|-------|------|
| `info` | Business events, successful operations, health checks |
| `warn` | Degraded but recoverable, unexpected-but-handled conditions, authorization failures |
| `error` | Failures that need attention, unrecoverable errors |

## Reusable Patterns

### LOG_TAG + Shared Payload

For jobs/services with multiple log points, use a constant tag and a shared payload method:

```ruby
class SomeJob < ApplicationJob
  LOG_TAG = '[SomeJob]'

  def perform(token)
    @record = Record.find_by(token: token)
    return Rails.logger.warn("#{LOG_TAG} Record not found", token:) unless @record

    result = SomeInteractor.call(record: @record)
    if result.success?
      Rails.logger.info("#{LOG_TAG} Succeeded", **log_payload, count: result.items.size)
    else
      Rails.logger.error("#{LOG_TAG} Failed", **log_payload, error: result.error)
    end
  end

  private

  def log_payload = { record_id: @record.public_id, record_token: @record.token }
end
```

### Tagged Logging for Scoped Context

Request-scoped context (url, ip, path, request_id, dd trace) is already added via `config.log_tags` in `application.rb`. You do not need to re-add these fields manually.

Use `Rails.logger.tagged` to push context down to all logs within a block — useful at the controller level, in organizers, or for batch operations:

```ruby
# Controller-level tagging — all logs from this action (including interactors/services) get the tag
around_action :tag_logs

def tag_logs
  Rails.logger.tagged(controller: self.class.name, action: action_name) { yield }
end

# Organizer/interactor wrapping
Rails.logger.tagged('Auth0') do
  # all logs inside get the Auth0 tag
  Auth0::SyncUser.call(user: user)
end

# Batch operations
Rails.logger.tagged(org_id: org.public_id) do
  Rails.logger.info('batch_processing_started', batch_size: items.count)
end
```

## Where to Add Logging

**Log when it aids debugging:** business decisions (assignments, state transitions, skips), job success/failure, external service calls and outcomes, authorization denials, error recovery paths, scheduled/cron operations.

**Don't log:** routine happy-path reads, simple CRUD without business logic, sensitive data (passwords, tokens, PII beyond public_ids/emails), data already in request context (url, ip, request_id — these come from `log_tags`). Not every interactor or method needs logging — add it where the information would help someone investigating a production issue.

## Testing Structured Logs

Only test log output when the log is the sole observable side effect of a code branch (e.g., an early return that only logs). Don't add specs for routine info logs — they create brittle tests without meaningful coverage.

```ruby
# When a log is the only way to verify a branch was reached:
expect(Rails.logger).to receive(:warn).with(
  'record_not_found',
  hash_including(token: 'abc123'),
)
```

## Quick Reference

| Want to... | Do this |
|------------|---------|
| Log a business event | `Rails.logger.info('event_name', field: value, ...)` |
| Log a warning | `Rails.logger.warn('event_name', field: value, ...)` |
| Log an error with exception | `Rails.logger.error('event_name', error_class: e.class.name, error_message: e.message, ...)` |
| Add scoped context | `Rails.logger.tagged(key: val) { ... }` |
| Reuse context across log points | Define `LOG_TAG` constant + shared payload method |
| Test log (only if sole branch indicator) | `expect(Rails.logger).to receive(:warn).with('event', hash_including(...))` |
