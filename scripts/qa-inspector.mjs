#!/usr/bin/env node
// Inspector rail acceptance runner (ADR-0078). Start picode-docs-fixture on a
// private port and an agent-browser session first; pass the session's CDP URL
// and an installed Playwright module via PICODE_QA_CDP / PICODE_PLAYWRIGHT_MODULE.
// No app dependency added. Refuses real workspace paths: the fixture's
// synthetic picode workspace is a seeded dirty repository, and this runner
// only edits files there and only starts one disposable fixture terminal.
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve, join } from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';

const playwright = await import(process.env.PICODE_PLAYWRIGHT_MODULE || 'playwright');
const chromium = playwright.chromium || playwright.default?.chromium;
assert.ok(chromium, 'PICODE_PLAYWRIGHT_MODULE must resolve to a Playwright module exporting chromium');
const base = process.env.PICODE_QA_BASE || 'http://127.0.0.1:18752';
const out = resolve(process.env.PICODE_QA_OUT || 'var/inspector');
mkdirSync(out, { recursive: true });
const evidence = { base, checks: [], screenshots: [], audits: [] };
const api = async (path, method = 'GET', body) => {
  const response = await fetch(base + path, { method, headers: { 'Content-Type': 'application/json' }, body: body === undefined ? undefined : JSON.stringify(body) });
  assert.ok(response.ok, `${method} ${path}: ${response.status}`);
  return response.status === 204 ? null : response.json();
};
const fleet = await api('/api/workspaces');
const workspaces = Array.isArray(fleet) ? fleet : fleet.workspaces;
const workspace = workspaces.find((w) => w.name === 'picode');
assert.ok(workspace?.path && /^\/tmp\/picode-docs-fixture-[^/]+\/work\/picode$/.test(workspace.path), 'requires an isolated docs fixture');
const atlas = workspace.agents.find((a) => a.name === 'Atlas');
assert.ok(atlas, 'requires the synthetic Atlas agent');
const website = workspaces.find((w) => w.name === 'website');
const kepler = website?.agents?.find((a) => a.name === 'Kepler');
assert.ok(kepler, 'requires the synthetic Kepler agent on the non-git website workspace');
const root = workspace.path;
const status = await api(`/api/agents/${atlas.id}/gitstatus`);
assert.equal(status.git, true, 'the fixture must seed a repository under the picode workspace');

