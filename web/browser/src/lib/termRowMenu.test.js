import { test } from "node:test";
import assert from "node:assert/strict";
import { termRowMenu } from "./termRowMenu.js";

const ids = (rows) => rows.filter((r) => !r.sep).map((r) => r.id);
const row = (rows, id) => rows.find((r) => r.id === id);

// Decision table: the menu is a function of the terminal's running state.
// | t.running | lifecycle rows                |
// | --------- | ----------------------------- |
// | truthy    | Restart terminal, Stop terminal |
// | falsy     | Start terminal                  |
// Every other row is identical in both states, and both surfaces render
// from this one module.

test("a running terminal offers restart and stop, not start", () => {
  const rows = termRowMenu({ id: "t1", running: true });
  assert.deepEqual(ids(rows), ["rename", "launch", "settings", "restart", "stop", "remove"]);
});

test("a stopped terminal offers start, not restart or stop", () => {
  const rows = termRowMenu({ id: "t1", running: false });
  assert.deepEqual(ids(rows), ["rename", "launch", "settings", "start", "remove"]);
});

test("an unknown terminal degrades to the stopped shape", () => {
  assert.deepEqual(ids(termRowMenu()), ids(termRowMenu({ running: false })));
  assert.deepEqual(ids(termRowMenu({})), ids(termRowMenu({ running: false })));
});

test("a CLI without an adapter drops Launch settings", () => {
  // Muse Code and Antigravity open a terminal with no launch settings to
  // edit; the item would otherwise open an empty form.
  const catalog = [{ id: "muse", integrationCapable: false }, { id: "pi", integrationCapable: true }];
  assert.deepEqual(ids(termRowMenu({ id: "t1", running: true, launchCli: "muse" }, { clis: catalog })), ["rename", "settings", "restart", "stop", "remove"]);
  assert.deepEqual(ids(termRowMenu({ id: "t2", running: false, launchCli: "pi" }, { clis: catalog })), ["rename", "launch", "settings", "start", "remove"]);
});

test("remove is the only dangerous row, everywhere", () => {
  for (const t of [{ running: true }, { running: false }]) {
    const rows = termRowMenu(t);
    assert.equal(rows.filter((r) => !r.sep && r.danger).length, 1);
    assert.equal(row(rows, "remove").danger, true);
  }
});

test("the two surfaces render the same options in the same order", () => {
  // The order is the contract: identity rows, then lifecycle, then the
  // dangerous one last. Asserting the full sequence here pins the sidebar
  // and the Agent CLIs list to one menu.
  assert.deepEqual(termRowMenu({ running: true }).map((r) => (r.sep ? "sep" : r.id)),
    ["rename", "launch", "settings", "sep", "restart", "stop", "sep", "remove"]);
  assert.deepEqual(termRowMenu({ running: false }).map((r) => (r.sep ? "sep" : r.id)),
    ["rename", "launch", "settings", "sep", "start", "sep", "remove"]);
});

test("every row answers with a label and a one-line title", () => {
  for (const t of [{ running: true }, { running: false }]) {
    for (const r of termRowMenu(t)) {
      if (r.sep) continue;
      assert.ok(r.label, JSON.stringify(r));
      assert.match(r.title, /\.$/, JSON.stringify(r));
    }
  }
});

const all = { list: true, read: true, write: true, prompt: true };
const clis = [
  { id: "pi", name: "Pi", installed: true, sessions: all },
  { id: "codex", name: "Codex", installed: true, sessions: all },
  { id: "claude-code", name: "Claude Code", installed: false, sessions: all },
];
const pinned = { running: true, lastSession: { cli: "pi", sessionId: "s1", path: "/p", cwd: "/w" } };

test("Continue in… is dropped when the row cannot act", () => {
  assert.equal(row(termRowMenu({ running: true }, { clis }), "handoff"), undefined);
  assert.equal(row(termRowMenu({ running: true, lastSession: { cli: "pi" } }, { clis }), "handoff"), undefined);
  assert.equal(row(termRowMenu(pinned, { clis: [] }), "handoff"), undefined);
  assert.equal(row(termRowMenu({ running: false }, { clis }), "handoff"), undefined);
});

test("a pinned conversation offers Continue in… before lifecycle, running or stopped", () => {
  const running = termRowMenu(pinned, { clis });
  assert.deepEqual(running.map((r) => (r.sep ? "sep" : r.id)),
    ["rename", "launch", "settings", "sep", "handoff", "sep", "restart", "stop", "sep", "remove"]);
  const stopped = termRowMenu({ ...pinned, running: false }, { clis });
  assert.deepEqual(stopped.map((r) => (r.sep ? "sep" : r.id)),
    ["rename", "launch", "settings", "sep", "handoff", "sep", "start", "sep", "remove"]);
  const sub = row(running, "handoff").sub;
  assert.deepEqual(sub.map((s) => s.target.id), ["codex", "claude-code"]);
  assert.equal(sub.find((s) => s.target.id === "claude-code").label, "Claude Code (not installed)");
  assert.equal(sub.find((s) => s.target.id === "claude-code").disabled, true);
  assert.equal(sub.find((s) => s.target.id === "codex").disabled, false);
  for (const r of [...running, ...sub]) {
    if (r.sep) continue;
    assert.match(r.title, /\.$/, JSON.stringify(r));
  }
});
