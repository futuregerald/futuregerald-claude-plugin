# Security & Data-Flow Statement

This document exhaustively declares every network destination, credential touchpoint, subprocess, and deletion this skill can perform, so that users and automated security reviewers can verify the claims against the code.

**TL;DR: this vendored copy contains no outbound code that takes a credential. Upstream's cloud scripts are not included in this plugin, so there is no consent gate and no vendor API call. The only outbound request any script makes is an image search against the official Wikimedia Commons API (`scripts/fetch_images.py`). There is no telemetry. No data is ever sent to any server controlled by the skill author. The divergence from upstream is recorded in the repository's `docs/huashu-design-provenance.md`.**

## Complete list of network destinations

| Host | Where | What is sent | When |
|---|---|---|---|
| `commons.wikimedia.org` (official Wikimedia API) | `scripts/fetch_images.py` | Image search keywords; downloads CC/public-domain images with license info printed for review | Only when the agent fetches stock imagery for a content design |
| Brand official websites, `simpleicons.org`, Google favicon service | `references/brand-asset-protocol.md` (instructions, no script) | Plain GET requests to download publicly served logos/brand assets | Only when you ask for a brand-specific design |
| `fonts.googleapis.com`, `unpkg.com` and similar CDNs | Static `<link>`/`<script>` tags inside demo/output HTML | Standard browser font/library fetches when *you* open a generated HTML file | Browser-side only; render scripts work offline-first |

That is the entire list. `grep -rn "https://" --include="*.py" --include="*.mjs" --include="*.js" --include="*.sh" scripts/` to verify.

> **This copy diverges from upstream.** The cloud TTS script, the AI video-review script,
> their driver, that script's reference doc and the environment-variable template are not
> vendored here. Upstream ships them consent-gated and hostname-pinned; this plugin omits
> them so the repository carries no outbound-credential code at all. The exact path list
> and the re-sync recipe are in the repository's `docs/huashu-design-provenance.md`.

## API keys

- No key is hardcoded anywhere, and **no script in this copy reads or transmits a credential.**
- One credential touchpoint remains, and it is a document rather than a script: `references/react-setup.md` option B asks you to paste an Anthropic key into a demo page input, which the page then sends to `api.anthropic.com` from your browser. Upstream marks it local-demo-only and not recommended, and the default options require no key at all. It is declared here because this file promises to declare every credential touchpoint — not because anything runs it for you.

## Subprocesses

All subprocess calls invoke local media tools only: `ffmpeg`, `ffprobe`, `ffplay`, Playwright/Chromium for HTML rendering and screenshots. No shell-to-network combinations, no curl-pipe-sh patterns.

## File deletion

Recursive deletion is limited to temp directories the scripts themselves create with unique timestamp+PID names (`.video-tmp-*`, `.seek-tmp-*`, `_narration/.tmp`, Python `tempfile.TemporaryDirectory`). No script ever deletes user data or anything outside its own scratch space.

## Dependencies

Mainstream registry packages only (`playwright`, `sharp`, `pptxgenjs`, `pdf-lib`, `requests`), installed via standard `npm`/`pip`/`uv` — no binary downloads from arbitrary URLs. One documented exception to be aware of: `npx hyperframes init` (optional animation backend, see `references/hyperframes-backend.md`) installs 19 hyperframes documentation skills into `~/.claude/skills/`. This is called out with a warning in the docs before the command.

## Hooks

`scripts/design-gate-hook.sh` is **never installed automatically** — nothing in this skill writes to `settings.json`. If you manually opt in, its entire behavior is: block long-video render commands (exit 2) until a design-approval file exists. It makes no network calls, writes nothing, deletes nothing.

## Proxy handling note

`fetch_images.py` disables inheriting proxy environment variables (`trust_env = False` / clearing `ALL_PROXY` etc.) for its own requests. This exists to survive stale local proxy configurations that break TLS — not to evade monitoring. If you need these requests to go through your proxy, set it explicitly in the script invocation.

## Reporting

Found something that contradicts this document? Please open an issue — a mismatch between this file and the code is treated as a bug.
