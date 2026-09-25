#!/usr/bin/env node
// Exercise real native-settings APIs only against the synthetic docs fixture.
// The settings pane edits one layer at a time and keeps the keyboard map on
// its own sub-tab (docs/plans/cli-settings-ux.md): this script asserts the
// layer switcher, the route it writes, provenance and the reset round trip.
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
const base = new URL(process.argv[2] || "http://localhost:18861");
assert.ok(["localhost", "127.0.0.1"].includes(base.hostname) && base.protocol === "http:");
const out = resolve(process.argv[3] || "var/screenshots/cli-native-settings/browser");
mkdirSync(out, { recursive: true });
const api = async (path, method = "GET", body) => {
  const response = await fetch(new URL(path, base), { method, ...(body ? { headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) } : {}) });
  assert.ok(response.ok, `${method} ${path}: ${response.status}`);
  return response.status === 204 ? null : response.json();
};
const workspaces = await api("/api/workspaces");
const workspace = workspaces.find(w => w.name === "picode" && /[\\/]picode-docs-fixture-[^\\/]+[\\/]work[\\/]picode$/.test(w.path));
assert.ok(workspace, "Refuse to mutate a non-synthetic fixture");
const agent = workspace.agents.find(a => a.name === "Atlas");
assert.equal(agent?.mode, "stopped");
const id = agent.id;
const terminals = await api("/api/terminals");
const terminal = (terminals.terminals || terminals).find(t => t.running) || (terminals.terminals || terminals)[0];
assert.ok(terminal?.id, "the fixture must have a terminal for the sidebar-context row");
const session = "cli-settings-qa-" + process.pid;
const browser = (...args) => execFileSync("agent-browser", ["--session", session, ...args], { encoding: "utf8", maxBuffer: 4 * 1024 * 1024 }).trim();
const evaluate = code => JSON.parse(browser("eval", code));
const navigate = hash => evaluate(`location.hash=${JSON.stringify(hash)}`);
// The pane is ready when its sub-tabs are painted and the visible form is
// editable (the keyboard map has no form of its own).
const ready = () => browser("wait", "--fn", '!!document.querySelector("#pi-settings-view .settings-section") && !document.querySelector(".pi-settings-fields[disabled]:not([hidden])")');
// The pane tabs are links, not buttons: click the real one so the router, the
// href and the app's own rewrite effect are all exercised (a URL the harness
// typed itself cannot catch a route the app rewrites away — 2026-09-12).
const paneClick = label => evaluate(`[...document.querySelectorAll(".cli-pane-tabs a")].find(a=>a.textContent.trim()===${JSON.stringify(label)}).click(); true`);
// The layer marker sits on the body wrapper, not on each section (the machine
// layer renders one section per native settings group since internal/clisettings
// reached pi). Ask the wrapper; an earlier version read the first section and
// silently answered "".
const layerOf = () => evaluate('document.querySelector("#pi-settings-view [data-layer]")?.dataset.layer || ""');
const settingsHash = "#/clis/pi/settings";
// The machine layer, named: the pane follows the selected workspace (the
// ADR-0179 context rule), so a bare address lands on that workspace's layer
// once the sidebar holds one of its terminals.
const globalHash = settingsHash + "?scope=global";
const canonicalContext = "#/clis/pi/settings?agentId=" + encodeURIComponent(id);
const contextHash = canonicalContext;
// The fixture's machine layer is disposable: start every run from empty so a
// leftover key from an earlier run cannot shadow the provenance assertions.
writeFileSync((await api("/api/pi-settings")).global.path, "{}\n");
// The keyboard rows assume the machine's map is untouched — the Off facet
// counts the actions pi ships unbound, and Find by key expects a chord nobody
// uses — so clear every override this catalog knows before the first capture,
// the same reason the settings file above starts empty (a previous run, or a
// manual probe on this fixture, otherwise leaves one behind).
await api("/api/cli-keys", "PUT", { cli: "pi", resetAll: true });
// Same for the guest CLI this run edits (P2b): its map lives in the fixture's
// isolated HOME, and a leftover row from an earlier run would read as a change
// this run did not make.
await api("/api/cli-keys", "PUT", { cli: "omp", resetAll: true });
const compactRowSet = () => evaluate('[...document.querySelectorAll("#pi-settings-view .set-row")].find(r=>r.querySelector("#g-compactionEnabled"))?.classList.contains("is-set") === true');
// Toasts are the app's, not the pane's: a save two steps earlier leaves a stack
// at the viewport's bottom edge, which on a phone lands on top of whatever the
// next pane draws there. Wait for them to leave before judging a capture.
// On a phone the pane sits below the CLI's header and the shell scrolls its own
// column (.m-screen.m-more-page, 844px tall with 1042 of content), so a capture
// at scrollTop 0 photographs the header and the pane's edge. Center it first —
// the same reason every other mobile target in this file is centered.
const centerPane = (cli, block = "center") => evaluate(`(() => { const p = document.querySelector('#cli-settings-view .key-pane[data-cli="${cli}"]'); p.scrollIntoView({ block: ${JSON.stringify(block)} }); return true; })()`);
// A tall pane cannot be centered: its head is what a reader judges, and on a
// phone the shell's sticky header sits over the first line — so scroll it to the
// top, then back off by the header's height.
const headPane = (cli) => evaluate(`(() => {
  const p = document.querySelector('#cli-settings-view .key-pane[data-cli="${cli}"]');
  p.scrollIntoView({ block: "start" });
  for (let el = p.parentElement; el; el = el.parentElement) {
    const oy = getComputedStyle(el).overflowY;
    if (/auto|scroll/.test(oy) && el.scrollHeight > el.clientHeight + 1) { el.scrollTop -= 120; break; }
  }
  return true;
})()`);
const settleToasts = async () => {
  for (let i = 0; i < 40; i++) {
    // The element leaving the DOM is not the pixels leaving: a capture two
    // frames later caught the fade as an unlabelled pill.
    if (evaluate('!document.querySelector("[data-sonner-toast]")')) {
      await new Promise(resolve => setTimeout(resolve, 450));
      return;
    }
    await new Promise(resolve => setTimeout(resolve, 200));
  }
};
const results = [];
async function capture(name, audit = false) {
  // Let native controls finish painting before judging pixels.
  await new Promise(resolve => setTimeout(resolve, 250));
  if (audit) {
    // Toasts and mobile sheets animate in from the bottom edge, so for ~200ms
    // they are genuinely outside the viewport and the audit reports a clipped
    // overlay that no reader would ever see. Wait for the stack to hold still
    // (two identical reads) before asking it — a settled clip is still a FAIL.
    browser("wait", "--fn", '(() => { const box = [...document.querySelectorAll("[data-sonner-toast], .dlg, [role=alertdialog]")].map(e => { const r = e.getBoundingClientRect(); return Math.round(r.top) + ":" + Math.round(r.bottom); }).join(","); const still = box === window.__qaOverlayBox; window.__qaOverlayBox = box; return still; })()');
    const report = evaluate("window.__picodeOverlayAudit()");
    writeFileSync(resolve(out, name + "-audit.json"), JSON.stringify(report, null, 2));
    assert.equal(report.ok, true, name);
  }
  browser("screenshot", resolve(out, name + ".png"));
}
try {
  for (const app of ["desktop", "mobile"]) {
    browser("set", "viewport", ...(app === "desktop" ? ["1365", "1000"] : ["390", "844"]));
    browser("open", new URL(`/${app}/?theme=dark${globalHash}`, base).href);
    ready();
    // No agent in the route: one body (this machine), no agent pill at all.
    // The section count is not the acceptance: the machine layer now renders
    // one section per native settings group (Session, Model, Approvals,
    // Interface — internal/clisettings declares them for pi too since native
    // settings reached every CLI), so pinning it to 1 was pinning a count that
    // a legitimate change moved. What this row means is the assertions below:
    // the layer, the pills and the agent body.
    assert.ok(evaluate('document.querySelectorAll("#pi-settings-view .settings-section").length') >= 1);
    assert.equal(layerOf(), "global");
    assert.equal(evaluate('document.querySelectorAll("#pi-settings-view .settings-layer .pkg-scope-btn").length'), 1);
    assert.equal(evaluate('document.querySelectorAll("[data-layer=agent]").length'), 0);
    assert.equal(evaluate('!!document.querySelector("#pi-settings-view .settings-ctx")'), false);
    await capture(app + "-global", true);
    const pattern = `qa-${app}-${process.pid}-*`;
    const before = await api("/api/pi-settings");
    browser("click", "#g-compactionEnabled"); ready();
    assert.equal((await api("/api/pi-settings")).global.compactionEnabled, !before.global.compactionEnabled);
    results.push(app + ": global round-trip");

    // The keyboard map is a pane next to Settings (no sub-tab row anywhere),
    // machine-wide, and rebuilt (docs/plans/keyboard-pane.md): a toolbar whose
    // facets count, one row per action with the row's own actions in a fixed
    // column, and the notes pi's own defaults need — the file it writes, the
    // platform it binds for, the chords a browser keeps.
    paneClick("Keyboard");
    browser("wait", "--fn", '!!document.querySelector("#keys-filter")');
    assert.equal(evaluate('location.hash.split("?")[0].endsWith("/keyboard")'), true);
    assert.equal(evaluate('document.querySelectorAll("#pi-settings-view .pkg-tabs, #pi-settings-view .settings-layer").length'), 0);
    const keyReport = await api("/api/cli-keys?cli=pi");
    assert.equal(evaluate('document.querySelectorAll("#pi-settings-view .key-row").length'), keyReport.actions.length);
    // The period rides inside the code box with the path (the P0 visual fix), so
    // the assertion accepts it rather than the end of the string.
    assert.match(evaluate('document.querySelector("#pi-settings-view .settings-desc code").textContent'), /keybindings\.json\.?$/, "the pane names the file it writes");
    assert.equal(evaluate('document.querySelector("#pi-settings-view .key-bar .key-facet").textContent'), "All" + keyReport.actions.length);
    // A chord is a value, not a control: the keycap is its own height (24px on
    // a pointer, 28px on a phone), never the 36px --ctl-h it shipped with.
    assert.equal(evaluate('document.querySelector("#pi-settings-view .key-chip").offsetHeight'), app === "desktop" ? 24 : 28);
    // Density, which nothing measured before the owner photographed 216px rows
    // and a 700px void between a label and its chord (2026-09-21). Each row is
    // one line of action, one line of chords, one optional note; the label cell
    // is never taller than a line; and the keycaps sit next to the action they
    // belong to, not at the far edge of the card. A two-chord row with a
    // browser-reserved warning (Page down: PageDown + Ctrl+PageDown) wraps its
    // note to three lines at 1365px — 106px measured 2026-09-25 — so the
    // per-row ceiling is 112; the total still catches the "screens of air".
    assert.ok(evaluate('[...document.querySelectorAll("#pi-settings-view .key-label")].every(l => l.getBoundingClientRect().height <= 44)'), "a label cell must be one line, not a stretched flex item");
    assert.ok(evaluate('(() => { const rows = [...document.querySelectorAll("#pi-settings-view .key-row")]; const name = rows[0].querySelector(".key-name"); const keys = rows[0].querySelector(".key-keys"); const label = name.getBoundingClientRect(); const chip = keys.querySelector(".key-chip").getBoundingClientRect(); return Math.abs(label.top - chip.top) <= 12; })()'), "the action and its keycap belong on one line");
    assert.ok(evaluate('(() => { const row = document.querySelector("#pi-settings-view .key-row"); const gap = row.querySelector(".key-keys").getBoundingClientRect().left - row.querySelector(".key-name").getBoundingClientRect().right; return gap <= 48; })()'), "the keycaps follow the label instead of floating at the card's edge");
    assert.ok(evaluate(`(() => { const rows = [...document.querySelectorAll("#pi-settings-view .key-row")]; const heights = rows.map(r => r.getBoundingClientRect().height); const total = heights.reduce((a, b) => a + b, 0); return Math.max(...heights) <= ${app === "desktop" ? 112 : 128} && total <= ${app === "desktop" ? 4000 : 6200}; })()`), "90 rows stay a list, not six screens of air");
    assert.ok(evaluate('document.querySelectorAll("#pi-settings-view .key-row .key-alt").length') >= 8, "the rows pi binds differently elsewhere say so");
    assert.ok(evaluate('document.querySelectorAll("#pi-settings-view .key-note.is-warn").length') >= 5, "the chords the browser keeps are marked");
    assert.ok(evaluate('document.querySelectorAll("#pi-settings-view .key-note:not(.is-warn)").length') > 20, "shared chords are reported, not hidden");
    await capture(app + "-keyboard", true);

    // A facet narrows the list to its own state; the count is the number of
    // rows on screen, not a badge that lies.
    evaluate('[...document.querySelectorAll("#pi-settings-view .key-facet")].find(b=>b.textContent.startsWith("Off")).click(); true');
    browser("wait", "--fn", 'document.querySelectorAll("#pi-settings-view .key-row").length === document.querySelectorAll("#pi-settings-view .key-none").length');
    const offRows = evaluate('document.querySelectorAll("#pi-settings-view .key-row").length');
    assert.equal(offRows, keyReport.actions.filter(a => !a.alt && !(a.defaults || []).length).length, "the Off facet is the actions pi ships unbound");
    await capture(app + "-keyboard-off");
    evaluate('[...document.querySelectorAll("#pi-settings-view .key-facet")].find(b=>b.textContent.startsWith("All")).click(); true');
    browser("wait", "--fn", 'document.querySelectorAll("#pi-settings-view .key-row").length > 50');

    // Find by key: press a chord, get the actions that answer to it — none on
    // a fresh fixture — and the chip that clears the filter.
    evaluate('document.querySelector("#pi-settings-view .key-find").click(); true');
    browser("press", "Control+Alt+k");
    browser("wait", "--fn", '!!document.querySelector("#pi-settings-view .key-facet.is-chord")');
    assert.match(evaluate('document.querySelector("#pi-settings-view .key-facet.is-chord").textContent'), /Ctrl\+Alt\+K/);
    assert.equal(evaluate('document.querySelectorAll("#pi-settings-view .key-row").length'), 0);
    await capture(app + "-keyboard-keyfilter");
    // A DOM click after centering the target: the mobile shell's notice banner
    // covers the top of the pane and a role-based click on a covered control is
    // refused (the same reason the capture step below centers its row).
    evaluate('(() => { const b = [...document.querySelectorAll("#pi-settings-view .key-pane button")].find(x => x.textContent.trim() === "Show all"); b.scrollIntoView({ block: "center" }); b.click(); return true; })()');
    browser("wait", "--fn", 'document.querySelectorAll("#pi-settings-view .key-row").length > 50');
    results.push(app + ": facets and the key filter narrow the map, and both clear");

    // The capture itself: Add key, press a chord, and the map takes it. The row
    // wears the changed bar while it is set, and Reset all hands every changed
    // key back — the fixture is pristine again afterwards.
    const firstAction = keyReport.actions[0];
    // Center the row first: the mobile shell's notice banner covers the top of
    // the form and a ref click on a covered Add is refused (2026-09-12).
    evaluate('(() => { const row = document.querySelectorAll("#pi-settings-view .key-row")[0]; row.scrollIntoView({ block: "center" }); [...row.querySelectorAll("button")].find(b => b.textContent === "Add key").click(); return true; })()');
    browser("wait", "--fn", '!!document.querySelector("#pi-settings-view .key-chip.is-listen")');
    await capture(app + "-keyboard-capture");
    browser("press", "Control+Alt+k");
    let bound = "";
    for (let i = 0; i < 20 && !bound; i++) {
      await new Promise(resolve => setTimeout(resolve, 150));
      bound = ((await api("/api/cli-keys?cli=pi")).user?.[firstAction.id] || []).find(key => /ctrl\+alt\+k/i.test(key)) || "";
    }
    assert.ok(bound, "the captured chord must be saved to the key map");
    browser("wait", "--fn", 'document.querySelectorAll("#pi-settings-view .key-row.is-changed").length === 1');
    assert.equal(evaluate('!!document.querySelector("#pi-settings-view .key-bar .btn")'), true, "a changed map offers Reset all");
    await capture(app + "-keyboard-changed");
    evaluate('document.querySelector("#pi-settings-view .key-bar .btn").click(); true');
    browser("wait", '[role="alertdialog"], .dlg');
    // A mobile dialog is a sheet that slides in from the bottom: judge its
    // geometry once it is fully inside the viewport, not mid-animation.
    browser("wait", "--fn", '(() => { const d = document.querySelector("[role=alertdialog], .dlg"); if (!d) return false; const r = d.getBoundingClientRect(); return r.top >= 0 && r.bottom <= innerHeight; })()');
    await capture(app + "-keyboard-reset-all", true);
    evaluate('(() => { const d = document.querySelector("[role=alertdialog]") || document.querySelector(".dlg"); [...d.querySelectorAll("button")].find(b => b.textContent.trim() === "Reset all").click(); return true; })()');
    let cleared = false;
    for (let i = 0; i < 20 && !cleared; i++) {
      await new Promise(resolve => setTimeout(resolve, 150));
      const user = (await api("/api/cli-keys?cli=pi")).user || {};
      cleared = !Object.keys(user).some(k => keyReport.actions.some(a => a.id === k));
    }
    assert.ok(cleared, "Reset all must clear every override this catalog knows");
    // The sheet animates out; the audit refuses an overlay clipped by the
    // viewport, so wait for it to leave before judging the pane again.
    browser("wait", "--fn", '!document.querySelector("[role=alertdialog], .dlg")');
    browser("wait", "--fn", 'document.querySelectorAll("#pi-settings-view .key-row.is-changed").length === 0');
    await capture(app + "-keyboard-reset");
    await api("/api/cli-keys", "PUT", { cli: "pi", action: firstAction.id, reset: true });
    results.push(app + ": a captured chord is saved, marks the row, and Reset all hands every key back");

    // The same screen for the CLIs whose maps PiCode writes through their own
    // files (P2b, P3): Omp's flat YAML and Codex's nested TOML render from the
    // registry's catalog, a captured chord lands in that CLI's file in that
    // CLI's own spelling, and Reset all hands every row back. The fixture's HOME
    // is its own, so none of this touches the machine's real ~/.omp or ~/.codex.
    for (const guest of [
      { cli: "omp", rows: 70, chord: "Control+Alt+o", spelled: /ctrl[+-]alt[+-]o/i, file: /keybindings\.(yml|yaml|json)/ },
      { cli: "codex", rows: 149, chord: "Control+Alt+m", spelled: /ctrl-alt-m/i, file: /config\.toml/ },
      { cli: "agy", rows: 36, chord: "Control+Alt+a", spelled: /ctrl[+-]alt[+-]a/i, file: /keybindings\.json/ },
      { cli: "opencode", rows: 162, chord: "Control+Alt+p", spelled: /ctrl[+-]alt[+-]p/i, file: /tui\.json/ },
    ]) {
      const { cli } = guest;
      navigate("#/clis/" + cli + "/keyboard");
      browser("wait", "--fn", `document.querySelectorAll('#cli-settings-view .key-pane[data-cli=${cli}] .key-row').length > ${guest.rows - 10}`);
      const report = await api("/api/cli-keys?cli=" + cli);
      assert.equal(report.state, "shipped", cli + " ships an editor");
      assert.equal(evaluate(`document.querySelectorAll('#cli-settings-view .key-pane[data-cli=${cli}] .key-row').length`), report.actions.length, cli + "'s pane lists the catalog the CLI has");
      assert.equal(evaluate(`document.querySelector('#cli-settings-view .key-pane[data-cli=${cli}] .key-bar .key-facet').textContent`), "All" + report.actions.length);
      assert.match(evaluate(`document.querySelector('#cli-settings-view .key-pane[data-cli=${cli}] .settings-desc code').textContent`), guest.file, cli + "'s pane names the file it writes");
      headPane(cli);
      await settleToasts();
      await capture(app + "-keyboard-" + cli);
      const row = report.actions[0];
      // The capture above is a pause in the flow, not a state to lean on: the
      // pane is re-asserted here, so a re-render between the two cannot turn
      // this click into a TypeError on a missing row (2026-09-21).
      browser("wait", "--fn", `!!document.querySelector('#cli-settings-view .key-pane[data-cli=${cli}] .key-row')`);
      evaluate(`(() => { const r = document.querySelector('#cli-settings-view .key-pane[data-cli=${cli}] .key-row'); r.scrollIntoView({ block: "center" }); [...r.querySelectorAll("button")].find(b => b.textContent === "Add key").click(); return true; })()`);
      browser("wait", "--fn", `!!document.querySelector('#cli-settings-view .key-pane[data-cli=${cli}] .key-chip.is-listen')`);
      browser("press", guest.chord);
      let bound = "";
      for (let i = 0; i < 20 && !bound; i++) {
        await new Promise(resolve => setTimeout(resolve, 150));
        bound = (((await api("/api/cli-keys?cli=" + cli)).user || {})[row.id] || []).find(k => guest.spelled.test(k)) || "";
      }
      assert.ok(bound, cli + ": the captured chord must reach the CLI's own file, in its own spelling");
      assert.equal(evaluate(`document.querySelectorAll('#cli-settings-view .key-pane[data-cli=${cli}] .key-row.is-changed').length`), 1);
      assert.equal(evaluate(`!!document.querySelector('#cli-settings-view .key-pane[data-cli=${cli}] .key-bar .btn')`), true, cli + ": a changed map offers Reset all");
      headPane(cli);
      await settleToasts();
      await capture(app + "-keyboard-" + cli + "-changed");
      evaluate(`document.querySelector('#cli-settings-view .key-pane[data-cli=${cli}] .key-bar .btn').click(); true`);
      browser("wait", "[role=alertdialog], .dlg");
      browser("wait", "--fn", '(() => { const d = document.querySelector("[role=alertdialog], .dlg"); if (!d) return false; const r = d.getBoundingClientRect(); return r.top >= 0 && r.bottom <= innerHeight; })()');
      await capture(app + "-keyboard-" + cli + "-reset-all", true);
      evaluate('(() => { const d = document.querySelector("[role=alertdialog]") || document.querySelector(".dlg"); [...d.querySelectorAll("button")].find(b => b.textContent.trim() === "Reset all").click(); return true; })()');
      let cleared = false;
      for (let i = 0; i < 20 && !cleared; i++) {
        await new Promise(resolve => setTimeout(resolve, 150));
        cleared = Object.keys((await api("/api/cli-keys?cli=" + cli)).user || {}).length === 0;
      }
      assert.ok(cleared, cli + ": Reset all must hand every row back");
    }
    results.push(app + ": another CLI's map is edited through the same screen and its own file");

    // Every CLI the registry knows answers its own Keyboard pane (ADR-0174): a
    // guest gets its state and one action, and never Pi's rows — which is what
    // a pane that only knew one CLI used to render behind "coming soon".
    for (const [cli, want] of [["hermes", /a few keys/], ["grok", /built in/], ["claude-code", /has not shipped yet/]]) {
      navigate(`#/clis/${cli}/keyboard`);
      browser("wait", "--fn", `!!document.querySelector('#cli-settings-view .key-pane[data-cli="${cli}"] .cli-notice a')`);
      assert.match(evaluate(`document.querySelector('#cli-settings-view .key-pane[data-cli="${cli}"] .cli-notice span').textContent`), want, cli + " must state its own case");
      assert.equal(evaluate('document.querySelectorAll("#cli-settings-view .key-row").length'), 0, cli + " must not render Pi's rows");
      assert.equal(evaluate('!!document.querySelector("#pi-settings-view")'), false, cli + " must not borrow Pi's frame");
      if (cli !== "grok") { centerPane(cli); await settleToasts(); await capture(app + "-keyboard-" + cli); }   // grok is captured below with the overlay audit
    }
    // The action is a real control, and which kind it is shows in how it opens:
    // a vendor's page is a new tab, an in-app route is navigation. Both are
    // asserted, because a note whose action does nothing is not an action.
    navigate("#/clis/grok/keyboard");
    browser("wait", "--fn", '!!document.querySelector("#cli-settings-view .key-pane[data-cli=grok] .cli-notice a")');
    assert.equal(evaluate('document.querySelector("#cli-settings-view .key-pane[data-cli=grok] .cli-notice a").getAttribute("target")'), "_blank", "a vendor's page opens in a new tab");
    centerPane("grok");
    await settleToasts();
    await capture(app + "-keyboard-grok", true);
    navigate("#/clis/hermes/keyboard");
    browser("wait", "--fn", '!!document.querySelector("#cli-settings-view .key-pane[data-cli=hermes] .cli-notice a")');
    assert.equal(evaluate('document.querySelector("#cli-settings-view .key-pane[data-cli=hermes] .cli-notice a").getAttribute("href")'), "#/clis/hermes/settings");
    assert.equal(evaluate('document.querySelector("#cli-settings-view .key-pane[data-cli=hermes] .cli-notice a").getAttribute("target")'), null, "an in-app route stays in the tab");
    evaluate('document.querySelector("#cli-settings-view .key-pane[data-cli=hermes] .cli-notice a").click(); true');
    browser("wait", "--fn", 'location.hash === "#/clis/hermes/settings"');
    // Hermes' three rebindable keys are a group in its Settings pane, which is
    // where its Keyboard pane sends the reader.
    browser("wait", "--fn", '!!document.querySelector("#cli-settings-view .settings-section h3")');
    assert.ok(evaluate('[...document.querySelectorAll("#cli-settings-view .settings-section")].some(s => s.querySelector("h3")?.textContent === "Keyboard" && s.querySelectorAll(".set-row").length === 3)'), "hermes must show its three keyboard rows");
    results.push(app + ": a CLI with no key-map editor gets its state and one action, and its own keys are rows");
    navigate(globalHash);
    ready();
    paneClick("Settings");
    browser("wait", "--fn", '!!document.querySelector("#pi-settings-view .settings-layer")');

    // The pane tab keeps working while the sidebar holds a terminal: the
    // settings route stays unscoped and the pane's "carry the selected agent"
    // rewrite watches it — the shape that silently reverted to Settings when
    // the map was a `?tab=keys` sub-tab (2026-09-12).
    if (app === "desktop") {
      navigate("#/term/" + encodeURIComponent(terminal.id));
      await new Promise(resolve => setTimeout(resolve, 400));
      navigate(settingsHash);
      // The route carries the terminal's workspace now; the pane under it may
      // be that folder's (untrusted) layer, so wait for the route, not a form.
      browser("wait", "--fn", 'location.hash.startsWith("#/clis/pi/settings") && !!document.querySelector(".cli-pane-tabs a")');
      paneClick("Keyboard");
      browser("wait", "--fn", '!!document.querySelector("#keys-filter")');
      await new Promise(resolve => setTimeout(resolve, 1200));
      assert.equal(evaluate('location.hash.split("?")[0].endsWith("/keyboard")'), true, "the pane rewrite must not undo a pane click");
      results.push(app + ": a Keyboard click survives the pane's own route rewrite");
      // Back to the machine layer: the provenance step below reads its toggle.
      navigate(globalHash);
      ready();
    }

    // Provenance: a value this layer sets is marked, and Use inherited hands it
    // back to the parent instead of freezing a copy.
    browser("wait", "--fn", '[...document.querySelectorAll("#pi-settings-view .set-row")].find(r=>r.querySelector("#g-compactionEnabled"))?.classList.contains("is-set") === true');
    assert.equal(evaluate('[...document.querySelectorAll("#pi-settings-view .set-row")].find(r=>r.querySelector("#g-compactionEnabled"))?.querySelector(".set-src").textContent'), "Set here");
    assert.equal((await api("/api/pi-settings")).global.has.compactionEnabled, true);
    browser("find", "role", "button", "click", "--name", "Use inherited", "--exact");
    browser("wait", "--fn", '[...document.querySelectorAll("#pi-settings-view .set-row")].find(r=>r.querySelector("#g-compactionEnabled"))?.classList.contains("is-set") === false');
    const afterReset = (await api("/api/pi-settings")).global;
    assert.equal(afterReset.has?.compactionEnabled ?? false, false);
    assert.equal(evaluate('[...document.querySelectorAll("#pi-settings-view .set-row")].find(r=>r.querySelector("#g-compactionEnabled"))?.querySelector(".set-src").textContent'), "Pi default");
    await capture(app + "-reset", true);
    results.push(app + ": provenance marker and Use inherited round trip");

    // Another CLI's pane never renders Pi's editor. The row used to wait on
    // the "in development" notice, which no managed CLI shows any more: the
    // native-settings landing gave every one of them an editor, so it waits on
    // the pane instead and keeps the assertion that mattered.
    navigate(globalHash.replace("/pi", "/codex"));
    browser("wait", "#cli-settings-view");
    assert.equal(evaluate('document.querySelectorAll("#g-compactionEnabled").length'), 0);
    await capture(app + "-other-cli");
    results.push(app + ": another CLI's pane refuses the Pi editor");

    navigate(settingsHash + "?agentId=missing-settings-fixture-agent");
    browser("wait", "[role=alert]");
    assert.equal(evaluate('document.querySelectorAll("#g-compactionEnabled").length'), 0);
    await capture(app + "-missing-agent");
    results.push(app + ": missing agent refuses fallback");

    // An agent in the route opens that agent's layer, and the switcher offers
    // exactly the layers this context has.
    navigate(canonicalContext);
    ready();
    assert.equal(evaluate("location.hash"), canonicalContext);
    assert.equal(layerOf(), "agent");
    assert.deepEqual(evaluate('[...document.querySelectorAll("#pi-settings-view .settings-layer .pkg-scope-btn")].map(b=>b.textContent)'), ["Global", "picode", "Atlas"]);
    await capture(app + "-context", true);
    if (!(await api("/api/pi-settings?agentId=" + encodeURIComponent(id))).writable.project) {
      browser("find", "role", "radio", "click", "--name", "picode", "--exact");
      browser("wait", "--fn", '!!document.querySelector("#pi-settings-view .cli-notice")');
      assert.equal(evaluate('document.querySelectorAll("#w-steeringMode").length'), 0);
      browser("find", "role", "radio", "click", "--name", "Atlas", "--exact");
      browser("wait", "--fn", '!!document.querySelector("#ag-set-thinking")');
      results.push(app + ": untrusted workspace blocks project controls");
    }
    const layerBeforeReload = layerOf();
    browser("reload"); ready();
    assert.equal(evaluate("location.hash").startsWith(canonicalContext), true);
    assert.equal(layerOf(), layerBeforeReload);
    browser("wait", "--fn", '[...document.querySelector("#ag-set-thinking").options].some(o=>o.value==="medium")');
    browser("select", "#ag-set-thinking", "medium"); ready();
    assert.equal((await api("/api/pi-settings?agentId=" + encodeURIComponent(id))).agent.thinking, "medium");
    results.push(app + ": agent context route, reload and correct agent write");
    await api("/api/agents/" + encodeURIComponent(id), "PATCH", { thinking: "high" });
    browser("wait", "--fn", 'document.querySelector("#ag-set-thinking")?.value==="high"');
    results.push(app + ": external agent changes arrive through feed");

    // Switching layer rewrites the route; a reload lands on the same layer.
    browser("find", "role", "radio", "click", "--name", "Global", "--exact");
    // The hash moves first; wait for the rendered layer, not the URL.
    browser("wait", "--fn", 'document.querySelector("#pi-settings-view [data-layer]")?.dataset.layer === "global"');
    assert.equal(evaluate('location.hash.includes("scope=global")'), true);
    browser("reload"); ready();
    assert.equal(layerOf(), "global");
    assert.equal(evaluate('!!document.querySelector("#pi-settings-view .settings-file")'), true);
    results.push(app + ": the switcher writes the layer onto the route and survives a reload");

    navigate(globalHash);
    ready();
    evaluate('window.qaFetch=window.fetch;window.fetch=(url,opts)=>opts?.method==="PUT"&&String(url).includes("/api/pi-settings")?Promise.resolve(new Response(JSON.stringify({error:"QA: save unavailable"}),{status:503,headers:{"Content-Type":"application/json"}})):window.qaFetch(url,opts)');
    const compact = evaluate('document.querySelector("#g-compactionEnabled").getAttribute("aria-checked")');
    browser("click", "#g-compactionEnabled"); ready(); browser("wait", "[role=alert]");
    assert.equal(evaluate('document.querySelector("#g-compactionEnabled").getAttribute("aria-checked")'), compact);
    browser("fill", '[aria-label="Model pattern"]', pattern);
    browser("click", ".set-pat-add button"); ready();
    assert.equal(evaluate('document.querySelector(".set-pat-add input").value'), pattern);
    await capture(app + "-save-error");
    evaluate('window.fetch=window.qaFetch');
    browser("click", ".set-pat-add button"); ready();
    assert.equal(evaluate('document.querySelector(".set-pat-add input").value'), "");
    results.push(app + ": failed save rolls back and retains pattern for retry");

    navigate("#/preferences");
    evaluate('window.qaFetch=window.fetch;window.fetch=(url,opts)=>String(url).includes("/api/pi-settings")?Promise.resolve(new Response(JSON.stringify({error:"QA: settings unavailable"}),{status:503,headers:{"Content-Type":"application/json"}})):window.qaFetch(url,opts)');
    navigate(globalHash); browser("wait", "#pi-settings-view [role=alert]");
    await capture(app + "-read-error");
    evaluate('window.fetch=window.qaFetch');
    browser("click", "#pi-settings-view [role=alert] button"); ready();
    results.push(app + ": initial failure retries");

    // Stay inside Agent CLIs: unmounting it with the invention down would
    // (correctly) replace the whole detail with its outage notice, and this
    // row is about the settings editor, not the shell's error reporting.
    evaluate('window.qaFetch=window.fetch;window.fetch=(url,opts)=>/^\\/api\\/(clis|terminals|cli-jobs)(?:[/?]|$)/.test(String(url))?Promise.resolve(new Response(JSON.stringify({error:"QA: terminal manager unavailable"}),{status:503,headers:{"Content-Type":"application/json"}})):window.qaFetch(url,opts)');
    navigate("#/clis/pi");
    browser("wait", "--fn", '!document.querySelector("#pi-settings-view")');
    navigate(globalHash); ready();
    // The settings editor is the acceptance: it renders and stays editable
    // while the terminal/CLI inventory is down. The Agent CLIs shell may
    // report that outage above it (an honest notice, not a settings failure).
    assert.equal(evaluate('document.querySelectorAll("#pi-settings-view [role=alert]").length'), 0);
    assert.equal(evaluate('!!document.querySelector("#g-compactionEnabled")'), true);
    assert.equal(evaluate('document.querySelector(".pi-settings-fields").disabled'), false);
    evaluate('window.fetch=window.qaFetch');
    results.push(app + ": independent from terminal manager requests");

    // The Palette shortcut opens the layer that holds the row it names.
    navigate(contextHash + "&focus=scoped-models");
    ready();
    assert.equal(layerOf(), "global");
    browser("wait", "--fn", '(()=>{const r=document.querySelector("#scoped-models")?.getBoundingClientRect();return r&&r.top>=0&&r.bottom<=innerHeight})()');
    results.push(app + ": scoped models revealed after load, on the layer that owns it");
    if (app === "mobile") {
      browser("find", "role", "button", "click", "--name", "Back");
      browser("wait", "--fn", 'location.hash==="#/clis"');
      results.push("mobile: Back returns to Agent CLIs");
    }
  }
  // A fresh owned folder proves the blocked branch even on repeat runs.
  const untrustedPath = resolve(workspace.path, "..", "settings-untrusted-" + process.pid);
  mkdirSync(untrustedPath);
  const untrustedWorkspace = await api("/api/workspaces", "POST", { name: "Settings untrusted", path: untrustedPath });
  const untrustedAgent = await api("/api/workspaces/" + untrustedWorkspace.id + "/agents", "POST", { name: "Settings scope" });
  browser("open", new URL("/mobile/?theme=dark" + settingsHash + "?agentId=" + encodeURIComponent(untrustedAgent.id) + "&scope=workspace", base).href);
  // An untrusted folder's layer is a notice with the trust link, not a form.
  browser("wait", "--fn", '!!document.querySelector(\'[data-layer="project"]\')');
  assert.equal(evaluate('document.querySelectorAll("#w-steeringMode").length'), 0);
  // A plain anchor: the shell's own notices can sit over it on a phone, so
  // navigate it directly and assert the route the app lands on.
  browser("wait", "--fn", '!!document.querySelector(\'[data-layer="project"] a\')');
  evaluate('document.querySelector(\'[data-layer="project"] a\').click(); true');
  browser("wait", "--fn", `location.hash===${JSON.stringify("#/agent/" + untrustedAgent.id)}`);
  browser("back");
  browser("wait", "--fn", '!!document.querySelector(\'[data-layer="project"]\')');
  evaluate('document.querySelector("[data-layer=project]").scrollIntoView({block:"center"})');
  await capture("mobile-untrusted-workspace");
  const rejected = await fetch(new URL("/api/pi-settings", base), { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ agentId: untrustedAgent.id, layer: "project", patch: { steeringMode: "all" } }) });
  assert.equal(rejected.status, 409);
  results.push("untrusted workspace: Trust action targets its agent; server refuses project write");
  // An unknown reset name is refused instead of quietly leaving the override.
  const badReset = await fetch(new URL("/api/pi-settings", base), { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ layer: "global", patch: { reset: ["model"] } }) });
  assert.equal(badReset.status, 400);
  results.push("unknown reset key refused");
  // Trust acceptance uses the real endpoint on this disposable workspace.
  await api("/api/agents/" + encodeURIComponent(id) + "/trust", "POST");
  browser("open", new URL("/mobile/?theme=light" + contextHash + "&scope=workspace", base).href); ready();
  browser("wait", "#w-steeringMode");
  browser("select", "#w-steeringMode", "all"); ready();
  assert.equal((await api("/api/pi-settings?agentId=" + encodeURIComponent(id))).project.steeringMode, "all");
  await capture("mobile-trusted-workspace");
  results.push("trusted workspace: project round-trip");
  const free = await api("/api/agents", "POST", { name: "Settings QA free" });
  assert.ok(free.id);
  browser("open", new URL("/mobile/" + settingsHash + "?agentId=" + encodeURIComponent(free.id), base).href); ready();
  assert.equal(evaluate('document.querySelectorAll("[data-layer=project]").length'), 0);
  assert.equal(evaluate('document.querySelectorAll("[data-layer=agent]").length'), 1);
  assert.equal(evaluate('document.querySelectorAll("#pi-settings-view .settings-layer .pkg-scope-btn").length'), 2);
  results.push("free agent: agent layer without workspace");
  await api("/api/agents/" + encodeURIComponent(free.id), "DELETE");
  browser("wait", "[role=alert]");
  assert.equal(evaluate('document.querySelectorAll("#ag-set-thinking").length'), 0);
  results.push("deleted open agent: feed removes editable fields");
  browser("set", "viewport", "560", "900");
  browser("open", new URL("/browser/?theme=light" + globalHash, base).href); ready();
  await capture("desktop-narrow");
  console.log(JSON.stringify({ ok: true, results }, null, 2));
  writeFileSync(resolve(out, "results.json"), JSON.stringify({ ok: true, results }, null, 2));
} catch (error) {
  // A row that fails must not hide the rows that passed: this script has no
  // other record of which screens it exercised before the throw.
  try {
    writeFileSync(resolve(out, "results.json"), JSON.stringify({ ok: false, error: String(error), results }, null, 2));
    writeFileSync(resolve(out, "failure.json"), JSON.stringify({ error: String(error), snapshot: browser("snapshot", "-i") }, null, 2));
    browser("screenshot", resolve(out, "failure.png"));
  } catch {}
  throw error;
} finally { browser("close"); }
