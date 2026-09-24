# huashu-design — vendoring provenance

`skills/huashu-design` is a vendored copy of a third-party skill. This file records
what it was copied from and how it differs, so a re-sync is mechanical rather than
archaeological.

## Upstream

| | |
|---|---|
| Repository | `https://github.com/alchaincyf/huashu-design` |
| Default branch | `master` |
| Licence | MIT (relicensed upstream 2026-05-14; the previous pin predated this and was Personal-Use-Only) |
| **Current pin** | **`0830494`** — 2026-09-22 |
| Previous pin | `23f60d9` — 2026-04-26 |

Upstream publishes exactly one tag (`v2.0`, 2026-04-21) and has not tagged since,
so syncs pin a **commit SHA**, not a version. Do not look for a release to track.

The `23f60d9 → 0830494` range is 71 commits, 58 files, +13,205 / −2,219, with 36 new
paths. Reproduce with:

```bash
git clone https://github.com/alchaincyf/huashu-design.git /tmp/hs && cd /tmp/hs
git diff --stat 23f60d9..0830494
```

## Removed from this copy (5 paths)

Removed so the repository carries no outbound code that takes a credential. Upstream
ships all of them consent-gated (`--yes` / `HUASHU_CLOUD_OK=1`) and hostname-pinned,
and they are inert without API keys — the omission is a policy choice, not a defect
report.

| Path | Why |
|---|---|
| `scripts/cloud/tts-doubao.mjs` | Sends narration text to `openspeech.bytedance.com` with your key |
| `scripts/cloud/ai-review-video.py` | Sends rendered video segments to `ark.cn-beijing.volces.com` with your key |
| `.env.example` | Existed only to configure the two scripts above |
| `references/ai-video-review.md` | Is the deleted Python script's manual and nothing else |
| `scripts/narrate-pipeline.mjs` | A pure driver for the deleted TTS script — hard-wired at its line 43, called at 147 and 157, with no fallback, so it is dead code the moment that script goes |

**Consequence:** the narrated-video route (SKILL.md step 9.5) still works, but
`voiceover.mp3` and `timeline.json` are user-supplied rather than generated. The
design guidance in `references/voiceover-pipeline.md` — continuous-motion narrative,
hero morph, move-on-pause, Subtitles — is independent of TTS and is kept in full, as
are `scripts/render-narration.sh`, `scripts/mix-voiceover.sh` and
`assets/narration_stage.jsx`.

**Kept deliberately:** `scripts/fetch_images.py`. It reaches `commons.wikimedia.org`
un-gated, but it is a read-only public image search carrying no credential, and the
content-design workflow depends on it. It is this copy's only outbound script.

## Local patches to re-apply on every sync (7)

| # | File | Patch |
|---|---|---|
| 1 | `references/voiceover-pipeline.md` | Banner after the opening blockquote: TTS scripts not vendored, audio and timeline self-supplied. Plus 7 in-file repairs — the ASCII diagram's middle box, the front-matter `voice:` comment, the karaoke `words` dependency, the `三个脚本` table (→ `两个脚本`), the whole `## .env 配置` section (deleted), workflow step 2, and the TTS troubleshooting row |
| 2 | `SKILL.md` | Three lines: the post-render review bullet (step 9), the narration bullet (step 9.5), and the narrated-video routing-table row |
| 3 | `SKILL.md` | `## 版本自检（静默）` neutralised — see below |
| 4 | `SECURITY.md` | TL;DR narrowed, both cloud destination rows and the consent-gate section removed, divergence note added, proxy note reduced to `fetch_images.py`. **The `API keys` section is kept, reduced to the `references/react-setup.md` option B disclosure** — see below |
| 5 | `README.md` / `README.en.md` | The security paragraph in each |
| 6 | `references/hyperframes-backend.md` | Extra warning: `~/.claude/skills` may be a symlink into a plugin repo's working tree |
| 7 | `scripts/render-narration.sh`, `scripts/mix-voiceover.sh`, `assets/narration_stage.jsx` | Reworded text that named the deleted driver. **Patches 7 change shipped-script output**, not just docs: `render-narration.sh:73` is an error message a user reads, and `mix-voiceover.sh:8` is its usage comment |

### Why the version self-check was removed (patch 3)

