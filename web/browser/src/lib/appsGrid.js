// Ordering and filtering for the Apps sidebar tab. The server returns
// manifests in registration order (internal/apps Registry.All); the grid
// is presentation, so it sorts alphabetically here — same criterion the
// sidebar uses for free agents (localeCompare, base sensitivity) — and
// filters by the visible name as you type. Pure: never mutates the input.
export function visibleApps(list, q) {
  const query = (q || "").trim().toLowerCase();
  const sorted = [...(list || [])].sort((a, b) =>
    String(a?.name || "").localeCompare(String(b?.name || ""), undefined, { sensitivity: "base" }));
  if (!query) return sorted;
  return sorted.filter((a) => String(a?.name || "").toLowerCase().includes(query));
}
