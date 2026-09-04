# Framework Detection

Detect the project framework BEFORE dispatching the investigation sub-agent. The framework determines what the runtime guarantees — findings that ignore framework behavior produce misdiagnoses that waste engineers' time and erode trust in grooming notes.

## Detection

Detect from project files:

| File | Framework |
|------|-----------|
| `Gemfile` with `rails` | Ruby on Rails (ActiveRecord, Interactors, Pundit, Packwerk) |
| `package.json` with `next` / `react` | Next.js / React |
| `package.json` with `express` | Express.js |
| `requirements.txt` / `pyproject.toml` with `django` | Django |
| `go.mod` | Go (statically typed) |

Set `{FRAMEWORK_CONTEXT}` to the relevant rules below. Pass to BOTH the investigation sub-agent (full rules) and the staff review sub-agent (compact checklist — see staff-review-prompt.md).

## Ruby on Rails Investigation Rules

Understand how Rails actually works before diagnosing root causes or suggesting fixes. Every claim about Rails behavior must be verified against the actual code — not inferred from method names or grep results.

**The cardinal rule: if you didn't read the model/concern/config, you don't know what it does. "HIGH confidence" without reading the source is a lie.**

### ActiveRecord association resolution (CRITICAL)

Rails `where(table: {key: value})` checks if `key` matches a `belongs_to` association name on the model BEFORE treating it as a literal column name. If `key` matches an association, Rails resolves it to the foreign key column automatically.

**Mandatory verification steps for any where-hash investigation:**
1. Read the MODEL file (`app/models/<table>.rb`) — find all `belongs_to` declarations
2. Check which association name resolves to which FK column
3. Association names DIFFER between repos (e.g. one app declares `belongs_to :owner` while another declares `belongs_to :assignee` — both resolving to the same FK column)
4. When counting "affected files," verify EACH file individually against its repo's model. Never batch-flag from grep results.

**For PG::UndefinedColumn errors in where hashes:**
The key is likely a WRONG ASSOCIATION NAME (not resolved by the model), not a dropped column. Check the model to distinguish.

**Anti-pattern: raw FK column names.** Don't recommend `where(orgs: {owner_id: user&.id})` when the Rails-idiomatic fix is `where(orgs: {owner: user})` using the correct association name.

> **Case study:** Grooming once claimed an association was referencing a "removed column" and flagged 8 files as buggy. In reality the name it expected simply wasn't the association name on that app's model, which declared a different one over the same FK. Most flagged files were correct in their own repo. The fix was a one-file association rename, not an 8-file rewrite. Root failure: never read the model's `belongs_to` definitions.

### ActiveRecord query building

- `where(column: value)` is parameterized — not injection
- `find_by(column: value)` is parameterized — not injection
- `pluck(:column)` returns typed Ruby values from the DB adapter
- For raw SQL with interpolation: only dangerous with STRING input, not integers (`.to_i` always returns Integer)
- `conn.quote(value)` is for string values in raw SQL. Using it on integers is unnecessary.
- `conn.quote_column_name` / `conn.quote_table_name` are for dynamic identifiers. Hardcoded string constants do NOT need quoting.

### Type safety in Ruby

- `.to_i` ALWAYS returns Integer. `"anything".to_i` -> `0`. An Integer interpolated into SQL can ONLY produce a decimal number. This is NOT injection.
- `.to_f` ALWAYS returns Float. Same type safety applies.
- `.to_s` on Integer/Float cannot produce SQL metacharacters.
- Only flag injection when STRINGS from user/external input are interpolated without parameterization.

### Callbacks and lifecycle hooks

Callbacks fire IMPLICITLY. They are the #1 source of "invisible" behavior that investigations miss or misattribute.

**Mandatory verification for any bug involving unexpected data state:**
1. Read the MODEL file and ALL included concerns — look for `before_save`, `after_save`, `before_validation`, `after_commit`, `after_create`, etc.
2. Check the ORDER callbacks fire: validation -> before_save -> SQL -> after_save -> (transaction commit) -> after_commit
3. `after_save` fires INSIDE the transaction. `after_commit` fires AFTER. Sidekiq jobs enqueued in `after_save` can execute before the transaction commits — leading to "record not found" errors. This is a known Rails gotcha. Don't flag it as a race condition in application code.
4. Callbacks in concerns are invisible in the model file — you MUST read every `include`'d module.

**Anti-pattern: blaming application code for callback behavior.** If data is being modified "unexpectedly," check callbacks before blaming the controller or interactor. Read the full callback chain.

### Concerns and module inclusion

Methods available on a model may come from `include`'d concerns, not the model file itself. A method that "doesn't exist" in the model may be defined in a concern.

**Mandatory verification when claiming a method is missing:**
1. Read the model file — find all `include` statements
2. Read each included concern
3. Check `ApplicationRecord` and any intermediate base classes
4. Only THEN claim a method doesn't exist

**Anti-pattern: grepping for a method name, not finding it in the model file, and claiming it's missing.** It's probably in a concern.

### Enums

`enum status: { draft: 0, active: 1, archived: 2 }` generates:
- Scopes: `Model.draft`, `Model.active`
- Predicates: `record.draft?`, `record.active?`
- Transitions: `record.active!` (saves immediately)
- The DB stores INTEGER values, not strings

**Misdiagnosis trap:** Seeing `status = 0` in DB and thinking data is corrupt. It's the enum's integer mapping — read the enum definition.

