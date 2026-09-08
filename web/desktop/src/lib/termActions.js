// What the terminal context menu does once a row is chosen.
// Study: docs/benchmarks/2026-09-07-terminal-context-menu.md.
//
// Everything a pane can answer on its own (clipboard, selection, scroll,
// font, clear) runs here against the live xterm instance; everything that
// belongs to the application (attach, rename, tabs, routes) is handed back
// to App through `handlers`. The split keeps App free of xterm internals
// and keeps this file free of routing.
//
// Copy and paste deliberately reuse xterm's own doors — `getSelection` and
// `paste` — the same ones Ctrl+Shift+C/V already drive in
// web/shared/domain/termKeys.js. Paste is a user gesture on the user's own
// keyboard equivalent, not PiCode writing into a CLI on its own initiative
// (the line ADR-0078 draws for the Inspector).

import { terms } from "./terms.js";
import { bumpTermFontSize } from "@picode/shared/domain/termTheme.js";
import { cellFromMouse, linkAt } from "@picode/shared/domain/termLinks.js";
import { toast } from "./toast.js";

const key = (id) => "sh:" + id;

// The pane under a right-click, or null when the event came from anywhere
// else. ShellTerm stamps the ids; the instance itself lives in the terms map.
export function paneAt(target) {
  const el = target && target.closest ? target.closest(".term-pane") : null;
  if (!el || !el.dataset || !el.dataset.termId) return null;
  const id = el.dataset.termId;
  return {
    id,
    kind: el.dataset.termKind === "agent" ? "agent" : "term",
    cwd: el.dataset.termCwd || "",
    entry: terms.get(key(id)) || null,
  };
}

export function paneSelection(entry) {
  const term = entry && entry.term;
  if (!term || typeof term.getSelection !== "function") return "";
  return term.getSelection() || "";
}

// The token under the cursor, resolved by the same rules that draw the
// Ctrl+click underline (termLinks.js) — so the menu can only offer to open
// what the pane already treats as a link. A cwd that lagged behind a `cd`
// makes this miss, exactly as the underline does.
export function paneLink(entry, ev, cwd) {
  const term = entry && entry.term;
  if (!term) return null;
  const cell = cellFromMouse(term, ev);
  const buf = term.buffer && term.buffer.active;
  const line = cell && buf ? buf.getLine(cell.lineIndex) : null;
  if (!line) return null;
  const hit = linkAt(line.translateToString(true), cell.col, cwd || "");
  if (!hit) return null;
  if (hit.kind === "http") {
    let label = hit.href;
    try { label = new URL(hit.href).hostname || hit.href; } catch { /* keep the raw href */ }
    return { kind: "http", label, href: hit.href };
  }
  const path = hit.path || "";
  return { kind: "file", label: path.split("/").pop() || path, path };
}

function sendPane(entry, text) {
  const sock = entry && entry.sock;
  if (!sock || sock.readyState !== 1) {
    toast.error("This terminal is not connected.");
    return false;
  }
  sock.send(new TextEncoder().encode(text));
  return true;
}

export function runTermCommand(cmd, ctx, handlers = {}) {
  const entry = terms.get(key(ctx.id));
  const term = entry && entry.term;
  const focus = () => { if (term && term.focus) term.focus(); };
  switch (cmd) {
    case "copy":
      if (!ctx.selection) return;
      navigator.clipboard.writeText(ctx.selection)
        .then(() => toast.ok("Copied."))
        .catch(() => toast.error("Clipboard blocked — copy manually with Ctrl+Shift+C."));
      focus();
      return;
    case "paste":
      navigator.clipboard.readText()
        .then((text) => { if (text && term && term.paste) term.paste(text); focus(); })
        .catch(() => toast.error("Clipboard blocked — paste manually with Ctrl+Shift+V."));
      return;
    case "select-all":
      if (term) term.selectAll();
      focus();
      return;
    case "scroll-end":
      if (term) term.scrollToBottom();
      focus();
      return;
    case "text-bigger": bumpTermFontSize(1); focus(); return;
    case "text-smaller": bumpTermFontSize(-1); focus(); return;
    case "text-reset": bumpTermFontSize(0); focus(); return;
    // Ctrl-L: the keystroke the user would type, sent to the pane. Nothing
    // is wiped locally — a local clear would only desynchronise the view
    // from the tmux pane that owns the scrollback.
    case "clear":
      if (sendPane(entry, "\f")) focus();
      return;
    default: {
      const fn = handlers[cmd];
      if (fn) fn(ctx);
    }
  }
}

// Put the caret back where the user was working. Called after the menu
// closes: Radix restores focus to the virtual trigger, which is nowhere.
export function focusPane(id) {
  const entry = terms.get(key(id));
  if (entry && entry.term && entry.term.focus) entry.term.focus();
}
