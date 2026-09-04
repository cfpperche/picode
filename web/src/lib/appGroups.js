// Host presentation state, scoped by app + the block's stable identity.
// Search starts with matching groups open but never overwrites saved folds.
const STORAGE_KEY = "picode-app-groups-v1";
const queryKey = (query) => String(query || "").trim().toLowerCase();

export function readGroupPreferences(storage) {
  try {
    const raw = JSON.parse((storage || globalThis.localStorage).getItem(STORAGE_KEY));
    if (!raw || typeof raw !== "object" || Array.isArray(raw)) return {};
    return Object.fromEntries(Object.entries(raw).filter(([, value]) => typeof value === "boolean"));
  } catch { return {}; }
}

export function writeGroupPreferences(saved, storage) {
  try { (storage || globalThis.localStorage).setItem(STORAGE_KEY, JSON.stringify(saved)); }
  catch { /* Folding still works when storage is unavailable. */ }
}

export function groupIsOpen(state, id, query) {
  const q = queryKey(query);
  if (!q) return state.saved[id] === true;
  const values = state.search.query === q ? state.search.values : {};
  return values[id] !== false;
}

export function resetGroupSearch(state, query) {
  const q = queryKey(query);
  if (state.search.query === q) return state;
  return { ...state, search: { query: q, values: {} } };
}

export function toggleGroup(state, id, query) {
  const open = !groupIsOpen(state, id, query);
  const q = queryKey(query);
  if (!q) return { ...state, saved: { ...state.saved, [id]: open } };
  const values = state.search.query === q ? state.search.values : {};
  return { ...state, search: { query: q, values: { ...values, [id]: open } } };
}
