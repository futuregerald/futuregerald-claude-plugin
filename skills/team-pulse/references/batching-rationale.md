# Sub-Agent Batching vs Fan-Out Rationale

## One Per Source vs Per Day Trade-Off

The default pattern is **one agent per source covering the whole window** (3 agents total).
Fan out per day instead (24 agents) when attribution isolation matters more than tokens.

### Trade Table

| | One per source (default) | One per source per day |
|---|---|---|
| Dispatch floors | 3 | 24 |
| Tokens | ~171,000 | ~1,365,000 |
| Wall clock | slower | faster — fan-out parallelises |
| Raw input per agent | 8x larger | bounded at one day |
| Attribution risk | real, and silent | structurally prevented |

### Attribution Isolation

An agent that holds only Tuesday cannot report Tuesday's work under Wednesday, cannot merge two people's tickets into one summary, and cannot carry a stray detail from an adjacent day into a sentence about this one. An agent holding eight days for seven people can do all three, and **nothing downstream will flag it** — the digest will be fluent, well-formed, and wrong in a way only the person it describes would catch.

**Default to one per source.** Take the per-day fan-out when the report will be acted on personally — a 1:1, a performance conversation, anything where a name attached to the wrong piece of work is the expensive failure — or when you need the report fast.

### Capacity vs Attribution Splits

When splitting a source:
- **For capacity:** The source cannot be capped at the query, or activity is unusually heavy. Split into **halves or thirds** (not days).
- **For attribution:** Split by **day** so the temporal boundary cannot be crossed.

Whichever the trigger, **a split source must write numbered digests** (`.updates/<source>-1.md`, `.updates/<source>-2.md`, or `.updates/<source>-<date>.md`) to prevent silent file overwrites.
