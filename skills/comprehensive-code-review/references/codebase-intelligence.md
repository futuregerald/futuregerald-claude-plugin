# Codebase Intelligence Gathering

**Always populate `{CODEBASE_CONTEXT}` and `{CONVENTIONS_CONTEXT}` with actual output —
never with instructions to search.**

Two questions decide whether a review reads like a colleague who knows the codebase:

1. **Does this already exist?** — answered from the AST index below.
2. **How do we do this here?** — answered from the repo's own written conventions, and from
   the most recently changed sibling of the file being reviewed.

Both are gathered by the orchestrator, by running commands, before any sub-agent is dispatched.

---

## Part 1 — The AST index (`{CODEBASE_CONTEXT}`)

Everything here uses the `codebase-memory-mcp` **CLI**. No MCP server and no client
configuration, so the same commands work in a local session and on a CI runner — but the
*starting state* differs: a laptop usually has an index already, while a CI runner has neither
the binary (~295 MB) nor an index, and takes either the from-scratch build in Step 1 or the
graceful-failure path below.

```bash
CBM=codebase-memory-mcp
command -v "$CBM" >/dev/null 2>&1 || CBM="$HOME/.local/bin/codebase-memory-mcp"
command -v "$CBM" >/dev/null 2>&1 || CBM=""   # unavailable — see Graceful Failure
```

### Step 1 — Resolve or build the project

```bash
PROJECT=$("$CBM" cli list_projects 2>/dev/null \
  | grep -o '"name":"[^"]*","root_path":"'"$PWD"'"' | head -1 \
  | sed 's/.*"name":"\([^"]*\)".*/\1/')

if [ -z "$PROJECT" ]; then
  # Stable per-repo name, re-indexed in place. A per-SHA name would leave one
  # dead project per review on the machine, with nothing cleaning them up.
  PROJECT="review-$(basename "$PWD")"
  "$CBM" cli index_repository --repo-path "$PWD" --mode fast --name "$PROJECT"
fi
```

Building from scratch is cheap enough not to plan around: **16.3s** for a 164,000-node Rails
repo, **5.4s** for a 26,000-node one. A single query costs about **1.5s**.

**Staleness.** An existing index may sit on a different branch than the PR. A stale *hit* is
still a real symbol and can be cited. A stale *miss* is not evidence of anything.

### Step 2 — Coverage guard (run before any query)

```bash
"$CBM" cli check_index_coverage --project "$PROJECT" --paths '["path/one.rb","path/two.rb"]'
```

Every changed file that comes back partial or missing goes into the `GREP-ONLY FILES` line of
the emitted header. This is not hypothetical — a large Rails monolith I measured had 21
partially-parsed files, so without the guard a parse gap reads to the sub-agent as an absence.

### Step 3 — Extract symbols from the diff

Deterministic, no model judgment. Ruby:

```bash
# \w+ does NOT match "::" — without the colons, Billing::Invoices::CreateAdjustment
# extracts as "Billing", and every downstream query then sweeps a whole namespace.
NEW_CLASSES=$(grep -oE '^\+\s*(class|module)\s+[A-Z][A-Za-z0-9_:]*' <<<"$DIFF" \
  | awk '{print $NF}' | sed 's/.*:://' | sort -u)
NEW_METHODS=$(grep -oE '^\+\s*def\s+(self\.)?[a-zA-Z_][a-zA-Z0-9_?!]*' <<<"$DIFF" \
  | sed 's/.*def *//; s/^self\.//' | sort -u)
```

**Drop framework verbs before querying.** `call`, `create`, `index`, `show`, `update`,
`destroy`, `new`, `perform`, `initialize` and other single short words are defined by every
class of their kind — searching them returns the whole codebase and tells you nothing. A stem
worth a query is specific and usually multi-word.

TypeScript: `^\+\s*(export\s+)?(async\s+)?function\s+(\w+)` and
`^\+\s*(export\s+)?class\s+(\w+)`. Go: `^\+func\s+(\([^)]*\)\s*)?(\w+)`.

### Step 4 — Five query classes

Run them at the frequencies in **Budget** below — not all five per symbol.

