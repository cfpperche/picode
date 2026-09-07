// Integrated mobile tool navigation, on a disposable local docs fixture only.
import assert from "node:assert/strict";
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
const { chromium } = await import(process.env.PICODE_PLAYWRIGHT_MODULE || "/tmp/picode-owned-playwright/node_modules/playwright-core/index.mjs");
const base = process.argv[2] || "http://127.0.0.1:18841";
const url = new URL(base);
assert.ok(url.protocol === "http:" && ["127.0.0.1", "localhost"].includes(url.hostname));
const fleet = await fetch(base + "/api/workspaces").then(response => response.json());
assert.ok(fleet.length && fleet.every(workspace => /^\/tmp\/picode-docs-fixture-[^/]+\//.test(workspace.path)), "synthetic fixture required");
const workspace = fleet.find(row => row.name === "picode");
const agent = workspace.agents.find(row => row.name === "Atlas");
const out = resolve(process.argv[3] || "var/screenshots/mobile-files-git/navigation");
mkdirSync(out, { recursive: true });
const browser = await chromium.launch({ headless: true, executablePath: process.env.PICODE_QA_CHROME || "/usr/bin/google-chrome" });
const report = { screenshots: [], decisions: [], errors: [] };
let terminal;
const preparedTerminals = new Set();
const deliveryRequests = [];
const page = await browser.newPage({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });
page.on("pageerror", error => report.errors.push(error.message));
page.on("request", request => { if (request.method() === "POST" && /\/api\/terminals\/[^/]+\/(type|run)$/.test(new URL(request.url()).pathname)) deliveryRequests.push({ path: new URL(request.url()).pathname, body: request.postDataJSON() }); });
page.on("response", async response => { if (response.request().method() === "POST" && new URL(response.url()).pathname === "/api/terminals" && response.ok()) { const created = await response.json(); if (created.id) preparedTerminals.add(created.id); } });
async function shot(name) {
  await page.evaluate(() => document.fonts.ready);
  await page.waitForTimeout(850);
  const audit = await page.evaluate(() => window.__picodeOverlayAudit());
  const path = resolve(out, name + ".png");
  await page.screenshot({ path }); report.screenshots.push(path);
  assert.equal(audit.ok, true, name + ": " + JSON.stringify(audit));
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), true, name + " page overflow");
}
async function go(hash) { await page.goto(base + "/mobile/?theme=dark" + hash, { waitUntil: "domcontentloaded" }); }
try {
  for (const [width, height] of [[320, 720], [390, 844], [844, 390]]) {
    await page.setViewportSize({ width, height });
    await go("#/work");
    const group = page.getByRole("region", { name: "picode", exact: true });
    await group.getByRole("button", { name: "Files", exact: true }).waitFor(); await shot("work-project-tools-" + width);
    await group.getByRole("button", { name: "Files", exact: true }).click();
    await page.locator(".m-files-screen").waitFor(); await page.getByText("README.md", { exact: true }).first().waitFor();
    await shot("files-from-work-" + width);
    assert.ok(!await page.getByRole("navigation", { name: "Sections", exact: true }).count());
    await page.getByRole("button", { name: "Back", exact: true }).click();
    await group.getByRole("button", { name: "Git", exact: true }).click();
    await page.locator(".m-git").waitFor(); await page.getByRole("button", { name: "Git actions", exact: true }).waitFor();
    await shot("git-from-work-" + width);
    await page.getByRole("button", { name: "Git actions", exact: true }).click();
    await page.getByRole("dialog").waitFor(); await shot("git-actions-" + width);
    await page.keyboard.press("Escape");
    await go("#/agent/" + agent.id);
    await page.getByRole("button", { name: "Project tools", exact: true }).click();
    await shot("agent-project-tools-" + width);
    await page.getByRole("dialog").getByRole("button", { name: "Files", exact: true }).click();
    await page.locator(".m-files-screen").waitFor();
    assert.match(page.url(), new RegExp("#/tree/a/" + agent.id));
    report.decisions.push("Work and agent tool navigation at " + width);
  }
  const response = await fetch(base + "/api/terminals", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name: "mobile-project-tools-qa", workspaceId: workspace.id }) });
  assert.ok(response.ok); terminal = await response.json();
  await page.setViewportSize({ width: 320, height: 720 });
  await go("#/term/" + terminal.id);
  await page.getByRole("button", { name: "Terminal actions", exact: true }).click(); await shot("terminal-project-tools-320");
  await page.getByRole("dialog").getByRole("button", { name: /^Git(?: \d+)?$/ }).click();
  await page.locator(".m-git").waitFor(); assert.match(page.url(), new RegExp("#/git/t/" + terminal.id));
  report.decisions.push("Terminal tools preserve terminal identity");
  // The same parsed route can occur twice in browser history. An internal
  // Follow replaces only the current URL, so accepted Back must remount it.
  const pinned = "#/tree/t/" + terminal.id + "?root=" + encodeURIComponent(workspace.path);
  await go(pinned); await page.getByText("README.md", { exact: true }).first().waitFor();
  await page.evaluate(() => history.pushState(history.state, "", location.href));
  const nextRoot = workspace.path + "/docs";
  const quoted = "'" + nextRoot.replaceAll("'", "'\\''") + "'";
  const moved = await fetch(base + "/api/terminals/" + terminal.id + "/run", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ text: "cd -- " + quoted, root: workspace.path }) });
  assert.ok(moved.ok, "move only the owned fixture shell");
  for (let attempt = 0; attempt < 30; attempt++) {
    const folder = await fetch(base + "/api/terminals/" + terminal.id + "/browse").then(response => response.json());
    if (folder.root === nextRoot) break;
    await page.waitForTimeout(100);
  }
  await page.getByRole("button", { name: "File actions", exact: true }).click();
  await page.getByRole("button", { name: "Follow working folder", exact: true }).click();
  await page.waitForURL(value => value.hash.includes(encodeURIComponent(nextRoot)));
  await page.goBack();
  await page.getByRole("button", { name: "Follow folder", exact: true }).waitFor();
  assert.equal(new URLSearchParams(new URL(page.url()).hash.split("?")[1]).get("root"), workspace.path);
  await shot("files-duplicate-history-pinned-root-320");
  report.decisions.push("Back after internal Follow restores the original root precondition, including duplicate history entries");
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(base + "/mobile/?theme=light#/file/w/" + workspace.id + "/README.md?root=" + encodeURIComponent(workspace.path), { waitUntil: "domcontentloaded" });
  await page.getByRole("button", { name: "Edit", exact: true }).click(); await page.locator(".cm-content").waitFor();
  await shot("file-editor-light-390");
  await page.getByRole("button", { name: "File actions", exact: true }).click(); await shot("file-actions-light-390");
  await page.keyboard.press("Escape");
  await page.goto(base + "/mobile/?theme=light#/git/w/" + workspace.id, { waitUntil: "domcontentloaded" });
  await page.getByRole("button", { name: "History", exact: true }).click();
  await page.locator(".m-git-commit-row").first().waitFor(); await shot("git-history-light-390");
  report.decisions.push("File editor, actions and Git history in light theme");
  await page.getByRole("button", { name: "Git actions", exact: true }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Commit", exact: true }).click();
  await page.getByLabel("Commit message", { exact: true }).fill("QA draft only");
  await page.setViewportSize({ width: 390, height: 430 });
  await page.getByRole("button", { name: "Prepare", exact: true }).scrollIntoViewIfNeeded();
  await shot("git-commit-short-viewport-390");
  await page.getByRole("dialog").getByRole("button", { name: "Back", exact: true }).click();
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole("dialog").getByRole("button", { name: "Fetch", exact: true }).click();
  await page.getByRole("button", { name: "Prepare", exact: true }).click();
  await page.waitForURL(value => value.hash.startsWith("#/term/"));
  await page.locator(".xterm-screen").waitFor();
  assert.equal(deliveryRequests.length, 1);
  assert.ok(deliveryRequests[0].path.endsWith("/type"));
  assert.deepEqual(deliveryRequests[0].body, { text: "git fetch --prune", root: workspace.path });
  await shot("git-prepare-real-terminal-390");
  report.decisions.push("Prepare creates an owned plain shell and types exact command/root without Enter");
  assert.deepEqual(report.errors, []);
  writeFileSync(resolve(out, "result.json"), JSON.stringify({ ok: true, ...report }, null, 2) + "\n");
  console.log("PASS: " + report.decisions.length + " journeys; " + report.screenshots.length + " screenshots and overlay audits");
} finally {
  await browser.close();
  if (terminal?.id) preparedTerminals.add(terminal.id);
  for (const id of preparedTerminals) {
    const response = await fetch(base + "/api/terminals/" + encodeURIComponent(id), { method: "DELETE" });
    assert.ok(response.ok, "owned QA terminal cleanup");
  }
}
