import { test } from "node:test";
import assert from "node:assert/strict";
import { agentRowMenu } from "./agentRowMenu.js";
import { agentSubtitle, principalLabel } from "./managedPrincipal.js";

const ids = (rows) => rows.filter((r) => !r.sep).map((r) => r.id);
const row = (rows, id) => rows.find((r) => r.id === id);
const RUNNING = { id: "t1", running: true, state: "idle", cli: "claude-code" };
const STOPPED = { id: "t1", running: false };
const PINNED = { id: "t1", running: false, lastSession: { cli: "claude-code", sessionId: "s1", path: "/p", cwd: "/w" } };
const CATALOG = [
  { id: "pi", name: "Pi", installed: true, sessions: { list: true, read: true, write: true, prompt: true, agent: true } },
  { id: "claude-code", name: "Claude Code", installed: true, sessions: { list: true, read: true, write: true, prompt: true } },
  { id: "codex", name: "Codex", installed: true, sessions: { list: true, read: true, write: true, prompt: true } },
];

// Decision table (ADR-0160): the menu is a function of kind × terminal
// state × adapter.
// | CLI agent | bound terminal  | integrationCapable | lifecycle + launch        |
// | ----- | --------------- | ------------------ | ------------------------- |
// | Pi stopped | —         | —                  | Start, settings, chat     |
// | Pi managed/interactive | — | —               | Restart/Stop, settings, chat |
// | Pi any mode | present  | —                  | launch + settings         |
// | yes   | stopped/missing | true               | Start agent + launch      |
// | yes   | running         | true               | Restart/Stop + launch     |
// | yes   | any             | false              | lifecycle, no launch      |

test("Pi without a bound terminal offers scoped settings, not launch settings", () => {
  assert.deepEqual(ids(agentRowMenu({ cli: "pi", mode: "stopped" })), ["start", "settings", "chat", "term", "rename", "remove"]);
  for (const mode of ["managed", "interactive"]) {
    const rows = agentRowMenu({ id: "pi/qa", cli: "pi", mode });
    assert.deepEqual(ids(rows), ["restart", "stop", "settings", "chat", "term", "rename", "remove"]);
    assert.equal(row(rows, "restart").label, "Restart agent");
    assert.equal(row(rows, "settings").href, "#/clis/pi/settings?agentId=pi%2Fqa");
  }
  assert.equal(row(agentRowMenu({ cli: "pi", mode: "managed" }), "stop").label, "Stop agent");
});

test("Pi bound terminals use the same launch route in every mode, with no duplicate", () => {
  for (const mode of ["stopped", "managed", "interactive"]) {
    const rows = agentRowMenu({ id: "pi/qa", cli: "pi", mode, terminalId: "term/qa" });
    assert.equal(row(rows, "launch").href, "#/clis/terminal/term%2Fqa");
    assert.equal(row(rows, "settings").href, "#/clis/pi/settings?agentId=pi%2Fqa");
    assert.equal(row(rows, "terminal-settings"), undefined);
    assert.equal(rows.filter((r) => r.label === "Launch settings").length, 1);
  }
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

// ADR-0088 on an agent row: a conversation pinned on the bound terminal
// continues in another CLI — the same submenu the terminal row offers.
test("a Pi agent with a pinned session offers Continue in… toward other CLIs", () => {
  const rows = agentRowMenu({ cli: "pi", sessionPath: "/p/s.jsonl", mode: "interactive" }, { clis: CATALOG });
  const handoff = row(rows, "handoff");
  if (!handoff || !Array.isArray(handoff.sub)) assert.fail("no handoff for pinned pi");
  assert.ok(!handoff.sub.map((s) => s.target.id).includes("pi"), "pi is the source, not a target");
  assert.equal(row(rows, "chat").label, "Open chat");
});

test("a Pi agent without a pinned session stays flat", () => {
  const rows = agentRowMenu({ cli: "pi", mode: "interactive" }, { clis: CATALOG });
  assert.equal(rows.some((r) => r.id === "handoff"), false);
  assert.equal(rows.some((r) => r.sep), false);
});

test("a CLI agent with a pinned conversation offers Continue in…", () => {
  const rows = agentRowMenu({ cli: "claude-code", terminalId: "t1" }, { clis: CATALOG, term: PINNED });
  const handoff = row(rows, "handoff");
  if (!handoff || !Array.isArray(handoff.sub) || !handoff.sub.length) {
    assert.fail("no handoff submenu: " + JSON.stringify(rows));
  }
  assert.equal(handoff.label, "Continue in…");
  const targets = handoff.sub.map((s) => s.target.id);
  assert.ok(targets.includes("pi"), "pi is a handoff target");
  assert.ok(!targets.includes("claude-code"), "the source is not its own target");
  // The submenu rides before the lifecycle, separated by a divider.
  assert.ok(rows.findIndex((r) => r.sep) > rows.findIndex((r) => r.id === "handoff"), "divider after the submenu");
});

test("no pin, no Continue in… — the row stays flat", () => {
  const rows = agentRowMenu({ cli: "claude-code", terminalId: "t1" }, { clis: CATALOG, term: STOPPED });
  assert.equal(rows.some((r) => r.id === "handoff"), false);
  assert.equal(rows.some((r) => r.sep), false);
});

// Fork agent… | pin | CLI forks natively (sessions.fork) | row
//             | no  | —                                  | absent
//             | yes | no                                 | absent
//             | yes | yes                                | first, before Continue in…
test("Fork agent… follows the pin and the CLI's native fork", () => {
  const forking = CATALOG.map((c) => (c.id === "claude-code" ? { ...c, sessions: { ...c.sessions, fork: true } } : c));
  const rows = agentRowMenu({ cli: "claude-code", terminalId: "t1" }, { clis: forking, term: PINNED });
  const order = ids(rows);
  assert.equal(order[0], "fork");
  assert.equal(order[1], "handoff");
  assert.equal(row(rows, "fork").label, "Fork agent…");
  assert.ok(rows.findIndex((r) => r.sep) > order.indexOf("handoff"), "one divider after both");
  assert.equal(ids(agentRowMenu({ cli: "claude-code", terminalId: "t1" }, { clis: CATALOG, term: PINNED })).includes("fork"), false, "no native fork, no row");
  assert.equal(ids(agentRowMenu({ cli: "claude-code", terminalId: "t1" }, { clis: forking, term: STOPPED })).includes("fork"), false, "no pin, no row");
});
