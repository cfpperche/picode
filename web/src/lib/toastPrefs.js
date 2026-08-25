const KEY = "picode-toast";

export const TOAST_POSITIONS = [
  "top-left",
  "top-center",
  "top-right",
  "bottom-left",
  "bottom-center",
  "bottom-right",
];

export function defaultToastPrefs() {
  return {
    position: "top-right",
    expand: false,
    richColors: false,
    closeButton: true,
    duration: 4000,
    visibleToasts: 3,
  };
}

export function readToastPrefs() {
  const d = defaultToastPrefs();
  try {
    const j = JSON.parse(localStorage.getItem(KEY) || "{}");
    if (TOAST_POSITIONS.includes(j.position)) d.position = j.position;
    d.expand = !!j.expand;
    d.richColors = !!j.richColors;
    d.closeButton = j.closeButton !== false;
    const dur = Number(j.duration);
    if (Number.isFinite(dur)) d.duration = Math.min(15000, Math.max(1500, dur));
    const n = Number(j.visibleToasts);
    if (Number.isFinite(n)) d.visibleToasts = Math.min(5, Math.max(1, n));
  } catch { /* ignore */ }
  return d;
}

export function persistToastPrefs(prefs) {
  const next = { ...defaultToastPrefs(), ...prefs };
  localStorage.setItem(KEY, JSON.stringify(next));
  if (typeof window !== "undefined") window.dispatchEvent(new Event("picode-toast-prefs"));
  return next;
}
