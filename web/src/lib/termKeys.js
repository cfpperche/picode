// Terminal keys (Warp / Windows Terminal / VS Code terminal).
// Shift+Enter (or Ctrl/Alt+Enter) inserts a line; Enter still sends.
// Ctrl+C copies when text is selected, else SIGINT. Ctrl+Shift+C/V always
// copy/paste. Sequences: xterm modifyOtherKeys CSI 27 ; mod ; 13 ~.

import { readTermPrefs, TERM_NEWLINES } from "./termTheme.js";

const ENTER = 13;
const SEQ = {
  "shift-enter": "\x1b[27;2;" + ENTER + "~",
  "alt-enter": "\x1b[27;3;" + ENTER + "~",
  "ctrl-enter": "\x1b[27;5;" + ENTER + "~",
};

export function newlineSeq(ev, newlineKey) {
  if (!ev || ev.type === "keyup" || ev.key !== "Enter" || ev.repeat) return null;
  const want = newlineKey || "shift-enter";
  const shift = !!ev.shiftKey;
  const alt = !!ev.altKey;
  const ctrl = !!(ev.ctrlKey || ev.metaKey);
  if (want === "ctrl-enter" && ctrl && !shift && !alt) return SEQ["ctrl-enter"];
  if (want === "alt-enter" && alt && !shift && !ctrl) return SEQ["alt-enter"];
  if (want === "shift-enter" && shift && !alt && !ctrl) return SEQ["shift-enter"];
  return null;
}

export function copyPasteAction(ev, prefs) {
  if (!ev || ev.type === "keyup") return null;
  const ctrl = !!(ev.ctrlKey || ev.metaKey);
  if (!ctrl) return null;
  const k = (ev.key || "").toLowerCase();
  const shift = !!ev.shiftKey;
  if (shift && k === "c") return "copy";
  if (shift && k === "v") return "paste";
  if (!shift && k === "c" && (prefs ? prefs.copyIfSelection !== false : true)) return "copy-if-sel";
  return null;
}

export function termShortcutRows(prefs) {
  const p = prefs || readTermPrefs();
  const nl = (TERM_NEWLINES.find((k) => k.id === p.newlineKey) || TERM_NEWLINES[0]).label;
  const rows = [
    { key: "Ctrl+`", label: "New terminal" },
    { key: nl, label: "New line" },
  ];
  if (p.copyIfSelection !== false) {
    rows.push({ key: "Ctrl+C", label: "Copy if selected; else interrupt" });
  }
  rows.push(
    { key: "Ctrl+Shift+C", label: "Copy" },
    { key: "Ctrl+Shift+V", label: "Paste" },
    { key: "Ctrl++ / − / 0", label: "Font size" },
  );
  return rows;
}

export function wireTermKeys(term, send) {
  if (!term || typeof term.attachCustomKeyEventHandler !== "function") return;
  term.attachCustomKeyEventHandler((ev) => {
    if (!ev || ev.type !== "keydown") return true;
    const prefs = readTermPrefs();
    const clip = copyPasteAction(ev, prefs);
    if (clip === "copy" || clip === "copy-if-sel") {
      const sel = term.getSelection && term.getSelection();
      if (clip === "copy-if-sel" && !sel) return true;
      if (sel && navigator.clipboard && navigator.clipboard.writeText) navigator.clipboard.writeText(sel);
      if (typeof ev.preventDefault === "function") ev.preventDefault();
      return false;
    }
    if (clip === "paste") {
      if (navigator.clipboard && navigator.clipboard.readText) {
        navigator.clipboard.readText().then((t) => { if (t && term.paste) term.paste(t); }).catch(() => {});
      }
      if (typeof ev.preventDefault === "function") ev.preventDefault();
      return false;
    }
    const seq = newlineSeq(ev, prefs.newlineKey);
    if (!seq) return true;
    if (typeof ev.preventDefault === "function") ev.preventDefault();
    if (send) send(new TextEncoder().encode(seq));
    return false;
  });
}