const browser = await chromium.connectOverCDP(process.env.PICODE_QA_CDP);
const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
const page = await context.newPage();
page.setDefaultTimeout(12000);
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
const rail = () => page.locator('#inspector');
const row = (path) => rail().locator(`.ft-row[data-path=${JSON.stringify(path)}]`);
const tabs = () => page.locator('.main-tabs .mtab');
const check = async (name, fn) => { await fn(); evidence.checks.push(name); console.log('PASS', name); };
async function shot(name) {
  await page.evaluate(() => document.fonts.ready);
  await delay(180);
  const audit = await page.evaluate(() => window.__picodeOverlayAudit());
  const clipped = await page.evaluate(() => [...document.querySelectorAll('#inspector button, #inspector input')].flatMap((el) => {
    if (!el.getClientRects().length) return [];
    const r = el.getBoundingClientRect();
    const host = document.getElementById('inspector').getBoundingClientRect();
    return r.left < host.left - 1 || r.right > host.right + 1 || r.top < 0 || r.bottom > innerHeight + 1 ? [el.getAttribute('aria-label') || el.textContent.trim()] : [];
  }));
  evidence.audits.push({ name, ...audit, clipped });
  await page.screenshot({ path: join(out, name + '.png') });
  evidence.screenshots.push(name + '.png');
  assert.equal(audit.ok, true, JSON.stringify(audit));
  assert.deepEqual(clipped, [], 'Every rail control must fit inside the rail and the viewport');
}
async function open(hash) {
  await page.goto(base + '/desktop/?desktop=1', { waitUntil: 'domcontentloaded' });
  await page.evaluate((h) => { localStorage.setItem('picode-theme', 'dark'); localStorage.setItem('picode-inspector-open', '1'); location.hash = h; }, hash);
  await page.reload({ waitUntil: 'domcontentloaded' });
  await rail().waitFor();
}
const term = await api('/api/terminals', 'POST', { name: 'QA', cwd: root, workspaceId: workspace.id });
try {
  await open(`#/agent/${atlas.id}`);
  await check('g1 agent anchor lists the working tree with counts and totals', async () => {
    await rail().getByRole('tab', { name: /^Changes/ }).waitFor();
    await row('web/desktop/src/App.jsx').waitFor();
    const text = await rail().innerText();
    assert.match(text, /Uncommitted/);
    assert.match(text, /\+11/);
    assert.match(text, /−2/);
    assert.equal(await rail().locator('.insp-branch-name').innerText(), 'main');
    assert.equal(await rail().locator('.ft-tab-badge').innerText(), '4');
    await shot('inspector-changes-dark');
  });
  await check('g2 a change opens the file tab in Diff view; Open file swaps to the editor', async () => {
    await row('web/desktop/src/App.jsx').click();
    await page.locator('.file-surface[aria-label="Changes to web/desktop/src/App.jsx"]').waitFor();
    assert.equal(await tabs().count(), 2);
    await row('web/desktop/src/App.jsx').click();
    assert.equal(await tabs().count(), 2, 'a second click is idempotent');
    assert.equal(await rail().locator('.ft-row-on').getAttribute('data-path'), 'web/desktop/src/App.jsx');
    await shot('inspector-diff-tab-dark');
    await page.locator('.file-surface').getByRole('button', { name: 'Open file', exact: true }).click();
    await page.locator('.file-surface .file-pane-name').waitFor();
    await page.locator('.file-surface').getByRole('button', { name: 'View diff', exact: true }).waitFor();
    await page.locator('.file-surface').getByRole('button', { name: 'View diff', exact: true }).click();
    await page.locator('.file-surface[aria-label="Changes to web/desktop/src/App.jsx"]').waitFor();
  });
  await check('g3 the file tab keeps the agent anchor; Files opens the editor', async () => {
    assert.match(await rail().locator('.insp-head').innerText(), /Atlas/);
    await rail().getByRole('tab', { name: 'Files', exact: true }).click();
    await row('README.md').waitFor();
    await row('README.md').click();
    await page.locator('.file-surface[aria-label="README.md"]').waitFor();
    assert.equal(await tabs().count(), 3);
  });
  await check('g4 the Files filter covers loaded rows only and clears', async () => {
    await row('web').click();
    await row('web/desktop').waitFor();
    await rail().locator('.insp-filter').fill('readme');
    await page.waitForFunction(() => document.querySelectorAll('#inspector .ft-row').length === 1);
    await rail().locator('.insp-filter').fill('zzz-nothing');
    await rail().getByText('No loaded file matches.').waitFor();
    await shot('inspector-files-filter-empty-dark');
    await rail().getByRole('button', { name: 'Clear', exact: true }).click();
    await row('README.md').waitFor();
    await page.evaluate(() => { document.documentElement.setAttribute('data-theme', 'light'); localStorage.setItem('picode-theme', 'light'); });
    await shot('inspector-files-light');
    await page.evaluate(() => { document.documentElement.setAttribute('data-theme', 'dark'); localStorage.setItem('picode-theme', 'dark'); });
  });
  await check('g5 the This-agent scope is an empty state with one action when the session named no files', async () => {
    await page.evaluate((h) => { location.hash = h; }, `#/agent/${atlas.id}`);
    await rail().getByRole('tab', { name: /^Changes/ }).click();
    await rail().getByRole('button', { name: 'This agent', exact: true }).click();
    await rail().getByText('No files from this agent yet.').waitFor();
    await shot('inspector-scope-empty-dark');
    await rail().getByRole('button', { name: 'Show all', exact: true }).click();
    await row('README.md').waitFor();
  });
  await check('g6 toggle by button and shortcut persists; app tabs keep the anchor', async () => {
    await page.locator('.insp-toggle').click();
    await page.waitForFunction(() => document.getElementById('inspector').hidden);
    assert.equal(await page.evaluate(() => localStorage.getItem('picode-inspector-open')), '0');
    await page.keyboard.press('Control+.');
    await page.waitForFunction(() => !document.getElementById('inspector').hidden);
    assert.equal(await page.evaluate(() => localStorage.getItem('picode-inspector-open')), '1');
    await page.evaluate(() => { location.hash = '#/app/inbox'; });
    await page.waitForFunction(() => document.querySelectorAll('.main-tabs .mtab').length === 4);
    assert.match(await rail().locator('.insp-head').innerText(), /Atlas/, 'an app tab keeps the last anchor');
    await page.reload({ waitUntil: 'domcontentloaded' });
    await rail().waitFor();
    assert.equal(await page.evaluate(() => document.getElementById('inspector').hidden), false);
  });
  await check('g7 the sizer resizes by keyboard within bounds and remembers', async () => {
    await rail().locator('.insp-sizer').focus();
    await page.keyboard.press('ArrowLeft');
    await page.keyboard.press('ArrowLeft');
    await page.waitForFunction(() => Math.round(document.getElementById('inspector').getBoundingClientRect().width) === 360);
    assert.equal(await page.evaluate(() => localStorage.getItem('picode-inspector-w')), '360');
    for (let i = 0; i < 30; i++) await page.keyboard.press('ArrowLeft');
    const max = await page.evaluate(() => Number(document.querySelector('.insp-sizer').getAttribute('aria-valuemax')));
    assert.equal(Math.round(await page.evaluate(() => document.getElementById('inspector').getBoundingClientRect().width)), max);
    await page.evaluate(() => { localStorage.setItem('picode-inspector-w', '320'); });
    await page.reload({ waitUntil: 'domcontentloaded' });
    await rail().waitFor();
    assert.equal(Math.round(await page.evaluate(() => document.getElementById('inspector').getBoundingClientRect().width)), 320);
  });
  await check('g8 the rail shrinks before it hides and never squeezes the conversation under 640px', async () => {
    await page.setViewportSize({ width: 1200, height: 900 });
    await page.waitForFunction(() => Math.round(document.getElementById('inspector').getBoundingClientRect().width) <= 270 && !document.getElementById('inspector').hidden);
    assert.ok(await page.evaluate(() => document.getElementById('main').getBoundingClientRect().width) >= 640 + 40);
    await shot('inspector-shrunk-1200-dark');
    await page.setViewportSize({ width: 1100, height: 900 });
    await page.waitForFunction(() => document.getElementById('inspector').hidden);
    assert.equal(await page.locator('.insp-toggle').getAttribute('title'), 'Window too narrow for the inspector');
    await page.setViewportSize({ width: 700, height: 900 });
    await page.waitForFunction(() => getComputedStyle(document.getElementById('inspector')).display === 'none');
    await shot('inspector-narrow-700-dark');
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.waitForFunction(() => !document.getElementById('inspector').hidden);
  });
  await check('g9 a terminal anchor pins its root; a cd elsewhere is a blocked line with Follow', async () => {
    await page.evaluate((h) => { location.hash = h; }, `#/term/${term.id}`);
    await rail().getByRole('tab', { name: /^Changes/ }).waitFor();
    assert.equal(await rail().locator('.insp-scope').count(), 0, 'terminals have no session scope');
    execFileSync('tmux', ['send-keys', '-t', term.session + ':', 'cd /tmp', 'Enter']);
    await delay(1200);
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    await rail().locator('.insp-notice').getByRole('button', { name: 'Follow', exact: true }).waitFor();
    assert.match(await rail().locator('.insp-notice').innerText(), /This terminal moved to \/tmp\./);
    assert.ok(await rail().locator('.ft-row').count() > 0, 'the previous lists stay visible');
    await shot('inspector-blocked-terminal-moved-dark');
    await rail().locator('.insp-notice').getByRole('button', { name: 'Follow', exact: true }).click();
    await rail().getByText('Not a git repository').waitFor();
    assert.match(await rail().locator('.insp-head').innerText(), /\/tmp$/);
    execFileSync('tmux', ['send-keys', '-t', term.session + ':', 'cd ' + root, 'Enter']);
  });
  await check('g10 a folder without git keeps Files and says so; no anchor is one line', async () => {
    await page.evaluate((h) => { location.hash = h; }, `#/agent/${kepler.id}`);
    await rail().getByText('Not a git repository').waitFor();
    assert.equal(await rail().getByRole('tab').count(), 1);
    await rail().getByText('Empty folder.').waitFor();
    await shot('inspector-nongit-dark');
    await page.evaluate(() => { localStorage.removeItem('picode-tabs'); location.hash = '#/app/inbox'; });
    await page.reload({ waitUntil: 'domcontentloaded' });
    await rail().getByText('Open an agent or terminal to inspect its files.').waitFor();
    await shot('inspector-no-anchor-dark');
  });
  await check('g11 the More menu opens inside the viewport and lists its actions', async () => {
    await page.evaluate((h) => { location.hash = h; }, `#/agent/${atlas.id}`);
    await rail().getByRole('tab', { name: /^Changes/ }).waitFor();
    await rail().getByRole('button', { name: 'More', exact: true }).click();
    await page.getByRole('menuitem', { name: 'Reveal folder' }).waitFor();
    await page.getByRole('menuitem', { name: 'Open git graph' }).waitFor();
    await page.getByRole('menuitem', { name: 'Open as tab' }).waitFor();
    await shot('inspector-more-menu-dark');
    await page.keyboard.press('Escape');
    await page.getByRole('menuitem', { name: 'Reveal folder' }).waitFor({ state: 'hidden' });
    await page.getByRole('menuitem', { name: 'Open as tab' }).waitFor({ state: 'hidden' }).catch(() => {});
  });
  assert.deepEqual(errors, [], 'no page errors');
  evidence.result = 'PASS';
} catch (error) {
  evidence.result = 'FAIL';
  evidence.error = String(error && error.stack || error);
  try { await page.screenshot({ path: join(out, 'failure.png') }); } catch { /* keep the original error */ }
  throw error;
} finally {
  writeFileSync(join(out, 'qa.json'), JSON.stringify(evidence, null, 2));
  await context.close().catch(() => {});
  // A connected browser is only disconnected by close(); the agent-browser
  // session that owns it keeps running.
  await browser.close().catch(() => {});
  await fetch(base + '/api/terminals/' + term.id, { method: 'DELETE' }).catch(() => {});
}
console.log(`inspector QA: ${evidence.checks.length} groups passed → ${out}`);
