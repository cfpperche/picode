import { test } from "node:test";
import assert from "node:assert/strict";
import { buildTermMenu, paneCapabilities, planAsk, ASK_INLINE_MAX } from "./termMenu.js";

const ids = (rows) => rows.filter((r) => !r.sep).map((r) => r.id);
const row = (rows, id) => rows.find((r) => r.id === id);

test("copy stays in place with its reason when nothing is selected", () => {
  const rows = buildTermMenu({ kind: "term" });
  const copy = row(rows, "copy");
  assert.equal(copy.disabled, true);
  assert.match(copy.reason, /Select text/);
  assert.equal(row(rows, "paste").disabled, undefined);
});

test("a selection arms copy and, on a running CLI, the ask row", () => {
  const rows = buildTermMenu({ kind: "term", selection: "panic: nil map", cli: "Claude Code", running: true });
  assert.equal(row(rows, "copy").disabled, false);
  assert.equal(row(rows, "ask").label, "Ask Claude Code about this");
  assert.ok(ids(rows).includes("attach"));
});

test("whitespace is not a selection", () => {
  const rows = buildTermMenu({ kind: "term", selection: "  \n ", cli: "Codex", running: true });
  assert.equal(row(rows, "copy").disabled, true);
  assert.equal(row(rows, "ask"), undefined);
});

test("the prompt door only exists while the CLI runs", () => {
  const stopped = ids(buildTermMenu({ kind: "term", selection: "boom", cli: "Codex", running: false }));
  assert.equal(stopped.includes("ask"), false);
  assert.equal(stopped.includes("attach"), false);
  assert.equal(stopped.includes("snippet"), false);
});

test("send-to-terminal follows the prompt door (ADR-0089)", () => {
  const rows = buildTermMenu({ kind: "term", cli: "Pi", running: true });
  assert.equal(row(rows, "snippet").label, "Send to terminal…");
  assert.equal(ids(buildTermMenu({ kind: "term" })).includes("snippet"), false);
  assert.equal(ids(buildTermMenu({ kind: "term", shell: true })).includes("snippet"), false);
});

test("run-command lives on a bare shell pane (ADR-0130 Q2a)", () => {
  const rows = buildTermMenu({ kind: "term", shell: true });
  assert.equal(row(rows, "snippet-cmd").label, "Run command…");
  assert.equal(ids(buildTermMenu({ kind: "term", cli: "Pi", running: true })).includes("snippet-cmd"), false);
  assert.equal(ids(buildTermMenu({ kind: "agent", shell: true })).includes("snippet-cmd"), false);
});

test("find is offered on every pane, with the chord the app really answers", () => {
  for (const ctx of [{ kind: "term" }, { kind: "term", cli: "Pi", running: true }, { kind: "agent" }]) {
    assert.ok(ids(buildTermMenu(ctx)).includes("find"), JSON.stringify(ctx));
  }
  assert.equal(row(buildTermMenu({ kind: "term" }), "find").key, "Ctrl+Shift+F");
  assert.equal(row(buildTermMenu({ kind: "term", findKey: "Alt+F" }), "find").key, "Alt+F");
});

test("only a bare shell prompt gets clear", () => {
  assert.ok(ids(buildTermMenu({ kind: "term", shell: true })).includes("clear"));
  // A launched CLI, a TUI someone started by hand, and an agent's own TUI
  // all own their screen.
  assert.equal(ids(buildTermMenu({ kind: "term", cli: "Pi", running: true })).includes("clear"), false);
  assert.equal(ids(buildTermMenu({ kind: "term" })).includes("clear"), false);
  assert.equal(ids(buildTermMenu({ kind: "agent", shell: true })).includes("clear"), false);
});

test("the token under the cursor becomes one open row", () => {
  const file = row(buildTermMenu({ kind: "term", link: { kind: "file", label: "termLinks.js" } }), "open-link");
  assert.equal(file.label, "Open termLinks.js");
  assert.equal(file.icon, "file");
  const http = row(buildTermMenu({ kind: "term", link: { kind: "http", label: "github.com" } }), "open-link");
  assert.equal(http.icon, "external");
});

test("an agent pane without a fleet row has no lifecycle rows", () => {
  const rows = ids(buildTermMenu({ kind: "agent", selection: "x" }));
  for (const id of ["rename", "settings", "files", "close-tab", "remove", "ask", "attach", "snippet"]) {
    assert.equal(rows.includes(id), false, id + " must not reach a missing fleet row");
  }
  assert.deepEqual(rows.slice(0, 3), ["copy", "paste", "select-all"]);
});

