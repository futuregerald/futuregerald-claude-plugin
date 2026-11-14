# Components

Every class below is styled in `assets/base.html`. Read only the entries the plan needs. Examples use a made-up plan (turning a warehouse team's shipment events into customer delivery alerts); replace every value with the plan's own.

Each section is `<section id="…">` with an `.eyebrow` (what kind of section), an `h2` that states the point, then one component. The `id` is a bare word so it can be linked as `#flow`.

## Summary points — what the reader must take away

Three cards right under the header. Use for the plan's source, its main mechanism and its main rule. Not a feature list.

```html
<section>
  <div class="points">
    <div class="point">
      <span class="eyebrow">Source</span>
      <h3>The warehouse team's shipment events</h3>
      <p>They already record each status change. We don't track parcels ourselves.</p>
    </div>
    <!-- two more .point cards -->
  </div>
</section>
```

## Flow diagram — data moving between parts or teams

Hand-drawn SVG with one horizontal lane per owner. Recipe and sizes in `diagrams.md`. Follow it with a `.legend` and a `.caption` saying what does NOT happen (the edge a reader would wrongly assume).

```html
<div class="legend">
  <span><span class="swatch" style="background: var(--theirs)"></span>Warehouse team owns</span>
  <span><span class="swatch" style="background: var(--accent)"></span>We build</span>
  <span><span class="swatch" style="background: var(--stop)"></span>Stops here</span>
</div>
<div class="diagram" role="img" aria-label="One sentence describing the whole flow.">
  <svg viewBox="0 0 960 420" xmlns="http://www.w3.org/2000/svg">…</svg>
</div>
<p class="caption">Re-sends from the warehouse never reach our job, so they can't duplicate alerts.</p>
```

`--theirs` is the "other team" colour and `--accent` is "ours"; keep that meaning on every component.

## Worked example — input rows become output

Real-shaped values: plausible ids, dates, paths. Show the arithmetic beside derived numbers.

```html
<div class="example">
  <div class="panel">
    <div class="panel-head"><h3>Their events</h3><span class="owner theirs">Warehouse team</span></div>
    <table class="rows">
      <thead><tr><th>parcel</th><th>event</th></tr></thead>
      <tbody><tr><td class="mono">PX-4471</td><td><span class="chip new">delivered</span></td></tr></tbody>
    </table>
  </div>
  <div class="arrow" aria-hidden="true">→</div>
  <div class="panel alert-card">
    <div class="panel-head"><h3>Our alert</h3><span class="owner ours">delivery_alerts</span></div>
    <div class="counts">
      <div class="count new"><div class="n">3</div><div class="l">delivered</div></div>
      <div class="count gone"><div class="n">1</div><div class="l">returned</div></div>
    </div>
    <dl class="kv"><dt>Open before</dt><dd>12</dd><dt>Open after</dt><dd>8 <span class="math">= 12 − 3 − 1</span></dd></dl>
  </div>
</div>
```

Chips: `.chip.new` (added/good), `.chip.gone` (removed/bad), `.chip.stop` (neutral or "nothing happens").

## Steps — an ordered process

Only when order matters. The number is drawn by CSS; don't type it.

```html
<ol class="steps">
  <li><div class="body">
    <h3>Check the switch is on</h3>
    <p><code>return unless Flags.on?(:delivery_alerts)</code>. When it's off nothing is written.</p>
  </div></li>
</ol>
```

A step that branches gets a `.grid-table` inside its `.body` with "Situation | What happens" rows, one row per branch, including the "nothing happens" row.

## Handoff — what each step is sent

Use when work passes between jobs, services or teams. One `.steps` item per hop: heading `Sender → Receiver`, an `.owner` tag (`theirs` existing / `ours` new), a `<pre>` with the exact call, then one sentence on what the receiver works out from it.

## Choice cards — a decision between options

The chosen option gets `.option.chosen` and a `.pill.done` tag; the other gets `.pill.open` "Not chosen". Bullets say consequences, not features. When the choice changes behaviour across states (on/off, enabled for whom), add a `.grid-table` of states → results.

```html
<div class="compare">
  <div class="option chosen"><span class="pill done tag">Chosen</span><h3>…</h3><pre>…</pre><ul><li>…</li></ul></div>
  <div class="option"><span class="pill open tag">Not chosen</span><h3>…</h3><ul><li>…</li></ul></div>
</div>
```

## Problem → answer table — why a design choice

For any choice a reviewer will question ("why our own table?", "why one, not two?"). Left column: what goes wrong without it. Right: what the choice does. Close with a short `h3` + paragraph for the one follow-up question.

## Schema cards — tables we create

```html
<div class="tables">
  <div class="schema">
    <h3>delivery_alerts</h3>
    <table>
      <tr><td>parcel_id</td><td>FK parcels</td></tr>
      <tr><td>kind</td><td>baseline or change; only change rows are alerts</td></tr>
    </table>
    <div class="foot">Unique (parcel_id, event_id) · insert-only</div>
  </div>
</div>
```

The right column says why the column exists or where its value comes from, never just its type.

## Entity relationships — how tables connect

Mermaid `erDiagram` inside `<div class="diagram erd"><pre class="mermaid">…</pre></div>`. Rules in `diagrams.md`. Follow it with a `.legend` naming which tables belong to whom.

## Tiles — numbers the UI shows

`.tiles` of `.tile`: eyebrow label, `.n` big number, `.d` delta, `.src` saying exactly how it's worked out. Caption the numbers as illustrative unless they're real.

## Endpoint cards — an API

One `.endpoint` per route: left `.side` has `.verb` (`<b>GET</b>path`) and "Send" (exactly what the caller provides); right `.side` has "Returns" and a trimmed JSON `<pre>`. Put access rules once, above, in an `.access` box.

## Patterns table — existing code reused

`.grid-table` with "Piece | Copied from". Every right-hand cell names a real file, class or method from the plan. If the plan names none, leave the section out.

## Work table, affected code, rollback

Stories as a `.grid-table` (Story | Repo | Waits on). Then a `.two` row of `.access` boxes: "Affected existing code" (one line per changed symbol and its test) and "Rollback" (the steps, in order).

## Accepted gaps and decisions

Accepted gaps: a `.list` with a bold lead per item. Decisions: `.decision` rows, `.pill.done` "Settled" or `.pill.open` "Open", the question in `.q`, the answer or what's blocking in `<p>`. Open items say who decides.
