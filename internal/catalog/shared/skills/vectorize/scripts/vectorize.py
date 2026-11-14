#!/usr/bin/env python3
"""vectorize.py -- convert raster images (PNG/JPG) into clean SVG vector art.

This is the ORCHESTRATOR. It is designed to be launched with whatever python
happens to be on PATH (which on this machine is 3.14, where the vtracer native
wheel SEGFAULTS -- see the module docstring / SKILL.md troubleshooting).

To sidestep that, the orchestrator does NOT import vtracer/pillow/cairosvg
itself. Instead it:

  1. Bootstraps (once) a cached Python 3.12 virtualenv via `uv`, into which it
     installs vtracer + pillow (+ cairosvg for verification).
  2. Re-invokes itself with that venv's python via the hidden `__worker__`
     subcommand. When running as the worker, the heavy libraries import fine
     because we're now on 3.12.

So: `python3 vectorize.py <input> ...` (any python) transparently hands the
real work to the 3.12 venv. Subsequent runs reuse the cached venv and are fast.

Proven workflow encoded here (Apple Silicon macOS):
  * COLOR art  -> vtracer (color mode, stacked, spline).
  * MONO art   -> potrace CLI if available, else vtracer greyscale.
  * Optional edge flood-fill background removal (--transparent).
  * svgo optimize pass (npx --yes svgo --multipass), skipped gracefully if absent.
  * cairosvg verification render to <name>.check.png, skipped gracefully if absent.
"""

from __future__ import annotations

import argparse
import glob
import os
import shutil
import subprocess
import sys
from pathlib import Path

# --------------------------------------------------------------------------- #
# Configuration / constants
# --------------------------------------------------------------------------- #

# Stable cache location for the bootstrapped 3.12 venv. Lives inside the skill
# dir so it is discoverable and easy to nuke if it ever gets corrupted.
SKILL_DIR = Path(__file__).resolve().parent.parent
VENV_DIR = SKILL_DIR / ".venv"
VENV_PY = VENV_DIR / "bin" / "python"

UV_BIN = shutil.which("uv") or "/opt/homebrew/bin/uv"

# The Python version that actually works. vtracer's wheel segfaults on 3.14.
TARGET_PYTHON = "3.12"

# Packages the worker needs. cairosvg is for the optional --verify step.
WORKER_PACKAGES = ["vtracer", "pillow", "cairosvg"]

IMAGE_EXTS = ("*.png", "*.jpg", "*.jpeg", "*.PNG", "*.JPG", "*.JPEG")

# Proven vtracer defaults for flat/vector-style art.
DEFAULT_FILTER_SPECKLE = 6
DEFAULT_COLOR_PRECISION = 7
DEFAULT_CORNER_THRESHOLD = 60
DEFAULT_PATH_PRECISION = 3

# Sentinel color used for edge flood-fill background removal. Chosen to be an
# unlikely-in-art magenta so we can reliably map it to transparency afterward.
SENTINEL = (255, 0, 255)
FLOODFILL_THRESH = 50


def log(msg: str) -> None:
    print(msg, flush=True)


def err(msg: str) -> None:
    print(msg, file=sys.stderr, flush=True)


# --------------------------------------------------------------------------- #
# Venv bootstrap (runs in the orchestrator, on whatever python invoked us)
# --------------------------------------------------------------------------- #

