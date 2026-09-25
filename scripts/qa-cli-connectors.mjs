#!/usr/bin/env node
// Connectors pane boundaries against an owned scratch instance only
// (scripts/qa-scratch.sh). Refuses any other HOME: the fixture is the
// scratch's isolated home, and nothing here touches real config.
//
//   scripts/qa-scratch.sh start <name> && scripts/qa-scratch.sh seed <name>
//   node scripts/qa-cli-connectors.mjs http://localhost:<port> [outDir]
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdirSync, rmSync, writeFileSync } from "node:fs";
import { resolve, dirname, join } from "node:path";

const base = new URL(process.argv[2]);
assert.ok(base.protocol === "http:" && ["localhost", "127.0.0.1"].includes(base.hostname), "Use the scratch's http://localhost URL");
const out = resolve(process.argv[3] || "var/screenshots/cli-connectors/browser");
mkdirSync(out, { recursive: true });

const api = async (path, method = "GET", body) => {
  const res = await fetch(new URL(path, base), { method, ...(body ? { headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) } : {}) });
  assert.ok(res.ok, `${method} ${path}: ${res.status} ${res.ok ? "" : await res.text()}`);
  return res.status === 204 ? null : res.json();
};

const workspaces = await api("/api/workspaces");
const workspace = workspaces.find((w) => w.agents && w.agents.length);
assert.ok(workspace, "Seed the scratch first: scripts/qa-scratch.sh seed <name>");
const agent = workspace.agents[0];
const ws = workspace.id;
const id = agent.id;

const first = await api(`/api/mcp?workspace=${ws}&agent=${id}`);
const layer = (first.layers || []).find((l) => l.id === "pi-global");
assert.ok(layer && /[/\\]var[/\\]qa[/\\][^/\\]+[/\\]home[/\\]/.test(layer.path), `Refuse non-scratch instance: ${layer && layer.path}`);
const settingsFile = join(dirname(layer.path), "settings.json");
const serversFile = layer.path;
const writeSettings = (packages) => writeFileSync(settingsFile, JSON.stringify({ packages }));
const writeServers = (servers) => writeFileSync(serversFile, JSON.stringify({ mcpServers: servers }));

let session = "cli-connectors-" + process.pid;
const browser = (...args) => execFileSync("agent-browser", ["--session", session, ...args], { encoding: "utf8", maxBuffer: 4 * 1024 * 1024 }).trim();
const ev = (code) => JSON.parse(browser("eval", code));
// Expression waits poll the page instead of relying on the CLI's --fn parsing.
async function until(code, ms = 15000) {
  const started = Date.now();
  for (;;) {
    if (ev(code)) return true;
    if (Date.now() - started > ms) throw new Error("timeout waiting for: " + code);
    await new Promise((done) => setTimeout(done, 150));
  }
}
const fault = (code) => ev(`window.qaFetch ||= window.fetch; ${code}`);
const settle = (ms = 400) => new Promise((done) => setTimeout(done, ms));
const ready = () => until('!!document.querySelector("#connectors-view .connectors-head")');
const context = `#/clis/pi/connectors?workspaceId=${ws}&agentId=${id}`;
const root = "#/clis/pi/connectors";
let visit = 0;
const open = (app, hash, width = app === "mobile" ? 390 : 1365) => {
  browser("open", new URL(`/${app}/?qa=${process.pid}-${++visit}${hash}`, base).href);
  browser("set", "viewport", String(width), String(app === "mobile" ? 844 : 1000));
  assert.deepEqual(ev("[innerWidth,innerHeight]"), [width, app === "mobile" ? 844 : 1000]);
};
async function capture(name, audit = true) {
  await new Promise((done) => setTimeout(done, 500));
  if (audit) {
    const report = ev("window.__picodeOverlayAudit()");
    writeFileSync(join(out, name + "-audit.json"), JSON.stringify(report, null, 2));
    assert.equal(report.ok, true, name + " geometry");
  }
  browser("screenshot", join(out, name + ".png"));
}

