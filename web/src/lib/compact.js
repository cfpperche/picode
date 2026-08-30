const KEY = "picode-compacting";
// Entries older than this are stale (compaction died with the page); drop on load.
const MAX_AGE = 30 * 60 * 1000;

export function readCompacting(store = localStorage) {
  try {
    const raw = JSON.parse((store && store.getItem(KEY)) || "{}");
    const now = Date.now();
    const out = {};
    for (const [k, v] of Object.entries(raw)) {
      if (typeof v === "number" && v > 0 && now - v < MAX_AGE) out[k] = v;
    }
    return out;
  } catch {
    return {};
  }
}

export function writeCompacting(map, store = localStorage) {
  try {
    store && store.setItem(KEY, JSON.stringify(map));
  } catch {
    /* storage unavailable */
  }
}
