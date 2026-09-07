import { test } from "node:test";
import assert from "node:assert/strict";
import { sessionClis, handoffModes, handoffTargets, handoffSummaryLine, handoffRequest, lineageBadges } from "./sessionHandoff.js";

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

test("summary line omits zero counts and names the biggest omissions", () => {
  assert.equal(handoffSummaryLine({ messages: 42, toolCalls: 17 }, { dropped: { thinking: 12, "claude.attachment": 3, image: 1, "codex.event_msg": 9 } }), "42 messages · 17 tool calls · thinking left out (12) · event msg left out (9) · 4 other items skipped");
  assert.equal(handoffSummaryLine({ messages: 1, toolCalls: 0 }, { dropped: {} }), "1 message");
  assert.equal(handoffSummaryLine({}, null), "");
});

test("handoffRequest is the exact body the server accepts", () => {
  const body = handoffRequest({ id: "cc-1", path: "/p", cwd: "/w", workspaceId: "ws" }, { to: "codex", mode: "native", window: "all", tools: "text" }, { force: true });
  assert.deepEqual(Object.keys(body).sort(), ["cwd", "force", "id", "mode", "path", "to", "tools", "window", "workspaceId"]);
  assert.equal(body.force, true);
  assert.equal(handoffRequest({ id: "x" }, { to: "pi" }).window, "recent");
  assert.equal(handoffRequest({ id: "x" }, { to: "pi" }).mode, "");
});

test("lineage badges", () => {
  const names = { codex: "Codex", "claude-code": "Claude Code" };
  assert.deepEqual(lineageBadges(null, names), []);
  const badges = lineageBadges({ from: { cli: "claude-code", id: "cc-1" }, to: [{ cli: "codex", id: "cx-1" }, { cli: "unknown" }] }, names);
  assert.deepEqual(badges.map((b) => b.label), ["from Claude Code", "continued in Codex", "continued in unknown"]);
});