const results = [];
try {
  for (const app of (process.env.QA_APPS?.split(",") || ["desktop", "mobile"])) {
    try { browser("close"); } catch {}
    session = "cli-connectors-" + process.pid + "-" + app;

    // Adapter absent: one installation action, no catalog and no scope control.
    writeSettings([]);
    writeServers({});
    open(app, context);
    browser("wait", "#connectors-view .mcp-empty");
    assert.match(ev('document.querySelector("#connectors-view .mcp-empty").innerText'), /Install the MCP adapter/);
    assert.equal(ev('document.querySelectorAll("#connectors-view .connector-option").length'), 0);
    assert.equal(ev('!!document.querySelector("#connectors-view .connectors-head .btn")'), false);
    await capture(app + "-blocked");
    results.push(app + ": adapter absent shows one installation action only");

    // Adapter present, nothing configured.
    writeSettings(["npm:pi-mcp-adapter"]);
    browser("reload");
    // The add door is the Marketplace tab beside Installed (since the
    // connectors pane took the one-pane shape; the old Add dialog is gone).
    await until('[...document.querySelectorAll("#connectors-view .pkg-fine")].some(e=>/No connectors yet/.test(e.textContent))');
    assert.equal(ev('[...document.querySelectorAll("#connectors-view [role=tab]")].some(t=>t.textContent.startsWith("Marketplace"))'), true);
    assert.equal(ev('document.querySelectorAll("#connectors-view .mcp-row").length'), 0);
    await capture(app + "-empty");
    results.push(app + ": empty state is one line, with the Marketplace tab as the add door");

    // Configured: rows, a disabled row, the stopped-agent line and control rhythm.
    writeServers({
      context7: { url: "https://mcp.context7.com/mcp" },
      github: { url: "https://api.githubcopilot.com/mcp", auth: "oauth" },
      "local-files": { command: "npx", args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"] },
      notion: { url: "https://mcp.notion.com/mcp", auth: "oauth", disabled: true },
    });
    browser("reload");
    browser("wait", '#connectors-view .mcp-row');
    assert.equal(ev('document.querySelectorAll("#connectors-view .mcp-row").length'), 4);
    assert.equal(ev('document.querySelectorAll("#connectors-view .mcp-row.off").length'), 1);
    assert.equal(ev('document.querySelector("#connectors-view .mcp-row.off .rx-switch").getAttribute("data-state")'), "unchecked");
    assert.equal(ev('document.querySelectorAll("#connectors-view .rx-switch[data-state=checked]").length'), 3);
    assert.equal(ev('document.querySelector("#connectors-view .pkg-fine").textContent.trim()'), "Live status comes from a running agent.");
    assert.equal(ev('!!document.querySelector("#connectors-view .mcp-live")'), false, "no live claim while the agent is stopped");
    assert.equal(ev('!!document.querySelector("#connectors-view .settings-ctx")'), false, "no glued workspace line");
    const actionHeights = ev('JSON.stringify([...document.querySelectorAll("#connectors-view .mcp-row-actions")].map(r=>[...r.children].map(c=>Math.round(c.getBoundingClientRect().height))))');
    for (const row of JSON.parse(actionHeights)) for (const h of row) assert.equal(h, 36, "row controls share --ctl-h");
    await capture(app + "-configured");
    results.push(app + ": rows carry scope, target, a real switch and the absent-live-status line");

    // The Marketplace: search filters, Add writes to the chosen scope, the card flips to Added.
    const card = '#connectors-view .pkg-card';
    // The connector cards on screen: visible, and named (the phone mounts
    // other card lists in the same view).
    const cards = `[...document.querySelectorAll(${JSON.stringify(card)})].filter(c=>c.offsetParent&&c.querySelector(".pkg-card-name"))`;
    fault(`window.qaWrites=[];window.fetch=(url,opts)=>{if(opts?.method&&opts.method!=="GET")window.qaWrites.push({url:String(url),method:opts.method,body:opts.body?JSON.parse(opts.body):null});return window.qaFetch(url,opts);}`);
    browser("find", "role", "tab", "click", "--name", "Marketplace");
    await until(`${cards}.length >= 6`);
    // A service already in the selected layer reads Added (and cannot be added twice).
    assert.equal(ev(`${cards}.some(c=>c.querySelector(".pkg-card-name").textContent==="Context7"&&c.querySelector(".pkg-card-foot button").textContent==="Added"&&c.querySelector(".pkg-card-foot button").disabled)`), true);
    browser("fill", '[aria-label="Search connectors"]', "library documentation");
    await until(`${cards}.length === 1`);
    assert.equal(ev(`${cards}[0].querySelector(".pkg-card-name").textContent`), "Context7");
    browser("fill", '[aria-label="Search connectors"]', "deepwiki");
    await until(`${cards}.length === 1`);
    await settle();
    ev(`${cards}[0].querySelector(".pkg-card-foot button").click(); true`);
    await until('window.qaWrites.some(w=>w.url==="/api/mcp")');
    let write = ev('window.qaWrites.filter(w=>w.url==="/api/mcp").at(-1)');
    assert.equal(write.method, "POST");
    assert.equal(write.body.name, "deepwiki");
    assert.equal(write.body.scope, "user");
    // workspaceId/agentId are the pane's context; scope is the decision.
    await until(`${cards}.some(c=>{const b=c.querySelector(".pkg-card-foot button");return b.textContent==="Added"&&b.disabled})`);
    await capture(app + "-marketplace");
    results.push(app + ": the Marketplace searches, and Add carries the machine scope");

    // Switching the target writes the scope onto the route and onto the request.
    const pillChecked = `[...document.querySelectorAll("#connectors-view .pkg-scope [role=radio]")].some(b=>b.textContent===${JSON.stringify(workspace.name)}&&b.getAttribute("aria-checked")==="true")`;
    browser("find", "role", "radio", "click", "--name", workspace.name, "--exact");
    await until(pillChecked);
    await until('location.hash.includes("scope=workspace")');
    browser("find", "role", "tab", "click", "--name", "Marketplace");
    browser("wait", card);
    browser("fill", '[aria-label="Search connectors"]', "chrome");
    await until(`${cards}.length === 1`);
    await settle();
    ev(`${cards}[0].querySelector(".pkg-card-foot button").click(); true`);
    await until('window.qaWrites.filter(w=>w.url==="/api/mcp"&&w.body.name==="chrome-devtools").length>0');
    write = ev('window.qaWrites.filter(w=>w.url==="/api/mcp").at(-1)');
    assert.equal(write.body.scope, "project");
    assert.equal(write.body.workspaceId, ws);
    assert.equal(write.body.agentId, id);
    await capture(app + "-add-workspace-scope");
    results.push(app + ": the labelled target decides where the connector is written and survives reload");
    browser("reload");
    try { browser("find", "role", "tab", "click", "--name", "Installed"); } catch {}
    browser("wait", '#connectors-view .mcp-row');
    assert.equal(ev('document.querySelectorAll("#connectors-view .mcp-row").length'), 6);
    assert.equal(ev('location.hash.includes("scope=workspace")'), true);
    // A reload drops the in-page recorder; re-arm it for the removal that follows.
    fault(`window.qaWrites=[];window.fetch=(url,opts)=>{if(opts?.method&&opts.method!=="GET")window.qaWrites.push({url:String(url),method:opts.method,body:opts.body?JSON.parse(opts.body):null});return window.qaFetch(url,opts);}`);

    // Remove: the menu, the confirmation naming the file, the DELETE call.
    browser("click", "#connectors-view .mcp-row .mcp-more");
    browser("wait", ".um-popover .um-item");
    assert.deepEqual(ev('[...document.querySelectorAll(".um-popover .um-item")].map(i=>i.textContent)'), ["Remove"]);
    browser("click", ".um-popover .um-item");
    browser("wait", '[role="alertdialog"], [role="dialog"]');
    assert.match(ev('document.querySelector(".dlg").innerText'), /mcp\.json/, "the confirmation names the file");
    await capture(app + "-remove-confirm");
    browser("find", "role", "button", "click", "--name", "Remove", "--exact");
    await until('(window.qaWrites||[]).some(w=>w.method==="DELETE")');
    write = ev('(window.qaWrites||[]).filter(w=>w.method==="DELETE").at(-1)');
    assert.match(write.url, /^\/api\/mcp\?/);
    await until('!document.querySelector(".pkg-job")');
    assert.equal(ev('document.querySelectorAll("#connectors-view .mcp-row").length'), 5);
    ev("window.fetch=window.qaFetch");
    results.push(app + ": removal is confirmed by name and sends one DELETE");

    // Narrow widths keep every control reachable and every overlay contained.
    if (app === "desktop") {
      open("desktop", context, 1024);
      ready();
      await capture("desktop-1024", true);
      browser("find", "role", "tab", "click", "--name", "Marketplace");
      browser("wait", card);
      await capture("desktop-1024-marketplace", true);
    } else {
      open("mobile", context);
      ready();
      await until('[...document.querySelectorAll("#connectors-view [role=tab]")].some(t=>t.textContent.startsWith("Marketplace"))');
      browser("find", "role", "tab", "click", "--name", "Marketplace");
      browser("wait", card);
      await capture("mobile-marketplace", true);
    }
    results.push(app + ": narrow layout keeps controls reachable and overlays inside the viewport");
  }
  writeFileSync(join(out, "results.json"), JSON.stringify({ ok: true, results, realDownloads: false, realModelTurns: false }, null, 2));
  console.log(JSON.stringify({ ok: true, results }, null, 2));
} catch (error) {
  try {
    writeFileSync(join(out, "failure.json"), JSON.stringify({ error: String(error), snapshot: browser("snapshot", "-i") }, null, 2));
    browser("screenshot", join(out, "failure.png"));
  } catch {}
  throw error;
} finally {
  // The workspace layer is the worktree itself: leave no fixture file behind.
  try { rmSync(join(workspace.path, ".mcp.json"), { force: true }); } catch {}
  try { rmSync(join(workspace.path, ".pi", "mcp.json"), { force: true }); } catch {}
  try { browser("close"); } catch {}
}
