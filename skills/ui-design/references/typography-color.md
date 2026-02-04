# Typography & Color Reference

## Typography

### Type Scale

Hand-crafted scales beat mathematical formulas for UI work:

```css
--text-xs: 12px;
--text-sm: 14px;
--text-base: 16px;
--text-lg: 18px;
--text-xl: 20px;
--text-2xl: 24px;
--text-3xl: 30px;
--text-4xl: 36px;
--text-5xl: 48px;
--text-6xl: 60px;
--text-7xl: 72px;
```

**Why not modular scales?**

- Fractional pixels (31.25px) cause browser inconsistencies
- Jumps are often too limiting (need something between 12 and 16)
- End up picking ratios that match sizes you already want

**Units:**

- Use px or rem
- Never em (nesting compounds and breaks your scale)

### Font Selection

**Safe choices:**

- Neutral sans-serif (Helvetica-style)
- System font stack: `-apple-system, Segoe UI, Roboto, Noto Sans, Ubuntu, Cantarell, Helvetica Neue`

**Quality indicators:**

- 5+ weights available (10+ styles including italics is ideal)
- Popular fonts are popular for a reason

**Personality through fonts:**

- Serif = elegant, classic
- Rounded sans = playful
- Neutral sans = let other elements define personality

### Line Length

**Optimal: 45-75 characters per line**

```css
.prose {
  max-width: 65ch;
} /* or 20-35em */
```

Constrain paragraphs even in wide layouts—let images span full width while text stays readable.

### Line Height

**Inversely proportional to font size:**

```css
--leading-tight: 1.25; /* Headlines */
--leading-snug: 1.375; /* Large text */
--leading-normal: 1.5; /* Body copy */
--leading-relaxed: 1.625; /* Small text */
--leading-loose: 2; /* Wide columns */
```

- Small text: taller line-height (easier to track)
- Large headlines: can use 1 (tight)
- Wider content: taller line-height (eyes travel further)

### Alignment

**Left-align by default** (for LTR languages)

**Center-align:**

- Headlines
- Short independent blocks (<3 lines)
- If one block is too long, rewrite shorter

**Right-align:**

- Numbers in tables (decimal alignment)
- Justified: enable hyphenation to avoid awkward gaps

### Letter Spacing

**Tighten headlines:**

```css
h1 {
  letter-spacing: -0.025em;
}
```

**Widen ALL CAPS:**

```css
.uppercase {
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
```

### Link Styling

- In prose: color + underline
- In navigation: heavier weight or darker color
- Ancillary links: style on hover only

---

## Color

### HSL Over Hex/RGB

HSL represents color the way humans perceive it:

- **Hue**: Position on color wheel (0-360°)
- **Saturation**: Colorfulness (0-100%)
- **Lightness**: Black to white (0-100%)

```css
/* Related colors are obvious in HSL */
--blue-light: hsl(210, 100%, 80%);
--blue-base: hsl(210, 100%, 50%);
--blue-dark: hsl(210, 100%, 30%);
```

**HSL vs HSB:**

- HSB is common in design tools
- Browsers only understand HSL
- 100% brightness in HSB ≠ white (depends on saturation)

### Building a Color Palette

**You need:**

- 8-10 grey shades
- 5-10 shades of each primary color
- 5-10 shades of each accent color (red, yellow, green, etc.)

**That's 50-100+ colors, not 5.**

### Creating Grey Shades

```css
--gray-50: hsl(210, 20%, 98%); /* Near white */
--gray-100: hsl(210, 20%, 96%);
--gray-200: hsl(210, 16%, 90%);
--gray-300: hsl(210, 14%, 80%);
--gray-400: hsl(210, 12%, 65%);
--gray-500: hsl(210, 10%, 50%);
--gray-600: hsl(210, 12%, 40%);
--gray-700: hsl(210, 14%, 30%);
--gray-800: hsl(210, 16%, 20%);
--gray-900: hsl(210, 20%, 10%); /* Near black */
```

**Tips:**

- Start dark grey, not pure black (looks unnatural)
- Saturate slightly with blue (cool) or yellow/orange (warm)
- Increase saturation at extremes (light and dark ends)

### Creating Color Shades

1. **Pick base (500)**: Should work as button background
2. **Pick darkest (900)**: Works for text
3. **Pick lightest (100)**: Works for tinted backgrounds
4. **Fill gaps using midpoints**

```css
/* Example blue palette */
--blue-100: hsl(210, 100%, 95%); /* Background tint */
--blue-200: hsl(210, 100%, 85%);
--blue-300: hsl(210, 100%, 75%);
--blue-400: hsl(210, 100%, 65%);
--blue-500: hsl(210, 100%, 50%); /* Button background */
--blue-600: hsl(210, 100%, 45%);
--blue-700: hsl(210, 100%, 35%);
--blue-800: hsl(210, 100%, 25%);
--blue-900: hsl(210, 100%, 15%); /* Text on light bg */
```

### Saturation and Lightness

**Problem:** At 0% or 100% lightness, saturation has no effect—colors look washed out.

**Solution:** Increase saturation as lightness moves away from 50%.

### Perceived Brightness by Hue

Not all hues are equally bright at 50% lightness:

- **Bright hues**: Yellow (60°), Cyan (180°), Magenta (300°)
- **Dark hues**: Red (0°), Green (120°), Blue (240°)

**Use this to adjust brightness without washing out:**

- To lighten: rotate toward 60°, 180°, or 300°
- To darken: rotate toward 0°, 120°, or 240°
- Limit rotation to 20-30° or color identity changes

### Accessibility

**WCAG Contrast Ratios:**

- Normal text (<18px): 4.5:1 minimum
- Large text (≥18px bold or ≥24px): 3:1 minimum

**Strategies:**

- Flip contrast: dark text on light colored background
- Rotate hue toward brighter color for colored text on colored background
- Never rely on color alone—add icons or shape differences

### Don't Rely on Color Alone

Color-blind users can't distinguish red/green trends:

- Add icons (↑ ↓ arrows)
- Use contrast differences (light vs dark)
- Add patterns or shapes
