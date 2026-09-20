// Inspector change-shape logic shared by the desktop rail (ADR-0078) and the
// mobile Inspector screen: how a flat gitstatus change list becomes a
// folder-grouped tree with summed counts, how one agent's session scopes it,
// and which dirty checkout of a repository the viewer is looking at. Kept
// pure so node:test reaches it from both apps; neither app re-implements it.

function byName(a, b) {
  return a.name.localeCompare(b.name, undefined, { sensitivity: "base", numeric: true });
}

// groupChanges folds a flat change list into per-level rows (dirs first,
// then files, both sorted), plus a stats map keyed by path: a file's own
// counts, or a folder's sum over its descendants. Binary files have no
// lines and add nothing to their folder's counts.
export function groupChanges(changes) {
  const levels = { "": { dirs: [], files: [] } };
  const stats = new Map();
  const dirs = new Set();
  for (const c of changes || []) {
    const path = String((c && c.path) || "");
    if (!path) continue;
    const parts = path.split("/");
    let parent = "";
    for (let i = 0; i < parts.length - 1; i++) {
      const dir = parts.slice(0, i + 1).join("/");
      if (!dirs.has(dir)) {
        dirs.add(dir);
        levels[dir] = { dirs: [], files: [] };
        levels[parent].dirs.push({ name: parts[i], path: dir });
        stats.set(dir, { add: 0, del: 0, files: 0, binary: false });
      }
      const s = stats.get(dir);
      s.add += Number(c.add) || 0;
      s.del += Number(c.del) || 0;
      s.files += 1;
      parent = dir;
    }
    levels[parent].files.push({ name: parts[parts.length - 1], path });
    stats.set(path, { add: Number(c.add) || 0, del: Number(c.del) || 0, files: 1, binary: !!c.binary, kind: c.kind || "modified" });
  }
  for (const level of Object.values(levels)) {
    level.dirs.sort(byName);
    level.files.sort(byName);
  }
  return { levels, dirs, stats };
}

export function changeTotals(changes) {
  const out = { add: 0, del: 0, files: 0 };
  for (const c of changes || []) {
    if (!c || !c.path) continue;
    out.add += Number(c.add) || 0;
    out.del += Number(c.del) || 0;
    out.files += 1;
  }
  return out;
}

// scopeChanges narrows the working tree to the paths one agent touched.
// null means "no scope" (every change); an empty set is a real empty scope.
export function scopeChanges(changes, touched) {
  if (!touched) return changes || [];
  return (changes || []).filter((c) => c && touched.has(c.path));
}

// normalizeTouched turns the paths a session's edit/write tools named —
// relative to the cwd or absolute — into root-relative paths comparable to
// gitstatus. Paths outside the root are dropped: they are not this folder's.
export function normalizeTouched(paths, root) {
  const out = new Set();
  const base = String(root || "").replace(/\\/g, "/").replace(/\/+$/, "");
  for (const raw of paths || []) {
    let p = String(raw || "").replace(/\\/g, "/").trim();
    if (!p) continue;
    if (p.startsWith("/")) {
      if (!base || !(p === base || p.startsWith(base + "/"))) continue;
      p = p.slice(base.length + 1);
    } else {
      p = p.replace(/^(\.\/)+/, "");
    }
    p = p.replace(/\/+$/, "");
    if (p && p !== "." && !p.startsWith("../")) out.add(p);
  }
  return out;
}

// compactCount: 684 · 1.9k · 12k · 1.2M — the shape the benchmarks print.
export function compactCount(n) {
  const v = Math.max(0, Math.round(Number(n) || 0));
  if (v < 1000) return String(v);
  if (v < 10000) return (v / 1000).toFixed(1).replace(/\.0$/, "") + "k";
  if (v < 1000000) return Math.round(v / 1000) + "k";
  return (v / 1000000).toFixed(1).replace(/\.0$/, "") + "M";
}

export function totalsLabel(totals) {
  if (!totals) return "";
  const parts = [];
  if (totals.add) parts.push("+" + compactCount(totals.add));
  if (totals.del) parts.push("−" + compactCount(totals.del));
  return parts.join(" ");
}

// sessionGroups folds a gitstatus page into what the Changes tab draws: the
// anchor folder first, then every dirty linked worktree in git's list order.
// The root group carries the pinned browse root (not the repo toplevel), so
// feed matching and touched-path scoping keep working exactly as before.
export function sessionGroups(status, root) {
  const out = [{
    key: "root",
    isRoot: true,
    path: String(root || ""),
    ref: "",
    branch: (status && status.branch) || "",
    detached: !!(status && status.detached),
    worktree: (status && status.worktree) || "",
    upstream: (status && status.upstream) || "",
    ahead: Number(status && status.ahead) || 0,
    behind: Number(status && status.behind) || 0,
    changes: (status && status.changes) || [],
    totals: (status && status.totals) || null,
  }];
  for (const wt of (status && status.worktrees) || []) {
    if (!wt || !wt.path || !wt.ref) continue;
    out.push({
      key: "wt:" + wt.path,
      isRoot: false,
      path: String(wt.path),
      ref: String(wt.ref),
      branch: wt.branch || "",
      detached: !!wt.detached,
      worktree: wt.worktree || "",
      upstream: wt.upstream || "",
      ahead: Number(wt.ahead) || 0,
      behind: Number(wt.behind) || 0,
      changes: wt.changes || [],
      totals: wt.totals || null,
    });
  }
  return out;
}

function groupDirty(g) {
  return !!(g && g.changes && g.changes.length);
}

// resolveSessionView decides which groups the Changes tab shows and whether
// the viewer is following one of them (docs/plans/inspector-session-changes.md
// decision table). follow when empty, suggest when not: an explicit follow
// (or one clean root with one dirty sibling) shows that sibling with a pill
// back to the anchor; anything messier shows groups or a switcher line, and
// a stale follow — the checkout went clean or missing — falls through to
// the same rules as no follow at all.
export function resolveSessionView({ groups, followedRef = "", dismissed = false } = {}) {
  const list = groups || [];
  const root = list.find((g) => g.isRoot) || null;
  const dirtyWTs = list.filter((g) => !g.isRoot && groupDirty(g));
  const rootDirty = groupDirty(root);
  const followed = followedRef ? list.find((g) => !g.isRoot && g.ref === followedRef) || null : null;
  if (followed && groupDirty(followed)) {
    return { mode: "follow", shown: [followed], pill: followed, switcher: dirtyWTs.filter((g) => g !== followed) };
  }
  if (!dismissed && !rootDirty && dirtyWTs.length === 1) {
    return { mode: "follow", shown: dirtyWTs, pill: dirtyWTs[0], switcher: [] };
  }
  if (rootDirty && dirtyWTs.length > 0) {
    return { mode: "groups", shown: [root, ...dirtyWTs], pill: null, switcher: [] };
  }
  if (!rootDirty && dirtyWTs.length > 1) {
    return { mode: "groups", shown: [...dirtyWTs], pill: null, switcher: [] };
  }
  if (!rootDirty && dirtyWTs.length === 1 && dismissed) {
    return { mode: "root", shown: root ? [root] : [], pill: null, switcher: dirtyWTs };
  }
  return { mode: "root", shown: root ? [root] : [], pill: null, switcher: [] };
}
