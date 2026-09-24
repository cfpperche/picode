import test from "node:test";
import assert from "node:assert/strict";
import { normalizeTerminalCli, terminalActivityStamp, terminalCli, terminalCliFaviconUrls, terminalCliLabel, terminalCliMark, terminalCommandTitle, terminalDisplayCli, terminalIsIdleShellAt, terminalStatus, terminalStatusLabel } from "./terminalCli.js";

test("terminal CLI aliases use one canonical identity", () => {
  assert.equal(normalizeTerminalCli("claude"), "claude-code");
  assert.equal(normalizeTerminalCli(" Claude-Code "), "claude-code");
  assert.equal(normalizeTerminalCli("codex"), "codex");
  assert.equal(normalizeTerminalCli("hermes"), "hermes");
  assert.equal(normalizeTerminalCli("opencode"), "opencode");
  assert.equal(normalizeTerminalCli("muse"), "muse");
  assert.equal(normalizeTerminalCli("agy"), "agy");
  assert.equal(normalizeTerminalCli("omp"), "omp");
  assert.equal(normalizeTerminalCli("unknown"), "");
  assert.equal(terminalCliLabel("pi"), "Pi");
  assert.equal(terminalCliLabel("hermes"), "Hermes Agent");
  assert.equal(terminalCliLabel("opencode"), "OpenCode");
  assert.equal(terminalCliLabel("muse"), "Muse Code");
  assert.equal(terminalCliLabel("agy"), "Antigravity");
  assert.equal(terminalCliLabel("omp"), "Omp");
  assert.equal(terminalCliMark("codex"), "Cx");
  assert.equal(terminalCliMark("hermes"), "H");
  assert.equal(terminalCliMark("opencode"), "Oc");
  assert.equal(terminalCliMark("muse"), "Mu");
  assert.equal(terminalCliMark("agy"), "Ag");
  assert.equal(terminalCliMark("omp"), "Om");
});

