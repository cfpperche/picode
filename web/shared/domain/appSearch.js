// Client-side filter for the apps host's list blocks (ADR-0036 amendment,
// 2026-09-01). Host-generic, not Inbox-specific: any app's "list" blocks get
// filtered the same way. No wire field, no apiVersion bump — everything
// matched against is already delivered per row by appPrimitives.js.
//
// Unlike git graph's search (gitgraphSearch.js, ADR-0038) this filters
// (hides) non-matches instead of dimming them. Git graph can't hide rows —
// they're positional in a drawn graph — but an app's list has no such
// constraint, and hiding is what a "filter" is asked to do.

function str(v) {
  return typeof v === "string" ? v : "";
}

// matchesItem: substring, case-insensitive, over every text field a list
// row can show. An empty/blank query always matches.
export function matchesItem(item, query) {
  const q = String(query || "").trim().toLowerCase();
  if (!q) return true;
  const hay = [str(item?.title), str(item?.subtitle), str(item?.badge), ...(Array.isArray(item?.meta) ? item.meta : [])]
    .join(" ")
    .toLowerCase();
  return hay.includes(q);
}

// filterListBlocks: only "list" blocks are touched — detail/form/actions
// blocks (e.g. Inbox's "Clear all done" bulk action) pass through
// untouched regardless of query. A list block left with zero items after
// filtering is dropped entirely, not kept as an empty section.
export function filterListBlocks(blocks, query) {
  const q = String(query || "").trim();
  if (!q) return Array.isArray(blocks) ? blocks : [];
  const out = [];
  for (const b of Array.isArray(blocks) ? blocks : []) {
    if (!b || b.type !== "list") {
      if (b) out.push(b);
      continue;
    }
    const groupMatch = b.collapsible && matchesItem({ title: b.title }, query);
    const items = (Array.isArray(b.items) ? b.items : []).filter((it) => groupMatch || matchesItem(it, query));
    if (items.length) out.push({ ...b, items });
  }
  return out;
}

// countListItems: total items across "list" blocks only.
export function countListItems(blocks) {
  let n = 0;
  for (const b of Array.isArray(blocks) ? blocks : []) {
    if (b && b.type === "list" && Array.isArray(b.items)) n += b.items.length;
  }
  return n;
}
