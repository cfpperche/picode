// Pure helpers for the Branches picker and the Show Remote Branches toggle.
// Nothing here touches the DOM, localStorage or fetch.

// visibleRefs drops tags (never offered in the picker) and, when showRemotes
// is off, drops remote-kind refs too — the same filter the ref-pills use, so
// the popover and the row decorations never disagree.
export function visibleRefs(refs, showRemotes) {
  return (refs || []).filter((r) => r.kind !== "tag" && (showRemotes || r.kind !== "remote"));
}

// groupBranches splits the picker's option list into local/remote sections,
// each sorted by name. The remote section is simply absent (not present-but-
// empty) when showRemotes is off.
export function groupBranches(refs, showRemotes) {
  const local = (refs || []).filter((r) => r.kind === "head").map((r) => r.name).sort();
  const remote = showRemotes
    ? (refs || []).filter((r) => r.kind === "remote").map((r) => r.name).sort()
    : [];
  return { local, remote };
}

// triggerLabel: "Show All", one branch name verbatim, or "name & N more"
// once a second branch joins the selection.
export function triggerLabel(selected) {
  const list = selected || [];
  if (list.length === 0) return "Show All";
  if (list.length === 1) return list[0];
  return `${list[0]} & ${list.length - 1} more`;
}

// walkParams is what the fetch sends: the branches to seed the walk with,
// and whether remotes matter at all (only when selection is empty — Show
// All is the one state where the checkbox still changes what is walked).
export function walkParams(selected, showRemotes) {
  const branches = (selected || []).filter(Boolean);
  return { branches, remotes: branches.length === 0 ? !!showRemotes : true };
}

// resolveSelection drops any previously-picked branch name that no longer
// appears in a freshly loaded ref list (deleted branch, or a stale
// localStorage entry). Run this once against a fresh graph.refs, before the
// selection reaches the popover or a fetch.
export function resolveSelection(selected, refs) {
  const known = new Set((refs || []).filter((r) => r.kind === "head" || r.kind === "remote").map((r) => r.name));
  return (selected || []).filter((name) => known.has(name));
}

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