| Class | Command | Answers |
|-------|---------|---------|
| **A. Reuse candidates** | `"$CBM" cli search_graph --project "$PROJECT" --name-pattern '<stem>' --detail ids --limit 20` and `--query '<stem split on _>' --limit 8` | Does a symbol by this name, or these words, already exist anywhere? |
| **B. Sibling conventions** | `"$CBM" cli search_graph --project "$PROJECT" --file-pattern '%<dir of changed file>%' --label Method --fields lines --limit 25` | What do the neighbours look like? |
| **C. Namespace peers** | `"$CBM" cli search_graph --project "$PROJECT" --qn-pattern '<namespace>' --label Class --detail ids --limit 25` | What is the shape of this family of classes? |
| **D. Duplication census** | `"$CBM" cli query_graph --project "$PROJECT" --query "MATCH (m:Method) WHERE m.name CONTAINS '<stem>' RETURN m.name AS name, count(*) AS n ORDER BY n DESC" --max-rows 20` | Is this concept already implemented N times? |
| **E. Fan-in** | `"$CBM" cli search_graph --project "$PROJECT" --qn-pattern '<changed class>' --min-degree 10 --limit 10` | Is the changed symbol a hub? A severity multiplier, nothing more. |

Class D is the one that produces findings a reviewer would otherwise never reach. On a large
Rails monolith, `CONTAINS 'business_day'` returned one predicate duplicated across **seven**
strategy classes and another across **two** interactors — none of which any reviewer had
noticed.

### Step 5 — The body token pass

Searching a new method's **name** finds a duplicate only when someone happened to name it
similarly. Most real duplication is named differently.

So for every new method, also pull distinctive tokens from its **body** in the diff —
constant names, called method names, model names, library calls — drop language keywords and
anything under 4 characters, and run class A on those too.

> A new `add_two_business_days` is found by its name. An existing `skip_weekend` that
> references `Date::DAYNAMES` is not — the **body token** `DAYNAMES` is what finds it.

### Step 6 — One canonical exemplar, not a list of siblings

A list of fifteen siblings is weaker guidance than one file with a date on it. Rank the
changed file's directory by recency and hand over the top entry as *the* pattern:

```bash
DIR=$(dirname "$CHANGED_FILE"); EXT="${CHANGED_FILE##*.}"
shopt -s nullglob 2>/dev/null || setopt local_options null_glob 2>/dev/null
for f in "$DIR"/*."$EXT"; do
  case " $FILE_LIST " in *" $f "*) continue;; esac   # never rank a file under review
  echo "$(git log -1 --format=%ad --date=short "$BASE_SHA" -- "$f") $f"
done | sort -r | head -3
```

Two guards, both load-bearing:

- **Exclude every file in `{FILE_LIST}`.** The changed file is by construction the most
  recently changed file in its directory, so without this the diff is ranked as its own gold
  standard and Section B reports conformance every time.
- **Date from `$BASE_SHA`, not `HEAD`** — for the same reason.
- `nullglob`, because an empty directory otherwise emits the literal unexpanded glob with a
  blank date, which sorts to the top and becomes a fake exemplar.

Roughly 1.4s on a large repo. "This is how we did it most recently" is an instruction a
sub-agent can follow; "here are some examples" is not.

### `trace_path` — never use it for reachability

**Do not use `trace_path` in review context.** The graph is for **existence and shape**, never
for **reachability**.

Measured on a large Rails monolith. The colon-qualified name returns `function not found`, so
reproduce it with the bare name:

```
$ codebase-memory-mcp cli trace_path --project <p> --function-name 'CreateAdjustment' \
    --direction inbound
callers_total: 0
```

A real caller sat one directory away, naming it as a bare constant inside an
`Interactor::Organizer` list. Tracing a bare method name returned callees inside a serializer,
three unrelated notification classes and a controller — all matched by short-name collision.

