const KEY = "picode-layout";
export const LETTERBOX_MIN = 24;

// How tall the WebKit standalone letterbox is: a status-bar-sized strip
// parked below the layout viewport (WebKit 313800). Gaps of 24px or less
// are noise. With an opaque status bar the layout viewport already starts
// below the bar, so screen-minus-inner is bar + strip; the strip is
// status-bar-sized, and if the extra beyond one bar is noise there is no
// strip (a translucent bar overlays, so the gap IS the strip).
export function letterboxPx(screenHeight, innerHeight, opts = {}) {
  const gap = Math.round(Number(screenHeight) - Number(innerHeight));
  if (gap <= LETTERBOX_MIN) return 0;
  if (opts.opaqueStatusBar) {
    if (gap <= LETTERBOX_MIN * 2) return 0;
    return Math.round(gap / 2);
  }
  return gap;
}

// Where the bottom bar's buttons sit and how tall the bar is — the two
// dials a phone owner needs to fit the shell to their screen. "auto"
// docks the bar into the letterbox on a home-screen iPhone (edge);
// "low" keeps the buttons in the layout viewport if that clips.
// Unknown stored values fall back to the defaults.
export const BOTTOM_BAR_MODES = ["auto", "low", "edge"];
export const BAR_HEIGHTS = [48, 56, 64];

export function defaultLayoutPrefs() {
  return { bottomBar: "auto", barHeight: 56 };
}

export function readLayoutPrefs() {
  const d = defaultLayoutPrefs();
  try {
    const j = JSON.parse(localStorage.getItem(KEY) || "{}");
    if (BOTTOM_BAR_MODES.includes(j.bottomBar)) d.bottomBar = j.bottomBar;
    const h = Number(j.barHeight);
    if (BAR_HEIGHTS.includes(h)) d.barHeight = h;
  } catch { /* ignore */ }
  return d;
}

export function persistLayoutPrefs(prefs) {
  const next = { ...defaultLayoutPrefs(), ...prefs };
  localStorage.setItem(KEY, JSON.stringify(next));
  applyLayoutPrefs(next);
  return next;
}

// "auto" resolves against the bootstrap's measurement (index.html): a
// letterboxed standalone install docks to the screen edge, everything
// else stays the default (centered in the 56px bar).
export function resolveBottomBar(mode) {
  if (mode === "low" || mode === "edge") return mode;
  const root = typeof document !== "undefined" ? document.documentElement : null;
  return root && root.dataset.standalone === "1" ? "edge" : "default";
}

export function applyLayoutPrefs(prefs) {
  const root = document.documentElement;
  root.dataset.bottombar = resolveBottomBar(prefs.bottomBar);
  root.style.setProperty("--m-nav", prefs.barHeight + "px");
}
