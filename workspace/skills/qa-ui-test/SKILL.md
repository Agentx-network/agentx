---
name: qa-ui-test
description: "Browser-based UI testing with Playwright. Captures screenshots, tests forms, checks responsive layout, and detects console errors. Use for testing web page rendering, user flows, and visual verification. Requires `npx` and `node`."
metadata: {"nanobot":{"emoji":"🖥️","requires":{"bins":["npx","node"]}}}
---

# QA UI Test

Browser-based testing using Playwright.

## Setup

Run the setup script on first use:

```bash
bash {baseDir}/scripts/setup-playwright.sh
```

## Screenshot Capture

```bash
npx playwright screenshot --browser chromium "URL" test-results/ui/homepage.png
```

With full page:
```bash
npx playwright screenshot --browser chromium --full-page "URL" test-results/ui/homepage-full.png
```

## Responsive Testing

Test at standard breakpoints:

```bash
# Mobile
npx playwright screenshot --browser chromium --viewport-size "375,812" "URL" test-results/ui/mobile.png

# Tablet
npx playwright screenshot --browser chromium --viewport-size "768,1024" "URL" test-results/ui/tablet.png

# Desktop
npx playwright screenshot --browser chromium --viewport-size "1440,900" "URL" test-results/ui/desktop.png
```

## Console Error Capture

Use a Playwright script to capture JavaScript errors:

```bash
node -e "
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  const errors = [];
  page.on('console', msg => { if (msg.type() === 'error') errors.push(msg.text()); });
  page.on('pageerror', err => errors.push(err.message));
  await page.goto('URL');
  await page.waitForTimeout(3000);
  if (errors.length) {
    console.log('Console errors found:');
    errors.forEach(e => console.log('  -', e));
  } else {
    console.log('No console errors.');
  }
  await browser.close();
})();
"
```

## Form Testing

Test form inputs with a Playwright script:

```bash
node -e "
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  await page.goto('URL');

  // Test empty form submission
  const submitBtn = await page.\$('button[type=submit], input[type=submit]');
  if (submitBtn) {
    await submitBtn.click();
    await page.waitForTimeout(1000);
    await page.screenshot({ path: 'test-results/ui/form-empty-submit.png' });
  }

  await browser.close();
})();
"

```

## Navigation Testing

Check for broken links:

```bash
node -e "
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  await page.goto('URL');
  const links = await page.$$eval('a[href]', els => els.map(e => e.href));
  const unique = [...new Set(links)].filter(l => l.startsWith('http'));
  console.log('Found', unique.length, 'unique links');
  for (const link of unique.slice(0, 20)) {
    try {
      const resp = await page.request.get(link);
      if (resp.status() >= 400) console.log('BROKEN:', resp.status(), link);
    } catch (e) {
      console.log('ERROR:', link, e.message);
    }
  }
  await browser.close();
})();
"
```

## Recording Results

Save all screenshots to `test-results/ui/`:

```bash
mkdir -p test-results/ui
```