Rails' dominant dispatch — organizer arrays, `send`, `constantize`, callback and serializer
registration, job class names held as strings — is invisible to an AST call graph. A finding
built on `trace_path` output ("nothing calls this, it is dead code"; "no caller depends on
this return shape") is confidently wrong and unfalsifiable by the author. That is worse than
having no data.

Reachability questions go to Grep.

### Query syntax gotchas

- `--name-pattern` is a **regex substring, not SQL LIKE**. `%business%` returns 0 rows;
  `business` returns 26 nodes, or 13 with `--label Method`. Pass `--label Method` in class A
  unless you specifically want classes and constants too.
- `--file-pattern` **does** take `%` wildcards. The two flags do not share a syntax.
- `--detail ids` returns bare qualified names — use it for wide sweeps; full rows only for
  classes A and B.
- `--semantic-query` is **not usable**, for two separate reasons. On an index that has
  embeddings it ranks too poorly to trust — measured, `["business","days","weekend","deadline"]`
  topped out at 0.062 and returned a SCIM controller. On an index built with `--mode fast`, as
  Step 1 builds them, there are no similarity or semantic edges at all. Either way, do not use
  it; reuse detection here is lexical.

### Budget

Run the classes at these frequencies — "five classes per symbol" plus a body-token pass would
be 60+ calls on an ordinary diff, which is why the frequency is specified rather than a bare
call cap:

| | Frequency |
|---|---|
| **B, C, E** | once per changed **directory**, not per symbol |
| **A, D** | for at most the **5 most substantial** new symbols, after dropping framework verbs |
| **Body tokens** | at most **3 tokens** per new method, class A only |

That lands near **24 calls** and **6 KB** of emitted context. Over budget, cut symbols before
cutting query classes — a partial sweep of the important symbols beats a full sweep of
trivial ones.

### The emitted block

```
## Codebase Context (AST index — project <name>, <N> nodes, mode <from index_status>)

WHAT THIS PROVES: a symbol listed here EXISTS at the stated file:line.
WHAT THIS DOES NOT PROVE: that a symbol absent here does not exist. This index
has no working semantic search and does not resolve Ruby metaprogramming.
Files marked GREP-ONLY were not fully parsed — an absence there is unknown,
not negative. DO NOT infer reachability, dead code, or caller safety from
this section.

GREP-ONLY FILES: <list or "none">

### A. Reuse candidates for <symbol>
### B. Canonical sibling — <file> (last changed <date>)
### C. Namespace peers
### D. Duplication census
### E. Fan-in
```

### Graceful failure

If the binary is missing, the index build fails, or a query errors: set `{CODEBASE_CONTEXT}`
to `"(unavailable — <reason>; Section B searches fall back to Grep)"`, note it in the report
header, and run the review normally.

**Never fail a review because the index is missing.** Same contract as review-lens.

---

## Part 2 — Repo conventions (`{CONVENTIONS_CONTEXT}`)

The index says what exists. These files say what was **decided**, and they are where a
reviewer who knows the company gets their instincts. Discover them from the repo root:

```bash
ls docs/adr/*.md 2>/dev/null | head -60
ls docs/good-practices.md docs/*GUIDELINES*.md docs/*PATTERNS*.md \
   CONTRIBUTING.md docs/CONTRIBUTING.md 2>/dev/null
```

Mature repos usually carry several. One Rails monolith I checked had 54 ADRs plus a
`good-practices.md` stating rules as concrete as *"only use organizers if the interactor would
be 100+ lines"*; its front-end sibling had `CODE_GUIDELINES.md`, `COMPONENT_PATTERNS.md`,
`RTK_QUERY_PATTERNS.md` and `MONOREPO_GUIDELINES.md`. None of it was ever read by a review.

### Injection rules

- **ADRs: titles only.** Fifty-four documents will not fit and most are irrelevant to any one
  diff. Emit the `NNNN_slug` filenames as a list and let the sub-agent read the two or three
  whose titles bear on the change:
  `sed -n '1,80p' docs/adr/0007_code_boundaries.md`
- **Guideline docs: full text, capped at ~12 KB total.** (A real `good-practices.md` came to
  11.7 KB on its own — an 8 KB cap would truncate the very file that motivates this rule.) These are the actual "how we do it
  here" and earn their tokens. Over the cap, emit the headings plus only those sections whose
  headings match tokens drawn from the diff.
- **Nothing found** → `"(none — this repo has no written conventions docs)"`. A real and
  common answer, not a failure.

### The emitted block

```
## Repo Conventions

ADRs available (read the relevant ones with sed -n '1,80p' <path>):
  docs/adr/0007_code_boundaries.md
  docs/adr/0017_package_best_practices.md
  ...

--- docs/good-practices.md ---
<full text, or matched sections>
```

---

## Gem / Library Source Constraint

**Do NOT search for installed gem source files on the CI runner.** Common gems
(interactor, pundit, cancancan, active_model_serializers, devise, sidekiq,
dry-rb, etc.) are installed via bundler but their source paths are not
reliably locatable in the CI environment. Attempting to find them wastes
many turns on fruitless `find`, `bundle show`, and `gem which` calls.

Instead:
- **Describe gem behavior from your training knowledge** — you know how these
  gems work. State your understanding and flag it as "based on standard gem
  behavior" if relevant to a finding.
- **Check the Gemfile / Gemfile.lock** for version constraints if version-specific
  behavior matters.
- **Read the app's own wrapper/base classes** (e.g., `ApplicationInteractor`,
  `ApplicationPolicy`) to understand how the app customizes gem behavior.

**Never:** run `find / -iname`, `bundle show`, `gem which`, or fetch gem source
from GitHub to understand how a standard gem works. This is a hard constraint.
