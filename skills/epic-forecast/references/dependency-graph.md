# Dependency graph

Runs in **both modes**. A dependency graph needs no window — it is about order, not dates.

The flat dependency table answers "what does this item wait on". It cannot answer the two questions
that decide a schedule: **what is blocked by something that is itself blocked**, and **which chain of
work is the longest**. The chain is the date-bearing number. The capacity total is not
(`capacity-model.md` §"The sum is never a date").

## Where edges come from

**The research agents' dependency lists are authoritative. Tracker links only supplement them.**

This is the single most important rule in this file, and it is the opposite of the obvious design.
Sourcing the graph from tracker link types looks cleaner — structured data, no parsing — and it
produces a confidently wrong answer.

The evidence, from a real 13-row dependency list:

| Rows | What they were |
|---|---|
| 4 | Named a tracker key |
| 4 | Explicitly recorded status "No ticket" |
| 5 | Review queues, product decisions, legal decisions — nothing a tracker models |

The row the source document itself labelled **"THE actual constraint"** was
`"Code review on 3 children + an open PR"`, with no issue link and no ticket. A link-sourced graph
would have drawn 4 of 13 edges, omitted the real bottleneck, and then computed a critical chain over
a subgraph that excluded it — a date that is wrong and looks rigorous.

Reporting link coverage does not rescue that design. It labels the omission; it does not fill it.

The skill already collects the right input: `agent-prompts.md` asks every cluster agent for
"upstream (with team name if external) · downstream · undecided decisions and who owns them". That
output is the graph's source. Tracker links are then queried **second**, and any edge found only
there is drawn with a `tracker` source and flagged in the output — because an edge the agents missed
is itself a finding about the research.

### Querying tracker links

Pass a **narrow `fields` list**: `key, summary, status, issuelinks, assignee`. A links query without
one returns full nested objects for every link on every issue and blows the response budget — one
observed call returned ~40 KB for 5 issues.

Map the link types:

| Link type | Becomes |
|---|---|
| `Blocks` / `is blocked by` | A **hard** edge; drives ordering |
| `Relates` | A **soft** edge; drawn dashed, never drives ordering |
| Idea/roadmap links (e.g. Polaris work-item links) | Not a dependency. Trace to the idea, do not draw an edge |

## Node kinds

Every node declares a `kind`, and the dependency list gains the same field:

| Kind | What it is | Carries an estimate? |
|---|---|---|
| `epic` | In-scope work with an estimate | **Yes — the only kind that does** |
| `external` | Another team, vendor or service. Often has no ticket at all | No |
| `decision` | An undecided product, legal or management call. **Record the owner** | No |
| `queue` | Review or approval latency — e.g. a named PR awaiting review | No |

**Weights exist only on `epic` nodes, and `build_graph.py` rejects a manifest that puts one
elsewhere.** This is enforced rather than documented because the failure is silent: giving a legal
decision a weight of 0 makes it vanish from the total, and giving it a guessed weight publishes a
schedule that prices a conversation nobody has had. Most real `dep` values — "Production matching
service", "Code review on 3 open PRs" — are not estimable work, and that is the normal case, not an
edge case.

The consequence for the output: **the chain is reported as epic weeks *plus named unweighted
gates*, never as a single number.**

## The manifest needs a join key

`build_workbook.py` is a generic sheets/columns/rows renderer with no node identity, and the same
item is routinely named differently on different sheets — a Dependencies row saying
`"Search reindex (ABC-101)"` against an Estimates row saying `"Search reindex / backfill"` with the key
only inside the link text. There is no string that joins a dependency row to the optimistic and
pessimistic estimates a weighted longest path needs.

So the manifest gains a **top-level `graph` block beside `sheets`**:

```json
{
  "sheets": [ ... ],
  "graph": {
    "nodes": [
      {"id": "e1", "label": "Search reindex", "key": "ABC-101", "kind": "epic",
       "weight_opt": 3.0, "weight_pess": 6.0},
      {"id": "q1", "label": "Code review on 3 children", "kind": "queue"},
      {"id": "d1", "label": "Retention policy call", "kind": "decision", "note": "owner: Legal"},
      {"id": "e2", "label": "Consumer rollout", "key": "ABC-102", "kind": "epic",
       "weight_opt": 2.0, "weight_pess": 4.0}
    ],
    "edges": [
      {"from": "q1", "to": "e1", "type": "hard", "source": "agent", "note": "THE actual constraint"},
      {"from": "e1", "to": "e2", "type": "soft", "source": "tracker"}
    ]
  }
}
```

`id` is the join key: Dependencies sheet rows carry the same ids. Edge direction is **"blocks"** —
`from` must complete before `to` can finish.

`build_workbook.py` iterates `manifest["sheets"]` only, so it is untouched by this addition. Do not
modify it.

## What gets computed

`scripts/build_graph.py` reads the `graph` block and emits Mermaid plus an edge table. Standard
library only — no dependency, no venv.

```
python3 scripts/build_graph.py manifest.json graph.md
```

1. **Cycle detection, FIRST.** Not a preference — a correctness requirement. A longest-path walk over
   a cyclic graph never terminates. On a cycle: report it as a data-quality finding, name the nodes,
   and skip the chain for the affected nodes rather than attempting it. A cycle means somebody
   recorded a dependency backwards, or two pieces of work genuinely need splitting apart, and both
   are worth saying out loud.
2. **Transitive closure.** So "blocked by something that is blocked by something" is visible. This is
   the thing a flat table structurally cannot show, and it is where schedule surprises live.
3. **Longest path = the critical chain**, node-weighted by each epic's one-engineer estimate, run
   once for the optimistic bound and once for the pessimistic. If the two bounds run along
   *different* paths, say so — which item decides the date then depends on which bound holds, and
   that is a finding.
4. **Orphans.** *Epics* with no edge in either direction: parallelisable now, and the safest work to
   start. A **gate** with no edges is reported separately and is the opposite of safe — a decision or
   a queue connected to nothing either blocks something nobody recorded, or should not be in the
   graph.
5. **Link coverage** — n of m epics carry any edge. A sparse graph is ambiguous: genuinely
   independent work and thin dependency research look identical, and they have opposite
   consequences. The report must say which.

## Rendering

Mermaid `graph LR`, which renders natively in artifacts and most Markdown viewers.

- Hard edges solid, soft edges dashed
- Node kinds distinguished by shape: epic rectangle · external subroutine · decision rhombus ·
  queue stadium
- The critical chain highlighted
- Edges found only in the tracker labelled `tracker only`

## Degenerate cases

**One epic, or no edges at all:** emit the node(s), state plainly that no dependencies were found,
and do not fabricate an ordering. With a single item, sequencing degenerates to that item's internal
order and the graph shows its external edges — say so rather than emitting an empty section. An
empty section reads as an oversight; a sentence saying "nothing was found, here is what that means"
reads as a result.

## What this replaces in the prose

Wherever the report currently says "the long pole" or names a bottleneck in prose, it now cites the
computed chain: its length in epic weeks, the gates on it by name, and the path. Prose that asserts a
bottleneck without the chain behind it is the thing this file exists to remove.
