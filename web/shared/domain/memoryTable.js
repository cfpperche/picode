// The Memory pane's table: what a column shows, how a row sorts, and what the
// index budget means (2026-09-20).
//
// The list this replaces showed a title, a kind and a truncated description.
// The files carry more, and the server's survey adds what no single file can
// answer — whether the index the CLI loads at session start still names this
// memory, how many other memories cite it, and which of its own citations
// lead nowhere. Those are the columns worth having; everything here is pure,
// so the sorting and the health rules are tested without a DOM.

export const MEMORY_COLUMNS = [
  { id: "memory", label: "Memory", sort: "title", grow: true },
  { id: "citedBy", label: "Cited by", sort: "citedBy", numeric: true, needs: "survey" },
  { id: "bytes", label: "Size", sort: "bytes", numeric: true },
  { id: "modified", label: "Modified", sort: "modified", numeric: true },
  { id: "health", label: "Health" },
];

// columnsFor drops what a CLI cannot answer. A store with no index convention
// has no citation graph, and a zero there would read as a finding.
export function columnsFor(items = []) {
  const survey = items.some((i) => i && i.citedBy !== undefined && i.citedBy !== null);
  return MEMORY_COLUMNS.filter((c) => c.needs !== "survey" || survey);
}

// The index is the file the CLI loads at the start of every session, and it
// reads only so much of it. Past the limit the rest is dropped silently, so
// the meter is the only place a reader can see it coming.
export function indexBudget(index) {
  if (!index || !index.exists) return null;
  const byBytes = index.byteLimit ? index.bytes / index.byteLimit : 0;
  const byLines = index.lineLimit ? index.lines / index.lineLimit : 0;
  const used = Math.max(byBytes, byLines);
  return {
    used,
    percent: Math.min(100, Math.round(used * 100)),
    over: used >= 1,
    near: used >= 0.8 && used < 1,
    lines: index.lines,
    bytes: index.bytes,
    lineLimit: index.lineLimit,
    byteLimit: index.byteLimit,
    dangling: index.dangling || [],
    // Which limit bites first, so the line says what to shorten.
    limit: byBytes >= byLines ? "bytes" : "lines",
  };
}

// health returns the chips a row wears. Each one is a thing to act on, never
// decoration: a memory the index does not name is on disk and out of reach,
// and a citation with no file behind it is a dead end the reader only finds
// by following it.
//
// "Nothing cites this one" is a finding too, but it is already the zero in
// the Cited by column — a chip repeating it would cost the width of a second
// chip to say the same thing twice.
export function health(item) {
  if (!item || item.index) return [];
  const out = [];
  if (item.indexed === false) {
    out.push({ id: "unindexed", label: "Not indexed", tone: "warn", detail: "The index does not name this file, so the CLI does not load it at the start of a session." });
  }
  if (item.broken && item.broken.length) {
    out.push({ id: "broken", label: item.broken.length === 1 ? "1 broken link" : item.broken.length + " broken links", tone: "warn", detail: "Points at " + item.broken.join(", ") + ", which no memory here answers to." });
  }
  return out;
}

// facets counts the kinds present, so the filter chips can say how many each
// one holds instead of offering an empty filter.
export function facets(items = []) {
  const counts = new Map();
  for (const item of items) {
    if (item.index) continue;
    const kind = item.kind || "";
    if (!kind) continue;
    counts.set(kind, (counts.get(kind) || 0) + 1);
  }
  return [...counts.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])).map(([kind, count]) => ({ kind, count }));
}

// sortItems keeps the index pinned to the top whatever the sort: it is the
// file the CLI actually loads, not one row among many.
export function sortItems(items = [], { key = "modified", dir = "desc" } = {}) {
  const rows = [...items];
  const sign = dir === "asc" ? 1 : -1;
  rows.sort((a, b) => {
    if (a.index !== b.index) return a.index ? -1 : 1;
    const av = sortValue(a, key);
    const bv = sortValue(b, key);
    if (av === bv) return (a.id || "").localeCompare(b.id || "");
    if (av === undefined || av === null) return 1;
    if (bv === undefined || bv === null) return -1;
    return av < bv ? -sign : sign;
  });
  return rows;
}

function sortValue(item, key) {
  if (key === "title") return (item.title || item.id || "").toLowerCase();
  if (key === "citedBy") return item.citedBy ?? -1;
  if (key === "bytes") return item.bytes ?? 0;
  if (key === "modified") return item.modified || "";
  return "";
}

// filterItems applies the text filter and the kind facet together. The index
// row survives a kind filter — hiding the file the CLI loads would be a lie
// about what is in the store.
export function filterItems(items = [], { text = "", kinds = [] } = {}) {
  const needle = text.trim().toLowerCase();
  const wanted = new Set(kinds);
  return items.filter((item) => {
    if (item.index) return !needle || matches(item, needle);
    if (wanted.size && !wanted.has(item.kind || "")) return false;
    return !needle || matches(item, needle);
  });
}

function matches(item, needle) {
  return [item.title, item.summary, item.id, item.kind, ...(item.broken || [])]
    .filter(Boolean)
    .join(" ")
    .toLowerCase()
    .includes(needle);
}

// blastRadius says what a delete costs in this store, before it happens:
// how many citations in other memories it breaks, and how many rows in the
// index will be left pointing at nothing. Naming it is the difference
// between a confirm and a dare.
export function blastRadius(count, citedTotal = 0, stillIndexed = 0) {
  const parts = [];
  if (citedTotal) {
    parts.push(citedTotal === 1 ? "1 link in another memory will point at nothing" : citedTotal + " links in other memories will point at nothing");
  }
  if (stillIndexed) {
    parts.push(stillIndexed === 1 ? "the index still names 1 of them" : "the index still names " + stillIndexed + " of them");
  }
  if (!parts.length) return "Nothing else points at " + (count === 1 ? "it" : "them") + ".";
  const line = parts.join("; ") + ".";
  return line.charAt(0).toUpperCase() + line.slice(1);
}

// bytesLabel keeps a size in the row without a unit conversation.
export function bytesLabel(bytes) {
  if (!bytes && bytes !== 0) return "";
  if (bytes < 1024) return bytes + " B";
  return Math.round(bytes / 1024) + " KB";
}

// agoLabel is a length of time, not a date: the question a memory raises is
// "how stale is this", and the exact timestamp is a title attribute.
export function agoLabel(iso, now = Date.now()) {
  if (!iso) return "";
  const then = Date.parse(iso);
  if (Number.isNaN(then)) return "";
  const days = Math.floor((now - then) / 86400000);
  if (days <= 0) return "today";
  if (days === 1) return "yesterday";
  if (days < 30) return days + "d";
  const months = Math.floor(days / 30);
  if (months < 12) return months + "mo";
  return Math.floor(days / 365) + "y";
}
