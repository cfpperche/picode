const KEY = "picode-providers-recent";
const MAX = 20;

export function readRecents() {
  try {
    const a = JSON.parse(localStorage.getItem(KEY) || "[]");
    return Array.isArray(a) ? a.filter((id) => typeof id === "string") : [];
  } catch {
    return [];
  }
}

function write(ids) {
  localStorage.setItem(KEY, JSON.stringify(ids.slice(0, MAX)));
}

export function pushRecent(id) {
  if (!id) return readRecents();
  const next = [id, ...readRecents().filter((x) => x !== id)];
  write(next);
  return next;
}

export function rememberProviders(ids) {
  let next = readRecents();
  for (const id of ids || []) {
    if (id && !next.includes(id)) next = [...next, id];
  }
  write(next);
  return next;
}

export function removeRecent(id) {
  const next = readRecents().filter((x) => x !== id);
  write(next);
  return next;
}

export function clearRecents() {
  write([]);
  return [];
}
