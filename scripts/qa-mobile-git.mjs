// Isolated Git component acceptance against a disposable docs fixture.
// Usage: node scripts/qa-mobile-git.mjs http://127.0.0.1:18846 [output-dir]
// Seeds only the guarded fixture's Git repository and a local bare remote.
// All application writes are refused; delivery callbacks only record arguments.
import assert from "node:assert/strict";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const repo = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const base = process.argv[2] || "http://127.0.0.1:18846";
const address = new URL(base);
assert.ok(address.protocol === "http:" && ["localhost", "127.0.0.1"].includes(address.hostname), "loopback HTTP fixture required");
const fleet = await fetch(new URL("/api/workspaces", base)).then(r => r.json());
assert.ok(fleet.length && fleet.every(w => /^\/tmp\/picode-docs-fixture-[^/]+\//.test(w.path)), "synthetic docs fixture required");
const workspace = fleet.find(w => w.name === "picode");
assert.ok(workspace, "picode fixture workspace required");
const root = workspace.path, fixture = root.split("/work/")[0];
const git = (...args) => execFileSync("git", ["-C", root, "-c", "user.name=QA Fixture", "-c", "user.email=qa@example.invalid", ...args], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }).trim();
if (!git("log", "--all", "--format=%s").includes("QA merge branch topology")) {
  git("add", "-A"); if (git("status", "--porcelain")) git("commit", "-m", "QA preserve baseline fixture changes");
  git("switch", "-c", "qa-side"); writeFileSync(resolve(root, "qa-side.txt"), "Side branch review\n"); git("add", "qa-side.txt"); git("commit", "-m", "QA side branch adds review notes with a long subject that wraps on a narrow mobile screen");
  git("switch", "main"); writeFileSync(resolve(root, "qa-main.txt"), "Main branch implementation\n"); git("add", "qa-main.txt"); git("commit", "-m", "QA main branch implementation"); git("merge", "--no-ff", "qa-side", "-m", "QA merge branch topology");
}
const remote = resolve(fixture, "qa-remote.git");
execFileSync("git", ["init", "--bare", remote], { stdio: "ignore" });
git("remote", ...(git("remote").split("\n").includes("origin") ? ["set-url", "origin", remote] : ["add", "origin", remote]));
assert.equal(git("remote", "get-url", "--push", "origin"), remote, "push destination must be owned local bare fixture");
git("push", "-u", "origin", "main"); git("push", "origin", "qa-side");
const sibling = resolve(fixture, "work/qa-sibling");
if (!git("worktree", "list", "--porcelain").includes("worktree " + sibling)) git("worktree", "add", "-b", "qa-sibling", sibling, "HEAD~1");
writeFileSync(resolve(sibling, "sibling-change.txt"), "Sibling only\n");
writeFileSync(resolve(root, "qa-main.txt"), "Main branch implementation\nMobile diff refresh content\n");
mkdirSync(resolve(root, "src/deeply-nested-workspace-folder"), { recursive: true });
writeFileSync(resolve(root, "src/deeply-nested-workspace-folder/mobile-review-with-a-long-filename-and-readable-path.txt"), "New mobile file\n");

const out = resolve(process.argv[3] || "var/screenshots/mobile-git");
mkdirSync(out, { recursive: true });
const harness = resolve(repo, "web/mobile/var/git-qa");
mkdirSync(harness, { recursive: true });
writeFileSync(resolve(harness, "index.html"), '<!doctype html><html><head><meta name="viewport" content="width=device-width,initial-scale=1" /></head><body><div id="root"></div><script type="module" src="./harness.jsx"></script></body></html>');
writeFileSync(resolve(harness, "harness.jsx"), readFileSync(resolve(repo, "scripts/fixtures/mobile-git-harness.jsx")));
const { createServer } = await import(resolve(repo, "web/node_modules/vite/dist/node/index.js"));
const { applicationConfig } = await import(resolve(repo, "web/tools/vite-config.mjs"));
const port = Number(process.env.PICODE_GIT_QA_PORT || 18848);
const config = applicationConfig("mobile", port);
config.configFile = false; config.server.host = "127.0.0.1"; config.server.proxy = { "/api": { target: base } };
const server = await createServer(config);
await server.listen();
const { chromium } = await import(process.env.PICODE_PLAYWRIGHT_MODULE || "/tmp/picode-owned-playwright/node_modules/playwright-core/index.mjs");
const browser = await chromium.launch({ headless: true, executablePath: process.env.PICODE_QA_CHROME || "/usr/bin/google-chrome" });
const report = { states: [], screenshots: [], audits: [], reads: [], pageErrors: [], refusedWrites: [] };
const endpoint = "/api/workspaces/" + workspace.id;
const graph = await fetch(base + endpoint + "/git?root=" + encodeURIComponent(root)).then(r => r.json());
assert.ok(graph.commits.some(c => c.parents.length > 1), "real merge topology required");
const status = await fetch(base + endpoint + "/gitstatus?root=" + encodeURIComponent(root)).then(r => r.json());

async function capture(page, name) {
  await page.evaluate(() => document.fonts.ready); await page.waitForTimeout(850);
  const audit = await page.evaluate(() => window.__picodeOverlayAudit());
  const width = await page.evaluate(() => ({ viewport: innerWidth, page: document.documentElement.scrollWidth }));
  const path = resolve(out, name + ".png"); await page.screenshot({ path });
  report.screenshots.push(path); report.audits.push({ name, audit, width });
  assert.equal(audit.ok, true, name + ": " + JSON.stringify(audit));
  assert.ok(width.page <= width.viewport, name + ": horizontal page overflow");
}
try {
  for (const width of [320, 390]) {
    const context = await browser.newContext({ viewport: { width, height: width === 320 ? 740 : 844 }, isMobile: true, hasTouch: true });
    const state = { fail: "", empty: false, noGit: false, moved: false, following: false, diffVersion: 1, graphMode: "", binary: false, graphDelay: 0, pr: { status: "none", branch: "main" } };
    await context.route("**/api/**", async route => {
      const req = route.request(), url = new URL(req.url()), path = url.pathname;
      if (req.method() !== "GET") { report.refusedWrites.push(path); return route.fulfill({ status: 503, json: { error: "QA refuses application writes" } }); }
      if (!path.startsWith(endpoint + "/")) return route.continue();
      report.reads.push(url.pathname + url.search);
      if (state.fail && path.endsWith("/" + state.fail)) return route.fulfill({ status: 503, json: { error: "QA Git read unavailable" } });
      if (state.moved && url.searchParams.get("root") === root) return route.fulfill({ status: 409, json: { error: "owner moved", cwd: sibling } });
      if (state.moved && path.endsWith("/browse") && !url.searchParams.has("root")) { state.following = true; return route.fulfill({ json: { root: sibling, entries: [] } }); }
      if (state.following && path.endsWith("/gitstatus")) return route.fulfill({ json: { ...status, branch: "qa-sibling", changes: [] } });
      if (path.endsWith("/git")) {
        if (state.graphDelay) await new Promise(resolve => setTimeout(resolve, state.graphDelay));
        if (state.graphMode === "empty") return route.fulfill({ json: { ...graph, commits: [], worktrees: [], more: false } });
        if (state.graphMode === "many") {
          const baseCommit = graph.commits.at(-1);
          const commits = Array.from({ length: 12 }, (_, i) => ({ hash: (i + 1).toString(16).padStart(40, "0"), parents: [baseCommit.hash], subject: "Parallel branch " + (i + 1), author: "QA Fixture", at: baseCommit.at + i + 1 }));
          return route.fulfill({ json: { ...graph, commits: [...commits, baseCommit], refs: [], worktrees: [], more: false } });
        }
      }
      if (state.binary && path.endsWith("/gitstatus")) return route.fulfill({ json: { ...status, changes: [{ path: "qa-image.png", kind: "modified", binary: true }] } });
      if (state.binary && path.endsWith("/gitdiff")) return route.fulfill({ json: { binary: true, kind: "modified", path: "qa-image.png" } });
      if (state.binary && (path.endsWith("/blob"))) return route.fulfill({ contentType: "image/svg+xml", body: '<svg xmlns="http://www.w3.org/2000/svg" width="640" height="360"><rect width="640" height="360" fill="#2563eb"/><circle cx="320" cy="150" r="64" fill="white"/><text x="320" y="270" text-anchor="middle" font-family="sans-serif" font-size="36" fill="white">Mobile image preview</text></svg>' });
      if (path.endsWith("/pr")) return route.fulfill({ json: state.pr });
      if (path.endsWith("/gitstatus") && !url.searchParams.has("worktree") && (state.empty || state.noGit)) return route.fulfill({ json: { ...status, git: !state.noGit, changes: [], totals: { add: 0, del: 0 } } });
      if (path.endsWith("/gitdiff") && state.diffVersion === 2) return route.fulfill({ json: { patch: "@@ -1 +1 @@\n-Main branch implementation\n+QA refreshed diff content\n", truncated: true } });
      return route.continue();
    });
    const page = await context.newPage(); page.on("pageerror", e => report.pageErrors.push(e.message));
    await page.goto(`http://127.0.0.1:${port}/mobile/var/git-qa/index.html`);
    await page.getByRole("button", { name: "History", exact: true }).waitFor();
    await page.evaluate(theme => { document.documentElement.dataset.theme = theme; }, width === 320 ? "light" : "dark");
    const reset = async () => { await page.evaluate(() => renderGit()); await page.locator(".m-git").waitFor(); };
    await capture(page, `changes-${width}`);
    await page.getByRole("button", { name: /^qa-main.txt/ }).click(); await page.getByRole("region", { name: "Changes in qa-main.txt", exact: true }).waitFor();
    await capture(page, `diff-${width}`);
    state.diffVersion = 2; await page.evaluate(() => window.dispatchEvent(new Event("focus")));
    await page.getByText("QA refreshed diff content", { exact: true }).waitFor(); await page.getByText("This diff is too large", { exact: false }).waitFor();
    state.fail = "gitdiff"; await page.evaluate(() => window.dispatchEvent(new Event("focus"))); await page.getByRole("alert").waitFor();
    assert.equal(await page.getByText("QA refreshed diff content", { exact: true }).count(), 1, "refresh error retains last good diff");
    await capture(page, `diff-error-retained-${width}`);
    state.fail = ""; await page.getByRole("button", { name: "Retry", exact: true }).click(); await page.getByRole("alert").waitFor({ state: "hidden" });
    await page.getByRole("button", { name: "Back", exact: true }).click();
    await page.getByRole("button", { name: "History", exact: true }).click(); await page.locator(".m-git-ancestry circle").first().waitFor();
    const rows = await page.locator(".m-git-graph-row").count();
    assert.ok(rows > graph.commits.length, "dirty worktrees have anchored pseudo rows");
    await capture(page, `ancestry-${width}`);
    await page.getByRole("searchbox").fill("side branch");
    assert.equal(await page.locator(".m-git-graph-row").count(), rows, "search keeps graph rows truthful");
    assert.ok(await page.locator(".m-git-graph-row.is-dim").count());
    await capture(page, `ancestry-search-${width}`);
    await page.getByRole("searchbox").press("Enter"); await page.getByRole("button", { name: "Clear search" }).click();
    await page.getByRole("button", { name: /^QA merge branch topology/ }).click(); await page.getByText("Parents", { exact: true }).waitFor();
    await capture(page, `merge-detail-${width}`);
    await page.locator(".m-git-file-row").first().click(); await capture(page, `commit-file-${width}`);
    await page.getByRole("button", { name: "Back", exact: true }).click(); await page.getByRole("button", { name: "Back", exact: true }).click();
    await page.getByRole("button", { name: /^Uncommitted · qa-sibling/ }).click(); await page.getByRole("button", { name: /^sibling-change.txt/ }).click();
    assert.equal(await page.getByRole("button", { name: "Open file", exact: true }).count(), 0, "sibling worktree stays read only");
    assert.equal(await page.getByRole("button", { name: "Git actions", exact: true }).count(), 0);
    assert.ok(report.reads.some(r => r.includes("gitdiff?") && r.includes("worktree=qa-sibling") && r.includes(encodeURIComponent(sibling))));
    await capture(page, `sibling-diff-${width}`); await reset();
    await page.getByRole("button", { name: "History", exact: true }).click(); await page.getByRole("button", { name: "Branches", exact: true }).click(); await capture(page, `branches-${width}`);
    await page.getByLabel("Include remote branches").uncheck(); await Promise.all([page.waitForResponse(r => r.url().includes("/git?") && r.url().includes("remotes=0")), page.getByRole("button", { name: "Apply", exact: true }).click()]);
    state.fail = "git"; await page.getByRole("button", { name: "Refresh", exact: true }).click(); await page.getByRole("alert").waitFor();
    assert.ok(await page.locator(".m-git-graph-row").count(), "graph refresh error retains last good graph");
    await capture(page, `history-error-${width}`); state.fail = ""; await page.getByRole("button", { name: "Retry", exact: true }).click(); await page.getByRole("alert").waitFor({ state: "hidden" });
    await page.getByRole("button", { name: "Git actions" }).click(); await capture(page, `actions-${width}`);
    await page.getByRole("button", { name: "Commit", exact: true }).click(); await page.getByRole("button", { name: "Prepare", exact: true }).click(); await page.getByRole("alert").waitFor();
    await page.getByLabel("Commit message", { exact: true }).fill("QA's mobile change");
    await page.getByText("Review command", { exact: true }).click();
    await page.evaluate(() => { qaGit.delay = true; qaGit.fail = true; qaGit.deliveries = []; });
    await page.locator(".m-git-sheet form").evaluate(el => { el.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })); el.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })); });
    await page.getByRole("status").filter({ hasText: "Sending this action" }).waitFor();
    assert.equal(await page.evaluate(() => qaGit.deliveries.length), 1, "same-task action double submit guarded");
    assert.equal(await page.getByRole("button", { name: "Close", exact: true }).isDisabled(), true);
    await page.keyboard.press("Escape"); assert.equal(await page.getByRole("dialog").count(), 1, "pending sheet cannot close");
    await capture(page, `action-pending-${width}`);
    await page.evaluate(() => qaGit.release()); await page.getByRole("alert").filter({ hasText: "QA channel unavailable" }).waitFor();
    assert.equal(await page.getByLabel("Commit message", { exact: true }).inputValue(), "QA's mobile change");
    await capture(page, `action-error-${width}`);
    await page.evaluate(() => { qaGit.delay = false; qaGit.fail = false; });
    await page.getByLabel("Send through").selectOption("run"); await page.getByRole("button", { name: "Run when idle", exact: true }).click(); await page.getByText("Terminal ready.", { exact: true }).waitFor();
    const delivery = await page.evaluate(() => qaGit.deliveries.at(-1)); assert.equal(delivery.args[1], root); assert.equal(delivery.args[3].run, true); assert.ok(delivery.args[2].includes("'QA'\\''s mobile change'"));
    await page.getByRole("button", { name: "Done", exact: true }).click();
    await page.getByRole("button", { name: "Pull request", exact: true }).click(); await page.getByRole("button", { name: "Create in terminal", exact: true }).waitFor();
    await capture(page, `pr-empty-${width}`);
    for (const reason of ["none", "gh-unauth"]) {
      state.pr = reason === "none" ? { status: "none", branch: "main" } : { status: "blocked", reason, message: "Sign in to GitHub CLI to view this pull request." };
      await page.getByRole("button", { name: "Refresh", exact: true }).click();
      const button = page.getByRole("button", { name: reason === "none" ? "Create in terminal" : "Log in from terminal", exact: true }); await button.waitFor();
      await page.evaluate(() => { qaGit.delay = true; qaGit.fail = true; qaGit.deliveries = []; });
      await button.evaluate(el => { el.dispatchEvent(new MouseEvent("click", { bubbles: true })); el.dispatchEvent(new MouseEvent("click", { bubbles: true })); });
      await page.getByRole("status").filter({ hasText: "Preparing the terminal" }).waitFor(); assert.equal(await page.evaluate(() => qaGit.deliveries.length), 1, "PR duplicate submit guarded");
      await page.evaluate(() => qaGit.release()); await page.getByRole("alert").waitFor();
      await capture(page, `pr-${reason}-error-${width}`);
    }
    state.pr = { status: "ok", pr: { number: 42, title: "Improve mobile project navigation and Git review", state: "OPEN", head: "qa-side", base: "main", url: "https://example.invalid/fixture/pull/42", checks: { failed: 1, passed: 3, failing: [{ name: "Mobile visual review", url: "https://example.invalid/check" }] }, additions: 42, deletions: 7, changedFiles: 3, updatedAt: new Date().toISOString(), reviewDecision: "REVIEW_REQUIRED" } };
    await page.getByRole("button", { name: "Refresh", exact: true }).click(); await page.getByText(state.pr.pr.title, { exact: true }).waitFor(); await capture(page, `pr-populated-${width}`);
    state.graphMode = "empty"; state.graphDelay = 1800; await reset(); await page.getByRole("button", { name: "History", exact: true }).click();
    await page.getByRole("status", { name: "Loading Git" }).waitFor(); await capture(page, `history-loading-${width}`);
    await page.getByText("No commits in this history.", { exact: true }).waitFor(); await capture(page, `history-empty-${width}`);
    state.graphDelay = 0; state.graphMode = "many"; await page.getByRole("button", { name: "Show all branches", exact: true }).click(); await page.getByText("Parallel branch 1", { exact: true }).waitFor();
    const gutter = await page.locator(".m-git-ancestry-viewport").evaluate(el => ({ width: el.clientWidth, scroll: el.scrollWidth }));
    assert.ok(gutter.width <= 72 && gutter.scroll > gutter.width, "only graph gutter scrolls for many lanes");
    await capture(page, `many-lanes-${width}`); state.graphMode = "";
    state.binary = true; await reset(); await page.getByRole("button", { name: /^qa-image.png/ }).click(); await page.getByRole("button", { name: "Enlarge after version" }).waitFor();
    await page.locator(".m-git-image img").first().evaluate(img => img.decode()); await capture(page, `binary-diff-${width}`);
    const oldImage = await page.locator(".m-git-image img").last().getAttribute("src");
    await page.evaluate(() => window.dispatchEvent(new Event("focus")));
    await page.waitForFunction(old => document.querySelectorAll(".m-git-image img")[1]?.getAttribute("src") !== old, oldImage);
    await page.getByRole("button", { name: "Enlarge after version" }).click(); await capture(page, `image-overlay-${width}`); await page.keyboard.press("Escape");
    assert.equal(await page.getByRole("dialog", { name: "Image preview" }).count(), 0); state.binary = false;
    const askFleet = [{ ...workspace, agents: workspace.agents.map((a, i) => ({ ...a, running: i === 0, mode: i === 0 ? "managed" : "stopped", streaming: i === 0, workPath: root })) }];
    await page.evaluate(workspaces => { qaGit.delay = false; qaGit.fail = false; qaGit.deliveries = []; renderGit({ workspaces }); }, askFleet);
    await page.getByRole("button", { name: "Git actions" }).click(); await page.getByRole("button", { name: "Commit", exact: true }).click();
    await page.getByLabel("Send through").selectOption("ask:" + workspace.agents[0].id);
    assert.equal(await page.getByLabel("Commit message (optional)").inputValue(), "");
    await page.getByRole("button", { name: "Ask Atlas", exact: true }).click(); await page.getByText("Queued for Atlas: commit after its current turn.", { exact: true }).waitFor();
    assert.equal(await page.evaluate(() => qaGit.deliveries.at(-1).channel), "agent");
    assert.equal(await page.evaluate(() => qaGit.deliveries.at(-1).args[2]), root);
    await capture(page, `agent-queued-${width}`); await page.getByRole("button", { name: "Done", exact: true }).click();
    state.empty = true; await reset(); await page.getByText("No uncommitted changes.", { exact: true }).waitFor(); await capture(page, `changes-empty-${width}`);
    state.empty = false; state.noGit = true; await reset(); await page.getByText("This folder is not a Git repository.").waitFor(); await capture(page, `not-git-${width}`);
    state.noGit = false; state.fail = "browse"; await page.evaluate(() => renderGit()); await page.getByRole("alert").waitFor(); await capture(page, `initial-error-${width}`); state.fail = ""; await page.getByRole("button", { name: "Retry", exact: true }).click(); await page.getByRole("button", { name: "History", exact: true }).waitFor();
    state.moved = true; await page.evaluate(() => window.dispatchEvent(new Event("focus"))); await page.getByRole("button", { name: "Follow folder", exact: true }).waitFor(); assert.equal(await page.getByRole("button", { name: "Git actions" }).isDisabled(), true);
    const frozenReads = report.reads.length; await page.evaluate(() => window.dispatchEvent(new Event("focus"))); await page.waitForTimeout(350); assert.equal(report.reads.length, frozenReads, "moved owner freezes reads");
    await capture(page, `owner-moved-${width}`); await page.getByRole("button", { name: "Follow folder", exact: true }).click(); await page.getByText("qa-sibling", { exact: true }).waitFor(); assert.equal(state.following, true); await capture(page, `owner-followed-${width}`);
    report.states.push({ width, result: "PASS", rows }); await context.close();
  }
  assert.deepEqual(report.pageErrors, []); assert.deepEqual(report.refusedWrites, []);
  writeFileSync(resolve(out, "report.json"), JSON.stringify(report, null, 2));
  console.log(JSON.stringify({ result: "PASS", widths: report.states, screenshots: report.screenshots.length, reads: report.reads.length }));
} finally { await browser.close(); await server.close(); }
