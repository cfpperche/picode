import { test } from "node:test";
import assert from "node:assert/strict";
import { buildTermMenu, planAsk, ASK_INLINE_MAX } from "./termMenu.js";

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

test("an agent's TUI pane is not a terminal to rename, close or remove", () => {
  const rows = ids(buildTermMenu({ kind: "agent", selection: "x" }));
  for (const id of ["rename", "settings", "files", "close-tab", "remove"]) {
    assert.equal(rows.includes(id), false, id + " must not reach an agent pane");
  }
  assert.deepEqual(rows.slice(0, 3), ["copy", "paste", "select-all"]);
});

test("remove is the only destructive row, and text size is the only submenu", () => {
  const rows = buildTermMenu({ kind: "term" });
  assert.deepEqual(rows.filter((r) => r.danger).map((r) => r.id), ["remove"]);
  const subs = rows.filter((r) => r.sub);
  assert.deepEqual(subs.map((r) => r.id), ["text-size"]);
  assert.deepEqual(subs[0].sub.map((r) => r.id), ["text-bigger", "text-smaller", "text-reset"]);
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
