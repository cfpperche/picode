// The PiCode right-click menu for a terminal pane.
// Study: docs/benchmarks/2026-09-07-terminal-context-menu.md.
//
// Pure on purpose: what the menu offers is decided from what this pane can
// actually do, so the rows are testable without a browser and the component
// stays a renderer. A row that cannot act is dropped, not greyed — the two
// exceptions are Copy and Paste, which every terminal keeps in place (VS
// Code, Ghostty, Windows Terminal) so the muscle memory never misses. Copy
// then carries the reason it is idle, as the app's generic menu already does.
//
// Icons are names, not components: this module must stay importable by the
// node test runner.

import { focusRowLabel } from "@picode/shared/domain/focusMode.js";
import { terminalHandoffMenu } from "@picode/shared/domain/sessionHandoff.js";

const SEP = { sep: true };

// Shortcuts are the ones the terminal really binds (web/shared/domain/
// termKeys.js and TermSurface's own handler). Never advertise a chord the
// app does not answer — a menu that lies about its keys is worse than one
// that shows none.
export const TERM_MENU_KEYS = {
  copy: "Ctrl+Shift+C",
  find: "Ctrl+Shift+F",
  paste: "Ctrl+Shift+V",
  "text-bigger": "Ctrl+=",
  "text-smaller": "Ctrl+-",
  "text-reset": "Ctrl+0",
  "close-tab": "Alt+W",
};

// ctx:
//   kind       "term" (a terminal of its own) | "agent" (an agent's TUI pane)
//   selection  what xterm has selected at the click (rightClickSelectsWord
//              already turned a bare right-click into the word under it)
//   cli        the Agent CLI label ("Claude Code"), "" when this pane has no
//              launch of its own — the prompt door is that launch (ADR-0089)
//   running    the CLI is up — the door only exists then
//   shell      the pane is a bare shell prompt: no CLI launch, no TUI seen
//   link       { kind: "file" | "http", label } under the cursor, or null
//   findKey    the chord Find really answers to, when the caller read the
//              user's own bindings; the default stands in otherwise
//   focus      fullscreen (focus) mode is already on — the row leaves it
//   focusable  the shell can host the mode at all (false in the ≤767px
//              column shell, which has no chrome to hide)
//   focusKey   the chord for the mode, read from the user's own bindings
//   record     the terminal JSON (lastSession, cwd) for Continue in…
//   clis       GET /api/clis rows; Continue in… derives targets from these
export function buildTermMenu(ctx = {}) {
  const selection = (ctx.selection || "").trim();
  const cli = ctx.cli || "";
  const running = !!ctx.running;
  const own = ctx.kind !== "agent"; // an agent's TUI is not a terminal to rename or remove
  const rows = [
    { id: "copy", label: "Copy", icon: "copy", key: TERM_MENU_KEYS.copy, disabled: !selection, reason: "Select text in the terminal first." },
    { id: "paste", label: "Paste", icon: "paste", key: TERM_MENU_KEYS.paste },
    { id: "select-all", label: "Select all", icon: "select" },
  ];

  // The section that makes this menu PiCode's rather than an emulator's:
  // the pane's own output becomes the agent's context (Cursor's "Add to
  // Chat", Warp's block copy), and the token under the cursor opens where
  // Ctrl+click already opens it.
  const send = [];
  if (cli && running) {
    if (selection) send.push({ id: "ask", label: "Ask " + cli + " about this", icon: "ask" });
    send.push({ id: "attach", label: "Attach files…", icon: "clip" });
  }
  if (ctx.link) send.push({ id: "open-link", label: "Open " + ctx.link.label, icon: ctx.link.kind === "http" ? "external" : "file" });
  if (send.length) rows.push(SEP, ...send);

  const view = [
    { id: "find", label: "Find…", icon: "search", key: ctx.findKey || TERM_MENU_KEYS.find },
    { id: "scroll-end", label: "Go to the end", icon: "end" },
    {
      id: "text-size",
      label: "Text size",
      icon: "text",
      sub: [
        { id: "text-bigger", label: "Bigger", key: TERM_MENU_KEYS["text-bigger"] },
        { id: "text-smaller", label: "Smaller", key: TERM_MENU_KEYS["text-smaller"] },
        { id: "text-reset", label: "Reset", key: TERM_MENU_KEYS["text-reset"] },
      ],
    },
  ];
  // Clear is a shell courtesy, not a screen wipe: it sends the same Ctrl-L
  // the user would type. A TUI owns its screen — launched by us or started
  // by hand in a plain terminal — so the row stays away from it rather than
  // pretending to clear something it does not control. An agent's pane is
  // always its TUI, so the flag cannot reach it.
  if (own && ctx.shell) view.push({ id: "clear", label: "Clear", icon: "clear" });
  // Fullscreen belongs to the shell, not to this pane, but the owner asked
  // for it on every tab's menu — and a terminal pane never shows the
  // generic one, so this is the only place it can appear over a terminal.
  if (ctx.focusable !== false) {
    view.push({ id: "fullscreen", label: focusRowLabel(!!ctx.focus), icon: ctx.focus ? "collapse" : "expand", key: ctx.focusKey || "" });
  }
  rows.push(SEP, ...view);

  if (own) {
    const ownRows = [
      { id: "rename", label: "Rename terminal…", icon: "pencil" },
      { id: "settings", label: "Terminal settings", icon: "settings" },
      { id: "files", label: "Open folder in Files", icon: "folders" },
    ];
    // Continue in… is this pane's conversation, not a session picker — the
    // pin names it (ADR-0084). An agent's TUI never reaches this block.
    const handoff = terminalHandoffMenu(ctx.record || { lastSession: ctx.lastSession }, ctx.clis);
    if (handoff) ownRows.push(handoff);
    rows.push(
      SEP,
      ...ownRows,
      SEP,
      { id: "close-tab", label: "Close tab", icon: "x", key: TERM_MENU_KEYS["close-tab"], hint: "The terminal keeps running." },
      { id: "remove", label: "Remove terminal…", icon: "trash", danger: true },
    );
  }
  return rows;
}

// A selection handed to the CLI takes one of two shapes, because a prompt
// input is one line and terminal output rarely is. Anything that reads
// cleanly inline is pre-filled as the message; the rest is staged as a text
// attachment, so the CLI reads a whole file instead of a mangled paste
// (ADR-0089 already delivers files by path). Nothing is ever truncated.
export const ASK_INLINE_MAX = 400;

export function planAsk(selection) {
  const raw = String(selection || "");
  const trimmed = raw.trim();
  if (!trimmed) return { mode: "none" };
  if (!/[\r\n]/.test(trimmed) && trimmed.length <= ASK_INLINE_MAX) return { mode: "text", text: trimmed };
  return { mode: "file", name: "selection.txt", body: raw.replace(/\s+$/, "") + "\n" };
}
