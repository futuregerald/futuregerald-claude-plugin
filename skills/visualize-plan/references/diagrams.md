# Diagrams

## Which diagram

```
Data moves between systems, jobs or teams?          → Flow diagram (hand SVG, owner lanes)
Plan adds or changes tables?                         → ER diagram (Mermaid erDiagram)
Calls go back and forth between two or more parts?   → Sequence (Mermaid sequenceDiagram)
A record moves through named states?                 → State diagram (Mermaid stateDiagram-v2)
A process with a few branches?                       → No diagram: .steps + a branch table
None of these?                                       → No diagram
```

One diagram per question. Don't draw something a table already says.

## Flow diagram (hand SVG)

Hand SVG, not Mermaid, because ownership lanes and the stop branch need control Mermaid doesn't give.

- `viewBox="0 0 960 H"`. One full-width `<rect>` band per owner, top to bottom in the order data flows: other team first (`fill: var(--theirs-soft)`), ours below (`fill: var(--accent-soft)`). Lane label top-left, 12px, uppercase, the lane's strong colour.
- Boxes: `rx="4"`, `fill: var(--surface)`, stroke in the owner's colour. Title 13–14px weight 600; up to two 12px muted lines saying what the box does. Table and job names in `font-family: var(--mono)`.
- Arrows: one `<marker id="ah">` filled with `var(--muted)`. Solid for "calls / writes", dashed (`stroke-dasharray: 4 3`) for "reads". Label an arrow only when the box names don't make it obvious.
- The change we make to someone else's code sits on its crossing arrow as a small labelled box ("+1 line: enqueue").
- A grey dashed "Stop" box (`stroke: var(--stop)`) lists every condition that ends the flow early.
- Every colour comes from a CSS variable through `style=""`, so both themes work. Never a hex value in the SVG.
- Leave 20px inside the viewBox around the outermost box. Check that no label runs past its box at 960 wide.

## ER diagram (Mermaid)

```
erDiagram
  orgs ||--o{ parcels : "owns"
  parcels ||--o{ shipment_events : "events (warehouse)"
  parcels ||--o{ delivery_alerts : "alerts (ours)"
  shipment_batches |o..o{ delivery_alerts : "batch_id, no FK"

  delivery_alerts {
    bigint id PK
    string kind "baseline or change"
    bigint parcel_id FK
    bigint batch_id "unique with parcel_id"
  }
```

- Solid `--` lines are real foreign keys; dotted `..` lines are ids stored without one. Say so in the caption.
- Relationship labels say what the link means and whose table it is ("events (warehouse)").
- List only the columns this plan reads or writes, and say that in the caption.
- Include the existing tables the new ones hang off, up to the org or account, so a reader can see how rows are scoped.
- Attribute types: one word each (`text_array`, not `text[]`). Comments in double quotes.

## Sequence and state diagrams (Mermaid)

Same wrapper as the ER diagram: `<div class="diagram erd"><pre class="mermaid">…</pre></div>`. Participants and states use the plan's real job, service or status names. Never load the Mermaid library in the fragment.