**Misdiagnosis trap:** Seeing `.where(status: 'draft')` and thinking it queries by string. ActiveRecord translates enum names to integers automatically.

### Default scopes

`default_scope { where(deleted_at: nil) }` silently adds a WHERE clause to EVERY query on that model. This is invisible in the calling code.

**Mandatory verification for "missing records" bugs:**
1. Check if the model has a `default_scope`
2. Check for `acts_as_paranoid`, `discard`, or `paranoia` gems — these add soft-delete default scopes
3. `unscoped` removes ALL scopes including default — verify if the code under investigation uses it

**Anti-pattern: claiming a query "filters out records incorrectly" without checking for default scopes.** The WHERE clause may not be in the code you're reading.

### STI (Single Table Inheritance)

When a model has a `type` column, Rails uses it to load different Ruby classes from the same table. `Event.all` only returns records where `type = 'Event'` (or subclasses).

**Misdiagnosis trap:** Claiming records are missing when querying a parent class — they may have a `type` value that routes to a different subclass. Always check the `type` column values in the schema.

### Polymorphic associations

`belongs_to :commentable, polymorphic: true` uses TWO columns: `commentable_type` (string) and `commentable_id` (integer).

**Misdiagnosis trap:** Seeing `commentable_id = 42` and assuming it references a specific table. The `_type` column determines which table. Always check BOTH columns.

### Delegation

`delegate :name, to: :org` makes `record.name` call `record.org.name`. The method appears on the model but is defined on the association target.

**Misdiagnosis trap:** Reading the model, seeing `record.name` called somewhere, grepping the model for `def name`, not finding it, and claiming it's undefined. Check `delegate` declarations.

### has_many dependent behavior

`has_many :findings, dependent: :destroy` means deleting a parent CASCADE-DELETES all children (via Rails, not DB). This fires callbacks on each child.

`has_many :findings, dependent: :delete_all` bypasses callbacks and does a single SQL DELETE.

**Misdiagnosis trap:** Investigating slow deletes or "unexpected side effects on delete" without checking the `dependent:` option. The cascade may be intentional, or the callback-firing variant may be causing the slowness.

### ActiveModel::Dirty (change tracking)

Rails tracks attribute changes on models:
- `saved_changes` — what changed in the LAST save (available in after_save/after_commit)
- `changes` — what's changed but NOT YET saved (available in before_save)
- `previous_changes` — deprecated alias for saved_changes

**Misdiagnosis trap:** Seeing a callback that checks `changes` in `after_save` and finding it empty. `changes` is cleared after save — use `saved_changes` instead.

### Interactor patterns

- `context.fail!` halts execution and rolls back in an `organize` chain
- Check `ApplicationInteractor` or base class for shared behavior
- Interactor failures propagate via `context.failure?` — verify the actual failure handling, don't assume it raises an exception

### Pundit authorization

- `authorize @resource` in controllers delegates to `<Resource>Policy`
- Policy methods match controller actions (`show?`, `update?`, `destroy?`)
- Check the ACTUAL policy file, not just whether `authorize` is called

### Packwerk boundaries

- `package.yml` files define component boundaries
- Cross-package calls may be forbidden — check `enforce_dependencies`
- When suggesting fixes, verify the fix respects package boundaries

### Sidekiq / ActiveJob

- Without `retry_on`, Sidekiq uses its OWN retry mechanism (25 retries, exponential backoff). Don't flag absence of `retry_on` as a bug.
- `perform_now` runs synchronously (used in rake tasks, not queued)

### Strong parameters

`params.require(:report).permit(:title, :description)` is the controller-level protection. Internal service objects that receive pre-validated hashes do NOT need strong params.

**Misdiagnosis trap:** Flagging an interactor for "missing strong params" when it only receives data from a controller that already applied them.

### Route conventions

`resources :reports` generates 7 routes. `member` routes include `:id`, `collection` routes don't.

**Misdiagnosis trap:** Claiming a route doesn't exist without reading `config/routes.rb`. Nested resources, concerns, and namespace blocks generate routes that aren't obvious from a single line.

### Rails conventions — don't misdiagnose

- `ENV.fetch('KEY')` raising KeyError is intentional fail-fast, not a bug
- `Time.current` is the Rails convention (not `Time.now`)
- `present?` / `blank?` are Rails core extensions, not custom code
- `squish` collapses whitespace — it's a Rails method, not custom code

## Go Investigation Rules

- Go is statically typed. Integer values CANNOT contain SQL injection when used with `fmt.Sprintf("%d", val)` or direct interpolation.
- String values from user input CAN be dangerous in `fmt.Sprintf` SQL construction — flag these.
- `database/sql` with `?` placeholders is parameterized.

## JavaScript/TypeScript Investigation Rules

- JS has NO integer type — all numbers are IEEE 754 doubles. However, `parseInt(val, 10)` returns NaN for non-numeric strings, not a dangerous value. `NaN` in SQL causes a query error, not injection.
- Template literals with user strings ARE dangerous in raw SQL.
- ORMs (Prisma, TypeORM, Knex) parameterize by default when using their query builder APIs. Raw SQL methods (`.raw()`, `.$queryRaw()`) need manual parameterization.

## Adding rules for new frameworks

When encountering a framework not listed above, apply the same principle: **understand what the language runtime and framework guarantee before claiming a vulnerability or bug exists.** Add rules to this file as they are discovered.
