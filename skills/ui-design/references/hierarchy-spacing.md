# Hierarchy & Spacing Reference

## Visual Hierarchy Deep Dive

### The Hierarchy Problem

When everything in an interface competes for attention, it feels noisy and chaotic. The solution is deliberately de-emphasizing secondary and tertiary information while highlighting what's most important.

### Three Tools for Hierarchy

#### 1. Font Size

Don't rely on size alone—it leads to primary content that's too large and secondary content that's too small.

#### 2. Font Weight

- **Normal text**: 400 or 500 (depending on font)
- **Emphasized text**: 600 or 700
- **Never use <400** for UI work—too hard to read at small sizes

#### 3. Color (Contrast)

Use 2-3 colors for text hierarchy:

```
Primary:   hsl(0, 0%, 10%)   // Near black - headlines, important content
Secondary: hsl(0, 0%, 45%)   // Medium grey - supporting text, dates
Tertiary:  hsl(0, 0%, 65%)   // Light grey - less important info, footer
```

### Colored Backgrounds

Grey text on colored backgrounds looks bad. Instead:

1. Pick a color with the same hue as background
2. Adjust saturation and lightness
3. Hand-pick until it looks right

Don't use white with reduced opacity—it looks washed out and shows through on images/patterns.

### Labels Strategy

**Skip labels when possible:**

- `janedoe@example.com` is obviously an email
- `(555) 765-4321` is obviously a phone number
- `$19.99` is obviously a price

**Combine labels with values:**

- ❌ "In stock: 12"
- ✅ "12 left in stock"
- ❌ "Bedrooms: 3"
- ✅ "3 bedrooms"

**When labels are necessary:**

- Make them secondary (smaller, lighter, lower weight)
- The data matters more than the label
- Exception: info-dense pages where users scan for labels (e.g., product specs)

### Button Hierarchy

```
Primary:   background-color: blue-600; color: white;
Secondary: border: 1px solid gray-300; background: white;
Tertiary:  color: gray-600; (link style, no background/border)
```

**Destructive actions:**

- Not the primary action? Use secondary/tertiary styling
- Save bold red for the confirmation step

### Semantic vs Document Hierarchy

Don't let HTML tags dictate visual treatment:

- An h1 doesn't have to be huge
- Section titles are often better small—content should be the focus
- Style for visual hierarchy, not semantic meaning

---

## Spacing Systems

### The Problem with Arbitrary Values

Choosing between 120px and 125px wastes time and creates inconsistent designs.

### Building a Scale

Start with 16px (browser default), use factors and multiples:

```css
--space-1: 4px; /* 0.25rem */
--space-2: 8px; /* 0.5rem */
--space-3: 12px; /* 0.75rem */
--space-4: 16px; /* 1rem - base */
--space-5: 24px; /* 1.5rem */
--space-6: 32px; /* 2rem */
--space-7: 48px; /* 3rem */
--space-8: 64px; /* 4rem */
--space-9: 96px; /* 6rem */
--space-10: 128px; /* 8rem */
--space-11: 192px; /* 12rem */
--space-12: 256px; /* 16rem */
```

Adjacent values differ by ~25% minimum—noticeable but not jarring.

### Using the System

1. Start with a guess
2. Try values on either side
3. Pick the one that looks best
4. Process of elimination beats pixel-perfect tweaking

### White Space Philosophy

**Start with too much, remove until happy.**

Default approach (adding space) gives minimum breathing room. Starting generous produces cleaner results.

Dense UIs (dashboards) are the exception—make density a deliberate choice.

---

## Layout Principles

### Don't Fill the Screen

- If content needs 600px, use 600px
- Extra space around edges is fine
- Individual sections don't need to match container width

### Mobile First

Start with ~400px canvas. Design mobile, then adapt upward. You'll change less than expected.

### Columns Over Width

Narrow form that feels unbalanced? Split into columns (form + supporting text) instead of widening the form.

### Fixed vs Fluid

**Grids (fluid/percentage) work for:**

- Main content areas
- Card grids
- Responsive layouts

**Fixed widths work better for:**

- Sidebars (optimize for content)
- Forms (optimal input width)
- Navigation

### Sizing Doesn't Scale Proportionally

```
Desktop headline: 45px (2.5× body at 18px)
Mobile headline:  20-24px (1.5-1.7× body at 14px)
```

The ratio changes. Large elements shrink faster than small elements.

**Same for components:**

- Large buttons need more generous padding
- Small buttons need tighter padding
- Don't scale proportionally—adjust each size independently

### Ambiguous Spacing

**Problem:** Equal spacing between label and input, and between form groups, makes grouping unclear.

**Solution:** More space around groups than within them.

```css
/* Form example */
.form-label {
  margin-bottom: 4px;
}
.form-group {
  margin-bottom: 24px;
}
```

This applies to:

- Form labels/inputs
- Section headings
- List items
- Horizontal component groups
