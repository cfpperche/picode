// What the git graph offers for the row or pill the reader pointed at
// (ADR-0096). Pure: it reads the graph payload and the app's agent lists and
// returns a menu description. Nothing here runs git, navigates, or touches
// the DOM — the caller does that.
//
// Phase 1 carries tier 0 only: the actions that cost no git command at all
// (clipboard, opening a tab that already exists). Tiers A-C arrive with the
// delivery doors in phase 2. An item that git would refuse is never listed:
// a branch checked out in a sibling worktree offers the way into that
// checkout instead of a checkout git will reject.

function basename(p) {
  const s = String(p || "").replace(/\/+$/, "");
  const cut = s.lastIndexOf("/");
  return cut < 0 ? s : s.slice(cut + 1);
}

function normDir(p) {
  return String(p || "").replace(/\\/g, "/").replace(/\/+$/, "");
}

// agentsById indexes every agent the app knows, so the graph's own
// worktree -> agent mapping (computed on the server) can borrow liveness
// without re-deriving who lives where.
function agentsById(ctx) {
  const map = new Map();
  const add = (a) => {
    if (a && a.id) map.set(a.id, a);
  };
  for (const ws of (ctx && ctx.workspaces) || []) for (const a of (ws && ws.agents) || []) add(a);
  for (const a of (ctx && ctx.freeAgents) || []) add(a);
  return map;
}

// repoOccupants names every agent living in a worktree of this repository,
// with the liveness the sidebar already knows. The server decided who lives
// where; this only joins on id.
export function repoOccupants(graph, ctx = {}) {
  const known = agentsById(ctx);
  const seen = new Set();
  const out = [];
  for (const wt of (graph && graph.worktrees) || []) {
    for (const a of (wt && wt.agents) || []) {
      if (!a || !a.id || seen.has(a.id)) continue;
      seen.add(a.id);
      const live = known.get(a.id) || {};
      out.push({
        id: a.id,
        name: a.name || live.name || "agent",
        streaming: !!live.streaming,
        running: live.running !== false,
        worktree: wt.path || "",
      });
    }
  }
  return out;
}

// busyLine is the sentence the menu shows before the reader clicks — the one
// thing no other git client can say. Silence when nobody is mid-turn: a menu
// header that reports the absence of a problem is noise.
export function busyLine(occupants) {
  const busy = (occupants || []).filter((o) => o.streaming);
  if (!busy.length) return "";
  if (busy.length === 1) return `${busy[0].name} is mid-turn in this repository.`;
  return `${busy[0].name} and ${busy.length - 1} more are mid-turn in this repository.`;
}

// worktreeOf finds the checkout holding a branch. The ref carries it since
// ADR-0096; the worktree list is the fallback for a payload that predates it.
function worktreeOf(graph, ref) {
  const wts = (graph && graph.worktrees) || [];
  if (ref && ref.worktree) {
    const byPath = wts.find((wt) => normDir(wt.path) === normDir(ref.worktree));
    if (byPath) return byPath;
    return { path: ref.worktree };
  }
  return wts.find((wt) => wt.branch && ref && wt.branch === ref.name) || null;
}

function openAgentItems(worktree, occupants) {
  const here = (worktree && worktree.agents) || [];
  return here
    .filter((a) => a && a.id)
    .map((a) => {
      const live = occupants.find((o) => o.id === a.id);
      return {
        id: "open-agent:" + a.id,
        label: `Open ${a.name || (live && live.name) || "agent"}`,
        kind: "open-agent",
        agentId: a.id,
      };
    });
}

function copyItem(id, label, value) {
  return { id, label, kind: "copy", value };
}

const REF_NOUN = { head: "Branch", remote: "Remote branch", tag: "Tag" };

// refSubmenus turns the pills drawn on a row into one submenu each, so every
// action a pill offers is also reachable from the row itself.
function refSubmenus(refs, graph, ctx, occupants) {
  const out = [];
  for (const ref of refs || []) {
    if (!ref || !ref.name || !REF_NOUN[ref.kind]) continue;
    const menu = graphActions({ kind: "ref", ref }, graph, ctx, occupants);
    if (!menu.items.length) continue;
    out.push({
      id: "ref:" + ref.kind + ":" + ref.name,
      kind: "sub",
      label: `${REF_NOUN[ref.kind]} ${ref.name}`,
      // A submenu can open far from its trigger (Radix flips it away from a
      // viewport edge), so it names its own subject rather than relying on
      // the row above it.
      name: ref.name,
      state: menu.state,
      items: menu.items,
    });
  }
  return out;
}

