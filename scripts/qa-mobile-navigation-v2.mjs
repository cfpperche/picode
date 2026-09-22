#!/usr/bin/env node
// Disposable fixture only. Fleet reads and creation are mocked; the global
// mutation guard prevents real agent turns or changes to fixture resources.
import assert from "node:assert/strict";
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

const { chromium } = await import(process.env.PICODE_PLAYWRIGHT_MODULE || "/tmp/picode-owned-playwright/node_modules/playwright-core/index.mjs");
const base = new URL(process.argv[2] || "http://127.0.0.1:18831");
assert.ok(base.protocol === "http:" && ["localhost", "127.0.0.1"].includes(base.hostname), "Loopback HTTP fixture required");
const fleet = await fetch(new URL("/api/workspaces", base)).then(r => r.json());
assert.ok(fleet.length && fleet.every(w => /^\/tmp\/picode-docs-fixture-[^/]+\//.test(w.path)), "Synthetic fixture required");
const workspace = fleet.find(w => w.name === "picode");
const agent = workspace?.agents.find(a => a.name === "Atlas");
assert.equal(agent?.mode, "stopped");
const out = resolve(process.argv[3] || "var/screenshots/mobile-v2-navigation");
mkdirSync(out, { recursive: true });
const report = { states: [], screenshots: [], consoleErrors: [], noRealMutations: true };
const browser = await chromium.launch({ headless: true, executablePath: process.env.PICODE_QA_CHROME || "/usr/bin/google-chrome" });
const deferred = () => { let resolve; const promise = new Promise(done => { resolve = done; }); return { promise, resolve }; };
const mobile = hash => new URL("/mobile/?theme=dark" + hash, base).href;
async function waitFor(signal, label) {
  let timer;
  try { await Promise.race([signal.promise, new Promise((_, reject) => { timer = setTimeout(() => reject(new Error("Timed out: " + label)), 10000); })]); }
  finally { clearTimeout(timer); }
}

async function contextFor(width = 390) {
  const context = await browser.newContext({ viewport: { width, height: 844 }, isMobile: true, hasTouch: true, colorScheme: "dark", serviceWorkers: "block" });
  await context.route("**/api/**", route => ["GET", "HEAD"].includes(route.request().method()) ? route.fallback() : route.fulfill({ status: 403, json: { error: "QA blocks real mutations" } }));
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  page.on("pageerror", error => report.consoleErrors.push(error.message));
  return { context, page };
}

async function capture(page, name) {
  await page.evaluate(() => document.fonts.ready);
  await page.waitForTimeout(650);
  const audit = await page.evaluate(() => window.__picodeOverlayAudit());
  assert.equal(audit.ok, true, name + ": " + JSON.stringify(audit));
  assert.equal(await page.evaluate(() => [...document.querySelectorAll(".m-screen")].some(el => el.scrollWidth > el.clientWidth + 1)), false, name + ": page overflow");
  const path = resolve(out, name + ".png");
  await page.screenshot({ path });
  report.screenshots.push(path);
}

try {
  for (const width of [390, 320]) {
    const { context, page } = await contextFor(width);
    const agents = Array.from({ length: 40 }, (_, i) => i === 20 ? agent : { ...agent, id: "qa-navigation-" + i, name: "QA item " + String(i + 1).padStart(2, "0") });
    await page.route("**/api/workspaces", route => route.fulfill({ json: [{ ...workspace, name: "QA work", agents }] }));
    await page.route("**/api/agents?free=1", route => route.fulfill({ json: [] }));
    await page.route("**/api/terminals", route => route.fulfill({ json: { terminals: [] } }));
    await page.goto(mobile("#/work/workspaces"), { waitUntil: "domcontentloaded" });
    await page.locator(".m-agent-row").first().waitFor();
    await page.getByRole("button", { name: "Search work", exact: true }).click();
    const search = page.locator("#mobile-work-search");
    assert.equal(await search.evaluate(el => el === document.activeElement), true, "Opening search focuses it deliberately");
    await search.fill("QA");
    const openAgent = page.locator(".m-row-main").filter({ hasText: "Atlas" });
    for (const back of ["button", "history"]) {
      await openAgent.scrollIntoViewIfNeeded();
      await page.waitForTimeout(100);
      const scroll = await page.locator(".m-work-v2").evaluate(el => el.scrollTop);
      assert.ok(scroll > 300, "Test must leave the top of the list");
      await capture(page, `search-before-${back}-${width}`);
      await openAgent.click();
      await page.locator(".m-agent #task-input").waitFor();
      await page.evaluate(() => {
        window.qaSearchFocus = 0;
        document.addEventListener("focusin", event => { if (event.target.id === "mobile-work-search") window.qaSearchFocus++; });
      });
      if (back === "button") await page.getByRole("button", { name: "Back", exact: true }).click();
      else await page.goBack();
      await search.waitFor();
      await page.waitForTimeout(650);
      assert.equal(await search.inputValue(), "QA", "Query survives agent navigation");
      const restored = await page.locator(".m-work-v2").evaluate(el => el.scrollTop);
      assert.ok(Math.abs(restored - scroll) <= 1, `${back}: expected scroll ${scroll}, got ${restored}`);
      assert.equal(await search.evaluate(el => el === document.activeElement), false, "Returning never focuses search");
      assert.equal(await page.evaluate(() => window.qaSearchFocus), 0, "No transient focus that could reopen a keyboard");
      await capture(page, `search-restored-${back}-${width}`);
      report.states.push({ state: "search-return", back, width, scroll, restored, searchRefocused: false });
    }
    await context.close();
  }

  {
    const { context, page } = await contextFor();
    const unavailable = route => route.fulfill({ status: 503, json: { error: "QA unavailable" } });
    await page.route("**/api/terminals", unavailable);
    await page.route("**/api/agents?free=1", unavailable);
    await page.goto(mobile("#/agent/" + agent.id), { waitUntil: "domcontentloaded" });
    await page.locator("#task-input").waitFor();
    await capture(page, "known-agent-with-unrelated-errors");
    await page.goto(mobile("#/agent/qa-missing"), { waitUntil: "domcontentloaded" });
    await page.locator('.m-route-state[role="alert"]').waitFor();
    await capture(page, "unknown-agent-source-error");
    await page.unroute("**/api/agents?free=1", unavailable);
    await page.getByRole("button", { name: "Try again", exact: true }).click();
    await page.getByText("That agent is gone.", { exact: true }).waitFor();
    report.states.push("source-specific-resource-and-absence");
    await context.close();
  }

  {
    const { context, page } = await contextFor();
    await page.route("**/api/catalog", route => route.fulfill({ json: { providers: [{ id: "anthropic", signedIn: false, models: [] }], thinking: ["off", "low", "high"] } }));
    await page.goto(mobile("#/work/agents"), { waitUntil: "domcontentloaded" });
    await page.getByRole("button", { name: "New agent", exact: true }).first().click();
    await page.locator("#create-provider").selectOption("anthropic");
    await capture(page, "create-with-no-models");
    await page.getByRole("link", { name: "Open Providers", exact: true }).click();
    await page.waitForURL(url => url.hash === "#/clis/pi/providers");
    await page.locator(".dlg-create").waitFor({ state: "detached" });
    await page.getByRole("button", { name: "Add provider", exact: true }).waitFor();
    assert.equal(await page.evaluate(() => document.body.style.pointerEvents), "", "Providers is interactive");
    await capture(page, "providers-after-create-recovery");
    report.states.push("create-sheet-provider-recovery");
    await context.close();
  }

  {
    const { context, page } = await contextFor();
    const oldStarted = deferred(), oldRelease = deferred(), created = deferred(), freshStarted = deferred(), freshRelease = deferred();
    let holdNext = false, didCreate = false;
    const newAgent = { ...agent, id: "qa-created-agent", name: "QA created agent", workspaceId: "" };
    await page.route("**/api/workspaces", route => route.fulfill({ json: [] }));
    await page.route("**/api/terminals", route => route.fulfill({ json: { terminals: [] } }));
    await page.route("**/api/catalog", route => route.fulfill({ json: { providers: [{ id: "qa-provider", signedIn: true, models: [{ id: "qa-model" }] }], thinking: ["low"] } }));
    await page.route("**/api/agents?free=1", async route => {
      if (holdNext) { holdNext = false; oldStarted.resolve(); await oldRelease.promise; return route.fulfill({ json: [] }); }
      if (didCreate) { freshStarted.resolve(); await freshRelease.promise; return route.fulfill({ json: [newAgent] }); }
      return route.fulfill({ json: [] });
    });
    await page.route("**/api/agents", route => {
      assert.equal(route.request().method(), "POST");
      didCreate = true;
      created.resolve();
      return route.fulfill({ json: newAgent });
    });
    await page.goto(mobile("#/work/agents"), { waitUntil: "domcontentloaded" });
    await page.getByText("No agents yet.", { exact: true }).waitFor();
    await page.waitForTimeout(650);
    await page.getByRole("button", { name: "New agent", exact: true }).first().click();
    await page.locator('.dlg-create input[name="name"]').fill(newAgent.name);
    holdNext = true;
    await page.evaluate(() => window.dispatchEvent(new Event("focus")));
    await waitFor(oldStarted, "pre-mutation read starts");
    await page.getByRole("button", { name: "Create", exact: true }).click();
    await waitFor(created, "mock creation is accepted");
    await page.locator(".dlg-create").waitFor({ state: "detached" });
    assert.equal(new URL(page.url()).hash, "#/work/agents", "Creation waits for fresh fleet data");
    oldRelease.resolve();
    await waitFor(freshStarted, "post-mutation read starts");
    assert.equal(new URL(page.url()).hash, "#/work/agents", "The pre-mutation result cannot trigger navigation");
    freshRelease.resolve();
    await page.waitForURL(url => url.hash === "#/agent/qa-created-agent");
    await page.locator("#task-input").waitFor();
    await page.getByRole("heading", { name: newAgent.name, exact: true }).waitFor();
    report.states.push("forced-refresh-awaits-post-mutation-read");
    await context.close();
  }
  assert.deepEqual(report.consoleErrors, [], "No uncaught application errors");
  report.ok = true;
  console.log("Mobile navigation/source/reload/create regressions: PASS");
} finally {
  writeFileSync(resolve(out, "result.json"), JSON.stringify(report, null, 2) + "\n");
  await browser.close();
}
