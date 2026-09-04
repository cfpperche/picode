import test from "node:test";
import assert from "node:assert/strict";
import { normalizeTerminalCli, terminalActivityStamp, terminalCli, terminalCliFaviconUrl, terminalCliFaviconUrls, terminalCliLabel, terminalCliMark, terminalStatus, terminalStatusLabel } from "./terminalCli.js";

test("terminal CLI aliases use one canonical identity", () => {
  assert.equal(normalizeTerminalCli("claude"), "claude-code");
  assert.equal(normalizeTerminalCli(" Claude-Code "), "claude-code");
  assert.equal(normalizeTerminalCli("codex"), "codex");
  assert.equal(normalizeTerminalCli("unknown"), "");
  assert.equal(terminalCliLabel("pi"), "Pi");
  assert.equal(terminalCliMark("codex"), "Cx");
});

test("supported runtimes use their official favicons, best first", () => {
  // Anthropic's own icon first — claude.ai's favicon hangs or 403s for
  // browser subresources, so it never rendered the Claude badge.
  assert.equal(terminalCliFaviconUrl("claude"), "https://www.anthropic.com/images/icons/favicon-32x32.png");
  assert.deepEqual(terminalCliFaviconUrls("claude"), [
    "https://www.anthropic.com/images/icons/favicon-32x32.png",
    "https://claude.ai/favicon.ico",
  ]);
  // openai.com challenges some browsers; the official CDN touch icon is the
  // second link so Codex still gets a real logo when the first is blocked.
  assert.deepEqual(terminalCliFaviconUrls("codex"), [
    "https://openai.com/favicon.ico",
    "https://cdn.oaistatic.com/assets/apple-touch-icon-mz9nytnj.webp",
  ]);
  assert.deepEqual(terminalCliFaviconUrls("grok"), ["https://grok.com/images/favicon.svg"]);
  assert.deepEqual(terminalCliFaviconUrls("pi"), ["https://pi.dev/favicon.svg"]);
  assert.deepEqual(terminalCliFaviconUrls("shell"), []);
});

test("authoritative tui presence wins over legacy projection", () => {
  const term = { running: true, cli: "claude", state: "working", tui: { cli: "pi", startedAt: "2026-09-04T10:00:00Z" } };
  assert.equal(terminalCli(term), "pi");
  assert.equal(terminalCli({ cli: "claude", tui: { cli: "unknown" } }), "");
  assert.equal(terminalStatus(term), "working");
  assert.equal(terminalStatusLabel(term), "Working");
  assert.equal(terminalActivityStamp({ tui: { cli: "pi", startedAt: "2026-09-04T10:00:00Z" } }), "2026-09-04T10:00:00Z");
});

test("stale activity cannot attach to a newer runtime", () => {
  const term = { running: true, state: "working", runId: "old", stateAt: "old", tui: { cli: "pi", runId: "new", startedAt: "new" } };
  assert.equal(terminalStatus(term), "open");
  assert.equal(terminalActivityStamp(term), "new");
});

test("terminal state table distinguishes presence from activity", () => {
  assert.equal(terminalStatus({ running: true }), "open");
  assert.equal(terminalStatusLabel({ running: true }), "Terminal open");
  assert.equal(terminalStatus({ running: true, tui: { cli: "grok" } }), "open");
  assert.equal(terminalStatusLabel({ running: true, tui: { cli: "grok" } }), "Open");
  assert.equal(terminalStatus({ running: true, state: "idle", cli: "grok" }), "ready");
  assert.equal(terminalStatusLabel({ running: true, state: "idle", cli: "grok" }), "Ready");
  assert.equal(terminalStatus({ running: true, state: "needs-you", cli: "grok" }), "needs-you");
  assert.equal(terminalStatus({ running: true, state: "working", cli: "grok" }), "working");
  assert.equal(terminalStatus({ running: false }), "stopped");
  assert.equal(terminalStatus({ running: false, state: "working", tui: { cli: "pi" } }), "stopped");
});
