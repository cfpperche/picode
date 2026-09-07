#!/usr/bin/env node
// Decision table: idle action starts once; pending action blocks all other
// mutations; delete cancellation sends nothing; each success/failure releases
// controls; failed toggle restores its captured value even if refresh fails;
// accepted toggle keeps its new value if only refresh fails. No real writes.
import assert from "node:assert/strict";
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
const { chromium } = await import(process.env.PICODE_PLAYWRIGHT_MODULE || "/tmp/picode-owned-playwright/node_modules/playwright-core/index.mjs");
const base = new URL(process.argv[2] || "http://127.0.0.1:18768");
assert.ok(base.protocol === "http:" && ["localhost", "127.0.0.1"].includes(base.hostname));
const fleet = await fetch(new URL("/api/workspaces", base)).then(r => r.json());
assert.ok(fleet.length && fleet.every(w => /^\/tmp\/picode-docs-fixture-[^/]+\//.test(w.path)), "Synthetic fixture required");
const out = resolve(process.argv[3] || "var/screenshots/mobile-automation-mutations");
mkdirSync(out, { recursive: true });
const report = { states: [], screenshots: [], consoleErrors: [], noRealMutations: true };
const browser = await chromium.launch({ headless: true, executablePath: process.env.PICODE_QA_CHROME || "/usr/bin/google-chrome" });
const deferred = () => { let resolve; const promise = new Promise(done => { resolve = done; }); return { promise, resolve }; };
const doubleClick = locator => locator.evaluate(el => { el.click(); el.click(); });
async function waitCalls(state, count) {
  const deadline = Date.now() + 10000;
  while (state.calls.length < count && Date.now() < deadline) await new Promise(done => setTimeout(done, 20));
  assert.equal(state.calls.length, count, "Expected intercepted mutation count");
}

async function screen(enabled = true, detail = true) {
  const context = await browser.newContext({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, colorScheme: "dark", serviceWorkers: "block" });
  await context.route("**/api/**", route => ["GET", "HEAD"].includes(route.request().method()) ? route.fallback() : route.fulfill({ status: 403, json: { error: "QA blocks real mutations" } }));
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  page.on("pageerror", error => report.consoleErrors.push(error.message));
  const state = { item: { id: "qa-automation", name: "QA automation", action: "start", prompt: "Synthetic QA only", enabled, webhook: true, workspaceId: "ws_free", running: false, sparkline: [] }, calls: [], reply: deferred(), failRefresh: false };
  await page.route(url => url.pathname === "/api/automations" || url.pathname.startsWith("/api/automations/"), async route => {
    const req = route.request(), path = new URL(req.url()).pathname;
    if (req.method() === "GET") {
      if (path === "/api/automations") return state.failRefresh ? route.fulfill({ status: 503, json: { error: "QA refresh unavailable" } }) : route.fulfill({ json: { items: state.deleted ? [] : [state.item] } });
      return route.fulfill({ json: { items: [] } });
    }
    const call = { path, method: req.method(), body: req.postDataJSON() };
    state.calls.push(call);
    const response = await state.reply.promise;
    if (call.method === "PATCH" && response.status < 400) state.item = { ...state.item, enabled: call.body.enabled };
    if (call.method === "DELETE" && response.status < 400) state.deleted = true;
    await route.fulfill(response);
  });
  await page.goto(new URL("/mobile/?theme=dark#/automations" + (detail ? "/qa-automation" : ""), base).href, { waitUntil: "domcontentloaded" });
  await page.getByRole("switch").waitFor();
  await page.waitForTimeout(650);
  return { context, page, state };
}

async function capture(page, name) {
  await page.evaluate(() => document.fonts.ready);
  await page.waitForTimeout(650);
  const audit = await page.evaluate(() => window.__picodeOverlayAudit());
  assert.equal(audit.ok, true, name + ": " + JSON.stringify(audit));
  const path = resolve(out, name + ".png");
  await page.screenshot({ path });
  report.screenshots.push(path);
}

async function controls(page, disabled, detail = true) {
  assert.equal(await page.getByRole("switch").isDisabled(), disabled);
  for (const name of detail ? ["Run now", "Edit", "Delete", "Regenerate secret"] : ["Run now"]) {
    const button = page.getByRole("button", { name, exact: true });
    if (await button.count()) assert.equal(await button.isDisabled(), disabled, name);
  }
}

try {
  {
    const { context, page, state } = await screen();
    await doubleClick(page.getByRole("button", { name: "Regenerate secret", exact: true }));
    await page.getByText("Rotating secret…", { exact: true }).waitFor();
    await controls(page, true);
    await page.getByRole("button", { name: "Run now", exact: true }).evaluate(el => el.click());
    await waitCalls(state, 1);
    assert.equal(state.calls.length, 1, "Double tap and competing action send one rotation");
    assert.equal(state.calls[0].path, "/api/automations/qa-automation/secret");
    await capture(page, "rotation-pending");
    const secret = "a".repeat(64);
    state.reply.resolve({ status: 200, json: { webhookSecret: secret } });
    await page.locator(".auto-secret .auto-code").waitFor();
    await page.getByText("Rotating secret…", { exact: true }).waitFor({ state: "detached" });
    assert.equal(await page.locator(".auto-secret .auto-code").textContent(), secret);
    await controls(page, false);
    await capture(page, "rotation-accepted");
    await page.getByRole("button", { name: "Done, I copied it", exact: true }).click();
    state.reply = deferred();
    await doubleClick(page.getByRole("button", { name: "Regenerate secret", exact: true }));
    await page.getByText("Rotating secret…", { exact: true }).waitFor();
    await waitCalls(state, 2);
    assert.equal(state.calls.length, 2, "A later deliberate rotation can start once");
    state.reply.resolve({ status: 503, json: { error: "QA rotation unavailable" } });
    await page.getByText("Rotating secret…", { exact: true }).waitFor({ state: "detached" });
    await controls(page, false);
    report.states.push("rotation-double-tap-serialized-success-and-error");
    await context.close();
  }

  for (const detail of [true, false]) for (const enabled of [true, false]) {
    const { context, page, state } = await screen(enabled, detail);
    await doubleClick(page.getByRole("switch"));
    await page.getByText("Saving automation…", { exact: true }).waitFor();
    assert.equal(await page.getByRole("switch").getAttribute("aria-checked"), String(!enabled), "Toggle is optimistic");
    await controls(page, true, detail);
    await waitCalls(state, 1);
    assert.equal(state.calls.length, 1, "Rapid toggle sends one PATCH");
    state.failRefresh = true;
    state.reply.resolve({ status: 503, json: { error: "QA save unavailable" } });
    await page.getByText("Saving automation…", { exact: true }).waitFor({ state: "detached" });
    await page.getByText("QA refresh unavailable", { exact: false }).first().waitFor();
    assert.equal(await page.getByRole("switch").getAttribute("aria-checked"), String(enabled), "Failed PATCH restores captured state despite failed GET");
    await controls(page, false, detail);
    if (enabled) await capture(page, "toggle-rollback-" + (detail ? "detail" : "list"));
    report.states.push({ state: "toggle-patch-and-refresh-failure", detail, enabled, restored: enabled });
    await context.close();
  }

  {
    const { context, page, state } = await screen();
    await page.getByRole("switch").click();
    await page.getByText("Saving automation…", { exact: true }).waitFor();
    await waitCalls(state, 1);
    state.failRefresh = true;
    state.reply.resolve({ status: 200, json: {} });
    await page.getByText("Saving automation…", { exact: true }).waitFor({ state: "detached" });
    assert.equal(await page.getByRole("switch").getAttribute("aria-checked"), "false", "Accepted PATCH is not rolled back when only GET fails");
    await controls(page, false);
    report.states.push("accepted-toggle-survives-refresh-failure");
    await context.close();
  }

  {
    const { context, page, state } = await screen();
    await doubleClick(page.getByRole("button", { name: "Run now", exact: true }));
    await page.getByText("Starting automation…", { exact: true }).waitFor();
    await controls(page, true);
    await waitCalls(state, 1);
    assert.equal(state.calls.length, 1);
    assert.equal(state.calls[0].path, "/api/automations/qa-automation/run");
    state.reply.resolve({ status: 200, json: {} });
    await page.getByText("Starting automation…", { exact: true }).waitFor({ state: "detached" });
    await controls(page, false);
    state.reply = deferred();
    await doubleClick(page.getByRole("button", { name: "Run now", exact: true }));
    await page.getByText("Starting automation…", { exact: true }).waitFor();
    await waitCalls(state, 2);
    state.reply.resolve({ status: 503, json: { error: "QA run unavailable" } });
    await page.getByText("Starting automation…", { exact: true }).waitFor({ state: "detached" });
    await controls(page, false);
    state.reply = deferred();
    await doubleClick(page.getByRole("button", { name: "Delete", exact: true }));
    await page.getByRole("dialog", { name: "Delete QA automation", exact: true }).waitFor();
    assert.equal(await page.getByRole("dialog", { name: "Delete QA automation", exact: true }).count(), 1);
    await capture(page, "delete-confirmation");
    await page.getByRole("dialog", { name: "Delete QA automation", exact: true }).getByRole("button", { name: "Cancel", exact: true }).click();
    await page.getByRole("dialog", { name: "Delete QA automation", exact: true }).waitFor({ state: "detached" });
    await controls(page, false);
    assert.equal(state.calls.length, 2, "Cancelled delete sends nothing");
    await page.getByRole("button", { name: "Delete", exact: true }).click();
    await doubleClick(page.getByRole("dialog", { name: "Delete QA automation", exact: true }).getByRole("button", { name: "Delete", exact: true }));
    await page.getByRole("dialog", { name: "Delete QA automation", exact: true }).waitFor({ state: "detached" });
    await controls(page, true);
    await waitCalls(state, 3);
    assert.equal(state.calls.length, 3, "Double confirmation sends one DELETE");
    assert.equal(state.calls[2].method, "DELETE");
    state.reply.resolve({ status: 503, json: { error: "QA delete unavailable" } });
    await page.getByText("Delete pending…", { exact: true }).waitFor({ state: "detached" });
    await controls(page, false);
    state.reply = deferred();
    await page.getByRole("button", { name: "Delete", exact: true }).click();
    await page.getByRole("dialog", { name: "Delete QA automation", exact: true }).getByRole("button", { name: "Delete", exact: true }).click();
    await waitCalls(state, 4);
    state.reply.resolve({ status: 200, json: {} });
    await page.getByText("No automations yet.", { exact: true }).waitFor();
    await page.getByText("Delete pending…", { exact: true }).waitFor({ state: "detached" });
    report.states.push("run-double-tap-success-error-and-delete-confirm-cancel-error-success");
    await context.close();
  }
  assert.deepEqual(report.consoleErrors, []);
  report.ok = true;
  console.log("Mobile automation mutation regressions: PASS");
} finally {
  writeFileSync(resolve(out, "result.json"), JSON.stringify(report, null, 2) + "\n");
  await browser.close();
}
