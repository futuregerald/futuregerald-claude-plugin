# Shared UI Implementation Conventions

These conventions apply to all frameworks. Framework-specific instructions are in the corresponding `react.md`, `vue.md`, or `svelte.md` file.

## Styling

**Detect the project's styling approach before writing any CSS:**

1. Check `package.json` for `tailwindcss`
2. If Tailwind is installed, check the version:
   - **Tailwind 4+**: Uses CSS-based config (`@import "tailwindcss"`), no `tailwind.config.js`. Use utility classes directly. Avoid `@apply` (still works but discouraged).
   - **Tailwind 3.x**: Uses `tailwind.config.js`. Utility-first, `@apply` sparingly. Check the existing config for theme customizations.
3. If Tailwind is **not** installed, use the project's existing styling approach:
   - CSS Modules (`.module.css`)
   - Scoped styles (Vue `<style scoped>`, Svelte `<style>`)
   - Plain CSS / SCSS
   - Check existing component files to match the established pattern

**Never introduce a styling library the project doesn't already use.**

## Design Matching Checklist

Match the design reference exactly on these dimensions:

| Dimension | What to check |
|-----------|---------------|
| **Colors** | Theme tokens or exact hex/rgb values. Check backgrounds, text, borders, shadows. |
| **Typography** | Font family, size, weight, line-height, letter-spacing. Check headings, body, labels. |
| **Spacing** | Padding, margin, gap. Maintain consistent spacing scale. |
| **Layout** | Flexbox vs grid, alignment, wrapping, responsive breakpoints. |
| **Visual elements** | Border width/radius, box-shadow, opacity, gradients, dividers. |
| **Interactive states** | Hover, focus, active, disabled. Match transitions and cursors. |
| **Icons & images** | Correct icons, sizing, aspect ratios, object-fit. |

## Accessibility (WCAG 2.1 AA)

Every component must meet these requirements:

- **Semantic HTML**: Use `<button>`, `<nav>`, `<main>`, `<article>`, `<header>`, `<footer>` -- not `<div>` with click handlers
- **ARIA attributes**: Add `aria-label`, `aria-expanded`, `aria-hidden`, `role` where semantic HTML is insufficient
- **Keyboard navigation**: All interactive elements reachable via Tab, operable via Enter/Space, dismissible via Escape
- **Focus management**: Visible focus indicators, logical focus order, focus trapping in modals/dialogs
- **Color contrast**: Minimum 4.5:1 for normal text, 3:1 for large text (18px+ bold or 24px+ regular)
- **Screen reader**: Meaningful alt text for images, `sr-only` text for icon-only buttons, live regions for dynamic content

## Responsive Design

- **Mobile-first**: Start with mobile layout, add breakpoints for larger screens
- **Breakpoints**: Use the project's existing breakpoint system. If none exists, use standard breakpoints (640px, 768px, 1024px, 1280px)
- **Test at**: 375px (mobile), 768px (tablet), 1280px (desktop) minimum

## Quality Checks (Shared)

These commands apply to all frameworks:

```bash
npm run lint        # Linting
npm run build       # Production build
```

**Typecheck commands are framework-specific** -- see the framework file for the correct command.

## File Organization

- One component per file
- Co-locate tests, types, and styles with the component when the project convention supports it
- Use PascalCase for component file names
- Follow the project's existing directory structure -- do not invent a new layout
