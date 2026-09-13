import { test } from "node:test";
import assert from "node:assert/strict";
import { sessionClis, handoffModes, handoffTargets, handoffSummaryLine, handoffSessionLabel, handoffRequest, sessionFromTerminal, terminalHandoffSourceCli, terminalHandoffMenu, lineageBadges } from "./sessionHandoff.js";

const all = { list: true, read: true, write: true, prompt: true };
const clis = [
  { id: "pi", name: "Pi", installed: true, sessions: all },
  { id: "claude-code", name: "Claude Code", installed: true, sessions: all },
  { id: "codex", name: "Codex", installed: false, sessions: all },
  { id: "grok", name: "Grok", installed: true, sessions: { list: true, read: true, write: false, prompt: true } },
  { id: "hermes", name: "Hermes Agent", installed: true, sessions: { list: true, read: true, write: false, prompt: false } },
  { id: "future", name: "Future CLI", installed: true, sessions: { list: true, read: false, write: false, prompt: true } },
  { id: "plain", name: "No sessions", installed: true },
];

test("sessionClis follows the advertised list, not a hardcoded one", () => {
  assert.deepEqual(sessionClis(clis), ["pi", "claude-code", "codex", "grok", "hermes", "future"]);
  assert.deepEqual(sessionClis([]), []);
  assert.deepEqual(sessionClis(undefined), []);
});

test("handoffModes is the decision table", () => {
  assert.deepEqual(handoffModes(all, all), ["native", "brief"]);
  assert.deepEqual(handoffModes(all, { write: false, prompt: true }), ["brief"]);
  assert.deepEqual(handoffModes(all, { write: true, prompt: false }), ["native"]);
  assert.deepEqual(handoffModes(all, { write: false, prompt: false }), []);
  assert.deepEqual(handoffModes({ read: false }, all), []);
  assert.deepEqual(handoffModes(undefined, all), []);
});

test("handoffTargets excludes the source, unreadable sources and dead-end targets", () => {
  const targets = handoffTargets(clis, "claude-code");
  assert.deepEqual(targets.map((t) => t.id), ["pi", "codex", "grok", "future"]);
  assert.deepEqual(targets.find((t) => t.id === "grok").modes, ["brief"]);
  assert.equal(targets.find((t) => t.id === "codex").installed, false);
  assert.deepEqual(handoffTargets(clis, "future"), []); // cannot be read
  assert.deepEqual(handoffTargets(clis, "nope"), []);
});

test("summary line omits zero counts and names only what a reader understands", () => {
  // claude.attachment and codex.event_msg are a CLI's own record types:
  // counted, never spelled out, because "bridge session left out (4)" told
  // the reader nothing when it shipped.
  assert.equal(handoffSummaryLine({ messages: 42, toolCalls: 17 }, { dropped: { thinking: 12, "claude.attachment": 3, image: 1, "codex.event_msg": 9 } }), "42 messages · 17 tool calls · thinking left out (12) · images left out (1) · 12 other items skipped");
  assert.equal(handoffSummaryLine({ messages: 1, toolCalls: 0 }, { dropped: {} }), "1 message");
  assert.equal(handoffSummaryLine({ messages: 2 }, { dropped: { "claude.bridge-session": 4 } }), "2 messages · 4 other items skipped");
  assert.equal(handoffSummaryLine({}, null), "");
});

test("the dialog names a session by its title, or a short id", () => {
  assert.equal(handoffSessionLabel({ name: "Race fix", id: "219fb973-d3b2-445d-b679-821b1a4958d7" }), "Race fix");
  assert.equal(handoffSessionLabel({ id: "219fb973-d3b2-445d-b679-821b1a4958d7" }), "219fb973");
  assert.equal(handoffSessionLabel({ id: "cc-1" }), "cc-1");
  assert.equal(handoffSessionLabel(null), "");
});

test("handoffRequest is the exact body the server accepts", () => {
  const body = handoffRequest({ id: "cc-1", path: "/p", cwd: "/w", workspaceId: "ws" }, { to: "codex", mode: "native", window: "all", tools: "text" }, { force: true });
  assert.deepEqual(Object.keys(body).sort(), ["cwd", "force", "id", "mode", "path", "to", "tools", "window", "workspaceId"]);
  assert.equal(body.force, true);
  assert.equal(handoffRequest({ id: "x" }, { to: "pi" }).window, "recent");
  assert.equal(handoffRequest({ id: "x" }, { to: "pi" }).mode, "");
});

test("sessionFromTerminal is the pin, with cwd/name falling back to the terminal", () => {
  assert.equal(sessionFromTerminal(null), null);
  assert.equal(sessionFromTerminal({}), null);
  assert.equal(sessionFromTerminal({ lastSession: { cli: "pi" } }), null);
  assert.deepEqual(
    sessionFromTerminal({
      cwd: "/term",
      name: "communication",
      workspaceId: "ws1",
      lastSession: { cli: "pi", sessionId: "s1", path: "/p", cwd: "/sess", name: "Race" },
    }),
    { id: "s1", path: "/p", cwd: "/sess", name: "Race", workspaceId: "ws1" },
  );
  assert.equal(sessionFromTerminal({ cwd: "/term", lastSession: { sessionId: "s1" } }).cwd, "/term");
  assert.equal(terminalHandoffSourceCli({ lastSession: { cli: "claude-code", sessionId: "s1" } }), "claude-code");
  assert.equal(terminalHandoffSourceCli({ launchCli: "pi" }), "");
});

test("terminalHandoffMenu drops the row when it cannot act", () => {
  const pinned = { lastSession: { cli: "claude-code", sessionId: "s1" } };
  assert.equal(terminalHandoffMenu({}, clis), null);
  assert.equal(terminalHandoffMenu({ lastSession: { cli: "future", sessionId: "s1" } }, clis), null); // unread source
  assert.equal(terminalHandoffMenu(pinned, []), null);
  const row = terminalHandoffMenu(pinned, clis);
  assert.equal(row.id, "handoff");
  assert.deepEqual(row.sub.map((s) => s.id), ["handoff:pi", "handoff:codex", "handoff:grok", "handoff:future"]);
  const codex = row.sub.find((s) => s.target.id === "codex");
  assert.equal(codex.disabled, true);
  assert.equal(codex.label, "Codex (not installed)");
  assert.match(codex.title, /not installed\.$/);
  assert.equal(row.sub.find((s) => s.target.id === "pi").disabled, false);
});

test("lineage badges", () => {
  const names = { codex: "Codex", "claude-code": "Claude Code" };
  assert.deepEqual(lineageBadges(null, names), []);
  const badges = lineageBadges({ from: { cli: "claude-code", id: "cc-1" }, to: [{ cli: "codex", id: "cx-1" }, { cli: "unknown" }] }, names);
  assert.deepEqual(badges.map((b) => b.label), ["from Claude Code", "continued in Codex", "continued in unknown"]);
});
