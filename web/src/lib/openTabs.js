const KEY = "picode-tabs";

export function readOpenTabs() {
  try {
    const j = JSON.parse(localStorage.getItem(KEY) || "null");
    if (!j || !Array.isArray(j.ids)) return { ids: [], selected: null };
    const ids = j.ids.map((x) => String(x || "")).filter(Boolean);
    const selected = j.selected && ids.includes(j.selected) ? j.selected : (ids[0] || null);
    return { ids, selected };
  } catch {
    return { ids: [], selected: null };
  }
}

export function writeOpenTabs(ids, selected) {
  const clean = [...new Set((ids || []).map((x) => String(x || "")).filter(Boolean))];
  const sel = selected && clean.includes(selected) ? selected : (clean[0] || null);
  localStorage.setItem(KEY, JSON.stringify({ ids: clean, selected: sel }));
}

export function filterOpenTabs(saved, exists) {
  const ids = (saved.ids || []).filter(exists);
  const selected = saved.selected && ids.includes(saved.selected) ? saved.selected : (ids[0] || null);
  return { ids, selected };
}
