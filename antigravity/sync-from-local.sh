#!/usr/bin/env bash
set -euo pipefail

src="${GEMINI_HOME:-$HOME/.gemini}"
dest="$(cd "$(dirname "$0")" && pwd)"

rm -rf "$dest/hooks" "$dest/rules" "$dest/skills"
mkdir -p "$dest/hooks" "$dest/rules" "$dest/skills/antigravity-orchestration"

cp "$src/GEMINI.md" "$dest/GEMINI.md"
cp "$src/config/hooks.json" "$dest/hooks.json"
cp "$src/config/rules/skills-and-delegation.md" "$dest/rules/"
cp "$src/config/skills/antigravity-orchestration/SKILL.md" "$dest/skills/antigravity-orchestration/"
for f in autonomous-breaker.sh autonomous_breaker.py breaker_rules.py replay_breaker.py session_metrics.py \
         test_autonomous_breaker.py test_breaker_rules.py test_replay_breaker.py test_session_metrics.py; do
  cp "$src/config/hooks/$f" "$dest/hooks/$f"
done

files=("$dest/GEMINI.md" "$dest/hooks.json" "$dest/rules/"*.md "$dest/skills/antigravity-orchestration/SKILL.md" "$dest/hooks/"*)

for f in "${files[@]}"; do
  sed -E -i '' \
    -e 's|^BREAKER_SCRIPT = .*$|BREAKER_SCRIPT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "autonomous-breaker.sh")|' \
    -e 's|/Users/[A-Za-z0-9_.-]+/|/Users/you/|g' \
    -e 's|\(cloud ID `[0-9a-f-]+`; projects [^)]*\)|(cloud ID and project keys in your local config)|g' \
    -e 's|[a-z0-9-]+\.atlassian\.net|your-site.atlassian.net|g' \
    -e 's|`Closes DL-XXXX` on Cobalt|`Closes PROJ-XXXX` on work repos|g' \
    -e 's|cobalt-structured-logging|structured-logging|g' \
    -e 's|cobalt-review-action|review-corpus|g' \
    -e 's|cobalt-methodologies|example-repo|g' \
    -e 's|cobalthq|work-org|g' \
    -e 's|Ruby/Rails \(Cobalt\)|Ruby/Rails (work repos)|g' \
    -e 's|Cobalt repos|work-org repos|g' \
    -e 's|Cobalt branches|Work branches|g' \
    -e 's|Cobalt|work-org|g' \
    -e "s|Gerald's|the user's|g" \
    -e 's|Gerald|the user|g' \
    "$f"
done

if grep -rnE "geraldonyango|Gerald|[Cc]obalt|zombie|0d3ebe47|Molly|DL-[0-9X]" "$dest" --exclude=sync-from-local.sh; then
  echo "redaction check failed" >&2
  exit 1
fi
echo "synced to $dest"
