// Browser-owned chords — which side of the platform owns a key.
//
// A web page never sees the browser's reserved shortcuts: Ctrl+T opens a
// tab in the browser process before any keydown reaches the page, so no
// in-page preventDefault can stop it. The cancelable class (Ctrl+F, Ctrl+P,
// Alt+Left, F3, …) does fire keydown and is the page's to cancel — inside
// a terminal pane xterm.js already cancels and encodes every chord it maps
// (termKeys.js), so the reserved set is the only real gap for a guest CLI
// like Codex's Ctrl+T. The Keyboard Lock API (fullscreen only, Chromium)
// hands the reserved set to the page; focus mode locks it there
// (desktop/src/lib/useFocusMode.js, focusMode.js).
//
// Chords use the app's format (piKey.js fromEvent): modifiers in order
// ctrl, shift, alt, super, then the lowercased key — "ctrl+pageUp" keeps
// the SPECIAL map's capital U. Pure data in / data out so node:test drives
// the whole table without a browser.

// The reserved set on Chromium (Chrome/Edge/Opera — the browsers this
// project is served to, docs-site/guide/keyboard.md): the browser consumes
// these before the page and no script can observe or cancel them. The
// ctrl rows are the Windows/Linux truth; macOS reserves the Cmd
// equivalents instead (its Ctrl+T is an ordinary chord a page CAN see),
// so the same rows exist under super — which is what piKey fromEvent
// reports for metaKey. Whether Keyboard Lock can capture the Cmd rows on
// macOS follows Chrome's implementation; the guard's job (never default
// onto a dead chord) holds either way.
export const RESERVED_CHORDS = [
  { chord: "ctrl+t", browser: "new tab" },
  { chord: "ctrl+shift+t", browser: "reopen closed tab" },
  { chord: "ctrl+n", browser: "new window" },
  { chord: "ctrl+shift+n", browser: "new incognito window" },
  { chord: "ctrl+w", browser: "close tab" },
  { chord: "ctrl+shift+w", browser: "close window" },
  { chord: "ctrl+tab", browser: "next tab" },
  { chord: "ctrl+shift+tab", browser: "previous tab" },
  { chord: "ctrl+pageUp", browser: "previous tab" },
  { chord: "ctrl+pageDown", browser: "next tab" },
  { chord: "ctrl+1", browser: "first tab" },
  { chord: "ctrl+2", browser: "tab 2" },
  { chord: "ctrl+3", browser: "tab 3" },
  { chord: "ctrl+4", browser: "tab 4" },
  { chord: "ctrl+5", browser: "tab 5" },
  { chord: "ctrl+6", browser: "tab 6" },
  { chord: "ctrl+7", browser: "tab 7" },
  { chord: "ctrl+8", browser: "tab 8" },
  { chord: "ctrl+9", browser: "last tab" },
  { chord: "super+t", browser: "new tab (macOS)" },
  { chord: "super+shift+t", browser: "reopen closed tab (macOS)" },
  { chord: "super+n", browser: "new window (macOS)" },
  { chord: "super+shift+n", browser: "new incognito window (macOS)" },
  { chord: "super+w", browser: "close tab (macOS)" },
  { chord: "super+shift+w", browser: "close window (macOS)" },
];

const RESERVED = new Set(RESERVED_CHORDS.map((r) => r.chord));

// True when the browser, not the page, owns this chord in a normal window.
export function isReservedChord(chord) {
  return RESERVED.has(chord);
}

// The first reserved chord in a list, or null — the guard used by the
// app-keys tests: a default that the browser would eat can never work
// (appKeys.js deliberately stays off this set).
export function firstReservedChord(chords) {
  if (!Array.isArray(chords)) return null;
  for (const c of chords) if (RESERVED.has(c)) return c;
  return null;
}

// What the browser shows for a reserved chord, for honest UI copy.
export function reservedBrowserAction(chord) {
  const row = RESERVED_CHORDS.find((r) => r.chord === chord);
  return row ? row.browser : null;
}
