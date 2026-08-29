// Terminal keys (VS Code integrated-terminal defaults).
// Modified Enter goes as xterm modifyOtherKeys (CSI 27;mod;13~): tmux
// (extended-keys-format xterm, set on attach) re-encodes it to the pane,
// and pi's fallback parser expects exactly this. VS Code's own terminal
// reaches the same result via Kitty in its xterm fork; the OSS xterm.js
// we embed has neither, so we encode here.
//
// Three layers, because browsers disagree on what follows a canceled
// keydown (keypress on Windows Chrome, input from IME, CDP…):
//  1. customKeyEventHandler: keydown → send the sequence, block the rest;
//  2. keypress of a bound modified Enter → blocked (xterm _keyPress would
//     write \r even after a canceled keydown);
//  3. termDataFilter on term.onData: any \r/\n that still escapes within
//     120ms of a bound modified-Enter keydown is swapped for the sequence
//     (or dropped, if layer 1 already sent it).
// Copy/paste: Shift+drag selects (xterm bypasses tmux mouse). Ctrl+C
// copies if there is a selection, else interrupt. Ctrl+V pastes.
// Ctrl+Shift+C/V still copy/paste.

import { readTermPrefs, TERM_NEWLINES } from "./termTheme.js";

const SEQ = {
  "shift-enter": "\x1b[27;2;13~",
  "alt-enter": "\x1b[27;3;13~",
  "ctrl-enter": "\x1b[27;5;13~",
};

const WINDOW = 120;
let recent = null; // { t, seq, sent } — last modified-Enter keydown

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
  if (k === "c" && shift) return "copy";
  if (k === "v") return "paste";
  if (k === "c" && !shift) return "copy-if-sel";
  return null;
}

export function termShortcutRows(prefs) {
  const p = prefs || readTermPrefs();
  const nl = (TERM_NEWLINES.find((k) => k.id === p.newlineKey) || TERM_NEWLINES[0]).label;
  const rows = [
    { key: "Ctrl+`", label: "New terminal" },
    { key: nl, label: "New line" },
  ];
  rows.push(
    { key: "Shift+drag", label: "Select" },
    { key: "Ctrl+C", label: "Copy if selected; else interrupt" },
    { key: "Ctrl+V", label: "Paste" },
    { key: "Ctrl+Shift+C", label: "Copy" },
    { key: "Ctrl+Shift+V", label: "Paste" },
    { key: "Ctrl++ / − / 0", label: "Font size" },
  );
  return rows;
}

// Layer 3: pass term.onData through this. "\r" right after a bound
// modified-Enter keydown becomes the sequence (or "" when already sent).
export function termDataFilter(data) {
  if (data !== "\r" && data !== "\n") return data;
  const r = recent;
  if (!r || Date.now() - r.t > WINDOW) return data; // plain Enter
  if (r.sent) return ""; // echo after layer 1 — drop
  if (!r.seq) return data; // modified but not the bound key — VS Code parity: pass \r
  r.sent = true;
  return r.seq; // the path layer 1 never caught
}

function trackKeydown(ev) {
  if (!ev || ev.type !== "keydown" || ev.key !== "Enter") return;
  const mod = ev.shiftKey || ev.altKey || ev.ctrlKey || ev.metaKey;
  if (!mod) {
    recent = null;
    return;
  }
  recent = { t: Date.now(), seq: newlineSeq(ev, readTermPrefs().newlineKey), sent: false };
}

export function wireTermKeys(term, send) {
  if (!term || typeof term.attachCustomKeyEventHandler !== "function") return;
  const ta = term.textarea || (term.element && term.element.querySelector("textarea"));
  if (ta) ta.addEventListener("keydown", trackKeydown, true);
  term.attachCustomKeyEventHandler((ev) => {
    if (!ev || (ev.type !== "keydown" && ev.type !== "keypress")) return true;
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
    if (ev.type === "keydown") {
      if (send) send(new TextEncoder().encode(seq));
      if (recent) recent.sent = true;
    }
    return false;
  });
}
