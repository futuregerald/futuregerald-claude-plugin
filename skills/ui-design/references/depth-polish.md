# Depth & Polish Reference

## Creating Depth

### Light Source Principle

Light comes from above. Use this to make elements feel raised or inset.

**Raised elements:**

- Lighter top edge (faces sky)
- Shadow below (blocks light underneath)

**Inset elements:**

- Shadow at top (lip blocks light)
- Lighter bottom edge (faces up)

### Raised Element Recipe

```css
.raised-button {
  /* Lighter top edge */
  border-top: 1px solid hsl(210, 100%, 55%);
  /* or */
  box-shadow: inset 0 1px 0 hsl(210, 100%, 60%);

  /* Shadow below */
  box-shadow: 0 1px 3px hsla(0, 0%, 0%, 0.2);
}
```

**Tips:**

- Hand-pick the lighter color—don't use semi-transparent white (desaturates)
- Keep blur radius small (2-3px)—shadows should be sharp like real shadows

### Inset Element Recipe

```css
.inset-well {
  /* Shadow at top */
  box-shadow: inset 0 2px 4px hsla(0, 0%, 0%, 0.1);

  /* Lighter bottom edge */
  border-bottom: 1px solid hsl(0, 0%, 95%);
}
```

Works for: text inputs, checkboxes, wells/containers

---

## Shadow System

### Elevation Concept

Shadows communicate z-axis position:

- Small shadow = slightly raised
- Large shadow = close to user

### Define 5 Shadow Levels

```css
--shadow-sm: 0 1px 2px hsla(0, 0%, 0%, 0.05);
--shadow: 0 1px 3px hsla(0, 0%, 0%, 0.1), 0 1px 2px hsla(0, 0%, 0%, 0.06);
--shadow-md: 0 4px 6px hsla(0, 0%, 0%, 0.1), 0 2px 4px hsla(0, 0%, 0%, 0.06);
--shadow-lg: 0 10px 15px hsla(0, 0%, 0%, 0.1), 0 4px 6px hsla(0, 0%, 0%, 0.05);
--shadow-xl: 0 20px 25px hsla(0, 0%, 0%, 0.1), 0 10px 10px hsla(0, 0%, 0%, 0.04);
```

### Use Cases by Elevation

- **sm**: Buttons, subtle cards
- **default**: Hoverable cards
- **md**: Dropdowns, popovers
- **lg**: Floating panels
- **xl**: Modals, dialogs

### Two-Part Shadows

Combine two shadows for realistic effect:

1. **Direct light shadow**: Large, soft, vertical offset
   - Simulates light source casting shadow behind object

2. **Ambient shadow**: Small, tight, less offset
   - Simulates dark area directly underneath where ambient light can't reach

```css
.card {
  box-shadow:
    0 10px 15px hsla(0, 0%, 0%, 0.1),
    /* Direct light */ 0 4px 6px hsla(0, 0%, 0%, 0.05); /* Ambient */
}
```

**At higher elevations:** Ambient shadow becomes more subtle or invisible.

### Interactive Shadows

- **Drag item**: Increase shadow to show it's "lifted"
- **Button press**: Decrease/remove shadow to show it's "pushed down"

---

## Flat Design Depth

### Color-Based Depth

- Lighter than background = raised
- Darker than background = inset

```css
.raised-card {
  background: hsl(0, 0%, 100%); /* Lighter than page */
}

.inset-well {
  background: hsl(0, 0%, 93%); /* Darker than page */
}
```

### Solid Shadows

No blur, just offset:

```css
.flat-card {
  box-shadow: 0 4px 0 hsl(0, 0%, 80%);
}
```

Maintains flat aesthetic while adding dimension.

### Overlapping Elements

- Offset cards across background transitions
- Make elements taller than containers
- Add small overlapping controls (carousel arrows)

**Overlapping images:** Add invisible border (background-colored gap) to prevent clashing.

---

## Working with Images

### Quality Requirements

Bad photos ruin designs. Options:

1. Hire a professional photographer
2. Use quality stock (Unsplash, paid stock sites)

Never use placeholder images planning to swap in phone photos later.

### Text Over Images

**Problem:** Images have light and dark areas; no text color works everywhere.

**Solutions:**

1. **Semi-transparent overlay**

```css
.hero::before {
  background: hsla(0, 0%, 0%, 0.4);
}
```

2. **Lower image contrast + adjust brightness**

```css
.hero-image {
  filter: contrast(0.8) brightness(1.1);
}
```

3. **Colorize image**
   - Lower contrast
   - Desaturate
   - Add solid fill with multiply blend mode

4. **Text shadow (glow)**

```css
.hero-text {
  text-shadow: 0 0 20px hsla(0, 0%, 0%, 0.5);
}
```

### Scaling Rules

**Icons:**

- Don't scale up 16-24px icons—they look chunky
- Solution: Put small icon in colored circle/shape
- Don't scale down large icons—they look muddy
- Solution: Redraw simplified version at target size

**Screenshots:**

- Don't shrink full screenshots 70%—text becomes 4px
- Use mobile/tablet layouts for smaller spaces
- Use partial screenshots
- Draw simplified wireframe versions

### User-Uploaded Content

**Control the container:**

```css
.avatar {
  width: 64px;
  height: 64px;
  background-size: cover;
  background-position: center;
}
```

**Prevent background bleed:**

```css
.product-image {
  /* Subtle inner shadow better than border */
  box-shadow: inset 0 0 0 1px hsla(0, 0%, 0%, 0.1);
}
```

---

## Finishing Touches

### Supercharge Defaults

**Bulleted lists:** Replace bullets with icons (checkmarks, arrows, topic-specific icons)

**Quotes:** Large, colored quotation marks as visual elements

**Links:** Custom underlines (thick, colored, partially overlapping text)

**Form controls:** Brand-colored checkboxes and radio buttons

### Accent Borders

Add colorful borders to:

- Top of cards
- Side of alert messages
- Under headlines
- Active navigation items
- Top of entire layout

```css
.card {
  border-top: 4px solid hsl(210, 100%, 50%);
}
.alert {
  border-left: 4px solid hsl(45, 100%, 50%);
}
```

### Background Decoration

**Color changes:**

```css
.section-alt {
  background: hsl(210, 20%, 97%);
}
```

**Gradients:**

```css
.hero {
  background: linear-gradient(135deg, hsl(210, 100%, 50%), hsl(180, 100%, 50%));
}
```

Keep hues within 30° for best results.

**Patterns:**

- Use subtle repeating patterns (Hero Patterns)
- Can repeat on single edge
- Keep contrast low

**Geometric shapes:**

- Position decorative shapes at corners/edges
- Keep contrast low so content remains readable

### Empty States

Don't neglect them—they're first impressions:

- Add illustrations
- Clear call-to-action button
- Hide supporting UI (tabs, filters) until content exists

### Reducing Borders

**Alternatives:**

1. Box shadow (outlines without harsh lines)
2. Different background colors
3. Extra spacing

```css
/* Instead of border between items */
.card {
  box-shadow: 0 1px 3px hsla(0, 0%, 0%, 0.1);
}
.list-item + .list-item {
  margin-top: 16px;
}
.alternate:nth-child(odd) {
  background: hsl(0, 0%, 97%);
}
```

### Think Outside the Box

**Dropdowns:** Add columns, sections, icons, supporting text

**Tables:** Combine columns, add hierarchy, include images, use color

**Radio buttons:** Selectable cards instead of circles + labels

**Any component:** Question assumptions about how it "should" look
