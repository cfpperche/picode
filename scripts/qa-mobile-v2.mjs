// Mobile v2 route/layout acceptance. Only a disposable docs fixture is allowed.
// No model turns, installs, deployments or external messages are performed.
import assert from "node:assert/strict";
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

const { chromium } = await import(process.env.PICODE_PLAYWRIGHT_MODULE || "/tmp/picode-owned-playwright/node_modules/playwright-core/index.mjs");
const base = process.argv[2] || "http://127.0.0.1:18831";
const url = new URL(base);
assert.ok(url.protocol === "http:" && ["localhost", "127.0.0.1"].includes(url.hostname));
const fleet = await fetch(new URL("/api/workspaces", base)).then(r => r.json());
assert.ok(fleet.length && fleet.every(w => /^\/tmp\/picode-docs-fixture-[^/]+\//.test(w.path)), "synthetic fixture required");
const out = resolve(process.argv[3] || "var/screenshots/mobile-v2/integrated");
mkdirSync(out, { recursive: true });
const browser = await chromium.launch({ headless: true, executablePath: process.env.PICODE_QA_CHROME || "/usr/bin/google-chrome" });
const report = { routes: [], states: [], screenshots: [], consoleErrors: [] };
const workspace = fleet.find(w => w.name === "picode");
const agent = workspace.agents.find(a => a.name === "Atlas");
const routes = [
  ["now", "#/", ".m-now-v2"], ["work", "#/work", ".m-work-v2"],
  ["agents", "#/work/agents", ".m-work-v2"], ["terminals", "#/work/terminals", ".m-work-v2"],
  ["inbox", "#/inbox", ".m-inbox"], ["more", "#/more", ".m-more-v2"],
  ["agent", "#/agent/" + agent.id, ".m-agent"],
  ["changes", "#/inspector/w/" + workspace.id, ".m-changes"],
  ["clis", "#/clis", "#agent-clis-view"], ["sessions", "#/clis/pi/sessions", "#sessions-view"],
  ["automations", "#/automations", "#automations-view"], ["automation-new", "#/automations/new", ".auto-form"],
  ...["providers", "settings", "preferences", "packages", "integrations", "devices", "system", "notifications", "apps", "llama"].map(id => [id, "#/more/" + id, ".m-more-page"]),
];

async function capture(page, name) {
  await page.evaluate(() => document.fonts.ready);
  await page.waitForTimeout(250);
  const audit = await page.evaluate(() => window.__picodeOverlayAudit());
  const overflow = await page.evaluate(() => [...document.querySelectorAll(".m-screen")].filter(e => e.scrollWidth > e.clientWidth + 1).map(e => ({ class: e.className, client: e.clientWidth, scroll: e.scrollWidth })));
  const path = resolve(out, name + ".png");
  await page.screenshot({ path });
  report.screenshots.push(path);
  report.routes.push({ name, audit, overflow });
  assert.equal(audit.ok, true, name + ": overlay/control audit " + JSON.stringify(audit));
  assert.equal(overflow.length, 0, name + ": horizontal page overflow " + JSON.stringify(overflow));
}

try {
  for (const width of [390, 320]) {
    const context = await browser.newContext({ viewport: { width, height: 844 }, isMobile: true, hasTouch: true, colorScheme: width === 390 ? "dark" : "light" });
    const page = await context.newPage();
    page.on("pageerror", error => report.consoleErrors.push(error.message));
    for (const [name, hash, marker] of routes) {
      await page.goto(base + "/mobile/?theme=" + (width === 390 ? "dark" : "light") + hash, { waitUntil: "domcontentloaded" });
      await page.locator(marker).waitFor();
      await page.waitForTimeout(650);
      await capture(page, `${name}-${width}`);
      console.log(`mobile v2: ${name} ${width} PASS`);
    }
    await context.close();
  }

  // Wider mobile app windows keep the same focused tools and sheet policy.
  for (const viewport of [{ width: 720, height: 1000 }, { width: 844, height: 390 }]) {
    const context = await browser.newContext({ viewport, isMobile: true, hasTouch: true, reducedMotion: "reduce" });
    const page = await context.newPage();
    page.on("pageerror", error => report.consoleErrors.push(error.message));
    for (const [name, hash, marker] of routes.filter(([name]) => ["now", "work", "agent", "automation-new"].includes(name))) {
      await page.goto(base + "/mobile/#" + hash.slice(1), { waitUntil: "domcontentloaded" });
      await page.locator(marker).waitFor();
      await page.waitForTimeout(650);
      await capture(page, `${name}-${viewport.width}`);
    }
    await context.close();
  }

  // An initial fetch error must offer recovery, never claim the agent is gone.
  const context = await browser.newContext({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });
  const page = await context.newPage();
  const matcher = "**/api/workspaces";
  await page.route(matcher, route => route.fulfill({ status: 503, json: { error: "QA unavailable" } }));
  await page.goto(base + "/mobile/#/agent/" + agent.id, { waitUntil: "domcontentloaded" });
  await page.getByText("Couldn’t load your work.", { exact: true }).waitFor();
  assert.equal(await page.getByText("That agent is gone.", { exact: true }).count(), 0);
  await capture(page, "fleet-first-load-error");
  await page.unroute(matcher);
  await page.getByRole("button", { name: "Try again", exact: true }).click();
  await page.locator(".m-agent textarea").waitFor();
  report.states.push("initial-error-retry");
  await context.close();

  assert.deepEqual(report.consoleErrors, [], "no uncaught application errors");
  report.ok = true;
} finally {
  writeFileSync(resolve(out, "result.json"), JSON.stringify(report, null, 2) + "\n");
  await browser.close();
}
