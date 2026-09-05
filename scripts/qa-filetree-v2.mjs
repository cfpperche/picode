#!/usr/bin/env node
// Optional browser acceptance runner. Start picode-docs-fixture on a private
// port and an agent-browser session first; pass its CDP URL and a Playwright
// module via PICODE_QA_CDP / PICODE_PLAYWRIGHT_MODULE. No app dependency added.
// Refuses real workspace paths and only starts a disposable fixture terminal.
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdirSync, writeFileSync, readFileSync, copyFileSync, existsSync, unlinkSync, chmodSync, utimesSync } from 'node:fs';
import { resolve, join } from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';

const { chromium } = await import(process.env.PICODE_PLAYWRIGHT_MODULE || 'playwright');
const base = process.env.PICODE_QA_BASE || 'http://127.0.0.1:18746';
const out = resolve(process.env.PICODE_QA_OUT || 'var/filetree-v2');
mkdirSync(out, { recursive: true });
const evidence = { checks: [], screenshots: [], audits: [] };
const api = async (path, method = 'GET', body) => {
  const response = await fetch(base + path, { method, headers: { 'Content-Type': 'application/json' }, body: body === undefined ? undefined : JSON.stringify(body) });
  assert.ok(response.ok, `${method} ${path}: ${response.status} ${await (!response.ok ? response.text() : Promise.resolve(''))}`);
  return response.status === 204 ? null : response.json();
};
const fleet = await api('/api/workspaces');
const workspaces = Array.isArray(fleet) ? fleet : fleet.workspaces;
const workspace = workspaces.find(w => w.name === 'picode');
assert.ok(workspace?.path && /^\/tmp\/picode-docs-fixture-[^/]+\/work\/picode$/.test(workspace.path), 'requires an isolated docs fixture');
assert.ok(workspace.agents.some(a => a.name === 'Atlas'), 'requires the synthetic Atlas workspace');
const root = workspace.path;
if (existsSync(join(root, 'new-name.txt'))) unlinkSync(join(root, 'new-name.txt'));
const empty = workspaces.find(w => w.name === 'website');
assert.equal(empty?.path, join(root, '..', 'website'), 'requires the synthetic empty workspace');
if (existsSync(join(empty.path, 'plain.txt'))) unlinkSync(join(empty.path, 'plain.txt'));
const agent = workspace.agents.find(a => a.name === 'Atlas');
const git = (...args) => execFileSync('git', ['-C', root, ...args], { stdio: 'pipe' });
const initial = 'package main\n\n// File Tree v2 fixture\n' + Array.from({length: 150}, (_, i) => `// Project line ${i + 1}\n`).join('');
mkdirSync(join(root, 'src'), { recursive: true });
mkdirSync(join(root, 'src/empty'), { recursive: true });
writeFileSync(join(root, 'src/main.go'), initial);
writeFileSync(join(root, 'README.md'), '# File Tree v2\n\nReview files beside their navigation.\n\n- One detail pane\n- Drafts stay safe\n');
writeFileSync(join(root, 'deleted.txt'), 'A deleted file\n');
writeFileSync(join(root, 'old-name.txt'), 'A renamed file\n');
writeFileSync(join(root, 'slow.txt'), 'slow response\n');
writeFileSync(join(root, 'blocked.txt'), 'Permission fixture\n');
writeFileSync(join(root, 'unsupported.bin'), Buffer.from([0, 1, 2, 3]));
writeFileSync(join(root, 'large.txt'), 'x'.repeat(3 * 1024 * 1024));
copyFileSync('web/public/icon-192.png', join(root, 'preview.png'));
writeFileSync(join(root, 'diagram.svg'), '<svg xmlns="http://www.w3.org/2000/svg" width="360" height="160"><rect width="360" height="160" fill="#15181d"/><text x="24" y="88" fill="#e7eaf0" font-size="24">File Tree v2</text></svg>');
for (let i = 0; i < 30; i++) writeFileSync(join(root, `reference-${String(i).padStart(2,'0')}.txt`), `Reference ${i}\n`);
if (!existsSync(join(root, '.git'))) git('init', '-b', 'main');
git('add', '.');
git('-c', 'user.name=PiCode QA', '-c', 'user.email=qa@example.invalid', 'commit', '--allow-empty', '-m', 'qa: seed file tree acceptance');
writeFileSync(join(root, 'src/main.go'), initial + '// Updated by an agent\n');
unlinkSync(join(root, 'deleted.txt'));
if (existsSync(join(root, 'new-name.txt'))) unlinkSync(join(root, 'new-name.txt'));
git('mv', 'old-name.txt', 'new-name.txt');
writeFileSync(join(root, 'untracked.md'), '# New document\n');
const browser = await chromium.connectOverCDP(process.env.PICODE_QA_CDP);
const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
const page = await context.newPage();
page.setDefaultTimeout(12000);
const errors = [];
page.on('pageerror', error => errors.push(error.message));
const surface = () => page.locator('.ft-surface:visible');
const row = path => surface().locator(`.ft-row[data-path=${JSON.stringify(path)}]`);
const pane = () => surface().locator('.file-pane');
const title = () => pane().locator('.file-pane-name');
const tabs = () => page.locator('.main-tabs .mtab');
const check = async (name, fn) => { await fn(); evidence.checks.push(name); console.log('PASS', name); };
async function shot(name) {
  await page.evaluate(() => document.fonts.ready);
  await delay(180);
  const audit = await page.evaluate(() => window.__picodeOverlayAudit());
  const clippedControls = await page.evaluate(() => [...document.querySelectorAll('.ft-head button, .file-pane-bar button, .file-pane-notice button, .gg-detail-head button, .dlg button')].flatMap(button => {
    if (!button.getClientRects().length) return [];
    const rect = button.getBoundingClientRect();
    const owner = button.closest('.ft-detail, .ft-surface, .dlg')?.getBoundingClientRect();
    return rect.left < Math.max(0, owner?.left ?? 0) - 1 || rect.right > Math.min(innerWidth, owner?.right ?? innerWidth) + 1 || rect.top < 0 || rect.bottom > innerHeight + 1
      ? [button.textContent.trim()] : [];
  }));
  evidence.audits.push({ name, ...audit, clippedControls });
  await page.screenshot({ path: join(out, name + '.png') });
  evidence.screenshots.push(name + '.png');
  assert.equal(audit.ok, true, JSON.stringify(audit));
  assert.deepEqual(clippedControls, [], 'Every toolbar control must fit inside its panel and the viewport');
}
async function openTree(kind = 'w', id = workspace.id) {
  await page.goto(base + '/?desktop=1', { waitUntil: 'domcontentloaded' });
  await page.getByRole('tab', { name: 'Workspaces', exact: true }).waitFor();
  await page.evaluate(hash => { location.hash = hash; }, `#/tree/${kind}/${id}`);
  await surface().waitFor();
  await surface().locator('.ft-head').getByRole('button',{name:'Refresh',exact:true}).waitFor();
}
async function file(path) {
  if (path.startsWith('src/') && await row('src').getAttribute('aria-expanded') !== 'true') await row('src').click();
  await row(path).click();
  await page.waitForFunction(p => document.querySelector('.ft-surface:not([hidden]) .file-pane-name')?.textContent === p, path);
}
async function edit(text) {
  const code = pane().locator('.cm-content:visible');
  await code.click();
  await page.keyboard.press('Control+a');
  await page.keyboard.insertText(text);
  await pane().locator('.file-dirty').waitFor();
}
async function pick(choice) { await page.getByRole('alertdialog').getByRole('button', { name: choice, exact: true }).click(); }
const term = await api('/api/terminals', 'POST', { name: 'QA', cwd: root, workspaceId: workspace.id });
try {
  await page.goto(base + '/?desktop=1', { waitUntil: 'domcontentloaded' });
  await page.evaluate(() => localStorage.setItem('picode-theme', 'dark'));
  await page.reload({ waitUntil: 'domcontentloaded' });
  await openTree();
  await check('files share one detail, selection is idempotent, keyboard navigation works', async () => {
    const count = await tabs().count();
    assert.ok(count > 0, 'the global tab strip is visible');
    await file('src/main.go');
    await pane().locator('.cm-content').waitFor();
    await row('src/empty').click();
    await surface().locator('.ft-empty').getByText('Empty folder.',{exact:false}).waitFor();
    await row('src/main.go').click();
    assert.equal(await tabs().count(), count);
    assert.equal(await title().innerText(), 'src/main.go');
    assert.equal(await row('src/main.go').getAttribute('aria-selected'), 'true');
    await row('src/main.go').focus();
    await page.keyboard.press('ArrowLeft');
    assert.equal(await page.evaluate(() => document.activeElement.dataset.path), 'src');
    await page.keyboard.press('ArrowRight');
    assert.equal(await page.evaluate(() => document.activeElement.dataset.path), 'src/empty');
    await page.keyboard.press('ArrowDown');
    assert.equal(await page.evaluate(() => document.activeElement.dataset.path), 'src/main.go');
    await shot('filetree-v2-editor-dark');
  });
  await check('changes, file and diff use the same pane; switching lists preserves it', async () => {
    const count = await tabs().count();
    await surface().getByRole('tab', { name: 'Changes' }).click();
    assert.equal(await title().innerText(), 'src/main.go');
    await row('src/main.go').click();
    await surface().locator('.diff').waitFor();
    await shot('filetree-v2-diff-dark');
    await surface().getByRole('button', { name: 'Open file', exact: true }).click();
    await pane().locator('.cm-content').waitFor();
    await surface().getByRole('button', { name: 'View diff', exact: true }).click();
    await surface().locator('.diff').waitFor();
    assert.equal(await tabs().count(), count);
    await surface().getByRole('tab', { name: 'Files', exact: true }).click();
    await file('src/main.go');
  });
  await check('cancel, discard and save cover document replacement', async () => {
    await edit('unsaved draft');
    await row('README.md').click();
    await shot('filetree-v2-unsaved-dialog');
    await pick('Cancel');
    assert.equal(await title().innerText(), 'src/main.go');
    assert.equal(await pane().locator('.cm-content').innerText(), 'unsaved draft');
    await row('README.md').click();
    await pick('Discard');
    await pane().locator('.file-preview').waitFor();
    assert.equal(readFileSync(join(root, 'src/main.go'), 'utf8'), initial + '// Updated by an agent\n');
    await file('src/main.go');
    await edit('saved by browser QA');
    await row('README.md').click();
    await pick('Save');
    await page.waitForFunction(() => document.querySelector('.ft-surface:not([hidden]) .file-pane-name')?.textContent === 'README.md');
    assert.equal(readFileSync(join(root, 'src/main.go'), 'utf8'), 'saved by browser QA');
  });
  await check('Preview/Raw retains unsaved text and its editor instance', async () => {
    await pane().getByRole('button', { name: 'Raw', exact: true }).click();
    await edit('# Edited preview\n\nA draft in Markdown.');
    await page.evaluate(() => { window.__qaEditor = document.querySelector('.ft-surface:not([hidden]) .cm-editor'); });
    await pane().getByRole('button', { name: 'Preview', exact: true }).click();
    await pane().getByRole('heading', { name: 'Edited preview' }).waitFor();
    await pane().getByRole('button', { name: 'Raw', exact: true }).click();
    assert.equal(await page.evaluate(() => window.__qaEditor === document.querySelector('.ft-surface:not([hidden]) .cm-editor')), true);
    await pane().getByRole('button', { name: 'Save', exact: true }).click();
    await pane().locator('.file-dirty').waitFor({ state: 'detached' });
  });
  await check('failed save and external conflict preserve draft, selection and retry', async () => {
    await file('src/main.go');
    await edit('keep this draft after failed save');
    const failWrite = route => route.request().method() === 'PUT' ? route.fulfill({status:503, contentType:'application/json', body:JSON.stringify({error:'Could not save this file.'})}) : route.continue();
    await page.route('**/text*', failWrite);
    await row('README.md').click();
    await pick('Save');
    await pane().getByText('Could not save this file.', {exact:true}).waitFor();
    assert.equal(await title().innerText(), 'src/main.go');
    assert.equal(await pane().locator('.cm-content').innerText(), 'keep this draft after failed save');
    await shot('filetree-v2-save-error');
    await page.unroute('**/text*', failWrite);
    await pane().getByRole('button', {name:'Retry save', exact:true}).click();
    await pane().locator('.file-dirty').waitFor({state:'detached'});
    await edit('keep this draft after conflict');
    writeFileSync(join(root,'src/main.go'),'changed on disk');
    utimesSync(join(root,'src/main.go'), new Date(), new Date(Date.now()+2000));
    await pane().getByRole('button', {name:'Save', exact:true}).click();
    await pane().getByText('This file changed on disk.', {exact:true}).waitFor();
    await shot('filetree-v2-conflict');
    await pane().getByRole('button', {name:'Reload', exact:true}).click();
    await page.getByRole('alertdialog').getByRole('button',{name:'Cancel',exact:true}).click();
    assert.equal(await pane().locator('.cm-content').innerText(), 'keep this draft after conflict');
    await surface().locator('.ft-head').getByRole('button',{name:'Refresh',exact:true}).click();
    await surface().locator('.ft-head').getByRole('button',{name:'Refresh',exact:true}).waitFor();
    assert.equal(await pane().locator('.cm-content').innerText(), 'keep this draft after conflict');
    await pane().getByRole('button', {name:'Reload',exact:true}).click();
    await page.getByRole('alertdialog').getByRole('button',{name:'Reload',exact:true}).click();
    await page.waitForFunction(() => document.querySelector('.ft-surface:not([hidden]) .cm-content')?.textContent === 'changed on disk');
  });
  await check('hide/reveal retains drafts and scroll; outer tab close is guarded', async () => {
    await edit(initial);
    await pane().locator('.cm-scroller').evaluate(el => { el.scrollTop = 450; });
    await delay(100);
    const scroll = await pane().locator('.cm-scroller').evaluate(el => el.scrollTop);
    const treeURL = page.url();
    await page.getByRole('button', {name:'Atlas',exact:true}).click();
    assert.equal(await page.locator('.ft-surface:visible').count(), 0);
    await page.evaluate(hash => { location.hash = hash; }, new URL(treeURL).hash);
    await surface().waitFor();
    assert.ok((await pane().locator('.cm-scroller').evaluate(el=>el.scrollTop)) >= scroll - 2);
    assert.equal(await pane().locator('.file-dirty').count(), 1);
    await page.locator('.main-tabs .mtab.active .mtab-close').click();
    await pick('Cancel');
    assert.equal(await pane().locator('.file-dirty').count(), 1);
    await page.locator('.main-tabs .mtab.active .mtab-close').click();
    await pick('Discard');
    await surface().waitFor({state:'hidden'});
    await page.evaluate(hash => {location.hash=hash;}, new URL(treeURL).hash);
    await surface().waitFor();
    await surface().locator('.ft-head').getByRole('button',{name:'Refresh',exact:true}).waitFor();
  });
  await check('pending saves defer selection; clean refresh preserves editor identity', async () => {
    await file('src/main.go');
    await edit('saved after a delayed response');
    let release, ready;
    const held = new Promise(r => release = r), started = new Promise(r => ready = r);
    const delayWrite = async route => {
      if (route.request().method() !== 'PUT') return route.continue();
      ready(); await held; await route.continue();
    };
    await page.route('**/text*', delayWrite);
    await pane().getByRole('button',{name:'Save',exact:true}).click();
    await started;
    await row('README.md').click();
    assert.equal(await title().innerText(), 'src/main.go');
    assert.equal(await page.getByRole('alertdialog').count(), 0);
    release();
    await page.waitForFunction(() => document.querySelector('.ft-surface:not([hidden]) .file-pane-name')?.textContent === 'README.md');
    await page.unroute('**/text*',delayWrite);
    await file('src/main.go');
    await pane().locator('.cm-content').waitFor();
    await page.evaluate(() => { window.__qaEditor=document.querySelector('.ft-surface:not([hidden]) .cm-editor'); });
    await surface().locator('.ft-head').getByRole('button',{name:'Refresh',exact:true}).click();
    await surface().locator('.ft-head').getByRole('button',{name:'Refresh',exact:true}).waitFor();
    assert.equal(await page.evaluate(() => window.__qaEditor === document.querySelector('.ft-surface:not([hidden]) .cm-editor')),true);
    // Leave the later rapid-read assertion's known contents in place.
    await edit('changed on disk');
    await pane().getByRole('button',{name:'Save',exact:true}).click();
    await pane().locator('.file-dirty').waitFor({state:'detached'});
  });
  await check('removed files retain drafts and browser reload is protected', async () => {
    await file('reference-00.txt');
    await edit('draft after removal');
    unlinkSync(join(root,'reference-00.txt'));
    await pane().getByRole('button',{name:'Save',exact:true}).click();
    await pane().getByText('That file is gone.',{exact:true}).waitFor();
    assert.equal(await pane().locator('.cm-content').innerText(),'draft after removal');
    await shot('filetree-v2-removed');
    let protectedReload = false;
    const dismiss = async dialog => {
      protectedReload=dialog.type()==='beforeunload';
      await dialog.dismiss().catch(()=>{}); // another attached driver may dismiss first
    };
    page.on('dialog',dismiss);
    await page.reload({waitUntil:'domcontentloaded',timeout:2000}).catch(()=>{});
    page.off('dialog',dismiss);
    assert.equal(protectedReload,true);
    assert.equal(await pane().locator('.cm-content').innerText(),'draft after removal');
    await pane().getByRole('button',{name:'Close file panel'}).click();
    await pick('Cancel');
    assert.equal(await pane().locator('.file-dirty').count(),1);
    await pane().getByRole('button',{name:'Close file panel'}).click();
    await pick('Discard');
    await surface().locator('.ft-detail').waitFor({state:'detached'});
  });
  await check('previews, deleted/renamed changes and unavailable files have usable states', async () => {
    await file('preview.png');
    await pane().locator('img').waitFor();
    await shot('filetree-v2-image');
    await file('diagram.svg');
    await pane().locator('.file-preview').waitFor();
    await file('unsupported.bin');
    await pane().getByText("Can't display this file.", {exact:true}).waitFor();
    await shot('filetree-v2-unsupported');
    await file('large.txt');
    await pane().getByText('This file is too large to display.', {exact:true}).waitFor();
    chmodSync(join(root, 'blocked.txt'), 0);
    await file('blocked.txt');
    await pane().getByText("You don't have access to this file.", {exact:true}).waitFor();
    await shot('filetree-v2-blocked');
    chmodSync(join(root,'blocked.txt'),0o644);
    await pane().getByRole('button',{name:'Reload',exact:true}).click();
    await pane().locator('.cm-content').waitFor();
    await surface().getByRole('tab',{name:'Changes'}).click();
    await row('deleted.txt').click();
    await surface().locator('.diff').waitFor();
    assert.equal(await surface().getByRole('button',{name:'Open file',exact:true}).count(),0);
    await row('new-name.txt').click();
    await surface().locator('.diff').waitFor();
    await surface().getByRole('button',{name:'Open file',exact:true}).click();
    await pane().locator('.cm-content').waitFor();
    assert.equal(await title().innerText(), 'new-name.txt');
  });
  await check('late reads cannot replace a more recent file selection', async () => {
    await surface().getByRole('tab',{name:'Files',exact:true}).click();
    let ready, release;
    const started = new Promise(r=>ready=r);
    const held = new Promise(r=>release=r);
    await page.route('**/text*', async route => {
      if(new URL(route.request().url()).searchParams.get('path') !== 'slow.txt') return route.continue();
      ready();
      await held;
      try { await route.fulfill({status:200, contentType:'application/json', body:JSON.stringify({path:'slow.txt',text:'late stale response',mtime:1})}); } catch { /* request was aborted */ }
    });
    await row('slow.txt').click();
    await started;
    await file('src/main.go');
    await pane().locator('.cm-content').waitFor();
    release();
    await delay(100);
    assert.equal(await title().innerText(),'src/main.go');
    assert.equal(await pane().locator('.cm-content').innerText(),'changed on disk');
    await page.unroute('**/text*');
  });
  await check('failed tree refresh retains the open file and exposes retry', async () => {
    const failBrowse = route => route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({error:'Could not read this folder.'})});
    await page.route('**/browse*',failBrowse);
    await surface().locator('.ft-head').getByRole('button',{name:'Refresh',exact:true}).click();
    await surface().locator('.ft-notice').getByText('Could not read this folder.',{exact:true}).waitFor();
    assert.equal(await pane().locator('.cm-content').innerText(),'changed on disk');
    await shot('filetree-v2-tree-error');
    await page.unroute('**/browse*',failBrowse);
    await surface().locator('.ft-notice').getByRole('button',{name:'Try again',exact:true}).click();
    await surface().locator('.ft-notice').waitFor({state:'detached'});
  });
  await check('resizing, light theme, narrow layout and animated close remain usable', async () => {
    const sizer = surface().getByRole('separator',{name:'File tree width'});
    const before = Number(await sizer.getAttribute('aria-valuenow'));
    await sizer.focus(); await page.keyboard.press('ArrowRight');
    assert.ok(Number(await sizer.getAttribute('aria-valuenow')) > before);
    const box=await sizer.boundingBox();
    await page.mouse.move(box.x+box.width/2,box.y+50); await page.mouse.down(); await page.mouse.move(box.x+40,box.y+50); await page.mouse.up();
    await page.evaluate(()=>{localStorage.setItem('picode-theme','light'); document.documentElement.dataset.theme='light';});
    await shot('filetree-v2-editor-light');
    await page.setViewportSize({width:900,height:760});
    await shot('filetree-v2-narrow');
    assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth <= innerWidth));
    await file('README.md');
    await page.setViewportSize({width:720,height:760});
    await shot('filetree-v2-preview-narrow');
    await page.setViewportSize({width:550,height:760});
    await shot('filetree-v2-compact');
    await page.setViewportSize({width:390,height:760});
    await shot('filetree-v2-stacked');
    await pane().getByRole('button',{name:'Close file panel'}).click();
    await surface().locator('.ft-detail').waitFor({state:'detached'});
    assert.equal(await surface().locator('.ft-sizer').count(),0);
    await page.setViewportSize({width:1440,height:900});
  });
  await check('empty and non-Git folders expose a next action', async () => {
    await openTree('w', empty.id);
    await surface().getByText('Empty folder.',{exact:false}).waitFor();
    await shot('filetree-v2-empty');
    writeFileSync(join(empty.path,'plain.txt'),'A folder without Git.');
    await surface().locator('.ft-head').getByRole('button',{name:'Refresh',exact:true}).first().click();
    await row('plain.txt').waitFor();
    assert.equal(await surface().getByRole('tab',{name:'Changes'}).count(),0);
    await file('plain.txt');
    await pane().locator('.cm-content').waitFor();
  });
  await check('a clean Changes list and clean diff expose a useful next action', async () => {
    await openTree();
    await surface().getByRole('tab',{name:'Changes'}).click();
    await row('src/main.go').click();
    await surface().locator('.diff').waitFor();
    git('add','.'); git('-c','user.name=PiCode QA','-c','user.email=qa@example.invalid','commit','-m','qa: settle file changes');
    await surface().locator('.ft-head').getByRole('button',{name:'Refresh',exact:true}).click();
    await surface().getByText('No changes.',{exact:false}).waitFor();
    await surface().getByText('No changes in this file.',{exact:true}).waitFor();
    await shot('filetree-v2-clean-changes');
    await surface().getByRole('button',{name:'View files',exact:true}).click();
    await surface().getByRole('button',{name:'Open file',exact:true}).click();
    await pane().locator('.cm-content').waitFor();
  });
  await check('agent and terminal owners open files; terminal cd requires explicit refresh', async () => {
    // A fresh storage context exercises each owner instead of the folder-tab dedup.
    await page.evaluate(()=>{localStorage.removeItem('picode-tabs'); localStorage.removeItem('picode-tree-owners');});
    await page.goto('about:blank');
    await openTree('a',agent.id);
    await file('README.md');
    await pane().locator('.file-preview').waitFor();
    await page.evaluate(()=>{localStorage.removeItem('picode-tabs'); localStorage.removeItem('picode-tree-owners');});
    await page.goto('about:blank');
    await openTree('t',term.id);
    await file('src/main.go');
    await edit('draft in the original terminal folder');
    const shellWord = value => "'" + value.replaceAll("'", "'\\''") + "'";
    execFileSync('tmux',['send-keys','-t',term.session,'cd '+shellWord(empty.path),'Enter']);
    for(let i=0;i<50;i++) { if((await api(`/api/terminals/${term.id}/cwd`)).cwd===empty.path) break; await delay(50); }
    assert.equal((await api(`/api/terminals/${term.id}/cwd`)).cwd,empty.path);
    await pane().getByRole('button',{name:'Save',exact:true}).click();
    await pane().getByText('This folder changed. Refresh the file tree.',{exact:true}).waitFor();
    await shot('filetree-v2-root-changed');
    await pane().getByRole('button',{name:'Refresh tree',exact:true}).click();
    await pick('Cancel');
    assert.equal(await pane().locator('.cm-content').innerText(),'draft in the original terminal folder');
    await pane().getByRole('button',{name:'Refresh tree',exact:true}).click();
    await pick('Discard');
    await row('plain.txt').waitFor();
    await file('plain.txt');
    await pane().locator('.cm-content').waitFor();
  });
  assert.deepEqual(errors, [], 'browser runtime errors');
  evidence.runtimeErrors = errors;
  evidence.status = 'PASS';
} catch(error) {
  evidence.status='FAIL'; evidence.error=String(error.stack || error);
  await page.screenshot({path:join(out,'failure.png')}).catch(()=>{});
  console.error(error); process.exitCode=1;
} finally {
  chmodSync(join(root,'blocked.txt'),0o644);
  writeFileSync(join(out,'qa.json'),JSON.stringify(evidence,null,2)+'\n');
  try { await context.close(); await browser.close(); }
  finally { await api('/api/terminals/'+term.id,'DELETE'); }
}
