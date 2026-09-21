const SPECIAL = {
  Backspace: "backspace",
  Enter: "enter",
  Escape: "escape",
  Tab: "tab",
  " ": "space",
  ArrowUp: "up",
  ArrowDown: "down",
  ArrowLeft: "left",
  ArrowRight: "right",
  Home: "home",
  End: "end",
  PageUp: "pageUp",
  PageDown: "pageDown",
  Delete: "delete",
  Insert: "insert",
};

const MOD_ONLY = { Control: 1, Shift: 1, Alt: 1, Meta: 1 };

export function fromEvent(ev) {
  if (!ev || MOD_ONLY[ev.key]) return null;
  const parts = [];
  if (ev.ctrlKey) parts.push("ctrl");
  if (ev.shiftKey) parts.push("shift");
  if (ev.altKey) parts.push("alt");
  if (ev.metaKey && !ev.ctrlKey) parts.push("super");
  let key = SPECIAL[ev.key];
  if (!key) {
    if (ev.key.length === 1) key = ev.key.toLowerCase();
    else if (/^f\d{1,2}$/i.test(ev.key)) key = ev.key.toLowerCase();
    else return null;
  }
  if (!ev.ctrlKey && !ev.altKey && !ev.metaKey && key.length === 1) return null;
  parts.push(key);
  return parts.join("+");
}

// The default Pi binds on a platform. Nine of pi's own actions carry a
// different binding on Windows or WSL (the catalog's alt map, read out of
// pi's docs/keybindings.md on 2026-09-21); an alternate is the whole binding
// there, not an addition to the base list, and a declared empty list means pi
// binds nothing.
export function defaultKeys(action, platform) {
  const alt = action.alt || {};
  if (platform && Object.prototype.hasOwnProperty.call(alt, platform)) return alt[platform] || [];
  return action.defaults || [];
}

export function effectiveKeys(action, user, platform) {
  if (user && Object.prototype.hasOwnProperty.call(user, action.id)) return user[action.id] || [];
  return defaultKeys(action, platform);
}

export function isOverride(action, user) {
  return !!(user && Object.prototype.hasOwnProperty.call(user, action.id));
}

// The other platforms' bindings for this action, for the line under a row that
// has them. Excludes the platform the pane is reading, whose binding is the
// one already shown.
export function platformAlternates(action, platform) {
  const out = [];
  for (const [name, keys] of Object.entries(action.alt || {})) {
    if (name !== platform) out.push({ platform: name, keys: keys || [] });
  }
  return out;
}

export function matchKeys(action, user, q, platform) {
  const needle = (q || "").trim().toLowerCase();
  if (!needle) return true;
  const keys = effectiveKeys(action, user, platform).join(" ");
  return (action.label + " " + action.group + " " + action.id + " " + keys).toLowerCase().includes(needle);
}