def ensure_venv() -> Path:
    """Create (once) and return the path to the cached 3.12 venv python.

    Uses `uv` to build a 3.12 venv and install the worker packages. Reused on
    subsequent runs. Raises SystemExit with a clear message if uv is missing.
    """
    if VENV_PY.exists() and _venv_has_vtracer():
        return VENV_PY

    if not (shutil.which("uv") or Path(UV_BIN).exists()):
        err(
            "ERROR: `uv` is required to bootstrap the Python 3.12 environment "
            "but was not found.\n"
            "       vtracer's wheel segfaults on Python 3.14, so we pin 3.12 via uv.\n"
            "       Install it with:  brew install uv\n"
            "       (expected at /opt/homebrew/bin/uv)"
        )
        raise SystemExit(2)

    uv = shutil.which("uv") or UV_BIN

    if not VENV_PY.exists():
        log(f"[bootstrap] creating Python {TARGET_PYTHON} venv at {VENV_DIR} ...")
        _run([uv, "venv", "--python", TARGET_PYTHON, str(VENV_DIR)])

    log(f"[bootstrap] installing {', '.join(WORKER_PACKAGES)} into venv ...")
    _run([uv, "pip", "install", "--python", str(VENV_PY), *WORKER_PACKAGES])

    if not _venv_has_vtracer():
        err("ERROR: vtracer failed to import in the bootstrapped venv.")
        raise SystemExit(2)

    log("[bootstrap] environment ready.")
    return VENV_PY


def _venv_has_vtracer() -> bool:
    if not VENV_PY.exists():
        return False
    proc = subprocess.run(
        [str(VENV_PY), "-c", "import vtracer, PIL"],
        capture_output=True,
    )
    return proc.returncode == 0


def _run(cmd: list[str]) -> None:
    proc = subprocess.run(cmd, capture_output=True, text=True)
    if proc.returncode != 0:
        err(f"Command failed ({' '.join(cmd)}):\n{proc.stdout}\n{proc.stderr}")
        raise SystemExit(proc.returncode)


# --------------------------------------------------------------------------- #
# Orchestrator: collect inputs, dispatch each into the worker
# --------------------------------------------------------------------------- #

def collect_inputs(path: str) -> list[Path]:
    p = Path(path)
    if p.is_dir():
        files: list[Path] = []
        for pat in IMAGE_EXTS:
            files.extend(sorted(p.glob(pat)))
        return sorted(set(files))
    if p.is_file():
        return [p]
    err(f"ERROR: input not found: {path}")
    raise SystemExit(2)


def run_orchestrator(args: argparse.Namespace) -> int:
    inputs = collect_inputs(args.input)
    if not inputs:
        err(f"ERROR: no PNG/JPG images found at {args.input}")
        return 2

    venv_py = ensure_venv()

    outdir = Path(args.outdir) if args.outdir else None
    if outdir:
        outdir.mkdir(parents=True, exist_ok=True)

    failures = 0
    for src in inputs:
        dst = (outdir or src.parent) / (src.stem + ".svg")
        # Hand this single file to the worker running under the 3.12 venv.
        worker_cmd = [
            str(venv_py), os.path.abspath(__file__), "__worker__",
            "--input", str(src),
            "--output", str(dst),
            "--mode", args.mode,
            "--filter-speckle", str(args.filter_speckle),
            "--color-precision", str(args.color_precision),
        ]
        if args.transparent:
            worker_cmd.append("--transparent")
        if args.global_key:
            worker_cmd.append("--global-key")

        proc = subprocess.run(worker_cmd)
        if proc.returncode != 0:
            err(f"FAILED: {src}")
            failures += 1
            continue

        # Optional post-steps run from the orchestrator (external CLIs).
        if not args.no_optimize:
            optimize_svg(dst)
        if not args.no_verify:
            verify_svg(venv_py, dst)

        _print_summary(src, dst)

    if failures:
        err(f"\n{failures} file(s) failed.")
        return 1
    return 0


def _print_summary(src: Path, dst: Path) -> None:
    src_kb = src.stat().st_size / 1024
    dst_kb = dst.stat().st_size / 1024 if dst.exists() else 0
    log(f"  OK  {src.name} ({src_kb:.1f} KB)  ->  {dst.name} ({dst_kb:.1f} KB)")


# --------------------------------------------------------------------------- #
# Optional external steps (svgo optimize, cairosvg verify)
# --------------------------------------------------------------------------- #

