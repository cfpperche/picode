// One scrollbar per surface, and never one from the web terminal
// (2026-09-11). The web terminal is a tmux client and tmux attaches on the
// alternate screen (internal/term/bridge.go), so xterm's buffer has no
// scrollback: its viewport has nothing to scroll, and xterm's own scrollbar
// — one pixel wide, since `overviewRuler.width` is also its
// `verticalScrollbarSize` — was the stray bar a reader saw beside the TUI's.
// Both are hidden, and the fit keeps reserving 1px so no dead gutter comes
// back (the 14px one commit 95c097c8 removed). The bar a reader sees belongs
// to whoever holds the scrollback: tmux's copy-mode indicator, or the TUI's
// own scrollbar. Changing this means giving the reader a browser-side
// scrollbar instead — a decision, not a style fix.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

const read = (path) => readFileSync(new URL("../" + path, import.meta.url), "utf8");

const desktop = read("web/browser/src/styles/app.css");
const mobile = read("web/mobile/src/mobile.css");
const theme = read("web/shared/domain/termTheme.js");

test("desktop hides both scrollbars xterm can draw", () => {
  assert.match(desktop, /\.xterm \.xterm-viewport \{ scrollbar-width: none; \}/);
  assert.match(desktop, /\.xterm \.xterm-viewport::-webkit-scrollbar \{ display: none; \}/);
  assert.match(desktop, /\.xterm \.xterm-scrollable-element > \.scrollbar \{ display: none; \}/);
});

test("mobile hides xterm's own scrollbar beside the viewport's", () => {
  assert.match(mobile, /#m-app \.xterm-viewport \{ scrollbar-width: none; \}/);
  assert.match(mobile, /#m-app \.xterm-viewport::-webkit-scrollbar \{ display: none; width: 0; \}/);
  assert.match(mobile, /#m-app \.xterm-scrollable-element > \.scrollbar \{ display: none; \}/);
});

test("the fit still reserves 1px, not a 14px gutter", () => {
  assert.match(theme, /overviewRuler: \{ width: 1 \}/);
});

test("the find addon's overview ruler is not the scrollbar", () => {
  // The ruler is its own canvas; hiding the scrollbar must not hide it, or
  // search matches lose their marks in the terminal.
  assert.doesNotMatch(desktop, /decoration-overview-ruler[^{]*\{[^}]*display: none/);
});

test("the ruler paints no outline of its own", () => {
  // `overviewRulerBorder` is drawn at full alpha across the ruler's column;
  // at one pixel wide that column is a stray line at the terminal's right
  // edge. Transparent, so only real decorations (find marks) paint.
  assert.match(theme, /const RULER_BORDER = "#00000000";/);
  assert.equal((theme.match(/overviewRulerBorder: RULER_BORDER/g) || []).length, 2);
});
