#!/usr/bin/env bash
set -euo pipefail

URL="${1:?Usage: axe-scan.sh URL}"
OUTDIR="test-results/accessibility"
mkdir -p "$OUTDIR"

echo "Running axe-core accessibility scan: $URL"

node -e "
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  await page.goto('$URL', { waitUntil: 'networkidle' });
  await page.addScriptTag({ url: 'https://cdnjs.cloudflare.com/ajax/libs/axe-core/4.10.2/axe.min.js' });
  const results = await page.evaluate(() => axe.run());

  const fs = require('fs');
  fs.writeFileSync('$OUTDIR/axe-results.json', JSON.stringify(results, null, 2));

  console.log('URL:', '$URL');
  console.log('Violations:', results.violations.length);
  console.log('Passes:', results.passes.length);
  console.log('Incomplete:', results.incomplete.length);
  console.log('');

  if (results.violations.length > 0) {
    console.log('=== Violations ===');
    results.violations.forEach(v => {
      console.log('');
      console.log('[' + v.impact.toUpperCase() + '] ' + v.id);
      console.log('  ' + v.help);
      console.log('  WCAG:', v.tags.filter(t => t.startsWith('wcag')).join(', '));
      console.log('  Elements:', v.nodes.length);
      v.nodes.slice(0, 3).forEach(n => {
        console.log('    -', n.html.substring(0, 120));
      });
    });
  } else {
    console.log('No accessibility violations found.');
  }

  await browser.close();
})();
" 2>&1 | tee "$OUTDIR/axe-scan.txt"

echo ""
echo "Results saved to $OUTDIR/"
