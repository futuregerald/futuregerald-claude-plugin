#!/usr/bin/env bash
set -euo pipefail

src="${GEMINI_HOME:-$HOME/.gemini}"
dest="$(cd "$(dirname "$0")" && pwd)"

mkdir -p "$dest/hooks" "$dest/rules" "$dest/skills/antigravity-orchestration/references" "$dest/plugins/antigravity-subagents/agents"

cp "$src/GEMINI.md" "$dest/GEMINI.md"
cp "$src/config/hooks.json" "$dest/hooks.json"
rm -f "$dest/rules/skills-and-delegation.md"
cp "$src/config/skills/antigravity-orchestration/SKILL.md" "$dest/skills/antigravity-orchestration/SKILL.md"
cp "$src/config/skills/antigravity-orchestration/references/config.example.md" "$dest/skills/antigravity-orchestration/references/config.example.md"
cp "$src/config/plugins/antigravity-subagents/plugin.json" "$dest/plugins/antigravity-subagents/plugin.json"
rm -f "$dest/plugins/antigravity-subagents/agents/"*.md
cp "$src/config/plugins/antigravity-subagents/agents/"*.md "$dest/plugins/antigravity-subagents/agents/"

for f in autonomous-breaker.sh autonomous_breaker.py breaker_rules.py replay_breaker.py session_metrics.py \
         test_autonomous_breaker.py test_breaker_rules.py test_replay_breaker.py test_session_metrics.py; do
  cp "$src/config/hooks/$f" "$dest/hooks/$f"
done

if grep -rnE "geraldonyango|Gerald|[Cc]obalt|zombie|0d3ebe47|[Mm]olly|DL-[0-9X]|cobalthq|futuregerald|/Users/[a-ce-zA-Z]|/Users/d[^e]" "$dest" --exclude=sync-from-local.sh --exclude-dir=__pycache__ --exclude-dir=.pytest_cache; then
  echo "error: personal or org-specific data found in tracked files; move it to a local referenced file" >&2
  exit 1
fi
echo "synced to $dest"
