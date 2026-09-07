#!/usr/bin/env node
// Edits only files under a verified disposable docs-fixture folder. All
// other API mutations are blocked; no agent turns or terminals are started.
import assert from "node:assert/strict";
import { mkdirSync, writeFileSync, readFileSync, copyFileSync, utimesSync } from "node:fs";
import { resolve, join } from "node:path";
const { chromium } = await import(process.env.PICODE_PLAYWRIGHT_MODULE || "/tmp/picode-owned-playwright/node_modules/playwright-core/index.mjs");
const base = new URL(process.argv[2] || "http://127.0.0.1:18768");
assert.ok(base.protocol === "http:" && ["localhost", "127.0.0.1"].includes(base.hostname));
const fleet = await fetch(new URL("/api/workspaces", base)).then(r => r.json());
assert.ok(fleet.length && fleet.every(w => /^\/tmp\/picode-docs-fixture-[^/]+\//.test(w.path)), "Synthetic fixture required");
const workspace = fleet.find(w => w.name === "picode");
const prefix = "mobile-files-qa";
const dir = join(workspace.path, prefix);
mkdirSync(join(dir, "nested"), { recursive: true });
mkdirSync(join(dir, "empty"), { recursive: true });
const initial = "const greeting = 'Hello mobile';\nconsole.log(greeting);\n";
for (const [name, text] of Object.entries({ "edit.js": initial, "other.txt": "Another file\n", "slow.txt": "Slow file\n", "nested/find-me.txt": "Search found me\n", "preview.md": "# Mobile preview\n\nRead and edit this file.\n", "large.txt": "x".repeat(2 * 1024 * 1024) })) writeFileSync(join(dir, name), text);
writeFileSync(join(dir, "unsupported.bin"), Buffer.from([0, 1, 2, 3]));
copyFileSync(resolve("web/public/icon-192.png"), join(dir, "preview.png"));
const out = resolve(process.argv[3] || "var/screenshots/mobile-files");
mkdirSync(out, { recursive: true });
const report = { checks: [], screenshots: [], consoleErrors: [], syntheticOnly: true };
const browser = await chromium.launch({ headless: true, executablePath: process.env.PICODE_QA_CHROME || "/usr/bin/google-chrome" });
const deferred = () => { let resolve; const promise = new Promise(done => { resolve = done; }); return { promise, resolve }; };
const route = path => new URL("/mobile/?theme=dark#/" + (path ? "file" : "tree") + "/w/" + workspace.id + (path ? "/" + encodeURIComponent(path) : "") + "?root=" + encodeURIComponent(workspace.path), base).href;
let lastPage;
async function newPage(width = 390, height = 844) {
  const context = await browser.newContext({ viewport: { width, height }, isMobile: true, hasTouch: true, colorScheme: "dark", serviceWorkers: "block" });
  await context.route("**/api/**", request => {
    const req = request.request();
    if (["GET", "HEAD"].includes(req.method())) return request.fallback();
    if (req.method() === "PUT" && new URL(req.url()).pathname === "/api/workspaces/" + workspace.id + "/text" && req.postDataJSON().path.startsWith(prefix + "/")) return request.fallback();
    return request.fulfill({ status: 403, json: { error: "QA blocks other mutations" } });
  });
  const page = await context.newPage(); lastPage = page;
  page.setDefaultTimeout(12000);
  page.on("pageerror", error => report.consoleErrors.push(error.message));
  return { context, page };
}
const contents = page => page.locator(".m-file-editor .cm-content");
async function edit(page, text) {
  await contents(page).click();
  await page.keyboard.press("ControlOrMeta+A");
  await page.keyboard.insertText(text);
  if (!await page.getByText("Saving…", { exact: true }).count()) await page.getByText("Unsaved changes", { exact: true }).waitFor();
}
async function open(page, file = "edit.js") {
  await page.goto(route(prefix + "/" + file), { waitUntil: "domcontentloaded" });
  await page.locator(".m-files-screen").waitFor();
}
async function capture(page, name) {
  await page.evaluate(() => document.fonts.ready);
  await page.waitForTimeout(650);
  const audit = await page.evaluate(() => window.__picodeOverlayAudit());
  const path = resolve(out, name + ".png");
  await page.screenshot({ path }); report.screenshots.push(path);
  assert.equal(audit.ok, true, name + ": " + JSON.stringify(audit));
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1), false, name + ": horizontal page overflow");
}
async function actions(page) { await page.getByRole("button", { name: "File actions", exact: true }).click(); await page.getByRole("dialog", { name: "File actions", exact: true }).waitFor(); await page.waitForTimeout(650); }
async function leave(page, choice) { await page.getByRole("dialog", { name: "Save changes?", exact: true }).getByRole("button", { name: choice, exact: true }).click(); await page.getByRole("dialog", { name: "Save changes?", exact: true }).waitFor({ state: "detached" }); }
const textRoute = new RegExp("/api/workspaces/" + workspace.id + "/text(?:\\?|$)");

