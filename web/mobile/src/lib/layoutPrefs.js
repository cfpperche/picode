const KEY = "picode-layout";

// Where the bottom bar's buttons sit and how tall the bar is — the two
// dials a phone owner needs to fit the shell to their screen. "auto" keeps
// the platform-adaptive behavior (low when iOS letterboxes the standalone
// shell, WebKit 313800); "edge" extends the shell into the letterbox's
// unreachable strip, which some iOS versions paint and some clip — user
// choice, user risk. Unknown stored values fall back to the defaults.
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
// letterboxed standalone install gets "low", everything else "default".
export function resolveBottomBar(mode) {
  if (mode === "low" || mode === "edge") return mode;
  const root = typeof document !== "undefined" ? document.documentElement : null;
  return root && root.dataset.standalone === "1" ? "low" : "default";
}

export function applyLayoutPrefs(prefs) {
  const root = document.documentElement;
  root.dataset.bottombar = resolveBottomBar(prefs.bottomBar);
  root.style.setProperty("--m-nav", prefs.barHeight + "px");
}
