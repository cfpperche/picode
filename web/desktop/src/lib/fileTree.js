// File tree logic (ADR-0030), kept pure so node:test can reach it: the tree
// is a cache of one-level /browse answers plus a set of expanded dirs, and
// the git decoration is derived from the flat change list alone — a folder
// shows its dot without ever having been expanded.

export function treeApiBase(kind) {
  if (kind === "term") return "/api/terminals/";
  if (kind === "workspace") return "/api/workspaces/";
  return "/api/agents/";
}

// provisionalTreeKey names a tree tab before the first response tells us the
// real root — same trick as the git graph's provisional key.
export function provisionalTreeKey(kind, id) {
  const k = kind === "term" ? "t" : kind === "workspace" ? "w" : "a";
  return "@" + k + ":" + String(id || "");
}

// mergeLevel folds one /browse answer into the level cache. The cache maps
// a dir path ("" = the root) to its listed children.
export function mergeLevel(levels, resp) {
  const dir = String(resp?.dir || "");
  return { ...levels, [dir]: { dirs: resp?.dirs || [], files: resp?.files || [] } };
}

// flattenTree walks the cached levels depth-first through the expanded set,
// yielding one row per visible entry. A dir row with loaded:false is
// expanded but its listing has not arrived (or failed) yet.
export function flattenTree(levels, expanded) {
  const out = [];
  const walk = (dir, depth) => {
    const level = levels[dir];
    if (!level) return;
    for (const d of level.dirs) {
      const open = expanded.has(d.path);
      out.push({ path: d.path, name: d.name, depth, isDir: true, open, loaded: !open || !!levels[d.path] });
      if (open) walk(d.path, depth + 1);
    }
    for (const f of level.files) {
      out.push({ path: f.path, name: f.name, depth, isDir: false });
    }
  };
  walk("", 0);
  return out;
}

// changeKinds: path -> kind, for decorating file rows.
export function changeKinds(changes) {
  const out = new Map();
  for (const c of changes || []) {
    if (c && c.path) out.set(c.path, c.kind || "modified");
  }
  return out;
}

// changedDirs: every ancestor dir of a changed file, so collapsed folders
// carry the dot too ("a/b/c.go" -> "a", "a/b").
export function changedDirs(changes) {
  const out = new Set();
  for (const c of changes || []) {
    const parts = String(c?.path || "").split("/");
    for (let i = 1; i < parts.length; i++) {
      out.add(parts.slice(0, i).join("/"));
    }
  }
  return out;
}
