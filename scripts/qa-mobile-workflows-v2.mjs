// Mobile sessions/automations acceptance. Only disposable docs fixtures are allowed.
// Every mutation, including read-only POST handoff previews, is mocked in the browser.
import assert from "node:assert/strict";
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

const { chromium } = await import(process.env.PICODE_PLAYWRIGHT_MODULE || "/tmp/picode-owned-playwright/node_modules/playwright-core/index.mjs");
const base = process.argv[2] || "http://127.0.0.1:18831";
const address = new URL(base);
assert.ok(address.protocol === "http:" && ["localhost", "127.0.0.1"].includes(address.hostname), "loopback HTTP fixture required");
const fleet = await fetch(new URL("/api/workspaces", base)).then(r => r.json());
assert.ok(fleet.length && fleet.every(w => /^\/tmp\/picode-docs-fixture-[^/]+\//.test(w.path)), "synthetic fixture required");
const workspace = fleet.find(w => w.name === "picode");
const agent = workspace?.agents.find(a => a.name === "Atlas");
assert.ok(workspace && agent, "picode/Atlas docs fixture required");
const out = resolve(process.argv[3] || "var/screenshots/mobile-v2/workflows");
mkdirSync(out, { recursive: true });
const browser = await chromium.launch({ headless: true, executablePath: process.env.PICODE_QA_CHROME || "/usr/bin/google-chrome" });
const report = { states: [], screenshots: [], audits: [], mockedWrites: [], consoleErrors: [] };
const piRows = Array.from({ length: 7 }, (_, i) => ({
  id: "qa-pi-" + i, path: workspace.path + "/sessions/" + (i ? "pi-session-" + i + ".jsonl" : "2026-09-07T12-30-00_workspace-mobile-refinement-with-a-long-session-name.jsonl"),
  cwd: workspace.path, messages: 14 + i, size: 12345, updatedAt: new Date().toISOString(), model: "example-model",
  ...(i === 1 ? { inUseBy: { agentId: agent.id, agentName: "Atlas" } } : {}),
}));
const cliRows = [{ id: "qa-claude", cli: "claude-code", name: "Claude review session", path: workspace.path + "/claude-review.jsonl", cwd: workspace.path, messages: 12, size: 2345, preview: "Review the workspace navigation and command composer.", resumeArgs: ["--resume", "qa-claude"], updatedAt: new Date().toISOString() }];
const automation = { id: "qa-automation", name: "Review overnight workspace changes and report failures", prompt: "Review changes and report the test results.", workspaceId: workspace.id, action: "start", cron: "0 9 * * 1-5", enabled: false, webhook: true, maxCostUsd: 2, maxRuns: 1, maxRunsWindowMin: 60 };

async function capture(page, name) {
  await page.evaluate(() => document.fonts.ready);
  await page.waitForTimeout(850);
  const audit = await page.evaluate(() => window.__picodeOverlayAudit());
  const width = await page.evaluate(() => ({ viewport: innerWidth, page: document.documentElement.scrollWidth }));
  const path = resolve(out, name + ".png");
  await page.screenshot({ path });
  report.screenshots.push(path);
  report.audits.push({ name, audit, width });
  assert.equal(audit.ok, true, name + ": overlay/control audit " + JSON.stringify(audit));
  assert.ok(width.page <= width.viewport, name + ": horizontal page overflow " + JSON.stringify(width));
}

try {
  for (const width of [320, 390]) {
    const context = await browser.newContext({ viewport: { width, height: width === 320 ? 740 : 844 }, isMobile: true, hasTouch: true, colorScheme: width === 320 ? "light" : "dark" });
    const state = { sessionsFail: false, sessionsEmpty: false, delayPi: false, autoMode: "empty", runsError: false, runs: [], previewOK: false, piMissing: false, liveHandoff: false, forceRequests: 0, releaseForce: null };
    await context.route("**/api/**", async route => {
      const request = route.request();
      const path = new URL(request.url()).pathname;
      const method = request.method();
      if (method !== "GET") {
        report.mockedWrites.push({ path, method });
        if (path.endsWith("/sessions/handoff/preview") && state.previewOK) {
          await route.fulfill({ json: { mode: "native", modes: ["native", "brief"], counts: { messages: 14, toolCalls: 2 }, manifest: { warnings: [] }, live: { name: "Atlas" } } });
        } else if (path.endsWith("/sessions/handoff") && state.liveHandoff) {
          if (!request.postDataJSON().force) {
            await route.fulfill({ status: 409, json: { error: "QA session still active", live: { name: "Atlas" } } });
          } else {
            state.forceRequests++;
            await new Promise(resolve => { state.releaseForce = resolve; });
            await route.fulfill({ status: 503, json: { error: "QA forced handoff unavailable" } });
          }
        } else {
          await route.fulfill({ status: 503, json: { error: "QA simulated save failure. No data was changed." } });
        }
        return;
      }
      if (path.startsWith("/api/clis/") && path.endsWith("/sessions")) {
        if (state.sessionsFail) return route.fulfill({ status: 503, json: { error: "QA sessions unavailable" } });
        const isPi = path.includes("/pi/");
        const rows = state.sessionsEmpty ? [] : isPi ? piRows : cliRows;
        if (isPi && state.delayPi) await new Promise(r => setTimeout(r, 2500));
        return route.fulfill({ json: { sessions: rows, totalBytes: 123456, cleanupDays: 0 } });
      }
      if (path === "/api/automations") {
        if (state.autoMode === "error") return route.fulfill({ status: 503, json: { error: "QA automations unavailable" } });
        return route.fulfill({ json: { items: state.autoMode === "populated" ? [automation] : [] } });
      }
      if (path.startsWith("/api/automations/") && path.endsWith("/runs")) {
        if (state.runsError) return route.fulfill({ status: 503, json: { error: "QA runs unavailable" } });
        return route.fulfill({ json: { items: state.runs } });
      }
      if (path === "/api/clis" && state.piMissing) {
        const response = await route.fetch();
        const body = await response.json();
        return route.fulfill({ json: { ...body, clis: (body.clis || []).map((c) => (c.id === "pi" ? { ...c, installed: false } : c)) } });
      }
      await route.continue();
    });
    const page = await context.newPage();
    page.on("pageerror", error => report.consoleErrors.push(error.message));
    let navigation = 0;
    const goto = async hash => {
      await page.goto(base + "/mobile/?theme=" + (width === 320 ? "light" : "dark") + "&qa-workflows=" + ++navigation + hash, { waitUntil: "domcontentloaded" });
      await page.locator(".m-screen").first().waitFor();
    };
    const nav = hash => page.evaluate(value => { location.hash = value; }, hash);
    const button = name => page.getByRole("button", { name, exact: true });
    const shot = name => capture(page, name + "-" + width);
    const closeDialog = async () => { await page.keyboard.press("Escape"); await page.locator(".dlg-overlay").waitFor({ state: "detached" }); };

    state.sessionsEmpty = true;
    await goto("#/clis/pi/sessions");
    await page.getByText("No Pi sessions yet", { exact: true }).waitFor();
    await shot("sessions-empty");
    state.sessionsEmpty = false;
    await button("Refresh").click();
    await page.waitForFunction(() => document.querySelectorAll(".sess-row").length === 7);
    await shot("sessions-populated");
    await page.getByRole("textbox", { name: "Search sessions" }).fill("no-match");
    await shot("sessions-filter-empty");
    await button("Clear search").click();
    state.sessionsFail = true;
    await button("Refresh").click();
    await page.getByText("QA sessions unavailable", { exact: false }).waitFor();
    assert.equal(await page.locator(".sess-row").count(), 7, "last sessions survive refresh errors");
    await shot("sessions-refresh-error");
    state.sessionsFail = false;
    await button("Retry").click();
    await page.getByText("QA sessions unavailable", { exact: false }).waitFor({ state: "detached" });
    state.delayPi = true;
    await button("Refresh").click();
    await page.locator('.cli-catalog a[href="#/clis/claude-code/sessions"]').click();
    await page.getByText("Claude review session", { exact: true }).waitFor();
    await page.waitForTimeout(2700);
    assert.equal(await page.getByText("qa-pi-0", { exact: true }).count(), 0, "old Pi response cannot replace selected CLI");
    assert.equal(await page.locator(".sess-row").count(), 1);
    await shot("sessions-switch-race");
    state.delayPi = false;
    await button("Open in terminal").click();
    await page.getByText("QA simulated save failure. No data was changed.", { exact: true }).waitFor();
    await shot("sessions-launch-error");
    await nav("#/clis/pi/sessions");
    await page.waitForFunction(() => document.querySelectorAll(".sess-row").length === 7);
    await page.locator(".sess-row").first().getByRole("button", { name: "Open with…", exact: true }).click();
    await page.getByRole("dialog").waitFor();
    await shot("sessions-open-sheet");
    await button("Open").click();
    await page.getByText("QA simulated save failure. No data was changed.", { exact: true }).first().waitFor();
    assert.equal(await page.getByRole("dialog").count(), 1, "failed resume preserves picker");
    await shot("sessions-resume-error");
    await closeDialog();
    await page.locator(".sess-row").first().getByRole("button", { name: "Delete", exact: true }).click();
    await page.getByRole("dialog").waitFor();
    await shot("sessions-delete-confirm");
    await button("Cancel").click();
    await page.locator(".dlg-overlay").waitFor({ state: "detached" });
    assert.equal(await page.locator(".sess-row").nth(1).getByRole("button", { name: "Delete", exact: true }).isDisabled(), true, "in-use session cannot be deleted");
    await page.locator(".sess-row").first().getByRole("button", { name: /^More actions/ }).click();
    await page.getByRole("menu").waitFor();
    await shot("sessions-handoff-menu");
    await page.keyboard.press("ArrowDown");
    await page.keyboard.press("Enter");
    await page.locator(".handoff-error").waitFor();
    await shot("sessions-handoff-error");
    state.previewOK = true;
    await button("Retry").click();
    await page.locator(".handoff-summary").waitFor();
    await page.waitForFunction(() => ![...document.querySelectorAll(".dlg button")].find(b => b.textContent === "Continue")?.disabled);
    await shot("sessions-handoff-recovered");
    for (let i = 0; i < 8; i++) {
      await page.keyboard.press("Tab");
      assert.equal(await page.evaluate(() => document.querySelector('[role="dialog"]').contains(document.activeElement)), true, "sheet traps keyboard focus");
    }
    state.liveHandoff = true;
    await button("Continue").click();
    await page.getByRole("dialog", { name: "Session still active", exact: true }).waitFor();
    await shot("sessions-handoff-live-confirm");
    await page.getByRole("dialog", { name: "Session still active", exact: true }).getByRole("button", { name: "Cancel", exact: true }).click();
    await page.getByRole("dialog", { name: "Session still active", exact: true }).waitFor({ state: "detached" });
    assert.equal(state.forceRequests, 0, "canceling confirmation never forces a handoff");
    assert.equal(await button("Continue").isEnabled(), true, "canceling confirmation releases the controls");
    await button("Continue").click();
    await page.getByRole("dialog", { name: "Session still active", exact: true }).waitFor();
    await page.waitForTimeout(250);
    await button("Continue anyway").click();
    await page.getByRole("dialog", { name: "Session still active", exact: true }).waitFor({ state: "detached" });
    await page.waitForFunction(() => [...document.querySelectorAll(".dlg button")].some(b => b.textContent === "Continuing…" && b.disabled));
    assert.equal(state.forceRequests, 1, "confirmation makes exactly one forced request");
    const handoffDialog = page.getByRole("dialog", { name: "Continue in Claude Code", exact: true });
    assert.equal(await handoffDialog.getByRole("button", { name: "Cancel", exact: true }).isDisabled(), true, "cancel stays disabled during forced request");
    for (const control of await handoffDialog.locator("input").all()) assert.equal(await control.isDisabled(), true, "options stay disabled during forced request");
    await page.keyboard.press("Escape");
    await page.mouse.click(4, 4);
    await page.waitForTimeout(650);
    assert.equal(await handoffDialog.isVisible(), true, "forced handoff cannot close by Escape or backdrop");
    const primary = handoffDialog.getByRole("button", { name: "Continuing…", exact: true });
    assert.equal(await primary.isDisabled(), true);
    const box = await primary.boundingBox();
    await page.mouse.click(box.x + box.width / 2, box.y + box.height / 2);
    await page.waitForTimeout(200);
    assert.equal(state.forceRequests, 1, "disabled primary cannot duplicate forced request");
    await shot("sessions-handoff-forced-pending");
    state.releaseForce();
    await handoffDialog.getByText("QA forced handoff unavailable", { exact: true }).waitFor();
    await page.waitForFunction(() => [...document.querySelectorAll(".dlg button")].some(b => b.textContent === "Continue" && !b.disabled));
    assert.equal(await handoffDialog.getByRole("button", { name: "Cancel", exact: true }).isEnabled(), true, "forced failure releases the controls");
    await shot("sessions-handoff-forced-error");
    await closeDialog();
    report.states.push({ width, passed: "sessions empty/filter/error retry/retention/CLI race/launch+resume failures/delete cancel/blocked delete/handoff retry/focus trap/live409 confirmation/forced pending lock/no duplicate/failure recovery" });

    await nav("#/automations");
    await page.getByText("No automations yet.", { exact: true }).waitFor();
    await shot("automations-empty");
    state.autoMode = "error";
    await goto("#/automations");
    await page.getByText("QA automations unavailable", { exact: false }).waitFor();
    await shot("automations-load-error");
    state.autoMode = "empty";
    await button("Retry").click();
    await page.getByText("No automations yet.", { exact: true }).waitFor();
    await shot("automations-list-recovered");
    state.autoMode = "error";
    await goto("#/automations/qa-automation");
    await page.getByText("QA automations unavailable", { exact: false }).waitFor();
    assert.equal(await page.locator(".mcp-skel").count(), 0, "detail initial error must end loading");
    await shot("automations-detail-load-error");
    state.autoMode = "populated";
    state.runsError = true;
    await button("Retry").click();
    await page.getByText("QA runs unavailable", { exact: true }).waitFor();
    await shot("automations-runs-error");
    state.runsError = false;
    await button("Retry").click();
    await page.getByText("No runs yet. Use Run now to try it.", { exact: true }).waitFor();
    await shot("automations-runs-recovered");
    await button("Delete").click();
    await page.getByRole("dialog").waitFor();
    await shot("automations-delete-confirm");
    await button("Cancel").click();
    await page.locator(".dlg-overlay").waitFor({ state: "detached" });
    await nav("#/automations/new");
    await page.locator(".auto-form").waitFor();
    await shot("automations-new");
    await page.getByPlaceholder("Nightly test run", { exact: true }).fill("QA form remains unsaved");
    await page.locator(".auto-textarea").fill("Check the mobile layout.");
    await page.locator(".auto-textarea").focus();
    await page.keyboard.press("Tab");
    assert.equal(await page.evaluate(() => document.activeElement.id), "auto-schedule-on", "keyboard advances from prompt to schedule");
    await button("Create automation").click();
    await page.getByText("QA simulated save failure. No data was changed.", { exact: true }).waitFor();
    assert.equal(await page.getByPlaceholder("Nightly test run", { exact: true }).inputValue(), "QA form remains unsaved");
    await shot("automations-save-error");
    await button("All automations").click();
    await page.getByRole("dialog").waitFor();
    await shot("automations-discard-confirm");
    await button("Cancel").click();
    await page.locator(".dlg-overlay").waitFor({ state: "detached" });
    assert.equal(await page.getByPlaceholder("Nightly test run", { exact: true }).inputValue(), "QA form remains unsaved", "cancel discard preserves draft");
    await button("All automations").click();
    await page.getByRole("dialog").waitFor();
    await page.waitForTimeout(250);
    await button("Discard changes").click();
    await page.locator(".dlg-overlay").waitFor({ state: "detached" });
    await shot("automations-list");
    state.runs = [{ id: "qa-run", firedAt: new Date().toISOString(), trigger: "manual", status: "failed", reason: "QA model unavailable", costUsd: 0.02 }];
    await nav("#/automations/qa-automation");
    await page.getByText("QA model unavailable", { exact: false }).waitFor();
    await shot("automations-runs-populated");
    const runTable = page.locator(".auto-table-wrap");
    await runTable.focus();
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(250);
    assert.ok(await runTable.evaluate(e => e.scrollWidth <= e.clientWidth || e.scrollLeft > 0), "run table scrolls with the keyboard");
    await button("Edit").click();
    await page.getByPlaceholder("Nightly test run", { exact: true }).fill("QA edited automation");
    await button("Save").click();
    await page.getByText("QA simulated save failure. No data was changed.", { exact: true }).waitFor();
    assert.equal(await page.getByPlaceholder("Nightly test run", { exact: true }).inputValue(), "QA edited automation", "failed edit retains changes");
    await shot("automations-edit-save-error");
    await button("Cancel").click();
    await page.getByRole("dialog").waitFor();
    await shot("automations-edit-discard-confirm");
    await page.getByRole("dialog").getByRole("button", { name: "Cancel", exact: true }).click();
    await page.locator(".dlg-overlay").waitFor({ state: "detached" });
    assert.equal(await page.getByPlaceholder("Nightly test run", { exact: true }).inputValue(), "QA edited automation", "cancel discard retains edit");
    await button("Cancel").click();
    await page.getByRole("dialog").waitFor();
    await page.waitForTimeout(250);
    await button("Discard changes").click();
    await page.locator(".dlg-overlay").waitFor({ state: "detached" });
    await page.locator(".auto-detail").waitFor();
    state.piMissing = true;
    state.autoMode = "populated"; // a start automation needs Pi; an empty list shows no banner (ADR-0179)
    await goto("#/automations");
    await page.getByText("Pi is not installed, so automations that start or message a Pi agent cannot run.", { exact: true }).waitFor();
    await shot("automations-blocked");
    report.states.push({ width, passed: "automations empty/initial+detail+runs error retry/delete cancel/create+edit save failure/discard cancel+confirm/keyboard prompt+run table/blocked dependency" });
    await context.close();
    console.log(`Mobile workflows ${width}: PASS`);
  }
  assert.deepEqual(report.consoleErrors, [], "no uncaught application errors");
  report.ok = true;
} finally {
  writeFileSync(resolve(out, "result.json"), JSON.stringify(report, null, 2) + "\n");
  await browser.close();
}
