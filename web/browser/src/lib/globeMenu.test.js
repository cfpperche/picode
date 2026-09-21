import { test } from "node:test";
import assert from "node:assert/strict";
import { globeBindId, globeClickAction, buildGlobeMenu, GLOBE_MENU_KEYS } from "./globeMenu.js";
import { fileTabId, gitTabId, treeTabId, appTabId, webTabId, termTabId } from "./routes.js";

const ids = (rows) => rows.filter((r) => !r.sep).map((r) => r.id);
const row = (rows, id) => rows.find((r) => r.id === id);

test("globeBindId: agent tab binds, everything else does not unless it is a CLI terminal", () => {
  assert.equal(globeBindId("ag1", null), "ag1");
  assert.equal(globeBindId(termTabId("desk-1"), { launchCli: "pi" }), "t:desk-1");
  assert.equal(globeBindId(termTabId("desk-1"), { tui: { cli: "omp" } }), "t:desk-1");
  assert.equal(globeBindId(termTabId("desk-1"), { tui: { cli: "claude" } }), "t:desk-1");
  assert.equal(globeBindId(termTabId("desk-1"), {}), "");
  assert.equal(globeBindId(termTabId("desk-1"), { tui: { cli: "" } }), "");
  assert.equal(globeBindId(webTabId("3"), null), "");
  assert.equal(globeBindId(fileTabId("w", "w1", "src/App.jsx"), null), "");
  assert.equal(globeBindId(gitTabId("/home/goat/picode/.git"), null), "");
  assert.equal(globeBindId(treeTabId("/home/goat/picode"), null), "");
  assert.equal(globeBindId(appTabId("canvas"), null), "");
  assert.equal(globeBindId("", null), "");
  assert.equal(globeBindId(null, null), "");
});

// Each row is the click table: shift × bind × split → action.
const CLICK = [
  { shiftKey: false, bindId: "",     splitOn: false, action: "new-tab" },
  { shiftKey: false, bindId: "ag1",  splitOn: false, action: "split" },
  { shiftKey: false, bindId: "ag1",  splitOn: true,  action: "noop" },
  { shiftKey: false, bindId: "t:1",  splitOn: false, action: "split" },
  { shiftKey: false, bindId: "t:1",  splitOn: true,  action: "noop" },
  { shiftKey: true,  bindId: "",     splitOn: false, action: "new-tab" },
  { shiftKey: true,  bindId: "ag1",  splitOn: false, action: "new-tab" },
  { shiftKey: true,  bindId: "ag1",  splitOn: true,  action: "new-tab" },
];

test("globeClickAction covers every shift × bind × split row", () => {
  for (const row of CLICK) {
    assert.equal(globeClickAction(row), row.action, JSON.stringify(row));
  }
});

test("globe menu: Open in new tab is always present", () => {
  const empty = buildGlobeMenu({});
  assert.deepEqual(ids(empty), ["new-tab"]);
  assert.equal(row(empty, "new-tab").label, "Open in new tab");
  assert.equal(row(empty, "new-tab").key, GLOBE_MENU_KEYS["new-tab"]);
  assert.equal(row(empty, "new-tab").icon, "plus");
});

test("globe menu: agent tab offers Open browser or Close browser split, never both", () => {
  const open = buildGlobeMenu({ tabId: "ag1", splitOn: false });
  assert.deepEqual(ids(open), ["open-browser", "new-tab"]);
  assert.equal(row(open, "open-browser").label, "Open browser");
  assert.equal(row(open, "close-browser"), undefined);

  const close = buildGlobeMenu({ tabId: "ag1", splitOn: true });
  assert.deepEqual(ids(close), ["close-browser", "new-tab"]);
  assert.equal(row(close, "close-browser").label, "Close browser split");
  assert.equal(row(close, "open-browser"), undefined);
});
