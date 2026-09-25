const PREFS_KEY = "picode-term-prefs";

export const TERM_SIZE_MIN = 11;
export const TERM_SIZE_MAX = 22;
export const TERM_SIZE_DEFAULT = 14;
export const TERM_LINE_MIN = 1;
export const TERM_LINE_MAX = 1.8;
export const TERM_TRACK_MIN = -2;
export const TERM_TRACK_MAX = 4;
export const TERM_PAD_MIN = 0;
export const TERM_PAD_MAX = 24;
export const TERM_SCROLL_MIN = 1000;
export const TERM_SCROLL_MAX = 50000;

export const TERM_FONTS = [
  { id: "jetbrains", label: "JetBrains Mono", css: '"JetBrains Mono", ui-monospace, "SF Mono", "Cascadia Code", Menlo, monospace' },
  { id: "fira", label: "Fira Code", css: '"Fira Code", ui-monospace, "SF Mono", Menlo, monospace' },
  { id: "ui", label: "UI mono", css: 'ui-monospace, "SF Mono", "Cascadia Code", Menlo, monospace' },
  { id: "system", label: "System", css: "monospace" },
];

export const TERM_CURSORS = [
  { id: "block", label: "Block" },
  { id: "bar", label: "Bar" },
  { id: "underline", label: "Underline" },
];

export const TERM_NEWLINES = [
  { id: "shift-enter", label: "Shift+Enter" },
  { id: "ctrl-enter", label: "Ctrl+Enter" },
  { id: "alt-enter", label: "Alt+Enter" },
];

export function defaultTermPrefs() {
  return {
    theme: "dark",
    font: "jetbrains",
    fontSize: TERM_SIZE_DEFAULT,
    lineHeight: 1,
    letterSpacing: 0,
    cursorStyle: "block",
    cursorBlink: true,
    scrollback: 10000,
    padding: 8,
    newlineKey: "shift-enter",
    copyIfSelection: false,
  };
}

function clampInt(n, min, max, fallback) {
  const v = Math.round(Number(n));
  if (!Number.isFinite(v)) return fallback;
  return Math.min(max, Math.max(min, v));
}

function clampNum(n, min, max, fallback) {
  const v = Number(n);
  if (!Number.isFinite(v)) return fallback;
  return Math.min(max, Math.max(min, v));
}

function normalize(raw) {
  const d = defaultTermPrefs();
  const src = raw && typeof raw === "object" ? raw : {};
  d.theme = src.theme === "light" ? "light" : "dark";
  d.font = TERM_FONTS.some((f) => f.id === src.font) ? src.font : d.font;
  d.fontSize = clampInt(src.fontSize, TERM_SIZE_MIN, TERM_SIZE_MAX, d.fontSize);
  d.lineHeight = Math.round(clampNum(src.lineHeight, TERM_LINE_MIN, TERM_LINE_MAX, d.lineHeight) * 100) / 100;
  d.letterSpacing = clampNum(src.letterSpacing, TERM_TRACK_MIN, TERM_TRACK_MAX, d.letterSpacing);
  d.cursorStyle = TERM_CURSORS.some((c) => c.id === src.cursorStyle) ? src.cursorStyle : d.cursorStyle;
  d.cursorBlink = src.cursorBlink !== false;
  d.scrollback = clampInt(src.scrollback, TERM_SCROLL_MIN, TERM_SCROLL_MAX, d.scrollback);
  d.padding = clampInt(src.padding, TERM_PAD_MIN, TERM_PAD_MAX, d.padding);
  d.newlineKey = TERM_NEWLINES.some((k) => k.id === src.newlineKey) ? src.newlineKey : d.newlineKey;
  d.copyIfSelection = src.copyIfSelection === true;
  return d;
}

export function readTermPrefs() {
  let parsed = {};
  try {
    parsed = JSON.parse((typeof localStorage !== "undefined" && localStorage.getItem(PREFS_KEY)) || "{}") || {};
  } catch { parsed = {}; }
  return normalize(parsed);
}

export function readTermTheme() {
  return readTermPrefs().theme;
}

