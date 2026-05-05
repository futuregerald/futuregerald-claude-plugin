# Screenshot Capture Instructions

Capture implementation screenshots before handing off to the designer agent. This gives the designer concrete images to compare rather than relying on its own browser access.

## Implementation Screenshot

**Try these tools in order. Use the first one that works:**

### 1. Puppeteer (preferred, compatible with Puppeteer 20+)

Check availability: `npx puppeteer --version` or look for `puppeteer` in `node_modules`.

Write this script to `/tmp/ui-capture.cjs` (note: `.cjs` extension ensures CommonJS parsing regardless of the project's module system) and run it with `node /tmp/ui-capture.cjs`:

```javascript
const puppeteer = require('puppeteer');
const ts = Date.now();
const desktopPath = `/tmp/ui-impl-screenshot-${ts}.png`;
const mobilePath = `/tmp/ui-impl-screenshot-mobile-${ts}.png`;

(async () => {
  let browser;
  try {
    browser = await puppeteer.launch({ headless: true });
    const page = await browser.newPage();

    // Desktop screenshot
    await page.setViewport({ width: 1280, height: 800 });
    await page.goto(process.argv[2], { waitUntil: 'networkidle0', timeout: 30000 });
    await page.screenshot({ path: desktopPath, fullPage: true });

    // Mobile screenshot
    await page.setViewport({ width: 375, height: 812 });
    await page.screenshot({ path: mobilePath, fullPage: true });

    console.log(`Desktop: ${desktopPath}`);
    console.log(`Mobile: ${mobilePath}`);
  } catch (err) {
    if (err.message.includes('ERR_CONNECTION_REFUSED') || err.message.includes('net::ERR_')) {
      console.error(`ERROR: Cannot connect to ${process.argv[2]}. Is your dev server running?`);
      process.exit(2);
    }
    throw err;
  } finally {
    if (browser) await browser.close();
  }
})();
```

Run: `node /tmp/ui-capture.cjs [app_url]`

**If the script exits with code 2** (connection refused), prompt the user: "Cannot connect to [app_url]. Please start your dev server and confirm when ready." Wait for confirmation before retrying.

Parse the output to extract the timestamped file paths for `impl_screenshot` and `impl_screenshot_mobile`.

### 2. Playwright

If Puppeteer is not available, check for `playwright` or `@playwright/test` in `node_modules`.

```bash
TS=$(date +%s)
# Desktop
npx playwright screenshot --viewport-size=1280,800 --full-page [app_url] /tmp/ui-impl-screenshot-$TS.png
# Mobile
npx playwright screenshot --viewport-size=375,812 --full-page [app_url] /tmp/ui-impl-screenshot-mobile-$TS.png
```

### 3. Chrome DevTools MCP

If neither Puppeteer nor Playwright is installed, use the Chrome DevTools MCP tool if available in the current session to navigate to `[app_url]` and capture a screenshot.

### 4. Ask user

If none of the above are available:
- Ask the user to provide a screenshot of the running app
- Or offer to install Puppeteer: `npm install --save-dev puppeteer`

## Design Reference Screenshot

Capture depends on the reference type:

| Type | How to capture |
|------|----------------|
| **Figma** | Use Figma MCP to export the node as an image. If Figma MCP is unavailable, ask the user to export as PNG and provide the file path. |
| **Local file** | Already an image -- read it directly |
| **Remote URL** | `curl -o /tmp/ui-design-ref.png [design_reference]` |

## Output

Store these paths for the designer agent:
- `impl_screenshot` -- desktop implementation screenshot (timestamped path from capture output)
- `impl_screenshot_mobile` -- mobile implementation screenshot (timestamped path from capture output)
- `design_screenshot` -- design reference image