Upstream's guard is double-gated: it skips when a `.last-update-check` file is under
30 days old, and its step 1 is *"skip if this directory is not a git clone"*. The
misfire is therefore conditional, not certain — it happens only if the agent resolves
"is this a git clone" through `git`'s upward walk rather than by testing
`skills/huashu-design/.git`. In a vendored copy that is a coin flip, and the losing
side compares **this plugin repo** against **its own** origin and tells the user to
`git pull` an unrelated repository. Removed rather than bet on the resolution.

### Why the `API keys` section survives (patch 4)

`references/react-setup.md` is vendored, and its option B asks the user to paste a
real Anthropic key into a demo page input which then POSTs to `api.anthropic.com`
(`react-setup.md:149`, `:154`, `:157`). Upstream's `SECURITY.md` line 24 is the only
place that discloses it. `SECURITY.md` opens by promising to declare every credential
touchpoint, so deleting that line while asserting "no API keys" would make this copy
**less** accurate than upstream. The claim is therefore narrowed to "no outbound code
that takes a credential", which is true, and the disclosure is retained.

Upstream's adjacent claim that keys are read narrowly from `.env` was dropped: it was
an overstatement even upstream — `tts-doubao.mjs:44-55` loaded the entire file into
`process.env`, while only `ai-review-video.py:84` extracted a single variable.

### Not diverged, on purpose

The three-direction gate is upstream behaviour and is **kept as-is**: every new design
returns three direction drafts first, with no exemption for a named style or brand.
Softening it would add a divergence to re-apply on every sync. If it ever needs
softening, add it to the table above rather than patching quietly.

## Re-sync recipe

```bash
SKILL=skills/huashu-design
git clone https://github.com/alchaincyf/huashu-design.git /tmp/hs     # or fetch an existing clone
git -C /tmp/hs log --oneline 0830494..master                          # read what changed since the pin

rm -rf "$SKILL" && mkdir -p "$SKILL"
git -C /tmp/hs archive <new-sha> | tar -x -C "$SKILL"                 # archive, not cp -r: preserves modes,
                                                                      # excludes .git and untracked files
git add -A "$SKILL"                                                   # 36 paths were new last time; a bare
                                                                      # commit records deletions only
```

Then re-delete the 5 paths, re-apply the 7 patches, and re-run the gates:

- **`gate.sh`** — greps the nine removed tokens (`scripts/cloud`, `tts-doubao`,
  `ai-review-video`, `ai-video-review`, `narrate-pipeline`, `.env.example`, `DOUBAO`,
  `ARK_API_KEY`, `HUASHU_CLOUD_OK`) across `*.md *.mjs *.js *.jsx *.sh *.py *.html
  *.json`. **Must return zero hits with no exclusions.** That only works because no
  replacement prose names a deleted filename — keep it that way, and put exact paths
  here instead. Last sync: 31 hits across 9 files before repair, 0 after.
- **A markdown link checker** that resolves each link relative to the containing
  file's directory. Baseline is **1**, not 0: `references/hero-animation-case-study.md`
  links `../demos/hero-animation-v9.mp4`, which upstream ships via GitHub releases
  rather than in the repo. Assert "no new dead links versus baseline".
- `node --check` every `.mjs`/`.js`, `bash -n` every `.sh`, `python3 -m py_compile`
  every `.py`. **Not `.jsx`** — those are Babel-transformed in the browser and are not
  valid plain JS, so a parse check fails on correct files. Review `narration_stage.jsx`
  by reading it.
- `gofmt -l .`, `go vet ./...`, `go test ./...`, `go build` — this proves
  `//go:embed all:skills` (`main.go:41`) still resolves over the new tree, including
  the non-ASCII filename `demos/voiceover-demo/什么是token.html`. It does **not**
  validate this skill's frontmatter: the installer's tests run against synthetic
  in-memory fixtures and never touch the real `skills/` tree.
- Check the frontmatter directly — `description` must stay under 1024 characters
  (it was 164 at this pin).

A link checker cannot detect a dangling reference to a deleted script here, because
those references are inline code spans rather than markdown links. `gate.sh` is the
check that proves the removal was repaired; the link checker only catches link rot.

## Verifying the skill actually loads

`~/.claude/skills` may be a symlink to a *different* checkout of this repository, in
which case invoking the skill loads that copy, not the one you just changed. To test
a worktree's copy:

```bash
mkdir -p .claude/skills
ln -sfn ../../skills/huashu-design .claude/skills/huashu-design
grep -qxF '.claude/' .git/info/exclude || echo '.claude/' >> .git/info/exclude
```

Then start a session with that worktree as the working directory. Remove `.claude/`
afterwards.
