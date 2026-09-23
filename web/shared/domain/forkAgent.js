// Fork agent… (docs/architecture/cli-session-handoff.md, "Fork"): the pure
// half of the dialog. A fork either shares the source's folder or works in
// a worktree of its own on a new branch, created first through ADR-0096's
// visible git door; these helpers name that worktree, recognize it once
// git has made it, and shape the request the server takes.

// forkSlug names the worktree folder and its branch after the fork: lower
// case, letters, digits and dashes, at most 48 characters — one path
// segment, as the git composer requires. A name already taken by a
// worktree folder or a local branch gets -2, -3… so git never refuses it.
export function forkSlug(name, graph) {
  const base = String(name || "")
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[̀-ͯ]/g, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 48)
    .replace(/-+$/, "") || "fork";
  const taken = new Set();
  for (const w of (graph && graph.worktrees) || []) {
    const m = /\/\.worktrees\/([^/]+)\/?$/.exec(w.path || "");
    if (m) taken.add(m[1]);
  }
  for (const r of (graph && graph.refs) || []) {
    if (r && r.kind === "head" && r.name) taken.add(r.name);
  }
  if (!taken.has(base)) return base;
  for (let n = 2; ; n += 1) {
    const next = base + "-" + n;
    if (!taken.has(next)) return next;
  }
}

// forkWorktreePath is the folder git made for slug, or "" until it exists.
export function forkWorktreePath(graph, slug) {
  const hit = ((graph && graph.worktrees) || []).find((w) => (w.path || "").replace(/\/+$/, "").endsWith("/.worktrees/" + slug));
  return hit ? hit.path : "";
}

// forkRequest is the POST /api/agents/{id}/fork-agent body: staged files
// travel inline (the server writes them into the fork's folder), folder
// picks travel as paths relative to it.
export function forkRequest({ name, text, items, workPath }) {
  const files = [];
  const paths = [];
  for (const it of items || []) {
    if (it.path) paths.push(it.path);
    else files.push({ name: it.name, mime: it.mime, data: it.data });
  }
  const body = { name: String(name || "").trim(), prompt: String(text || "") };
  if (workPath) body.workPath = workPath;
  if (files.length) body.files = files;
  if (paths.length) body.paths = paths;
  return body;
}
