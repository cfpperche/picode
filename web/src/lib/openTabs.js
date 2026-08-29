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

export function moveTab(ids, fromId, toId) {
  const list = ids || [];
  const from = list.indexOf(fromId);
  const to = list.indexOf(toId);
  if (from < 0 || to < 0 || from === to) return list;
  const next = list.slice();
  next.splice(from, 1);
  next.splice(to, 0, fromId);
  return next;
}

export function filterOpenTabs(saved, exists) {
  const ids = (saved.ids || []).filter(exists);
  const selected = saved.selected && ids.includes(saved.selected) ? saved.selected : (ids[0] || null);
  return { ids, selected };
}