// graphActions describes the menu for one target.
//
//   target: {kind: "commit", commit}
//           {kind: "ref", ref}          ref.kind is head | remote | tag
//           {kind: "worktree", worktree}
//
// The answer is {title, kind, state, items}. `state` is one line of fact —
// never an instruction, never a disabled item's excuse. `items` may be empty;
// the caller renders the empty menu as nothing rather than an empty popup.
export function graphActions(target, graph = {}, ctx = {}, known = null) {
  const occupants = known || repoOccupants(graph, ctx);
  const busy = busyLine(occupants);
  const none = { title: "", kind: "", state: "", items: [], busy };
  if (!target || !target.kind) return none;

  if (target.kind === "commit") {
    const c = target.commit;
    if (!c || !c.hash) return none;
    return {
      title: c.hash.slice(0, 7),
      kind: "commit",
      state: "",
      busy,
      items: [
        copyItem("copy-hash", "Copy commit hash", c.hash),
        ...(c.subject ? [copyItem("copy-subject", "Copy commit subject", c.subject)] : []),
        // The pills on this row are spans inside the row's own button, so a
        // keyboard has no way to point at one. The row's menu therefore
        // carries each ref as a submenu: right-clicking the pill is the
        // shortcut, not the only door (WCAG 2.1.1).
        ...refSubmenus(target.refs, graph, ctx, occupants),
      ],
    };
  }

  if (target.kind === "worktree") {
    const wt = target.worktree;
    if (!wt || !wt.path) return none;
    return {
      title: basename(wt.path),
      kind: "worktree",
      state: wt.detached ? "Detached HEAD." : "",
      busy,
      items: [
        ...openAgentItems(wt, occupants),
        ...(wt.branch ? [copyItem("copy-branch", "Copy branch name", wt.branch)] : []),
        copyItem("copy-path", "Copy worktree path", wt.path),
      ],
    };
  }

  if (target.kind !== "ref" || !target.ref || !target.ref.name) return none;
  const ref = target.ref;

  if (ref.kind === "tag") {
    return {
      title: ref.name,
      kind: "tag",
      state: "",
      busy,
      items: [copyItem("copy-tag", "Copy tag name", ref.name)],
    };
  }

  if (ref.kind === "remote") {
    return {
      title: ref.name,
      kind: "remote",
      state: "",
      busy,
      items: [copyItem("copy-branch", "Copy branch name", ref.name)],
    };
  }

  // A local branch. The state line is the fact that decides what phase 2 may
  // offer here, and it is worth saying on its own: a branch checked out in a
  // sibling worktree cannot be checked out or deleted, and git says so by
  // naming that checkout. PiCode names it first, and opens it.
  const wt = worktreeOf(graph, ref);
  const elsewhere = wt && !wt.self;
  const state = wt
    ? elsewhere
      ? `Checked out in ${basename(wt.path)}.`
      : "Checked out here."
    : "";
  return {
    title: ref.name,
    kind: "head",
    state,
    busy,
    items: [
      ...(elsewhere ? openAgentItems(wt, occupants) : []),
      copyItem("copy-branch", "Copy branch name", ref.name),
    ],
  };
}

// trackingLabel is what a branch pill can say about its upstream, in the
// words the branch chip already uses (ADR-0078). Empty when there is nothing
// to report — a branch level with its upstream needs no decoration.
export function trackingLabel(ref) {
  if (!ref || ref.kind !== "head") return "";
  if (ref.gone) return "upstream gone";
  if (!ref.upstream) return "";
  const parts = [];
  if (ref.ahead) parts.push(`↑${ref.ahead}`);
  if (ref.behind) parts.push(`↓${ref.behind}`);
  return parts.join(" ");
}

// trackingTitle is the tooltip behind that label, spelled out for a reader
// who does not read arrows.
export function trackingTitle(ref) {
  if (!ref || ref.kind !== "head") return "";
  if (ref.gone) return `${ref.upstream || "The upstream"} no longer exists on the remote`;
  if (!ref.upstream) return `${ref.name} has no upstream yet`;
  const parts = [];
  if (ref.ahead) parts.push(`${ref.ahead} ahead`);
  if (ref.behind) parts.push(`${ref.behind} behind`);
  if (!parts.length) return `${ref.name} is level with ${ref.upstream}`;
  return `${ref.name} is ${parts.join(" and ")} ${ref.upstream}`;
}
