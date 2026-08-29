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

export function effectiveKeys(action, user) {
  if (user && Object.prototype.hasOwnProperty.call(user, action.id)) return user[action.id] || [];
  return action.defaults || [];
}

export function isOverride(action, user) {
  return !!(user && Object.prototype.hasOwnProperty.call(user, action.id));
}

export function matchKeys(action, user, q) {
  const needle = (q || "").trim().toLowerCase();
  if (!needle) return true;
  const keys = effectiveKeys(action, user).join(" ");
  return (action.label + " " + action.group + " " + action.id + " " + keys).toLowerCase().includes(needle);
}