def optimize_svg(svg: Path) -> None:
    """Run svgo via npx to shrink the SVG. Warn + skip if unavailable."""
    if not shutil.which("npx"):
        log("  [optimize] npx not found; skipping svgo optimization.")
        return
    before = svg.stat().st_size / 1024
    proc = subprocess.run(
        ["npx", "--yes", "svgo", "--multipass", str(svg)],
        capture_output=True, text=True,
    )
    if proc.returncode != 0:
        log(f"  [optimize] svgo unavailable/failed; skipping. ({proc.stderr.strip()[:120]})")
        return
    after = svg.stat().st_size / 1024
    log(f"  [optimize] svgo: {before:.1f} KB -> {after:.1f} KB")


def verify_svg(venv_py: Path, svg: Path) -> None:
    """Rasterize the SVG back to <name>.check.png via cairosvg. Warn + skip on failure."""
    check_png = svg.with_suffix(".check.png")
    code = (
        "import sys\n"
        "try:\n"
        "    import cairosvg\n"
        "except Exception as e:\n"
        "    sys.stderr.write('cairosvg unavailable: %s' % e); sys.exit(3)\n"
        "cairosvg.svg2png(url=sys.argv[1], write_to=sys.argv[2], output_width=512)\n"
    )
    # cairosvg (via cffi.dlopen) can't find Homebrew's libcairo unless the
    # loader path includes it. Inject Homebrew's lib dir so verification works
    # out of the box on Apple Silicon macOS.
    env = os.environ.copy()
    brew_lib = _brew_lib_dir()
    if brew_lib:
        existing = env.get("DYLD_FALLBACK_LIBRARY_PATH", "")
        env["DYLD_FALLBACK_LIBRARY_PATH"] = (
            f"{brew_lib}:{existing}" if existing else brew_lib
        )

    proc = subprocess.run(
        [str(venv_py), "-c", code, str(svg), str(check_png)],
        capture_output=True, text=True, env=env,
    )
    if proc.returncode != 0:
        log(f"  [verify] cairosvg unavailable/failed; skipping. ({proc.stderr.strip()[:120]})")
        return
    log(f"  [verify] wrote {check_png.name} for eyeball check.")


def _brew_lib_dir() -> str | None:
    """Return Homebrew's lib dir (holds libcairo) if brew is present."""
    brew = shutil.which("brew")
    if not brew:
        # Common Apple Silicon default.
        default = "/opt/homebrew/lib"
        return default if Path(default).is_dir() else None
    proc = subprocess.run([brew, "--prefix"], capture_output=True, text=True)
    if proc.returncode != 0:
        return None
    lib = Path(proc.stdout.strip()) / "lib"
    return str(lib) if lib.is_dir() else None


# --------------------------------------------------------------------------- #
# Worker: runs under the 3.12 venv, imports vtracer/pillow, does the tracing
# --------------------------------------------------------------------------- #

def run_worker(args: argparse.Namespace) -> int:
    import tempfile

    src = Path(args.input)
    dst = Path(args.output)
    dst.parent.mkdir(parents=True, exist_ok=True)

    # Decide engine.
    mode = args.mode
    if mode == "auto":
        mode = _detect_mode(src)

    # Prepare the image to trace (optionally with background removed).
    trace_src = src
    tmp = None
    if args.transparent or args.global_key:
        tmp = Path(tempfile.mkstemp(suffix=".png")[1])
        if args.global_key:
            _remove_background_global(src, tmp)
        else:
            _remove_background(src, tmp)
        trace_src = tmp

    try:
        if mode == "mono":
            _trace_mono(trace_src, dst, args)
        else:
            _trace_color(trace_src, dst, args)
    finally:
        if tmp and tmp.exists():
            tmp.unlink()

    if not dst.exists() or dst.stat().st_size == 0:
        err(f"ERROR: no SVG produced for {src}")
        return 1
    return 0


