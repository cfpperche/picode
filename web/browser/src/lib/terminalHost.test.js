import test from "node:test";
import assert from "node:assert/strict";
import { agentPaneKeys, canvasTerminalHost, canvasTerminalTools, closingPaneKeys, fleetAgents, terminalHost } from "./terminalHost.js";
import { paneCapabilities, buildTermMenu } from "./termMenu.js";

const agent = { id: "a", terminalId: "t", mode: "interactive", name: "Agent", cli: "pi" };
const term = { id: "t", name: "Terminal", session: "picode-sh-t", launchCli: "pi", running: true };
const fleet = { freeAgents: [agent], terminals: [term] };

test("canonical pane keeps agent ownership and the containing tab", () => {
  for (const [tabs, tabId] of [[["a"], "a"], [["a", "t:t"], "a"], [["a", "t:t"], "t:t"]]) {
    const mapping = terminalHost({ kind: "term", id: "t", tabId }, [agent], tabs);
    const ctx = paneCapabilities({ ...mapping, host: "term", termRecord: term, tabs });
    assert.equal(ctx.id, "t");
    assert.equal(ctx.ownerId, "a");
    assert.equal(ctx.tabId, tabId);
    assert.equal(ctx.promptDoor, "cli");
    assert.equal(ctx.lifecycle, "agent");
    assert.equal(ctx.tabOpen, true);
    const rows = buildTermMenu(ctx);
    assert.ok(rows.some((r) => r.id === "close-tab"));
    assert.equal(rows.find((r) => r.id === "remove").label, "Remove agent…");
  }
});

test("canvas menu chooses an open owner tab; standalone terminals retain terminal actions", () => {
  assert.equal(terminalHost({ kind: "term", id: "t" }, [agent], ["a"]).tabId, "a");
  assert.equal(terminalHost({ kind: "agent", id: "a" }, [agent], []).tabId, "a");
  const mapping = terminalHost({ kind: "term", id: "t" }, [], ["t:t"]);
  assert.deepEqual(mapping, { agent: null, tabId: "t:t", ownerKind: "term", ownerId: "t" });
});

test("canvas tools follow the runtime rather than the panel's agent reference", () => {
  const seed = { id: "t", token: "seed", text: "Question" };
  for (const kind of ["agent", "terminal"]) {
    const model = { kind, ref: kind === "agent" ? "a" : "t", runtimeId: "t" };
    assert.deepEqual(canvasTerminalTools(model, { termAttach: seed, termFind: "t" }), { attach: seed, find: true });
    assert.deepEqual(canvasTerminalTools(model, { termAttach: { id: "other" }, termFind: "other" }), { attach: null, find: false });
    assert.deepEqual(canvasTerminalTools(model, {}), { attach: null, find: false });
  }
  assert.deepEqual(canvasTerminalTools(null, {}), { attach: null, find: false });
});

test("canvas and tabs address the same runtime, including either tab alias", () => {
  for (const kind of ["agent", "terminal"]) {
    for (const tabs of [["a"], ["t:t"], ["a", "t:t"], []]) {
      const resolved = canvasTerminalHost(kind, kind === "agent" ? agent : term, "/tmp", fleet, tabs);
      assert.equal(resolved.id, "t");
      assert.equal(resolved.owned, tabs.length > 0);
    }
  }
  assert.equal(canvasTerminalHost("agent", { id: "old", mode: "interactive" }, "/tmp", {}, ["old"]).id, "old");
  for (const kind of ["agent", "terminal"]) {
    assert.equal(canvasTerminalHost(kind, kind === "agent" ? agent : term, "/tmp", { ...fleet, termEpochs: { a: 2 } }, []).epoch, 2);
  }
});

test("cleanup covers both legacy and bound aliases without duplicate disposal", () => {
  assert.deepEqual(agentPaneKeys({ ...agent, legacyInteractive: true }), ["a", "t"]);
  assert.deepEqual(agentPaneKeys({ id: "a", terminalId: "a" }), ["a"]);
  assert.deepEqual(agentPaneKeys(null), []);
});

test("closing either tab preserves the runtime owned by its other alias", () => {
  assert.deepEqual(closingPaneKeys("t:t", [agent], ["a", "t:t"]), []);
  assert.deepEqual(closingPaneKeys("a", [agent], ["a", "t:t"]), ["a"]);
  assert.deepEqual(closingPaneKeys("a", [agent], ["a"]), ["a", "t"]);
  assert.deepEqual(closingPaneKeys("t:t", [agent], ["t:t"]), ["t"]);
  assert.deepEqual(closingPaneKeys("t:shell", [], ["t:shell"]), ["shell"]);
  assert.deepEqual(closingPaneKeys("file:x", [], ["file:x"]), []);
});

test("workspace ownership supports both current and compatibility fleet shapes", () => {
  assert.deepEqual(fleetAgents({ workspaces: [{ agent }, { agents: [{ id: "b" }] }] }).map((a) => a.id), ["a", "b"]);
});

test("terminal menu never promises xterm scroll-to-end for application-owned history", () => {
  for (const localScrollback of [true, false]) {
    const ctx = paneCapabilities({ host: "term", termRecord: term, localScrollback });
    assert.equal(buildTermMenu(ctx).some(row => row.id === "scroll-end"), localScrollback);
  }
});
