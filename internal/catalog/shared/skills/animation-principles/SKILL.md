---
name: animation-principles
description: Apply Disney's 12 animation principles to any motion work — CSS transitions and keyframes, JavaScript animation (Web Animations API, GSAP, Motion), and Rive. Use whenever building, reviewing or tuning an animation, transition, hover or loading effect, or animated character.
author: Gerald Onyango
tags: [animation, motion, css, javascript, rive, frontend]
---

# Animation Principles

Motion that follows the 12 principles reads as intentional; motion that ignores them reads as
software moving boxes. This skill turns each principle into something you can type, in the tool
the work already uses. It does not pick the tool: use whatever the project has.

## Before animating

1. **Name the job.** Every animation either gives feedback, shows a change of state, guides
   attention, or adds character. If it does none of these, cut it.
2. **Respect reduced motion.** Wrap non-essential motion in
   `@media (prefers-reduced-motion: no-preference)` (CSS) or check
   `matchMedia('(prefers-reduced-motion: reduce)').matches` (JS). Under reduce, swap movement for
   a short fade or an instant change; never remove the state change itself.
3. **Animate cheap properties.** `transform` and `opacity` run on the compositor. Animating
   `width`, `height`, `top`, `left` or `box-shadow` forces layout or paint every frame.

## The 12 principles

| # | Principle | What it means | CSS | JavaScript | Rive |
|---|---|---|---|---|---|
| 1 | Squash and stretch | Volume stays constant as shape deforms | `@keyframes` on `scale(1.15, 0.85)` | `el.animate([{transform:'scale(1)'},{transform:'scale(1.15,0.85)'},{transform:'scale(1)'}], 300)` | Inverse scale X/Y keys on bones |
| 2 | Anticipation | A small move the opposite way first | Keyframe a 5–10% backwards step before the main move | GSAP timeline: short reverse tween, then the main tween | Idle → Anticipate → Action states |
| 3 | Staging | One thing moves at a time, clearly | Dim or blur the background with `opacity`/`filter` | Sequence with a timeline, not parallel tweens | Layers, nested artboards, background opacity input |
| 4 | Straight ahead / pose to pose | Plan key poses, let the tool fill between | Keyframe percentages are the poses | Keyframe arrays or timeline labels | Key poses on the timeline |
| 5 | Follow through and overlap | Parts settle at different times | `transition-delay` / `animation-delay` per child | GSAP `stagger`, Motion `stagger()` | Lagged child bones, spring constraints |
| 6 | Slow in, slow out | Ease, never linear, for physical motion | `cubic-bezier()`, `ease-out` for entrances, `ease-in` for exits | `easing` option, GSAP `power2.out`, Motion springs | Graph Editor curves |
| 7 | Arcs | Natural paths curve | `offset-path` with `offset-distance`, or split X and Y with different easings | Animate X and Y on separate easings, or GSAP MotionPath | IK and path constraints |
| 8 | Secondary action | A smaller motion supports the main one | A second property or pseudo-element on its own timing | A second tween at a lower amplitude | Blend states, a second layer |
| 9 | Timing | Duration carries weight and mood | 100–200ms feedback, 200–400ms transitions, 400ms+ only for large or deliberate moves | Same ranges; scale by distance travelled | Animation duration and playback speed |
| 10 | Exaggeration | Push past real to read clearly | Overshoot: `cubic-bezier(0.34, 1.56, 0.64, 1)` | GSAP `back.out(1.7)`, spring with low damping | Overshoot in curves, rotations past natural |
| 11 | Solid drawing | Keep form and depth consistent | `perspective` and 3D transforms that keep proportions | Same, with consistent `transform-origin` | Bones for volume, clipping for depth |
| 12 | Appeal | Clean, readable, pleasant | Consistent easing and durations across the product | Shared easing and duration tokens | Clean curves, consistent strokes |

## Do / don't

- **Do** define easing and duration as shared tokens (CSS custom properties or a JS constants
  module), so the product moves with one voice. **Don't** hand-tune each animation.
- **Do** make exits faster than entrances (roughly two-thirds of the time). **Don't** make
  someone wait for an element to leave.
- **Do** keep feedback under 200ms. **Don't** animate a button press for longer than it takes to
  read.
- **Do** interrupt cleanly: animations from user input must be reversible mid-flight (CSS
  transitions, WAAPI and GSAP all handle this; CSS `@keyframes` animations do not). **Don't** queue them.

## Tool notes

- **Rive:** editor steps and runtime calls for every principle, plus React wiring, are in
  [references/rive.md](references/rive.md). Read it when the work is in Rive.
- **Deeper tool guides** (GSAP, Motion, anime.js, Lottie, CSS-native, accessible motion) live in
  the opt-in `animation-principles` plugin. Suggest
  `claude plugin enable animation-principles@animation-principles` for large motion work.

## Check before done

- Every animation has a named job (step 1).
- Reduced motion is handled, and the state change still happens under it.
- Only `transform` and `opacity` animate, or the exception is deliberate.
- Durations and easings come from shared tokens.
- Nothing linear except progress bars, spinners and continuous loops.
