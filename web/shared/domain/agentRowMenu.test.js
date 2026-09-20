import { test } from "node:test";
import assert from "node:assert/strict";
import { agentRowMenu } from "./agentRowMenu.js";
import { agentSubtitle, principalLabel } from "./managedPrincipal.js";

const ids = (rows) => rows.map((r) => r.id);
const row = (rows, id) => rows.find((r) => r.id === id);
const RUNNING = { id: "t1", running: true, state: "idle", cli: "claude-code" };
const STOPPED = { id: "t1", running: false };

// Decision table (ADR-0160): the menu is a function of kind × terminal
// state × adapter.
// | CLI agent | bound terminal  | integrationCapable | lifecycle + launch        |
// | ----- | --------------- | ------------------ | ------------------------- |
// | no    | —               | —                  | Start/Stop agent, chat    |
// | yes   | stopped/missing | true               | Start terminal + launch   |
// | yes   | running         | true               | Restart/Stop + launch     |
// | yes   | any             | false              | lifecycle, no launch      |

test("a Pi agent keeps the managed menu with Open chat", () => {
  assert.deepEqual(ids(agentRowMenu({ cli: "pi", mode: "stopped" })), ["start", "chat", "term", "rename", "remove"]);
  assert.deepEqual(ids(agentRowMenu({ cli: "pi", mode: "managed" })), ["stop", "chat", "term", "rename", "remove"]);
  assert.equal(row(agentRowMenu({ cli: "pi", mode: "managed" }), "stop").label, "Stop agent");
});

test("a stopped CLI agent offers Start terminal and Launch settings", () => {
  const rows = agentRowMenu({ cli: "claude-code", terminalId: "t1" }, { clis: [{ id: "claude-code" }], term: STOPPED });
  assert.deepEqual(ids(rows), ["start", "launch", "term", "rename", "remove"]);
  assert.equal(row(rows, "start").label, "Start terminal");
  assert.equal(row(rows, "launch").label, "Launch settings");
});

test("a running CLI agent swaps the lifecycle for restart and stop", () => {
  const rows = agentRowMenu({ cli: "claude-code", terminalId: "t1" }, { clis: [], term: RUNNING });
  assert.deepEqual(ids(rows), ["restart", "stop", "launch", "term", "rename", "remove"]);
  assert.equal(row(rows, "stop").label, "Stop terminal");
});

test("a CLI agent with no terminal in the list degrades to Start", () => {
  assert.deepEqual(ids(agentRowMenu({ cli: "grok", terminalId: "t9" }, { clis: [] })), ["start", "launch", "term", "rename", "remove"]);
});

test("a CLI agent on a CLI without an adapter drops Launch settings", () => {
  const catalog = [{ id: "muse", integrationCapable: false }];
  for (const term of [STOPPED, RUNNING]) {
    const rows = agentRowMenu({ cli: "muse", terminalId: "t1" }, { clis: catalog, term });
    assert.equal(rows.some((r) => r.id === "launch"), false);
  }
});

test("a CLI agent never offers Open chat; remove is the only danger row", () => {
  for (const term of [STOPPED, RUNNING, undefined]) {
    const rows = agentRowMenu({ cli: "claude-code", terminalId: "t1" }, { clis: [], term });
    assert.equal(rows.some((r) => r.id === "chat"), false);
    assert.equal(rows.filter((r) => r.danger).length, 1);
    assert.equal(row(rows, "remove").danger, true);
  }
});

test("the CLI agent subtitle names the CLI; Pi keeps the old wording", () => {
  assert.equal(agentSubtitle({ cli: "claude-code" }), "Claude Code");
  assert.equal(agentSubtitle({ cli: "grok" }), "Grok");
  assert.equal(agentSubtitle({ mode: "interactive" }), "Interactive session");
  assert.equal(agentSubtitle({}), "Pi agent");
});

test("peer owner labels: CLI agents name the CLI, Pi and terminals keep their wording", () => {
  assert.equal(principalLabel({ kind: "agent", cli: "claude-code" }), "Claude Code");
  assert.equal(principalLabel({ kind: "agent", cli: "pi" }), "Pi agent");
  assert.equal(principalLabel({ kind: "agent" }), "Pi agent");
  assert.equal(principalLabel({ kind: "terminal", cli: "claude-code" }), "Pi agent");
  assert.equal(principalLabel(null), "Pi agent");
});
