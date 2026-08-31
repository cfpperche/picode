// Search over the loaded commit window (ADR-0038). Client-side and additive:
// matches highlight and the rest dims, but no row is ever hidden — the lane
// layout is positional, and a graph with rows removed no longer tells the
// truth about the history.

// A query shorter than this matches nothing: one character lights up half the
// graph, which reads as broken rather than as a result.
export const MIN_QUERY = 2;

// matchCommits returns the hashes whose subject or author contains the query
// (case-insensitive), or whose hash starts with it.
export function matchCommits(commits, query) {
  const q = String(query || "").trim().toLowerCase();
  const out = new Set();
  if (q.length < MIN_QUERY) return out;
  for (const c of commits || []) {
    if (
      (c.subject || "").toLowerCase().includes(q) ||
      (c.author || "").toLowerCase().includes(q) ||
      (c.hash || "").toLowerCase().startsWith(q)
    ) {
      out.add(c.hash);
    }
  }
  return out;
}
