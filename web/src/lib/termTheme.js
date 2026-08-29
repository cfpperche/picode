const THEME_KEY = "picode-term-theme";
const SIZE_KEY = "picode-term-size";
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

export const TERM_FONT = TERM_FONTS[0].css;

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

function readLegacy() {
  const o = {};
  const theme = typeof localStorage !== "undefined" ? localStorage.getItem(THEME_KEY) : null;
  const size = typeof localStorage !== "undefined" ? localStorage.getItem(SIZE_KEY) : null;
  if (theme === "light" || theme === "dark") o.theme = theme;
  const n = parseInt(size || "", 10);
  if (Number.isFinite(n)) o.fontSize = n;
  return o;
}

export function readTermPrefs() {
  let parsed = {};
  try {
    parsed = JSON.parse((typeof localStorage !== "undefined" && localStorage.getItem(PREFS_KEY)) || "{}") || {};
  } catch { parsed = {}; }
  return normalize({ ...readLegacy(), ...parsed });
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

export function xtermTheme(mode) {
  if (mode === "light") {
    return {
      background: "#ffffff",
      foreground: "#16181d",
      cursor: "#2f6fed",
      selectionBackground: "#c9d7f5",
    };
  }
  return {
    background: "#0e0e11",
    foreground: "#ececf1",
    cursor: "#7c8cf8",
    selectionBackground: "#33467c",
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
    localStorage.setItem(THEME_KEY, next.theme);
    localStorage.setItem(SIZE_KEY, String(next.fontSize));
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
