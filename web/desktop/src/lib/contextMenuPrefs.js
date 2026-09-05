const KEY = "picode-ctxmenu";

export const CTX_MODIFIERS = ["shift", "alt", "ctrl"];

export function defaultContextMenuPrefs() {
  return { bypassModifier: "shift" };
}

export function readContextMenuPrefs() {
  const d = defaultContextMenuPrefs();
  try {
    const j = JSON.parse(localStorage.getItem(KEY) || "{}");
    if (CTX_MODIFIERS.includes(j.bypassModifier)) d.bypassModifier = j.bypassModifier;
  } catch { /* ignore */ }
  return d;
}

export function persistContextMenuPrefs(prefs) {
  const next = { ...defaultContextMenuPrefs(), ...prefs };
  localStorage.setItem(KEY, JSON.stringify(next));
  return next;
}

export function modifierHeld(modifier, ev) {
  if (modifier === "alt") return !!ev.altKey;
  if (modifier === "ctrl") return !!(ev.ctrlKey || ev.metaKey);
  return !!ev.shiftKey;
}
