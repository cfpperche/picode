// Ordering and filtering for the Apps sidebar tab. With no saved order the
// grid is alphabetical (localeCompare, base sensitivity). A saved id list
// (PUT /api/apps/order) wins; apps that are not in it append, still by name.
// Pure: never mutates the input.

function byName(a, b) {
  return String(a?.name || "").localeCompare(String(b?.name || ""), undefined, { sensitivity: "base" });
}

export function orderApps(list, ids) {
  const rows = [...(list || [])];
  if (!ids || !ids.length) return [...rows].sort(byName);
  const byId = new Map(rows.map((a) => [a.id, a]));
  const out = [];
  const seen = new Set();
  for (const id of ids) {
    const row = byId.get(id);
    if (!row || seen.has(id)) continue;
    seen.add(id);
    out.push(row);
  }
  const rest = rows.filter((a) => !seen.has(a.id)).sort(byName);
  return out.concat(rest);
}

export function visibleApps(list, q, ids) {
  const query = (q || "").trim().toLowerCase();
  const sorted = orderApps(list, ids);
  if (!query) return sorted;
  return sorted.filter((a) => String(a?.name || "").toLowerCase().includes(query));
}
