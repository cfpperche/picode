// Splitting and filtering for the folder picker's address bar: the typed
// text is a directory to list plus a trailing fragment that filters it.

// splitPathQuery("/home/goat/pi") → { dir: "/home/goat", q: "pi" }.
// A trailing slash means "list this, no filter". Windows separators are
// honored for splitting; expansion of ~ and C:\ stays the backend's job.
export function splitPathQuery(input) {
  const raw = String(input || "").trim();
  if (!raw) return { dir: "", q: "" };
  const norm = raw.replace(/\\/g, "/");
  if (norm === "/" || norm === "~" || norm.endsWith("/")) return { dir: raw, q: "" };
  const i = norm.lastIndexOf("/");
  if (i < 0) return { dir: raw, q: "" };
  return { dir: raw.slice(0, i) || "/", q: raw.slice(i + 1) };
}

// Case-insensitive: prefix matches first, then includes; stable within groups.
export function filterDirs(dirs, q) {
  const query = String(q || "").toLowerCase();
  if (!query) return dirs || [];
  const pre = [];
  const inc = [];
  for (const d of dirs || []) {
    const name = String(d.name || "").toLowerCase();
    if (name.startsWith(query)) pre.push(d);
    else if (name.includes(query)) inc.push(d);
  }
  return [...pre, ...inc];
}

// Exact (case-insensitive) beats prefix beats includes; null without a query.
export function bestMatch(dirs, q) {
  const query = String(q || "").toLowerCase();
  if (!query) return null;
  const ranked = filterDirs(dirs, query);
  const exact = ranked.find((d) => String(d.name || "").toLowerCase() === query);
  return exact || ranked[0] || null;
}
