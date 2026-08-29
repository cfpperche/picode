// Terminal keys (VS Code integrated-terminal defaults).
// Modified Enter goes as xterm modifyOtherKeys (CSI 27;mod;13~): tmux
// (extended-keys-format xterm, set on attach) re-encodes it to the pane,
// and pi's fallback parser expects exactly this. VS Code's own terminal
// reaches the same result via Kitty in its xterm fork; the OSS xterm.js
// we embed has neither, so we encode here.
// Ctrl+Shift+C/V copy/paste (VS Code). Ctrl+C interrupts; "copy if
// selected" (Warp) is opt-in in Preferences → Terminal → Keys.

import { readTermPrefs, TERM_NEWLINES } from "./termTheme.js";

const SEQ = {
  "shift-enter": "\x1b[27;2;13~",
  "alt-enter": "\x1b[27;3;13~",
  "ctrl-enter": "\x1b[27;5;13~",
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
  if (!shift && k === "c" && (prefs ? prefs.copyIfSelection === true : false)) return "copy-if-sel";
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
