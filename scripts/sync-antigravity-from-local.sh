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

for skill in antigravity-orchestration peer-pipeline-swarm start begin lifecycle subagent-driven-development \
             writing-plans plan-review comprehensive-code-review fix-check editor-pass baton-pass future-model-router; do
  if [ -d "$src/config/skills/$skill" ]; then
    mkdir -p "$dest/skills/$skill"
    [ -f "$src/config/skills/$skill/SKILL.md" ] && cp "$src/config/skills/$skill/SKILL.md" "$dest/skills/$skill/SKILL.md"
    if [ -d "$src/config/skills/$skill/references" ]; then
      mkdir -p "$dest/skills/$skill/references"
      for ref in "$src/config/skills/$skill/references/"*.md; do
        [ -f "$ref" ] || continue
        base="$(basename "$ref")"
        case "$base" in
          config.md|*.local.md) continue ;;
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

if grep -rnE "geraldonyango|Gerald|[Cc]obalt|zombie|0d3ebe47|[Mm]olly|DL-[0-9X]|cobalthq|futuregerald|/Users/[a-ce-zA-Z]|/Users/d[^e]" "$dest" --exclude=sync-from-local.sh --exclude-dir=__pycache__ --exclude-dir=.pytest_cache; then
  echo "error: personal or org-specific data found in tracked files; move it to a local referenced file" >&2
  exit 1
fi
echo "synced to $dest"
