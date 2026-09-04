---
name: vectorize
description: Convert raster images (PNG/JPG) into clean SVG vector art. Use whenever the user wants to vectorize an image, do png to svg, raster to vector, convert image to svg, trace an image, turn a mascot/logo/illustration into vectors, or produce scalable SVG from a bitmap. Handles color art (vtracer) and mono/line art (potrace), with optional background removal, svgo optimization, and a verification render.
allowed-tools: Bash(python3:*), Bash(uv:*), Bash(npx:*), Bash(brew:*), Read
---

# Vectorize: Raster to Clean SVG

Turn PNG/JPG bitmaps into faithful, clean SVG vector art. Built for flat/vector-style
art (mascots, logos, illustrations, icons) on Apple Silicon macOS.

## When to Use

Use this skill when the user wants to:

- "vectorize" an image, or do "png to svg" / "raster to vector"
- "convert image to svg" or "trace" a bitmap
- Turn a mascot, logo, or illustration into scalable vectors
- Batch-convert a directory of PNG/JPG images to SVG

## What It Does

`scripts/vectorize.py` runs a proven pipeline:

1. **Engine selection** — COLOR art (fills, illustrations, logos) is traced with
   **vtracer**; MONO/line art is traced with **potrace** (if installed) or vtracer
   greyscale as a fallback. Default is `color`.
2. **Optional background removal** — two strategies, *neither universally correct*
   (see "Transparency: choosing a strategy" below). `--transparent` (safe default) uses
   an edge-seeded flood fill; `--global-key` uses a global color-key. Applied *before*
   tracing, so the SVG has no background shape.
3. **Vectorize** with tuned vtracer params for flat art (stacked, spline).
4. **Optimize** (`svgo --multipass` via npx) — roughly 50% smaller. Skipped gracefully
   if npx/svgo is unavailable.
5. **Verify** (`cairosvg`) — rasterizes the output SVG back to `<name>.check.png` next
   to the SVG so a human can eyeball fidelity. Skipped gracefully if unavailable.

## Prerequisites

- **`uv`** (required) — `brew install uv`. Used to build the pinned Python 3.12 venv.
- **`npx` / `svgo`** (optional) — for the optimize step. Provided by Node; `--yes svgo`
  is fetched on demand. If missing, optimization is skipped with a warning.
- **`potrace`** (optional) — `brew install potrace`. Only used in `--mode mono` for the
  crispest line-art tracing. Without it, mono falls back to vtracer greyscale.
- **`cairo`** (optional, for verification) — `brew install cairo`. The script auto-adds
  Homebrew's lib dir to the loader path so `cairosvg` finds `libcairo` on Apple Silicon.

## One-Time Setup (auto-bootstrapped)

There is **nothing to install by hand** beyond the prerequisites above. On first run the
script uses `uv` to create a cached Python 3.12 virtualenv at `.venv/` inside this skill
directory and installs `vtracer`, `pillow`, and `cairosvg` into it. Subsequent runs reuse
that venv and are fast. If the venv ever gets corrupted, delete `.venv/` and re-run.

**Why a separate venv?** vtracer's native wheel **segfaults (exit 139) on Python 3.14**
on Apple Silicon. It works reliably on **Python 3.12**. The script manages this itself —
it never relies on the system Python having vtracer.

## Usage

```bash
# Single file, output alongside the input
python3 scripts/vectorize.py mascot.png

# Single file, custom output dir, with SAFE background removal (edge flood fill)
python3 scripts/vectorize.py mascot.png -o out/ --transparent

# Aggressive background removal (global color-key) -- clears interior gaps
python3 scripts/vectorize.py mascot.png -o out/ --global-key

# Whole directory (globs *.png / *.jpg / *.jpeg)
python3 scripts/vectorize.py ./assets/ -o ./svg/

# Mono / line-art (uses potrace if present, else vtracer greyscale)
python3 scripts/vectorize.py logo_bw.png --mode mono

# Auto-detect color vs mono
python3 scripts/vectorize.py drawing.png --mode auto

# Tuning + skipping optional steps
python3 scripts/vectorize.py mascot.png --filter-speckle 10 --color-precision 8
python3 scripts/vectorize.py mascot.png --no-optimize --no-verify
```

## Parameter Reference