test("remove is the only destructive row, and text size is the only submenu without a pin", () => {
  const rows = buildTermMenu({ kind: "term" });
  assert.deepEqual(rows.filter((r) => r.danger).map((r) => r.id), ["remove"]);
  const subs = rows.filter((r) => r.sub);
  assert.deepEqual(subs.map((r) => r.id), ["text-size"]);
  assert.deepEqual(subs[0].sub.map((r) => r.id), ["text-bigger", "text-smaller", "text-reset"]);
});

test("a pinned CLI pane offers Continue in…; a missing agent fleet row does not", () => {
  const cap = { list: true, read: true, write: true, prompt: true };
  const clis = [
    { id: "pi", name: "Pi", installed: true, sessions: cap },
    { id: "codex", name: "Codex", installed: true, sessions: cap },
  ];
  const record = { lastSession: { cli: "pi", sessionId: "s1", path: "/p", cwd: "/w" }, cwd: "/w", name: "communication" };
  const pane = buildTermMenu({ kind: "term", record, clis });
  const handoff = row(pane, "handoff");
  assert.equal(handoff.label, "Continue in…");
  assert.deepEqual(handoff.sub.map((s) => s.target.id), ["codex"]);
  assert.ok(ids(pane).indexOf("handoff") > ids(pane).indexOf("settings"));
  assert.ok(ids(pane).indexOf("handoff") < ids(pane).indexOf("close-tab"));
  const agent = ids(buildTermMenu({ kind: "agent", record, clis }));
  assert.equal(agent.includes("handoff"), false);
});

test("no separator ever leads, trails or doubles", () => {
  const shapes = [
    { kind: "term" },
    { kind: "term", cli: "Pi", running: true },
    { kind: "agent" },
    { kind: "agent", selection: "x", link: { kind: "http", label: "a.dev" } },
    { kind: "term", selection: "x", cli: "Grok", running: true, link: { kind: "file", label: "a.go" } },
  ];
  for (const ctx of shapes) {
    const rows = buildTermMenu(ctx);
    assert.equal(!!rows[0].sep, false, "leading separator");
    assert.equal(!!rows[rows.length - 1].sep, false, "trailing separator");
    for (let i = 1; i < rows.length; i++) {
      assert.equal(!!(rows[i].sep && rows[i - 1].sep), false, "double separator");
    }
  }
});

test("one selected line is pre-filled; anything larger is staged as a file", () => {
  assert.deepEqual(planAsk("  panic: nil map  "), { mode: "text", text: "panic: nil map" });
  assert.equal(planAsk("").mode, "none");
  assert.equal(planAsk("   \n\t ").mode, "none");
  const multi = planAsk("stack:\n  a.go:3\n  b.go:9\n\n");
  assert.equal(multi.mode, "file");
  assert.equal(multi.name, "selection.txt");
  assert.equal(multi.body, "stack:\n  a.go:3\n  b.go:9\n");
  assert.equal(planAsk("x".repeat(ASK_INLINE_MAX)).mode, "text");
  assert.equal(planAsk("x".repeat(ASK_INLINE_MAX + 1)).mode, "file");
});

test("every terminal pane offers fullscreen, with the chord and the verb it will do", () => {
  for (const kind of ["term", "agent"]) {
    const off = buildTermMenu({ kind, focusKey: "Ctrl+Shift+Enter" });
    assert.equal(row(off, "fullscreen").label, "Fullscreen");
    assert.equal(row(off, "fullscreen").key, "Ctrl+Shift+Enter");
    assert.equal(row(off, "fullscreen").icon, "expand");
    const on = buildTermMenu({ kind, focus: true, focusKey: "Ctrl+Shift+Enter" });
    assert.equal(row(on, "fullscreen").label, "Leave fullscreen");
    assert.equal(row(on, "fullscreen").icon, "collapse");
  }
});

test("a shell too narrow to host the mode drops the row instead of greying it", () => {
  assert.equal(ids(buildTermMenu({ kind: "term", focusable: false })).includes("fullscreen"), false);
});

test("agent pane offers the browser split; own terminal does not (ADR-0135 slice 1)", () => {
  const ids = (rows) => rows.filter((r) => r && r.id).map((r) => r.id);
  assert.ok(ids(buildTermMenu({ kind: "agent" })).includes("open-browser"));
  assert.ok(!ids(buildTermMenu({ kind: "agent" })).includes("close-browser"));
  assert.ok(ids(buildTermMenu({ kind: "agent", splitOn: true })).includes("close-browser"));
  assert.ok(!ids(buildTermMenu({ kind: "agent", splitOn: true })).includes("open-browser"));
  // A terminal launched with an agent CLI hosts an agent too — pi included.
  assert.ok(ids(buildTermMenu({ kind: "term", cli: "Pi", running: true })).includes("open-browser"));
  const shell = ids(buildTermMenu({ kind: "term", shell: true }));
  assert.ok(!shell.includes("open-browser") && !shell.includes("close-browser"));
});

