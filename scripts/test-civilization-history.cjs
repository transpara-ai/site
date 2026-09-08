// Run against TestWorkbenchBrowserFixture with WORKBENCH_BROWSER_HISTORY=true.
// These actions use fixture data only, never a live Civilization deployment.
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fs = require('node:fs');
const assert = require('node:assert/strict');

(async () => {
  const base = fs.readFileSync(process.env.WORKBENCH_BROWSER_URL_FILE, 'utf8').trim();
  const browser = await chromium.launch({ headless: true });
  const errors = [];
  try {
    const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
    const colleague = await browser.newPage();
    for (const p of [page, colleague]) p.on('pageerror', e => errors.push(e.message));
    const open = id => page.goto(base + '/console/workbench?work=' + id);
    const selected = id => page.waitForFunction(id => document.querySelector('#workbench-detail')?.dataset.workId === id, id);
    const nextPoll = async () => {
      const before = await page.locator('.wb-freshness time').getAttribute('datetime');
      await page.waitForFunction(before => document.querySelector('.wb-freshness time')?.dateTime !== before, before);
    };
    await open('prepared');
    await page.locator('#workbench-team-view').click();
    await page.getByRole('region', { name: 'Team workstreams' }).waitFor();
    await page.locator('#workbench-focus-view').click();
    await selected('prepared');
    await colleague.goto(base + '/console/workbench?work=prepared');
    await page.locator('#workbench-composer > summary').click();
    await page.getByRole('textbox', { name: 'Outcome', exact: true }).fill('Keep my next-work draft');
    await page.getByRole('button', { name: 'Approve result', exact: true }).click();
    await selected('brief');
    await colleague.waitForFunction(() => document.querySelector('#workbench-detail')?.dataset.workId === 'brief');
    assert(!page.url().includes('work=prepared'));
    assert(!colleague.url().includes('work=prepared'));
    assert.equal(await page.locator('#workbench-history').getAttribute('open'), null);
    assert.equal(await page.locator('#work-select-prepared').count(), 0);
    assert.equal(await page.getByRole('textbox', { name: 'Outcome', exact: true }).inputValue(), 'Keep my next-work draft');
    await page.locator('#workbench-history > summary').click();
    await nextPoll();
    assert.equal(await page.locator('#workbench-history').getAttribute('open'), '');
    await page.locator('#history-work-prepared').click();
    await selected('prepared');
    assert(page.url().includes('view=history'));
    assert.equal(await page.locator('#workbench-detail').getAttribute('data-work-state'), 'approved');
    await page.getByRole('link', { name: 'Inspect prepared diff' }).click();
    await page.locator('.wb-diff').waitFor();
    await nextPoll();
    assert.equal(await page.locator('.wb-diff').count(), 1);
    await page.reload();
    await selected('prepared');
    assert((await page.locator('#workbench-detail').innerText()).includes('Recorded by alice'));
    await page.locator('#workbench-focus-view').click();
    await selected('brief');

    await open('reject');
    await page.getByRole('button', { name: 'Reject result', exact: true }).click();
    await page.getByRole('alert').filter({ hasText: 'Add a reason' }).waitFor();
    await page.locator('textarea[name="feedback"]').fill('The result needs a different scope.');
    await nextPoll();
    assert.equal(await page.locator('textarea[name="feedback"]').inputValue(), 'The result needs a different scope.');
    await page.getByRole('button', { name: 'Reject result', exact: true }).click();
    await selected('brief');
    assert.equal(await page.locator('#work-select-reject').count(), 0);

    await open('enhance');
    await page.setViewportSize({ width: 375, height: 900 });
    await page.locator('textarea[name="feedback"]').fill('Add a worked example.');
    await page.getByRole('button', { name: 'Request changes / enhance', exact: true }).click();
    await selected('revision');
    await page.getByRole('link', { name: 'the previously delivered result', exact: true }).click();
    await selected('enhance');
    assert(page.url().includes('view=history'));
    assert((await page.locator('#workbench-detail').innerText()).includes('Add a worked example.'));
    await page.getByRole('link', { name: 'Open requested revision' }).click();
    await selected('revision');

    // Completion in another session clears the entire queue without a refresh.
    for (const work of ['brief', 'blocked', 'active', 'revision']) {
      await page.request.post(base + '/__fixture/change', { form: { work, state: 'completed' } });
    }
    await page.getByRole('heading', { name: "You're all caught up" }).waitFor();
    assert.equal(await page.locator('#workbench-detail').count(), 0);
    assert.equal(await page.locator('#workbench-history').getAttribute('open'), null);
    assert.equal((await page.locator('#workbench-list-title').innerText()).replace(/\s+/g, ' ').trim(), 'Current work 0');
    for (const width of [320, 375, 768, 1440]) {
      await page.setViewportSize({ width, height: 1000 });
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `overflow at ${width}px`);
    }
    if (process.env.WORKBENCH_BROWSER_OUTPUT) {
      await page.screenshot({ path: process.env.WORKBENCH_BROWSER_OUTPUT + '/history-caught-up.png', fullPage: true });
    }
    await page.locator('#workbench-team-view').click();
    await page.getByRole('region', { name: 'Team workstreams' }).waitFor();
    assert.equal(await page.locator('[data-team-work]').count(), 0);
    assert.deepEqual(await page.locator('.wb-rollup strong').allTextContents(), ['0', '0', '0', '0']);
    await page.locator('#workbench-history > summary').click();
    await page.setViewportSize({ width: 320, height: 900 });
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'expanded history overflow');
    await page.locator('#history-work-reject').click();
    await selected('reject');
    assert((await page.locator('#workbench-detail').innerText()).includes('The result needs a different scope.'));
    const noJS = await browser.newPage({ javaScriptEnabled: false });
    await noJS.goto(base + '/console/workbench?view=history&work=prepared');
    await noJS.getByRole('link', { name: 'Inspect prepared diff' }).click();
    assert((await noJS.locator('body').innerText()).includes('+Run make verify.'));
    await noJS.close();
    await colleague.close();
    assert.deepEqual(errors, []);
    console.log('PASS: approve/reject advance focus; enhancement opens revision; remote review/completion retires selection and URL; quiet empty state; collapsed history survives polling; retained reviews and diff; draft preservation; history reload and native links; team totals; 320–1440px layouts.');
  } finally {
    await browser.close();
    await fetch(base + '/__fixture/stop', { method: 'POST' });
  }
})().catch(e => { console.error(e); process.exitCode = 1; });
