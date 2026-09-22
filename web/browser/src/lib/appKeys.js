import { fromEvent, effectiveKeys } from "@picode/shared/domain/piKey.js";
import { readAppKeyOverrides } from "./appKeyPrefs.js";

// Same-group entries must stay contiguous — AppKeys.jsx groups via a linear
// walk (copied from PiKeys.jsx), not a sort/groupBy.
export const CATALOG = [
  { id: "app.palette.toggle", group: "Global", label: "Command palette", defaults: ["ctrl+k", "super+k"] },
  { id: "app.terminal.new", group: "Global", label: "New terminal", defaults: ["ctrl+`", "super+`"] },
  { id: "app.inspector.toggle", group: "Global", label: "Toggle inspector", defaults: ["ctrl+.", "super+."] },
  // Ctrl+Shift+F, not Ctrl+F: the terminal's own family (Ctrl+Shift+C/V,
  // termKeys.js) keeps the plain chord for the CLI — `less`, `vim` and
  // readline all use Ctrl+F — and Ctrl+F still opens the browser's find
  // everywhere outside a pane. Windows Terminal and GNOME Terminal use the
  // same chord for find.
  { id: "app.terminal.find", group: "Global", label: "Find in terminal", defaults: ["ctrl+shift+f", "super+shift+f"] },
  // Browsers reserve Ctrl+Tab, Ctrl+PgUp/PgDn and Ctrl+W, so tabs cycle on
  // Alt+bracket (docs/benchmarks/2026-09-06-tab-strip-overflow.md, owner
  // decision 2026-09-06).
  // Fullscreen (focus) mode. Ctrl+Shift+Enter is free on every side of
  // this app: nothing else in CATALOG uses it, the terminal's own family
  // needs exactly one modifier for a newline (termKeys.js binds
  // Shift/Alt/Ctrl+Enter, never two at once) and reserves Ctrl+Shift only
  // for C and V, and no browser binds it inside a page (Chrome's
  // Ctrl+Shift+Enter belongs to the omnibox). Enter, unlike a punctuation
  // key, means the same chord on every keyboard layout — Ctrl+Shift+. is
  // ">" on US and ":" on ABNT2, so fromEvent would read two chords.
  { id: "app.fullscreen.toggle", group: "Global", label: "Fullscreen", defaults: ["ctrl+shift+enter", "super+shift+enter"] },
  { id: "app.tab.prev", group: "Global", label: "Previous tab", defaults: ["alt+["] },
  { id: "app.tab.next", group: "Global", label: "Next tab", defaults: ["alt+]"] },
  { id: "app.tab.close", group: "Global", label: "Close tab", defaults: ["alt+w"] },
  { id: "composer.voice.toggle", group: "Composer", label: "Toggle voice", defaults: ["ctrl+shift+o", "super+shift+o"] },
  { id: "composer.dictate", group: "Composer", label: "Dictate", defaults: ["ctrl+d", "super+d"] },
];

export function matchAction(id, ev, overrides = readAppKeyOverrides()) {
  const action = CATALOG.find((a) => a.id === id);
  if (!action) return false;
  const chord = fromEvent(ev);
  return !!chord && effectiveKeys(action, overrides).includes(chord);
}

export function primaryChord(id, overrides) {
  const action = CATALOG.find((a) => a.id === id);
  if (!action) return "";
  return effectiveKeys(action, overrides)[0] || "";
}

const PART_LABELS = { ctrl: "Ctrl", shift: "Shift", alt: "Alt", super: "Cmd" };

// A chord's separator is the CLI's own (the registry's vocab): codex writes
// `ctrl-t`, pi and omp `ctrl+alt+k`. The display keeps that separator and only
// title-cases the words — a vocabulary is never translated into another one
// (ADR-0174), and a chord the user can copy out of the pane is one they can
// paste into their own file.
export function formatChord(chord, vocab = "") {
  if (!chord) return "";
  const sep = vocab === "codex" ? "-" : "+";
  return chord
    .split(sep)
    .map((part) => PART_LABELS[part] || (part.length === 1 ? part.toUpperCase() : part[0].toUpperCase() + part.slice(1)))
    .join(sep);
}

// True when the event is any Global app chord. Terminals hand these keys
// back to the app instead of writing them to the shell (Ctrl+K used to
// open the palette *and* send \x0b; Alt+[ would have sent a bare CSI).
export function matchGlobalAction(ev, overrides = readAppKeyOverrides()) {
  return CATALOG.some((a) => a.group === "Global" && matchAction(a.id, ev, overrides));
}