try {
  const { context, page } = await newPage();
  await page.goto(route(""), { waitUntil: "domcontentloaded" });
  await page.getByRole("button", { name: prefix, exact: true }).click();
  await capture(page, "folder-390");
  await page.getByRole("searchbox", { name: "Find files in this folder" }).fill("find me");
  await page.getByRole("button", { name: prefix + "/nested/find-me.txt", exact: true }).waitFor();
  await capture(page, "recursive-search");
  await page.getByRole("button", { name: prefix + "/nested/find-me.txt", exact: true }).click();
  await contents(page).waitFor();
  assert.equal(await contents(page).evaluate(el => el === document.activeElement), false, "Selecting a file never opens the editor keyboard");
  await page.getByRole("button", { name: "Back", exact: true }).click();
  assert.equal(await page.getByRole("searchbox").inputValue(), "find me", "Search survives the document view");
  await page.getByRole("button", { name: "Clear", exact: true }).click();
  await page.getByRole("button", { name: "edit.js", exact: true }).click();
  await contents(page).waitFor();
  await capture(page, "editor-390");
  await edit(page, "cancel keeps this draft\n");
  await page.getByRole("button", { name: "Back", exact: true }).click();
  await capture(page, "dirty-navigation");
  await leave(page, "Cancel");
  assert.match(await contents(page).innerText(), /cancel keeps this draft/);
  await page.getByRole("button", { name: "Back", exact: true }).click();
  await leave(page, "Discard");
  assert.equal(readFileSync(join(dir, "edit.js"), "utf8"), initial);
  await page.getByRole("button", { name: "edit.js", exact: true }).click();
  await edit(page, "saved before leaving\n");
  await page.getByRole("button", { name: "Back", exact: true }).click();
  await leave(page, "Save");
  await page.getByRole("searchbox").waitFor();
  assert.equal(readFileSync(join(dir, "edit.js"), "utf8"), "saved before leaving\n");
  report.checks.push("browse-recursive-search-no-autofocus-save-discard-cancel");

  await page.getByRole("button", { name: "edit.js", exact: true }).click();
  await edit(page, "failed save draft\n");
  const failSave = request => request.request().method() === "PUT" ? request.fulfill({ status: 503, json: { error: "QA save unavailable" } }) : request.fallback();
  await page.route(textRoute, failSave);
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await page.getByRole("button", { name: "Retry save", exact: true }).waitFor();
  await capture(page, "save-error");
  await page.getByRole("button", { name: "Back", exact: true }).click();
  await leave(page, "Save");
  assert.match(await contents(page).innerText(), /failed save draft/);
  await page.unroute(textRoute, failSave);
  await page.getByRole("button", { name: "Retry save", exact: true }).click();
  await page.getByText("Unsaved changes", { exact: true }).waitFor({ state: "detached" });
  assert.equal(readFileSync(join(dir, "edit.js"), "utf8"), "failed save draft\n");

  await edit(page, "conflict keeps my text\n");
  writeFileSync(join(dir, "edit.js"), "changed on disk\n");
  const future = new Date(Date.now() + 2000); utimesSync(join(dir, "edit.js"), future, future);
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await page.getByText("This file changed on disk.", { exact: true }).waitFor();
  await capture(page, "mtime-conflict");
  assert.match(await contents(page).innerText(), /conflict keeps my text/);
  await page.getByRole("button", { name: "Reload", exact: true }).click();
  await page.getByRole("dialog", { name: "Reload file?", exact: true }).getByRole("button", { name: "Cancel", exact: true }).click();
  assert.match(await contents(page).innerText(), /conflict keeps my text/);
  await page.getByRole("button", { name: "Reload", exact: true }).click();
  await page.getByRole("dialog", { name: "Reload file?", exact: true }).getByRole("button", { name: "Reload", exact: true }).click();
  await page.getByText("Unsaved changes", { exact: true }).waitFor({ state: "detached" });
  assert.match(await contents(page).innerText(), /changed on disk/);
  report.checks.push("failed-save-retry-and-mtime-conflict-reload");

  const saving = deferred(), releaseSave = deferred();
  const holdSave = async request => { if (request.request().method() !== "PUT") return request.fallback(); saving.resolve(); await releaseSave.promise; return request.fallback(); };
  await page.route(textRoute, holdSave);
  await edit(page, "first save payload\n");
  await page.getByRole("button", { name: "Save", exact: true }).click(); await saving.promise;
  await capture(page, "save-pending");
  assert.equal(await page.getByRole("progressbar", { name: "Saving file" }).count(), 1);
  await edit(page, "new typing during save\n");
  releaseSave.resolve();
  await page.getByText("Saving…", { exact: true }).waitFor({ state: "detached" });
  await page.unroute(textRoute, holdSave);
  assert.equal(readFileSync(join(dir, "edit.js"), "utf8"), "first save payload\n");
  assert.match(await contents(page).innerText(), /new typing during save/);
  await page.evaluate(() => window.dispatchEvent(new Event("focus")));
  assert.match(await contents(page).innerText(), /new typing during save/);
  await capture(page, "edits-during-save");
  report.checks.push("concurrent-edits-and-dirty-refresh-preserved");

  // A failed Follow after Discard must not disable the next hash guard.
  const failBrowse = request => request.fulfill({ status: 503, json: { error: "QA folder unavailable" } });
  await page.route("**/api/workspaces/" + workspace.id + "/browse**", failBrowse);
  await actions(page);
  await page.getByRole("button", { name: "Follow working folder", exact: true }).click();
  await leave(page, "Discard");
  await page.getByText("QA folder unavailable", { exact: true }).waitFor();
  await page.evaluate(() => { location.hash = "#/work"; });
  await page.getByRole("dialog", { name: "Save changes?", exact: true }).waitFor();
  await leave(page, "Cancel");
  assert.match(await contents(page).innerText(), /new typing during save/);
  await page.unroute("**/api/workspaces/" + workspace.id + "/browse**", failBrowse);
  report.checks.push("failed-follow-keeps-hash-guard");
  await context.close();

  {
    const { context, page } = await newPage();
    const reading = deferred(), releaseRead = deferred();
    await page.route(textRoute, async request => {
      if (request.request().method() !== "GET" || new URL(request.request().url()).searchParams.get("path") !== prefix + "/slow.txt") return request.fallback();
      reading.resolve(); await releaseRead.promise;
      await request.fulfill({ json: { text: "Stale slow response", mtime: "1" } }).catch(() => {});
    });
    await open(page, "slow.txt"); await reading.promise;
    await page.getByRole("button", { name: "Back", exact: true }).click();
    await page.getByRole("button", { name: "other.txt", exact: true }).click();
    await contents(page).waitFor();
    releaseRead.resolve(); await page.waitForTimeout(150);
    assert.match(await contents(page).innerText(), /Another file/);
    assert.doesNotMatch(await contents(page).innerText(), /Stale slow response/);
    await contents(page).evaluate(el => { window.qaEditorIdentity = el; });
    const reloaded = page.waitForResponse(response => response.request().method() === "GET" && response.url().includes("/text?") && response.url().includes("other.txt"));
    await actions(page); await page.getByRole("button", { name: "Reload file", exact: true }).click();
    await reloaded; await page.waitForTimeout(100);
    assert.equal(await contents(page).evaluate(el => el === window.qaEditorIdentity), true, "Unchanged reload preserves the editor instance");
    assert.equal(await contents(page).evaluate(el => el === document.activeElement), false);
    report.checks.push("stale-file-read-ignored-and-unchanged-reload-keeps-editor");
    await context.close();
  }

  {
    const { context, page } = await newPage();
    let changed = false;
    const roots = [];
    await page.route(textRoute, request => {
      roots.push(new URL(request.request().url()).searchParams.get("root"));
      return changed ? request.fulfill({ status: 409, json: { error: "working folder changed" } }) : request.fallback();
    });
    await open(page); await contents(page).waitFor();
    await edit(page, "root conflict draft\n"); changed = true;
    await page.getByRole("button", { name: "Save", exact: true }).click();
    await page.getByText("The working folder changed.", { exact: true }).waitFor();
    await capture(page, "working-folder-conflict");
    assert.ok(roots.length >= 2 && roots.every(root => root === workspace.path), "Reads and writes preserve the pinned root");
    await page.getByRole("button", { name: "Follow folder", exact: true }).click(); await leave(page, "Cancel");
    assert.match(await contents(page).innerText(), /root conflict draft/);
    const following = deferred(), releaseFollow = deferred();
    const newRoot = workspace.path + "-moved";
    let held = true;
    await page.route("**/api/workspaces/" + workspace.id + "/browse**", async request => {
      const url = new URL(request.request().url());
      if (url.searchParams.has("root")) return request.fallback();
      following.resolve(); if (held) await releaseFollow.promise;
      return request.fulfill({ json: { root: newRoot, dir: "", parent: "", dirs: [], files: [], cwdOk: true } });
    });
    await page.getByRole("button", { name: "Follow folder", exact: true }).click(); await leave(page, "Discard"); await following.promise;
    await edit(page, "typing while following keeps the draft\n");
    releaseFollow.resolve();
    // The original root conflict remains primary; the rejected Follow cannot
    // replace the document or clear edits typed while its request was pending.
    await page.waitForTimeout(150);
    assert.match(await contents(page).innerText(), /typing while following keeps the draft/);
    assert.equal(new URL(page.url()).hash.includes(encodeURIComponent(workspace.path)), true);
    await page.evaluate(() => { location.hash = "#/work"; });
    await page.getByRole("dialog", { name: "Save changes?", exact: true }).waitFor(); await leave(page, "Cancel");
    held = false;
    await page.getByRole("button", { name: "Follow folder", exact: true }).click(); await leave(page, "Discard");
    await page.getByText("This folder is empty.", { exact: true }).waitFor();
    assert.equal(new URL(page.url()).hash.includes(encodeURIComponent(newRoot)), true);
    assert.equal(await contents(page).count(), 0);
    await capture(page, "follow-working-folder");
    report.checks.push("root-precondition-and-edits-during-follow-preserved");
    await context.close();
  }

  for (const [width, height] of [[320, 844], [844, 390]]) {
    const { context, page } = await newPage(width, height);
    await open(page); await contents(page).waitFor();
    await capture(page, `editor-${width}x${height}`);
    await actions(page); await capture(page, `actions-${width}x${height}`);
    await page.getByRole("button", { name: "Done", exact: true }).click();
    await open(page, "preview.md");
    await page.getByRole("heading", { name: "Mobile preview", exact: true }).waitFor();
    await capture(page, `preview-${width}x${height}`);
    await page.getByRole("button", { name: "Edit", exact: true }).click();
    await contents(page).waitFor();
    assert.equal(await contents(page).evaluate(el => el === document.activeElement), false);
    await context.close();
  }

  {
    const { context, page } = await newPage();
    for (const [file, name] of [["unsupported.bin", "unsupported"], ["large.txt", "oversized"], ["missing.txt", "missing"]]) {
      await open(page, file);
      await page.locator(".m-file-notice").waitFor();
      assert.ok((await page.locator(".m-file-notice p").innerText()).length > 0);
      assert.equal(await page.locator(".m-file-notice button").count(), 1);
      assert.equal(await page.locator(".m-file-notice button").innerText(), name === "missing" ? "Reload" : "Browse folder");
      await capture(page, name);
      if (name !== "missing") {
        await page.getByRole("button", { name: "Browse folder", exact: true }).click();
        await page.getByRole("button", { name: "edit.js", exact: true }).waitFor();
      }
    }
    await open(page, "preview.png"); await page.locator(".m-file-preview img").waitFor(); await capture(page, "image-preview");
    await page.getByRole("button", { name: "Back", exact: true }).click();
    await page.getByRole("button", { name: "empty", exact: true }).click();
    await page.getByText("This folder is empty.", { exact: true }).waitFor(); await capture(page, "empty-folder");
    report.checks.push("unsupported-oversize-missing-empty-and-media-preview");
    await context.close();
  }
  {
    const { context, page } = await newPage(320, 844);
    const denied = request => request.fulfill({ status: 403, json: { error: "permission denied" } });
    await page.route("**/api/workspaces/" + workspace.id + "/browse**", denied);
    await page.goto(route(""), { waitUntil: "domcontentloaded" });
    await page.getByRole("button", { name: "Try again", exact: true }).waitFor();
    await page.getByText("You don't have access to this folder.", { exact: true }).waitFor();
    await capture(page, "folder-denied-320");
    await page.unroute("**/api/workspaces/" + workspace.id + "/browse**", denied);
    await page.getByRole("button", { name: "Try again", exact: true }).click();
    await page.getByRole("button", { name: prefix, exact: true }).waitFor();
    report.checks.push("folder-permission-error-and-retry");
    await context.close();
  }
  assert.deepEqual(report.consoleErrors, [], "No uncaught browser errors");
  report.ok = true;
  console.log("Mobile Files browser regressions: PASS");
} catch (error) {
  if (lastPage && !lastPage.isClosed()) { await lastPage.screenshot({ path: resolve(out, "failure.png") }); report.failureText = await lastPage.locator("body").innerText(); }
  throw error;
} finally {
  writeFileSync(resolve(out, "result.json"), JSON.stringify(report, null, 2) + "\n");
  await browser.close();
}
