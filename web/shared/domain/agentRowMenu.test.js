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
// | Pi stopped | —         | —                  | Start, launch, chat       |
// | Pi managed/interactive | — | —               | Restart/Stop, launch, chat |
// | yes   | stopped/missing | true               | Start agent + launch      |
// | yes   | running         | true               | Restart/Stop + launch     |
// | yes   | any             | false              | lifecycle, no launch      |

test("Pi offers lifecycle actions in both modes and scoped settings", () => {
  assert.deepEqual(ids(agentRowMenu({ cli: "pi", mode: "stopped" })), ["start", "launch", "chat", "term", "rename", "remove"]);
  for (const mode of ["managed", "interactive"]) {
    const rows = agentRowMenu({ id: "pi/qa", cli: "pi", mode });
    assert.deepEqual(ids(rows), ["restart", "stop", "launch", "chat", "term", "rename", "remove"]);
    assert.equal(row(rows, "restart").label, "Restart agent");
    assert.equal(row(rows, "launch").href, "#/clis/pi/settings?agentId=pi%2Fqa");
  }
  assert.equal(row(agentRowMenu({ cli: "pi", mode: "managed" }), "stop").label, "Stop agent");
});

test("a stopped CLI agent offers Start agent and Launch settings", () => {
  const rows = agentRowMenu({ cli: "claude-code", terminalId: "t1" }, { clis: [{ id: "claude-code" }], term: STOPPED });
  assert.deepEqual(ids(rows), ["start", "launch", "term", "rename", "remove"]);
  assert.equal(row(rows, "start").label, "Start agent");
  assert.equal(row(rows, "launch").label, "Launch settings");
  assert.equal(row(rows, "launch").href, "#/clis/terminal/t1");
});

test("a running CLI agent swaps the lifecycle for restart and stop", () => {
  const rows = agentRowMenu({ cli: "claude-code", terminalId: "t1" }, { clis: [], term: RUNNING });
  assert.deepEqual(ids(rows), ["restart", "stop", "launch", "term", "rename", "remove"]);
  assert.equal(row(rows, "stop").label, "Stop agent");
  assert.equal(row(rows, "restart").label, "Restart agent");
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

test("all CLI agents share lifecycle wording and ordering", () => {
  for (const cli of ["codex", "claude-code", "grok", "hermes", "opencode", "omp", "muse", "agy"]) {
    for (const term of [RUNNING, STOPPED]) {
      const rows = agentRowMenu({ cli, terminalId: "t1" }, { term });
      assert.deepEqual(rows.filter((r) => ["start", "restart", "stop"].includes(r.id)).map((r) => r.label),
        term.running ? ["Restart agent", "Stop agent"] : ["Start agent"]);
      assert.equal(rows.at(-1).id, "remove");
    }
  }
});

test("an unbound CLI agent does not offer launch settings", () => {
  assert.equal(row(agentRowMenu({ cli: "codex" }), "launch"), undefined);
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