| Flag | Default | Meaning |
|------|---------|---------|
| `input` | — | A single PNG/JPG file **or** a directory (globs images). |
| `-o, --outdir DIR` | alongside input | Where to write `.svg` (and `.check.png`). |
| `--transparent` | off | Remove background via **edge flood fill** (safe default). Preserves same-colored interior regions; leaves enclosed background opaque. |
| `--global-key` (alias `--transparent-aggressive`) | off | Remove background via **global color-key**. Clears enclosed gaps too, but erases interior regions matching the background. |
| `--mode color\|mono\|auto` | `color` | Engine: color=vtracer, mono=potrace/greyscale, auto=heuristic. |
| `--filter-speckle N` | `6` | Drop specks smaller than N px. **Higher = fewer specks, smaller file.** |
| `--color-precision N` | `7` | Color bits of precision. **Higher = more faithful, larger file.** |
| `--no-optimize` | off | Skip the svgo pass. |
| `--no-verify` | off | Skip the cairosvg `.check.png` render. |

**Fixed vtracer params (proven on flat art):** `colormode=color`, `hierarchical=stacked`,
`mode=spline`, `corner_threshold=60`, `path_precision=3`.

## Transparency: choosing a strategy

Background removal has two strategies and **neither is universally correct**. Pick based
on the artwork, and always eyeball the `.check.png` afterward.

| Strategy | Flag | Removes | Preserves | Fails when |
|----------|------|---------|-----------|------------|
| **Edge flood fill** (safe default) | `--transparent` | Only background *connected to the image border* | Enclosed same-colored regions (e.g. white/cream eye interiors) | Background is **trapped inside** the art (gaps between limbs, holes in a logo) — those stay opaque |
| **Global color-key** (aggressive) | `--global-key` | *All* pixels near the background color, anywhere | Nothing color-matching is safe | An interior region **matches the background color** — it gets erased, leaving a hole (e.g. cream eye-whites become transparent voids, which look like creepy black holes on a dark page) |

**Rule of thumb:**
- Start with `--transparent`. It never eats interior detail.
- If enclosed gaps must be cleared **and** no interior region shares the background color,
  use `--global-key`.
- If the art has cream/white eyes, teeth, or highlights on a light background, **avoid
  `--global-key`** — it will punch them out. Use `--transparent` and accept the enclosed
  gaps, or hand-edit the SVG.

## Troubleshooting

- **vtracer segfault / exit 139 (the #1 gotcha).** This happens when vtracer runs on
  Python 3.14. The script avoids it by pinning a **Python 3.12** venv via `uv` — never run
  the tracing on the system Python directly. If you see a segfault, delete `.venv/` and
  re-run so the venv rebuilds on 3.12. Confirm `uv` is installed (`brew install uv`).
- **"uv not found."** Install it: `brew install uv` (expected at `/opt/homebrew/bin/uv`).
  The bootstrap cannot proceed without it.
- **SVG too large.** Raise `--filter-speckle` (e.g. 10–15) to drop small specks, and make
  sure `--optimize` is on (svgo typically halves the size). Lowering `--color-precision`
  also shrinks output.
- **Jagged / blocky output.** Raise `--color-precision` (e.g. 8) and/or lower
  `--filter-speckle` so fine detail survives.
- **Background not removed.** Use `--transparent`. If a solid background still remains,
  the flood-fill threshold may be too low for a noisy/gradient background — raise
  `FLOODFILL_THRESH` in `scripts/vectorize.py` (default 50). Note the fill is
  edge-connected, so same-colored *interior* regions are intentionally preserved.
- **Gaps *inside* the art stayed opaque.** `--transparent` only removes background touching
  the border. Enclosed background (between limbs, holes in a logo) needs `--global-key` —
  but read the next entry first.
- **Interior regions turned into holes / creepy voids.** You used `--global-key` on art
  whose interior (eye-whites, teeth, highlights) matches the background color, so the
  color-key erased them; on a dark page they read as black holes. Switch to `--transparent`
  (which preserves interiors), or hand-edit the SVG. See "Transparency: choosing a strategy".
- **`.check.png` skipped ("cairosvg unavailable").** Install cairo: `brew install cairo`.
  The script adds Homebrew's lib dir to `DYLD_FALLBACK_LIBRARY_PATH` automatically; if it
  still can't load, confirm `brew --prefix`/lib contains `libcairo.2.dylib`.
- **svgo skipped.** Node/npx isn't on PATH. Install Node, or run with `--no-optimize` to
  silence the warning. Optimization is optional and never blocks output.
