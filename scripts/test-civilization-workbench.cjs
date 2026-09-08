// Requires a running opt-in TestWorkbenchBrowserFixture, or Platform's persisted
// journey with WORKBENCH_BROWSER_KIND=persisted. No browser dependency is added
// to the production bundle; PLAYWRIGHT_MODULE can name an existing installation.
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fs = require('node:fs');
const assert = require('node:assert/strict');

(async () => {
  const base = fs.readFileSync(process.env.WORKBENCH_BROWSER_URL_FILE, 'utf8').trim();
  const output = process.env.WORKBENCH_BROWSER_OUTPUT;
  const persisted = process.env.WORKBENCH_BROWSER_KIND === 'persisted';
  const browser = await chromium.launch({ headless: true });
  try {
    const page = await browser.newPage({ viewport: { width: 1440, height: 1000 }, colorScheme: 'light' });
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    await page.goto(base + '/console/workbench');
    await page.locator('#workbench-detail').waitFor();
    const screenshot = async name => { if (output) await page.screenshot({ path: `${output}/${name}.png`, fullPage: true }); };
    const selected = async id => assert.equal(await page.locator('#workbench-detail').getAttribute('data-work-id'), id);
    const choose = async id => {
      await page.locator(`[data-work-select="${id}"]`).click();
      await page.waitForFunction(id => document.getElementById('workbench-detail')?.dataset.workId === id, id);
    };
    if (!persisted) {
      await selected('brief');
      await page.keyboard.press('Tab');
      assert.equal(await page.evaluate(() => document.activeElement.className), 'wb-skip');
      await page.keyboard.press('Enter');
      assert.equal(await page.locator('#workbench-composer').getAttribute('open'), null);
      await screenshot('workbench-light');
      await page.getByRole('combobox', { name: 'Appearance' }).selectOption('dark');
      await screenshot('workbench-dark');
      await page.reload();
      assert.equal(await page.locator('body').getAttribute('data-theme'), 'dark');
      await page.getByRole('combobox', { name: 'Appearance' }).selectOption('light');
      await page.getByText('Technical details', { exact: true }).click();
      await page.getByRole('button', { name: 'Refresh progress' }).click();
      await page.waitForTimeout(300);
      assert.equal(await page.locator('#evidence-brief').getAttribute('open'), '');
    }
    await page.locator('#workbench-composer > summary').click();
    const outcome = page.getByRole('textbox', { name: 'Outcome', exact: true });
    await outcome.fill('Keep this draft while progress refreshes');
    const identity = await page.locator('[name="source_identity"]').inputValue();
    await page.getByRole('button', { name: 'Refresh progress' }).click();
    await page.locator('h1').click();
    const before = await page.locator('.wb-freshness time').getAttribute('datetime');
    await page.waitForFunction(before => document.querySelector('.wb-freshness time')?.dateTime !== before, before);
    assert.equal(await outcome.inputValue(), 'Keep this draft while progress refreshes');
    assert.equal(await page.locator('[name="source_identity"]').inputValue(), identity);

    if (!persisted) {
      await page.getByRole('button', { name: 'Confirm implementation' }).click();
      await page.waitForFunction(() => document.getElementById('workbench-detail')?.dataset.workState === 'implementing');
      assert.equal(await outcome.inputValue(), 'Keep this draft while progress refreshes');
      assert(page.url().includes('work=brief'));
      await choose('blocked');
      const answer = page.getByRole('textbox', { name: 'Which term should the operator guide use?' });
      await answer.fill('Keep my answer');
      await choose('active');
      assert.equal(await page.evaluate(() => document.activeElement.id), 'workbench-detail-title');
      await choose('blocked');
      assert.equal(await answer.inputValue(), 'Keep my answer');
      await answer.fill('fail');
      await page.getByRole('button', { name: 'Record answer and retry' }).click();
      await page.getByRole('alert').filter({ hasText: 'Retry is temporarily unavailable' }).waitFor();
      assert.equal(await answer.inputValue(), 'fail');
      await answer.fill('Use work item');
      await page.getByRole('button', { name: 'Record answer and retry' }).click();
      await page.waitForFunction(() => document.getElementById('workbench-detail')?.dataset.workState === 'implementing');
      assert.equal(await page.locator('[data-workbench-draft]').count(), 0);
      await choose('prepared');
    }

    await page.getByRole('combobox', { name: 'Execution host' }).selectOption('claude');
    await page.locator('#workbench-model-options > summary').click();
    await page.getByRole('textbox', { name: 'Model (optional)', exact: true }).fill('unavailable-model');
    await page.getByRole('button', { name: 'Prepare brief', exact: true }).click();
    await page.getByRole('alert').filter({ hasText: persisted ? 'provider "claude" is unavailable' : 'Selected model is unavailable' }).waitFor();
    assert.equal(await outcome.inputValue(), 'Keep this draft while progress refreshes');
    assert.equal(await page.locator('[name="source_identity"]').inputValue(), identity);
    assert.equal(await page.getByRole('combobox', { name: 'Execution host' }).inputValue(), 'claude');
    assert.equal(await page.getByRole('textbox', { name: 'Model (optional)', exact: true }).inputValue(), 'unavailable-model');
    // Intake errors preserve the current selection even after several board-only swaps.
    if (!persisted) await selected('prepared');
    await page.locator('#workbench-composer > summary').click();
    await page.getByRole('link', { name: 'Inspect prepared diff' }).click();
    await page.getByRole('heading', { name: 'Prepared diff', exact: true }).waitFor();
    assert((await page.locator('.wb-diff').innerText()).includes('+Run make verify.'));
    assert.equal(await page.evaluate(() => window.diffExecuted), undefined);
    await page.getByRole('button', { name: 'Refresh progress' }).click();
    await page.waitForTimeout(300);
    assert.equal(await page.locator('.wb-diff').count(), 1, 'expanded artifact lost on refresh');
    const download = await page.getByRole('link', { name: 'Download full diff' }).getAttribute('href');
    const raw = await page.request.get(base + download);
    assert(raw.headers()['content-type'].startsWith('text/plain'));
    assert((await raw.text()).includes('+Run make verify.'));
    await screenshot(persisted ? 'workbench-persisted-artifact' : 'workbench-prepared-diff');
    for (const width of [320, 375, 768, 1280, 1440]) {
      await page.setViewportSize({ width, height: 1000 });
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `horizontal overflow at ${width}px`);
    }
    await page.setViewportSize({ width: 375, height: 900 });
    await screenshot(persisted ? 'workbench-persisted-mobile' : 'workbench-mobile');
    await page.locator('#workbench-picker > summary').click();
    assert.equal(await page.locator('#workbench-picker').getAttribute('open'), '');
    if (!persisted) {
      await choose('active');
      assert.equal(await page.locator('#workbench-picker').getAttribute('open'), null);
      await page.setViewportSize({ width: 1280, height: 900 });
      await page.evaluate(() => { document.documentElement.style.fontSize = '200%'; });
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'horizontal overflow with 200% text');
      await page.evaluate(() => { document.documentElement.style.fontSize = ''; });
      await page.route('**/work-fragment?*', route => route.abort('failed'));
      await page.getByRole('button', { name: 'Refresh progress' }).click();
      await page.getByRole('alert').filter({ hasText: 'Connection interrupted' }).waitFor();
      await page.unroute('**/work-fragment?*');
      await choose('prepared');
      await page.goBack();
      await page.waitForFunction(() => document.getElementById('workbench-detail')?.dataset.workId === 'active');
      await page.getByRole('combobox', { name: 'Appearance' }).selectOption('dark');
      assert.equal(await page.locator('body').getAttribute('data-theme'), 'dark');
      assert.equal(await page.evaluate(() => (localStorage.getItem('htmx-history-cache') || '').includes('Keep this draft')), false);
      await page.goto(base + '/console/workbench?work=missing');
      await page.getByRole('heading', { name: 'This work is no longer available' }).waitFor();
      const noJS = await browser.newPage({ javaScriptEnabled: false });
      await noJS.goto(base + '/console/workbench?work=prepared');
      await noJS.getByRole('link', { name: 'Inspect prepared diff' }).click();
      assert((await noJS.locator('body').innerText()).includes('+Run make verify.'));
      await noJS.close();
    }
    assert.deepEqual(errors, []);
    console.log(`PASS: ${persisted ? 'persisted Hive journey' : 'multi-state Site UI'}; polling and draft retention; errors; actual inline artifact; themes; keyboard focus; 320–1440px layouts; ${persisted ? '' : '200% text; no-JS fallback; confirmation and retry;'}`);
  } finally {
    await browser.close();
    await fetch(base + '/__fixture/stop', { method: 'POST' });
  }
})().catch(error => { console.error(error); process.exitCode = 1; });
