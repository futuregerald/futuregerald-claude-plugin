#!/usr/bin/env bash
set -euo pipefail

src="${GEMINI_HOME:-$HOME/.gemini}"
dest="$(cd "$(dirname "$0")" && pwd)"

mkdir -p "$dest/hooks" "$dest/rules" "$dest/skills/antigravity-orchestration/references"

cp "$src/GEMINI.md" "$dest/GEMINI.md"
cp "$src/config/hooks.json" "$dest/hooks.json"
[ -f "$src/config/rules/skills-and-delegation.md" ] && cp "$src/config/rules/skills-and-delegation.md" "$dest/rules/skills-and-delegation.md"
cp "$src/config/skills/antigravity-orchestration/SKILL.md" "$dest/skills/antigravity-orchestration/SKILL.md"
cp "$src/config/skills/antigravity-orchestration/references/config.example.md" "$dest/skills/antigravity-orchestration/references/config.example.md"

for f in autonomous-breaker.sh autonomous_breaker.py breaker_rules.py replay_breaker.py session_metrics.py \
         test_autonomous_breaker.py test_breaker_rules.py test_replay_breaker.py test_session_metrics.py; do
  cp "$src/config/hooks/$f" "$dest/hooks/$f"
done

if grep -rnE "geraldonyango|Gerald|[Cc]obalt|zombie|0d3ebe47|Molly|DL-[0-9X]|cobalthq|futuregerald" "$dest" --exclude=sync-from-local.sh --exclude-dir=__pycache__; then
  echo "error: personal or org-specific data found in tracked files; move it to a local referenced file" >&2
  exit 1
fi
echo "synced to $dest"
