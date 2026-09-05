const KEY = "picode-app-keys";

export function readAppKeyOverrides() {
  try { return JSON.parse(localStorage.getItem(KEY) || "{}"); } catch { return {}; }
}

export function persistAppKeyOverride(actionId, keys) {
  const next = { ...readAppKeyOverrides() };
  if (keys === null) delete next[actionId];
  else next[actionId] = keys;
  localStorage.setItem(KEY, JSON.stringify(next));
  return next;
}
