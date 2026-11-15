#!/usr/bin/env bash
set -euo pipefail

src="${GEMINI_HOME:-$HOME/.gemini}"
dest="$(cd "$(dirname "$0")/.." && pwd)/internal/catalog/antigravity"

mkdir -p "$dest/hooks" "$dest/rules" "$dest/skills/antigravity-orchestration/references" "$dest/plugins/antigravity-subagents/agents"

[ -f "$src/GEMINI.md" ] && cp "$src/GEMINI.md" "$dest/GEMINI.md"
if [ -f "$src/config/hooks.json" ]; then
  python3 - "$src/config/hooks.json" "$dest/hooks.json" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    data = json.load(f)
out = {}
if "autonomous-circuit-breaker" in data:
    out["autonomous-circuit-breaker"] = data["autonomous-circuit-breaker"]
with open(sys.argv[2], "w") as f:
    json.dump(out, f, indent=2)
    f.write("\n")
PY
fi
rm -f "$dest/rules/skills-and-delegation.md"
[ -f "$src/config/skills/antigravity-orchestration/references/config.example.md" ] && cp "$src/config/skills/antigravity-orchestration/references/config.example.md" "$dest/skills/antigravity-orchestration/references/config.example.md"

for skill in antigravity-orchestration peer-pipeline-swarm start start-small begin lifecycle subagent-driven-development \
             future-writing-plans plan-review comprehensive-code-review fix-check editor-pass baton-pass future-model-router; do
  if [ -d "$src/config/skills/$skill" ]; then
    mkdir -p "$dest/skills/$skill"
    [ -f "$src/config/skills/$skill/SKILL.md" ] && cp "$src/config/skills/$skill/SKILL.md" "$dest/skills/$skill/SKILL.md"
    [ -f "$src/config/skills/$skill/LICENSE.txt" ] && cp "$src/config/skills/$skill/LICENSE.txt" "$dest/skills/$skill/LICENSE.txt"
    if [ -d "$src/config/skills/$skill/references" ] && { [ "$skill" = "future-model-router" ] || [ ! -d "$dest/../shared/skills/$skill/references" ]; }; then
      mkdir -p "$dest/skills/$skill/references"
      for ref in "$src/config/skills/$skill/references/"*.md; do
        [ -f "$ref" ] || continue
        base="$(basename "$ref")"
        case "$base" in
          config.md|cobalt-org.md|*.local.md) continue ;;
        esac
        cp "$ref" "$dest/skills/$skill/references/$base"
      done
    fi
  fi
done

if [ -f "$src/config/plugins/antigravity-subagents/plugin.json" ]; then
  cp "$src/config/plugins/antigravity-subagents/plugin.json" "$dest/plugins/antigravity-subagents/plugin.json"
  rm -f "$dest/plugins/antigravity-subagents/agents/"*.md
  cp "$src/config/plugins/antigravity-subagents/agents/"*.md "$dest/plugins/antigravity-subagents/agents/"
fi

for f in autonomous-breaker.sh autonomous_breaker.py breaker_rules.py replay_breaker.py session_metrics.py \
         test_autonomous_breaker.py test_breaker_rules.py test_replay_breaker.py test_session_metrics.py; do
  [ -f "$src/config/hooks/$f" ] && cp "$src/config/hooks/$f" "$dest/hooks/$f"
done

python3 - "$dest" <<'PY'
import os
import re
import sys

dest = sys.argv[1]
allowed = (
    "author: Jesse Vincent, Gerald Onyango",
    "Forked by Gerald Onyango",
    "zombie UI",
    "zombie buildings",
)
pat = re.compile(
    r"[Gg]erald|[Oo]nyango|[Cc]obalt|zombie|0d3ebe47|[Mm]olly"
    r"|(?:DL|CPTX|TRIAGE|PRODIDEA|LEADZ)-[0-9X]+"
    r"|cobalthq|futuregerald"
    r"|/Users/[a-ce-zA-Z]|/Users/d[^e]|/home/[a-zA-Z]"
    r"|AIza[0-9A-Za-z_-]{20,}|sk-ant-|ghp_[0-9A-Za-z]{20,}|github_pat_|xox[baprs]-|AKIA[0-9A-Z]{16}"
)
hits = []
for root, dirs, files in os.walk(dest):
    dirs[:] = [d for d in dirs if d not in ("__pycache__", ".pytest_cache")]
    for fname in sorted(files):
        if fname in ("sync-from-local.sh", "LICENSE.txt") or fname.endswith(".pyc"):
            continue
        fpath = os.path.join(root, fname)
        with open(fpath, "r", encoding="utf-8", errors="replace") as fh:
            for lineno, raw in enumerate(fh, 1):
                cleaned = raw
                for phrase in allowed:
                    cleaned = cleaned.replace(phrase, "")
                if pat.search(cleaned):
                    hits.append(f"{fpath}:{lineno}:{raw.rstrip()}")
if hits:
    for h in hits:
        print(h, file=sys.stderr)
    print("error: personal, org-specific, or secret data found in tracked files; move it to a local referenced file", file=sys.stderr)
    sys.exit(1)
PY
echo "synced to $dest"
