---
name: qa-accessibility
description: "Accessibility testing for web applications. Runs Lighthouse audits, axe-core scans via Playwright, and manual WCAG checks (alt text, form labels, heading hierarchy, color contrast). Use when testing a web app for accessibility compliance. Requires `npx` and `node`."
metadata: {"nanobot":{"emoji":"♿","requires":{"bins":["npx","node"]}}}
---

# QA Accessibility Test

Accessibility testing using Lighthouse, axe-core, and manual checks.

## 1. Lighthouse Accessibility Audit

```bash
mkdir -p test-results/accessibility
npx lighthouse "URL" --only-categories=accessibility --output=json --output-path=test-results/accessibility/lighthouse.json --chrome-flags="--headless --no-sandbox" 2>/dev/null

# Extract score and failing audits
node -e "
const r = require('./test-results/accessibility/lighthouse.json');
console.log('Accessibility Score:', r.categories.accessibility.score * 100);
const fails = Object.values(r.audits).filter(a => a.score === 0 && a.details);
fails.forEach(a => console.log('FAIL:', a.title, '-', a.description.split('.')[0]));
"
```

## 2. Axe-Core via Playwright

Detailed WCAG violation scan:

```bash
bash {baseDir}/scripts/axe-scan.sh "URL"
```

Or inline:

```bash
node -e "
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  await page.goto('URL');
  await page.addScriptTag({ url: 'https://cdnjs.cloudflare.com/ajax/libs/axe-core/4.10.2/axe.min.js' });
  const results = await page.evaluate(() => axe.run());
  console.log('Violations:', results.violations.length);
  results.violations.forEach(v => {
    console.log('---');
    console.log('Rule:', v.id, '(' + v.impact + ')');
    console.log('Help:', v.help);
    console.log('Affected:', v.nodes.length, 'elements');
    v.nodes.slice(0, 3).forEach(n => console.log('  -', n.html.substring(0, 120)));
  });
  await browser.close();
})();
"
```

## 3. Manual Checks

### Images — Alt Text

```bash
node -e "
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  await page.goto('URL');
  const imgs = await page.\$\$eval('img', els => els.map(e => ({ src: e.src.substring(0, 80), alt: e.alt, hasAlt: e.hasAttribute('alt') })));
  const missing = imgs.filter(i => !i.hasAlt);
  console.log('Images:', imgs.length, '| Missing alt:', missing.length);
  missing.forEach(i => console.log('  -', i.src));
  await browser.close();
})();
"
```

### Forms — Labels

```bash
node -e "
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  await page.goto('URL');
  const inputs = await page.\$\$eval('input:not([type=hidden]), select, textarea', els => els.map(e => ({
    tag: e.tagName, type: e.type, id: e.id, name: e.name,
    hasLabel: !!e.labels?.length, hasAriaLabel: e.hasAttribute('aria-label') || e.hasAttribute('aria-labelledby')
  })));
  const unlabeled = inputs.filter(i => !i.hasLabel && !i.hasAriaLabel);
  console.log('Form controls:', inputs.length, '| Unlabeled:', unlabeled.length);
  unlabeled.forEach(i => console.log('  -', i.tag, i.type || '', i.name || i.id || '(no identifier)'));
  await browser.close();
})();
"
```

### Headings — Hierarchy

```bash
node -e "
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  await page.goto('URL');
  const headings = await page.\$\$eval('h1,h2,h3,h4,h5,h6', els => els.map(e => ({ level: parseInt(e.tagName[1]), text: e.textContent.trim().substring(0, 60) })));
  console.log('Heading hierarchy:');
  let prev = 0;
  headings.forEach(h => {
    const skip = h.level > prev + 1 && prev > 0;
    console.log((skip ? 'SKIP ' : '     ') + 'h' + h.level + ': ' + h.text);
    prev = h.level;
  });
  await browser.close();
})();
"
```

## WCAG Severity Mapping

| Impact | WCAG Level | Severity |
|--------|-----------|----------|
| critical | A | critical |
| serious | A | high |
| moderate | AA | medium |
| minor | AAA | low |

## Recording Results

```bash
mkdir -p test-results/accessibility
```