const cap = { list: true, read: true, write: true, prompt: true };
const handoffClis = [
  { id: "pi", name: "Pi", installed: true, sessions: cap },
  { id: "codex", name: "Codex", installed: true, sessions: cap },
];

function atui(extra = {}) {
  return paneCapabilities({
    host: "agent",
    agent: { id: "ag1", name: "Snippets", mode: "interactive", sessionPath: "/p/session.jsonl", workspaceId: "ws1" },
    workspace: { id: "ws1", path: "/home/goat/picode" },
    paneCwd: "/home/goat/picode",
    tabs: ["ag1"],
    clis: handoffClis,
    ...extra,
  });
}

test("A-TUI with a fleet row gets the /term/ catalog including Ask/Attach/Send", () => {
  const rows = buildTermMenu({ ...atui(), selection: "panic: nil map" });
  const got = ids(rows);
  for (const id of ["copy", "paste", "select-all", "ask", "attach", "snippet", "find", "open-browser", "rename", "settings", "files", "handoff", "close-tab", "remove"]) {
    assert.ok(got.includes(id), id);
  }
  assert.equal(row(rows, "ask").label, "Ask Pi about this");
  assert.equal(row(rows, "rename").label, "Rename agent…");
  assert.equal(row(rows, "remove").label, "Remove agent…");
  assert.equal(row(rows, "close-tab").hint, "The session keeps running.");
  assert.equal(got.includes("snippet-cmd"), false);
  assert.equal(got.includes("clear"), false);
});

test("A-TUI + selection has ask, attach and snippet (promptDoor tui)", () => {
  const caps = atui({ selection: "x" });
  assert.equal(caps.promptDoor, "tui");
  assert.equal(caps.kind, "agent");
  const got = ids(buildTermMenu(caps));
  assert.ok(got.includes("snippet"));
  assert.ok(got.includes("ask"));
  assert.ok(got.includes("attach"));
});

test("A-TUI without a selection keeps Attach and Send, drops Ask", () => {
  const got = ids(buildTermMenu(atui()));
  assert.equal(got.includes("ask"), false);
  assert.ok(got.includes("attach"));
  assert.ok(got.includes("snippet"));
});

test("A-TUI Close tab follows tabOpen; Canvas without a host tab omits it", () => {
  const open = ids(buildTermMenu(atui({ tabs: ["ag1"] })));
  assert.ok(open.includes("close-tab"));
  const closed = ids(buildTermMenu(atui({ tabs: ["x:canvas"] })));
  assert.equal(closed.includes("close-tab"), false);
  assert.ok(closed.includes("rename"));
  assert.ok(closed.includes("remove"));
});

test("A-TUI without sessionPath omits Continue in…", () => {
  const caps = atui({ agent: { id: "ag1", name: "x", mode: "interactive" } });
  assert.equal(ids(buildTermMenu(caps)).includes("handoff"), false);
});

test("workspace agent with no workPath still pins cwd from the workspace", () => {
  const caps = atui({ agent: { id: "ag1", name: "Snippets", mode: "interactive", sessionPath: "/p", workspaceId: "ws1" } });
  assert.equal(caps.record.cwd, "/home/goat/picode");
  assert.ok(ids(buildTermMenu(caps)).includes("handoff"));
});

test("missing fleet row omits lifecycle and the prompt door", () => {
  const caps = paneCapabilities({ host: "agent", tabs: ["ag1"], selection: "x" });
  assert.equal(caps.lifecycle, null);
  assert.equal(caps.promptDoor, null);
  const got = ids(buildTermMenu(caps));
  for (const id of ["rename", "remove", "settings", "files", "close-tab", "snippet", "ask", "attach"]) {
    assert.equal(got.includes(id), false, id);
  }
});

test("do not light Ask/Attach from cli/running on an agent pane — promptDoor is the gate", () => {
  // The forbidden shortcut: filling cli/running as if the agent were a terminal.
  const poison = buildTermMenu({ kind: "agent", cli: "Pi", running: true, selection: "x" });
  assert.equal(ids(poison).includes("ask"), false);
  assert.equal(ids(poison).includes("attach"), false);
});

test("T-CLI still uses cli && running when promptDoor is omitted", () => {
  const rows = buildTermMenu({ kind: "term", selection: "x", cli: "Claude Code", running: true });
  assert.ok(ids(rows).includes("ask"));
  assert.equal(row(rows, "rename").label, "Rename terminal…");
});
