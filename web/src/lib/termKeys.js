// Keys users already have in VS Code / Windows Terminal / Pi TUI:
// Shift+Enter (or Ctrl/Alt+Enter) inserts a line; Enter still sends.
// Ctrl+Shift+C/V copy/paste without sending SIGINT.
// Sequences: xterm modifyOtherKeys CSI 27 ; mod ; 13 ~  (Pi matches these).

import { readTermPrefs } from "./termTheme.js";

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

export function copyPasteAction(ev) {
  if (!ev || ev.type === "keyup") return null;
  const ctrl = !!(ev.ctrlKey || ev.metaKey);
  if (!ctrl || !ev.shiftKey) return null;
  const k = (ev.key || "").toLowerCase();
  if (k === "c") return "copy";
  if (k === "v") return "paste";
  return null;
}

export function wireTermKeys(term, send) {
  if (!term || typeof term.attachCustomKeyEventHandler !== "function") return;
  term.attachCustomKeyEventHandler((ev) => {
    if (!ev || ev.type !== "keydown") return true;
    const clip = copyPasteAction(ev);
    if (clip === "copy") {
      const sel = term.getSelection && term.getSelection();
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
    const seq = newlineSeq(ev, readTermPrefs().newlineKey);
    if (!seq) return true;
    if (typeof ev.preventDefault === "function") ev.preventDefault();
    if (send) send(new TextEncoder().encode(seq));
    return false;
  });
}