test("supported runtimes use their official favicons, best first", () => {
  // First links are the same transparent SVG marks the provider faces use —
  // vendor raster favicons are opaque white and read as a card behind the
  // bare favicon. Later links are the vendor's own assets.
  assert.deepEqual(terminalCliFaviconUrls("claude"), [
    "https://unpkg.com/@lobehub/icons-static-svg@1.73.0/icons/claude.svg",
    "https://www.anthropic.com/images/icons/favicon-32x32.png",
    "https://claude.ai/favicon.ico",
  ]);
  assert.deepEqual(terminalCliFaviconUrls("codex"), [
    "https://unpkg.com/@lobehub/icons-static-svg@1.73.0/icons/openai.svg",
    "https://openai.com/favicon.ico",
    "https://cdn.oaistatic.com/assets/apple-touch-icon-mz9nytnj.webp",
  ]);
  assert.deepEqual(terminalCliFaviconUrls("grok"), [
    "https://unpkg.com/@lobehub/icons-static-svg@1.73.0/icons/grok.svg",
    "https://grok.com/images/favicon.svg",
  ]);
  assert.deepEqual(terminalCliFaviconUrls("hermes"), [
    "https://unpkg.com/@lobehub/icons-static-svg@1.95.0/icons/hermes-agent.svg",
    "https://unpkg.com/@lobehub/icons-static-svg@1.73.0/icons/nousresearch.svg",
    "https://hermes-agent.nousresearch.com/favicon.ico",
  ]);
  assert.deepEqual(terminalCliFaviconUrls("opencode"), [
    "https://unpkg.com/@lobehub/icons-static-svg@1.95.0/icons/opencode.svg",
    "https://opencode.ai/favicon.svg",
    "https://opencode.ai/favicon.ico",
  ]);
  assert.deepEqual(terminalCliFaviconUrls("muse"), [
    "https://unpkg.com/@lobehub/icons-static-svg@1.73.0/icons/meta.svg",
    "https://www.meta.com/favicon.ico",
  ]);
  assert.deepEqual(terminalCliFaviconUrls("agy"), [
    "https://unpkg.com/@lobehub/icons-static-svg@1.95.0/icons/antigravity.svg",
    "https://antigravity.google/favicon.ico",
  ]);
  assert.deepEqual(terminalCliFaviconUrls("omp"), [
    "https://omp.sh/favicon.svg",
    "https://omp.sh/favicon.ico",
  ]);
  // pi's mark is local and theme-neutral: pi.dev's favicon follows the OS
  // scheme, not the app's theme.
  const [piMark] = terminalCliFaviconUrls("pi");
  assert.equal(terminalCliFaviconUrls("pi").length, 1);
  assert.ok(piMark.startsWith("data:image/svg+xml,"));
  assert.ok(decodeURIComponent(piMark).includes('fill="#8a8a96"'));
  assert.ok(!decodeURIComponent(piMark).includes("prefers-color-scheme"));
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

test("display identity falls back to the launch CLI without a runtime", () => {
  // Nothing reports a runtime for a CLI with no adapter: its terminal is
  // still that CLI's terminal, not a plain shell.
  const agy = { running: true, cli: null, launchCli: "agy", tui: null };
  assert.equal(terminalCli(agy), "");
  assert.equal(terminalDisplayCli(agy), "agy");
  assert.equal(terminalCliLabel(terminalDisplayCli(agy)), "Antigravity");
  assert.deepEqual(terminalCliFaviconUrls(terminalDisplayCli(agy)).slice(0, 1), ["https://unpkg.com/@lobehub/icons-static-svg@1.95.0/icons/antigravity.svg"]);
  assert.equal(terminalDisplayCli({ launchCli: "muse" }), "muse");
  // The runtime projection and the launch CLI still beat nothing.
  assert.equal(terminalDisplayCli({ cli: "claude", launchCli: "agy" }), "claude-code");
  assert.equal(terminalDisplayCli({}), "");
  assert.equal(terminalDisplayCli(null), "");
  // A present tui is authoritative: a Pi terminal whose CLI exited is a
  // shell again, whatever it was launched with.
  assert.equal(terminalDisplayCli({ cli: "pi", launchCli: "pi", tui: { cli: "" } }), "");
  assert.equal(terminalDisplayCli({ launchCli: "pi", tui: { cli: "claude" } }), "claude-code");
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
  // A CLI without an adapter reports no activity: same status, CLI identity.
  assert.equal(terminalStatus({ running: true, cli: null, launchCli: "agy" }), "open");
  assert.equal(terminalStatusLabel({ running: true, cli: null, launchCli: "agy" }), "Open");
  assert.equal(terminalStatusLabel({ running: true, cli: null, launchCli: "muse" }), "Open");
  assert.equal(terminalStatusLabel({ running: false, cli: null, launchCli: "muse" }), "Stopped");
  assert.equal(terminalStatus({ running: true, state: "idle", cli: "grok" }), "ready");
  assert.equal(terminalStatusLabel({ running: true, state: "idle", cli: "grok" }), "Ready");
  assert.equal(terminalStatus({ running: true, state: "needs-you", cli: "grok" }), "needs-you");
  assert.equal(terminalStatus({ running: true, state: "working", cli: "grok" }), "working");
  assert.equal(terminalStatus({ running: true, state: "compacting", cli: "codex" }), "compacting");
  assert.equal(terminalStatusLabel({ running: true, state: "compacting", cli: "codex" }), "Compacting");
  assert.equal(terminalStatus({ running: false }), "stopped");
  assert.equal(terminalStatus({ running: false, state: "working", tui: { cli: "pi" } }), "stopped");
});

// Where a git command may be typed (ADR-0096): a plain idle shell in the
// folder. An idle Agent CLI terminal reports no runtime `cli` but keeps its
// launchCli — typing there was refused as "This is an Agent CLI" and broke
// Fork agent… on 2026-09-23.
test("only a plain idle shell in the folder takes a git command", () => {
  assert.equal(terminalIsIdleShellAt({ cwd: "/r" }, "/r"), true);
  assert.equal(terminalIsIdleShellAt({ cwd: "/r", launchCli: "claude-code" }, "/r"), false);
  assert.equal(terminalIsIdleShellAt({ cwd: "/r", cli: "codex" }, "/r"), false);
  assert.equal(terminalIsIdleShellAt({ cwd: "/r", tui: { cli: "pi" } }, "/r"), false);
  assert.equal(terminalIsIdleShellAt({ cwd: "/r", state: "working" }, "/r"), false);
  assert.equal(terminalIsIdleShellAt({ cwd: "/other" }, "/r"), false);
  assert.equal(terminalIsIdleShellAt(null, "/r"), false);
  assert.equal(terminalIsIdleShellAt({ cwd: "" }, ""), false);
  // A row a feed patch stripped of its live fields is still an agent's.
  assert.equal(terminalIsIdleShellAt({ id: "fork-1", cwd: "/r" }, "/r", new Set(["fork-1"])), false);
  assert.equal(terminalIsIdleShellAt({ id: "sh-1", cwd: "/r" }, "/r", new Set(["fork-1"])), true);
});

test("a running shell command reads as work beside the hook state (ADR-0212)", () => {
  const since = "2026-09-24T10:00:00Z";
  const base = { id: "t", running: true, cli: "codex", tui: { cli: "codex", runId: "r" }, runId: "r", stateAt: "2026-09-24T09:00:00Z" };
  // "!make deploy" after the turn ended: the hook said idle, the tree says make.
  const bang = { ...base, state: "idle", command: { name: "make", since } };
  assert.equal(terminalStatus(bang), "working");
  assert.equal(terminalStatusLabel(bang), "Running");
  assert.equal(terminalCommandTitle(bang), "Running make");
  assert.equal(terminalActivityStamp(bang), since);
  // Inside a turn the hook's word wins; the command only names the tool.
  const turn = { ...base, state: "working", command: { name: "make", since } };
  assert.equal(terminalStatusLabel(turn), "Working");
  assert.equal(terminalActivityStamp(turn), base.stateAt);
  // Needs you outranks a command still running.
  assert.equal(terminalStatus({ ...base, state: "needs-you", command: { name: "make", since } }), "needs-you");
  // No command: nothing changes.
  assert.equal(terminalStatus({ ...base, state: "idle" }), "ready");
  assert.equal(terminalCommandTitle({ ...base, state: "idle" }), "");
});