def _detect_mode(src: Path) -> str:
    """Heuristic: mostly black/white/greyscale -> mono, else color."""
    from PIL import Image
    im = Image.open(src).convert("RGB")
    im.thumbnail((64, 64))
    raw = im.tobytes()  # small thumbnail (RGB), so cheap; supported API
    total = len(raw) // 3
    colorful = 0
    for i in range(0, len(raw), 3):
        r, g, b = raw[i], raw[i + 1], raw[i + 2]
        if max(abs(r - g), abs(g - b), abs(r - b)) > 24:
            colorful += 1
    return "color" if total and (colorful / total) > 0.05 else "mono"


def _remove_background(src: Path, dst: Path) -> None:
    """Edge-seeded flood fill -> transparency.

    Flood-fill from the 4 corners AND 4 edge midpoints with a sentinel color
    (connected-region fill, thresh~50), then map sentinel pixels to alpha 0.
    Because the fill only follows connected regions from the edges, same-colored
    interior areas are NOT punched out.
    """
    from PIL import Image, ImageDraw

    im = Image.open(src).convert("RGB")
    w, h = im.size
    draw = ImageDraw.Draw(im)

    seeds = [
        (0, 0), (w - 1, 0), (0, h - 1), (w - 1, h - 1),   # corners
        (w // 2, 0), (w // 2, h - 1), (0, h // 2), (w - 1, h // 2),  # edge midpoints
    ]
    for seed in seeds:
        ImageDraw.floodfill(im, seed, SENTINEL, thresh=FLOODFILL_THRESH)

    rgba = im.convert("RGBA")
    px = rgba.load()
    for y in range(h):
        for x in range(w):
            r, g, b, _ = px[x, y]
            if (r, g, b) == SENTINEL:
                px[x, y] = (0, 0, 0, 0)
    rgba.save(dst)


def _remove_background_global(src: Path, dst: Path) -> None:
    """Global color-key background removal -> transparency.

    Samples the background color from the 4 corners (median), then maps EVERY
    pixel within FLOODFILL_THRESH of that color to alpha 0 -- regardless of
    whether it is connected to the edge. This clears background trapped INSIDE
    the artwork (gaps between limbs, holes in a logo) that the edge flood fill
    leaves opaque.

    TRADEOFF: because it is not connected-region aware, it also erases enclosed
    regions that happen to share the background color -- e.g. cream/white eye
    interiors on a light background, which then read as holes (creepy voids on a
    dark page). Use --transparent (edge flood fill) when the art has same-colored
    interior regions you must preserve; use --global-key only when interior gaps
    must be cleared and no interior region matches the background.
    """
    from PIL import Image

    im = Image.open(src).convert("RGB")
    w, h = im.size
    px = im.load()

    # Estimate background color from the corners (median per channel is robust
    # to a single stray corner pixel).
    corners = [px[0, 0], px[w - 1, 0], px[0, h - 1], px[w - 1, h - 1]]
    bg = tuple(sorted(c[i] for c in corners)[len(corners) // 2] for i in range(3))

    rgba = im.convert("RGBA")
    rpx = rgba.load()
    thr = FLOODFILL_THRESH
    for y in range(h):
        for x in range(w):
            r, g, b = px[x, y]
            if abs(r - bg[0]) <= thr and abs(g - bg[1]) <= thr and abs(b - bg[2]) <= thr:
                rpx[x, y] = (0, 0, 0, 0)
    rgba.save(dst)


def _trace_color(src: Path, dst: Path, args: argparse.Namespace) -> None:
    import vtracer
    vtracer.convert_image_to_svg_py(
        str(src),
        str(dst),
        colormode="color",
        hierarchical="stacked",
        mode="spline",
        filter_speckle=args.filter_speckle,
        color_precision=args.color_precision,
        corner_threshold=DEFAULT_CORNER_THRESHOLD,
        path_precision=DEFAULT_PATH_PRECISION,
    )


def _trace_mono(src: Path, dst: Path, args: argparse.Namespace) -> None:
    """Prefer potrace CLI for crisp line art; fall back to vtracer greyscale."""
    if shutil.which("potrace"):
        _trace_mono_potrace(src, dst, args)
    else:
        log("  [mono] potrace not found; using vtracer greyscale (binary) instead.")
        import vtracer
        vtracer.convert_image_to_svg_py(
            str(src),
            str(dst),
            colormode="binary",
            hierarchical="stacked",
            mode="spline",
            filter_speckle=args.filter_speckle,
            color_precision=args.color_precision,
            corner_threshold=DEFAULT_CORNER_THRESHOLD,
            path_precision=DEFAULT_PATH_PRECISION,
        )


def _trace_mono_potrace(src: Path, dst: Path, args: argparse.Namespace) -> None:
    """potrace needs a bitmap (PBM/PGM). Convert with Pillow first, then trace to SVG."""
    import tempfile
    from PIL import Image

    pgm = Path(tempfile.mkstemp(suffix=".pgm")[1])
    try:
        Image.open(src).convert("L").save(pgm)
        proc = subprocess.run(
            ["potrace", "-s", "-o", str(dst), str(pgm)],
            capture_output=True, text=True,
        )
        if proc.returncode != 0:
            raise RuntimeError(f"potrace failed: {proc.stderr}")
    finally:
        if pgm.exists():
            pgm.unlink()


# --------------------------------------------------------------------------- #
# CLI
# --------------------------------------------------------------------------- #

def build_worker_parser() -> argparse.ArgumentParser:
    """Hidden parser used only when re-invoked internally under the 3.12 venv."""
    w = argparse.ArgumentParser(prog="vectorize.py __worker__", add_help=False)
    w.add_argument("--input", required=True)
    w.add_argument("--output", required=True)
    w.add_argument("--mode", default="color")
    w.add_argument("--transparent", action="store_true")
    w.add_argument("--global-key", action="store_true")
    w.add_argument("--filter-speckle", type=int, default=DEFAULT_FILTER_SPECKLE)
    w.add_argument("--color-precision", type=int, default=DEFAULT_COLOR_PRECISION)
    return w


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        prog="vectorize.py",
        description="Convert raster images (PNG/JPG) into clean SVG vector art.",
    )
    p.add_argument("input", nargs="?", help="input PNG/JPG file OR a directory")
    p.add_argument("-o", "--outdir", help="output directory (default: alongside input)")
    p.add_argument("--transparent", action="store_true",
                   help="remove background via edge flood fill (safe: preserves "
                        "same-colored interior regions; leaves enclosed background opaque)")
    p.add_argument("--global-key", "--transparent-aggressive", action="store_true",
                   dest="global_key",
                   help="remove background via global color-key (clears enclosed gaps "
                        "too, but ERASES interior regions that match the background)")
    p.add_argument("--mode", choices=["color", "mono", "auto"], default="color",
                   help="tracing engine: color (vtracer), mono (potrace/greyscale), or auto")
    p.add_argument("--filter-speckle", type=int, default=DEFAULT_FILTER_SPECKLE,
                   help=f"drop specks smaller than N px (default {DEFAULT_FILTER_SPECKLE}; higher = smaller file)")
    p.add_argument("--color-precision", type=int, default=DEFAULT_COLOR_PRECISION,
                   help=f"color bits of precision (default {DEFAULT_COLOR_PRECISION}; higher = more faithful)")
    p.add_argument("--no-optimize", action="store_true", help="skip svgo optimization pass")
    p.add_argument("--no-verify", action="store_true", help="skip cairosvg .check.png verification render")
    return p


def main(argv: list[str]) -> int:
    # Internal re-invocation under the 3.12 venv. Intercept before the public
    # parser so the image path is never mistaken for a subcommand.
    if argv and argv[0] == "__worker__":
        args = build_worker_parser().parse_args(argv[1:])
        return run_worker(args)

    parser = build_parser()
    args = parser.parse_args(argv)
    if not args.input:
        parser.print_help()
        return 2

    return run_orchestrator(args)


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