export function readTermFontSize() {
  return readTermPrefs().fontSize;
}

export function termFontFamily(id) {
  const f = TERM_FONTS.find((x) => x.id === id) || TERM_FONTS[0];
  return f.css;
}

// xterm paints the overview ruler's outline with `overviewRulerBorder` at
// full alpha, and that outline is the ruler's whole column while the ruler
// is one pixel wide (xtermOptions below) — a stray white line at the right
// edge of every terminal whose xterm is not on the alternate screen, right
// beside the scrollbar the stylesheets hide. Transparent leaves the ruler
// canvas and the find addon's marks untouched (term-scrollbar.test.mjs).
const RULER_BORDER = "#00000000";

export function xtermTheme(mode) {
  if (mode === "light") {
    return {
      background: "#ffffff",
      foreground: "#16181d",
      cursor: "#2f6fed",
      selectionBackground: "#c9d7f5",
      overviewRulerBorder: RULER_BORDER,
    };
  }
  return {
    background: "#0e0e11",
    foreground: "#ececf1",
    cursor: "#7c8cf8",
    selectionBackground: "#33467c",
    overviewRulerBorder: RULER_BORDER,
  };
}

export function xtermOptions() {
  const p = readTermPrefs();
  return {
    cursorBlink: p.cursorBlink,
    cursorStyle: p.cursorStyle,
    fontSize: p.fontSize,
    fontFamily: termFontFamily(p.font),
    lineHeight: p.lineHeight,
    letterSpacing: p.letterSpacing,
    theme: xtermTheme(p.theme),
    scrollback: p.scrollback,
    rightClickSelectsWord: true,
    // @xterm/addon-search paints its match decorations through xterm's
    // proposed decoration API and throws without this. Nothing else here
    // uses a proposed API, and the flag changes no behaviour on its own.
    allowProposedApi: true,
    // FitAddon reserves 14px for an overview ruler we never render (no
    // decorations use it) — shrink it so the right edge isn't a dead gutter.
    // This value is ALSO xterm's own scrollbar width (`verticalScrollbarSize:
    // overviewRuler?.width || 14` in its Scrollable) — a 1px bar that the
    // stylesheets hide, because a tmux client has no scrollback to draw:
    // the reader's bar is tmux's, or the TUI's own (term-scrollbar.test.mjs).
    overviewRuler: { width: 1 },
  };
}

export function applyXtermOptions(term) {
  if (!term) return;
  const o = xtermOptions();
  term.options.cursorBlink = o.cursorBlink;
  term.options.cursorStyle = o.cursorStyle;
  term.options.fontSize = o.fontSize;
  term.options.fontFamily = o.fontFamily;
  term.options.lineHeight = o.lineHeight;
  term.options.letterSpacing = o.letterSpacing;
  term.options.theme = o.theme;
  term.options.scrollback = o.scrollback;
}

export function applyTermTheme(mode) {
  if (typeof document === "undefined") return;
  const v = mode === "light" ? "light" : "dark";
  document.documentElement.dataset.termTheme = v;
}

export function applyTermChrome() {
  if (typeof document === "undefined") return;
  const p = readTermPrefs();
  applyTermTheme(p.theme);
  document.documentElement.style.setProperty("--term-pad", p.padding + "px");
}

function emitTerm() {
  applyTermChrome();
  if (typeof window !== "undefined") window.dispatchEvent(new Event("picode-term-theme"));
}

export function persistTermPrefs(patch) {
  const next = normalize({ ...readTermPrefs(), ...patch });
  if (typeof localStorage !== "undefined") {
    localStorage.setItem(PREFS_KEY, JSON.stringify(next));
  }
  emitTerm();
  return next;
}

export function persistTermTheme(mode) {
  return persistTermPrefs({ theme: mode === "light" ? "light" : "dark" }).theme;
}

export function persistTermFontSize(n) {
  return persistTermPrefs({ fontSize: n }).fontSize;
}

export function bumpTermFontSize(delta) {
  if (delta === 0) return persistTermFontSize(TERM_SIZE_DEFAULT);
  return persistTermFontSize(readTermFontSize() + delta);
}
