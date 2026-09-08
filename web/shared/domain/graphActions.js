// What the git graph offers for the row or pill the reader pointed at
// (ADR-0096). Pure: it reads the graph payload and the app's agent lists and
// returns a menu description. Nothing here runs git, navigates, or touches
// the DOM — the caller does that.
//
// Tier 0 rows cost no git command at all (clipboard, opening a tab that
// already exists). Tiers A-C compose a command on the server and travel to
// one of ADR-0078's three doors. An item git would refuse is never listed: a
// branch checked out in a sibling worktree offers the way into that checkout
// instead of a checkout git will reject.
//
// The tier of an action is *not* decided here. It comes from the server's
// catalog (GET /api/git/actions), because a client that believed a tier C
// action were tier B would skip the confirmation that tier exists for. With
// no catalog loaded, this module offers tier 0 only — it fails closed.

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

// terminalsById indexes the app's terminal list the same way, for the pi
// CLI terminals the server lists as occupants (ADR-0089 amendment).
function terminalsById(ctx) {
  const map = new Map();
  for (const t of (ctx && ctx.terminals) || []) if (t && t.id) map.set(t.id, t);
  return map;
}

// repoOccupants names everyone living in a worktree of this repository —
// agents, and pi launched as an Agent CLI terminal — with the liveness the
// sidebar already knows. The server decided who lives where; this only
// joins on id. A terminal's `running` is the server's own presence fact
// (`live`), refined by the sidebar's `running: false` when it has one.
export function repoOccupants(graph, ctx = {}) {
  const agents = agentsById(ctx);
  const terminals = terminalsById(ctx);
  const seen = new Set();
  const out = [];
  for (const wt of (graph && graph.worktrees) || []) {
    for (const a of (wt && wt.agents) || []) {
      if (!a || !a.id || seen.has(a.id)) continue;
      seen.add(a.id);
      if (a.kind === "terminal") {
        const live = terminals.get(a.id) || {};
        out.push({
          id: a.id,
          kind: "terminal",
          name: a.name || live.name || "terminal",
          streaming: String(live.state || "") === "working",
          running: !!a.live && live.running !== false,
          worktree: wt.path || "",
        });
        continue;
      }
      const live = agents.get(a.id) || {};
      out.push({
        id: a.id,
        kind: "agent",
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

// openAgentItems: one row per occupant of the worktree, opening the tab that
// already exists — an agent's conversation, or a pi terminal's pane.
function openAgentItems(worktree, occupants) {
  const here = (worktree && worktree.agents) || [];
  return here
    .filter((a) => a && a.id)
    .map((a) => {
      const live = occupants.find((o) => o.id === a.id);
      const name = a.name || (live && live.name) || (a.kind === "terminal" ? "terminal" : "agent");
      if (a.kind === "terminal") {
        return { id: "open-terminal:" + a.id, label: `Open ${name}`, kind: "open-terminal", terminalId: a.id };
      }
      return { id: "open-agent:" + a.id, label: `Open ${name}`, kind: "open-agent", agentId: a.id };
    });
}

function copyItem(id, label, value) {
  return { id, label, kind: "copy", value };
}

const REF_NOUN = { head: "Branch", remote: "Remote branch", tag: "Tag" };

// refSections lays each pill drawn on the row out as its own labelled section
// of the same menu, so every action a pill offers is reachable from the row —
// a pill is a span inside the row's button and a keyboard cannot point at one
// (WCAG 2.1.1).
//
// Sections rather than submenus, deliberately: a submenu is a second layer of
// interaction for rows that are one click each, and it puts the leaf a hover
// away from the label that gives it meaning. The menu grows taller instead,
// and scrolls inside itself when the window is short.
function refSections(refs, graph, ctx, occupants) {
  const out = [];
  for (const ref of refs || []) {
    if (!ref || !ref.name || !REF_NOUN[ref.kind]) continue;
    const menu = graphActions({ kind: "ref", ref }, graph, ctx, occupants);
    if (!menu.items.length) continue;
    out.push({
      id: "section:ref:" + ref.kind + ":" + ref.name,
      kind: "section",
      label: `${REF_NOUN[ref.kind]} ${ref.name}`,
      state: menu.state,
    });
    out.push(...menu.items);
  }
  return out;
}


// --- Write actions (ADR-0096 phases 2-4) ----------------------------------

// LABELS is UI copy: what an action is called where the reader meets it. The
// tier and the fields each needs come from the server's catalog, never from
// here.
export const LABELS = {
  fetch: "Fetch",
  "fetch-into-local": "Fetch into a local branch…",
  pull: "Pull (fast-forward only)",
  "pull-remote": "Pull into current branch",
  checkout: "Switch to this branch",
  "checkout-detach": "Check out this commit (detached)",
  "checkout-remote": "Check out as a local branch…",
  "create-branch": "Create a branch here…",
  "create-tag": "Create a tag here…",
  "create-worktree": "Create a worktree…",
  "prune-worktrees": "Prune stale worktrees",
  merge: "Merge into current branch",
  rebase: "Rebase current onto this",
  "cherry-pick": "Cherry-pick onto current",
  revert: "Revert this commit",
  "reset-soft": "Reset here, keep changes staged",
  "reset-mixed": "Reset here, keep changes",
  "reset-keep": "Move back here, keep local changes",
  "rename-branch": "Rename this branch…",
  "delete-branch": "Delete this branch",
  commit: "Commit…",
  "commit-push": "Commit and push…",
  pr: "Create a pull request",
  push: "Push",
  "push-force": "Force-push (with lease)",
  "push-tag": "Push this tag",
  "delete-branch-force": "Delete this unmerged branch",
  "delete-remote-branch": "Delete on the remote",
  "delete-tag": "Delete this tag",
  "reset-hard": "Reset here, discard changes",
  discard: "Discard all uncommitted changes",
  clean: "Delete every untracked file",
  "worktree-remove": "Remove this worktree",
  "worktree-remove-force": "Remove it and its uncommitted work",
};

// FIELD_LABELS name the one input an action collects, in the words that suit
// that action rather than the composer's field name.
export const FIELD_LABELS = {
  "fetch-into-local": { name: "Local branch name" },
  "checkout-remote": { name: "Local branch name" },
  "create-branch": { name: "Branch name" },
  "create-tag": { name: "Tag name" },
  "create-worktree": { name: "Worktree folder name" },
  "rename-branch": { name: "New branch name" },
  commit: { message: "Commit message" },
  "commit-push": { message: "Commit message" },
};

// actionItem turns an id into a menu row, or nothing when the catalog has
// never heard of it. The catalog is the only source of the tier.
function actionItem(id, catalog, target, refKind = "") {
  const info = catalog && catalog[id];
  if (!info || !LABELS[id]) return null;
  return {
    id: "act:" + id,
    kind: "action",
    action: id,
    label: LABELS[id],
    tier: info.tier,
    needs: info.needs || [],
    target,
    // A branch and a tag may share a name; what was pointed at is recorded
    // by kind as well, so an undo puts back the right one.
    refKind,
  };
}

function actionItems(ids, catalog, target, refKind = "") {
  const out = [];
  for (const id of ids) {
    const item = actionItem(id, catalog, target, refKind);
    if (item) out.push(item);
  }
  return out;
}

// hasRemote: without one, push, pull and fetch are commands git can only
// refuse, so the rows do not exist (the reference tool's own rule).
function hasRemote(graph) {
  return ((graph && graph.remotes) || []).length > 0;
}

function localNamed(graph, name) {
  return ((graph && graph.refs) || []).some((r) => r.kind === "head" && r.name === name);
}

// commitWriteActions: what may be done to a commit that is not the HEAD the
// graph was read through. Merging or rebasing onto the commit you are already
// on is a no-op git would report as "Already up to date" — a row that can
// only do nothing is not offered.
function commitWriteActions(commit, graph, catalog) {
  const isHead = commit.hash === graph.head;
  // Thirteen rows in one list is a menu nobody reads and a popup that runs
  // off the bottom of the window. They group by what they do to the
  // repository: make something new, apply this commit's work somewhere, or
  // move the branch you are on.
  const groups = [
    ["Create", ["create-branch", "create-tag", "create-worktree"]],
    ["Apply", isHead ? ["revert"] : ["cherry-pick", "revert", "merge", "rebase"]],
    ["Move this branch", isHead ? ["reset-soft", "reset-mixed", "reset-hard"] : ["checkout-detach", "reset-soft", "reset-mixed", "reset-hard"]],
  ];
  const out = [];
  for (const [label, ids] of groups) {
    const items = actionItems(ids, catalog, commit.hash);
    if (!items.length) continue;
    out.push({ id: "section:" + label, kind: "section", label });
    out.push(...items);
  }
  return out;
}

// branchWriteActions applies the rules git enforces anyway, before the click:
// a branch held by another checkout cannot be switched to or deleted, the
// branch you are on cannot be merged into itself, and only the current branch
// is what `git push` publishes.
function branchWriteActions(ref, graph, catalog, { checkedOut, isCurrent }) {
  const ids = [];
  if (!checkedOut) ids.push("checkout", "create-worktree");
  if (!isCurrent) ids.push("merge", "rebase");
  ids.push("rename-branch");
  if (isCurrent && hasRemote(graph)) ids.push("push", "push-force");
  if (!checkedOut) {
    ids.push(ref.merged ? "delete-branch" : "delete-branch-force");
  }
  return actionItems(ids, catalog, ref.name, "head");
}

function remoteWriteActions(ref, graph, catalog) {
  const local = ref.name.slice(ref.name.indexOf("/") + 1);
  const ids = [];
  if (!localNamed(graph, local)) ids.push("checkout-remote", "fetch-into-local");
  ids.push("pull-remote", "delete-remote-branch");
  const items = actionItems(ids, catalog, ref.name, "remote");
  // origin/feat-x wants a local branch called feat-x nine times in ten; the
  // form opens with it filled and lets the reader change it.
  for (const item of items) if (item.needs.includes("name")) item.name = local;
  return items;
}

function tagWriteActions(ref, graph, catalog) {
  const ids = hasRemote(graph) ? ["push-tag", "delete-tag"] : ["delete-tag"];
  return actionItems(ids, catalog, ref.name, "tag");
}

// worktreeWriteActions: only a sibling checkout under the repository's own
// .worktrees/ can be removed by name, because that is the only shape the
// composer builds. Removing the checkout you are reading through would pull
// the floor out from under the command itself.
function worktreeWriteActions(wt, graph, catalog) {
  const ids = ["prune-worktrees"];
  const items = actionItems(ids, catalog, "");
  if (!wt.self && worktreeSlug(wt, graph)) {
    for (const id of ["worktree-remove", "worktree-remove-force"]) {
      const item = actionItem(id, catalog, "");
      if (item) {
        item.name = worktreeSlug(wt, graph);
        items.push(item);
      }
    }
  }
  return items;
}

// worktreeSlug is the single segment under .worktrees/ that names a checkout,
// or "" when the worktree lives somewhere else — the composer only builds
// `.worktrees/<name>`, so anything else is not offered rather than guessed.
export function worktreeSlug(wt, graph) {
  const self = ((graph && graph.worktrees) || []).find((w) => w.self);
  const root = normDir(self && self.path);
  const path = normDir(wt && wt.path);
  if (!root || !path) return "";
  for (const base of [root, root.replace(/\/\.worktrees\/[^/]+$/, "")]) {
    const prefix = base + "/.worktrees/";
    if (path.startsWith(prefix)) {
      const rest = path.slice(prefix.length);
      if (rest && !rest.includes("/")) return rest;
    }
  }
  return "";
}

// uncommittedWriteActions: only the checkout the graph was read through can
// be committed from here, because the command runs in a terminal sitting in
// that folder. A sibling worktree's row keeps its Open <agent> row instead —
// the agent living there is that tree's own door.
function uncommittedWriteActions(wt, graph, catalog) {
  if (!wt.self) return [];
  const ids = ["commit"];
  // A detached HEAD has no branch for `git push` to publish; the composer
  // would refuse, so the row is absent rather than an error waiting.
  if (hasRemote(graph) && !wt.detached) ids.push("commit-push");
  ids.push("discard", "clean");
  return actionItems(ids, catalog, "");
}

// repoWriteActions are the ones that need no target: the header's own menu.
export function repoActions(graph, ctx = {}) {
  const catalog = ctx.catalog || null;
  const ids = [];
  if (hasRemote(graph)) ids.push("fetch", "pull", "push", "pr");
  ids.push("create-branch");
  const items = actionItems(ids, catalog, graph.head || "");
  const occupants = repoOccupants(graph, ctx);
  return {
    title: graph.name || "this repository",
    kind: "repo",
    state: "",
    busy: busyLine(occupants),
    occupants,
    items,
  };
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
  const catalog = ctx.catalog || null;
  const none = { title: "", kind: "", state: "", items: [], busy, occupants };
  if (!target || !target.kind) return none;

  if (target.kind === "commit") {
    const c = target.commit;
    if (!c || !c.hash) return none;
    return {
      title: c.hash.slice(0, 7),
      kind: "commit",
      state: "",
      busy,
      occupants,
      items: [
        copyItem("copy-hash", "Copy commit hash", c.hash),
        ...(c.subject ? [copyItem("copy-subject", "Copy commit subject", c.subject)] : []),
        ...commitWriteActions(c, graph, catalog),
        // Right-clicking a pill is the shortcut; the row's own menu is the
        // door that a keyboard can reach.
        ...refSections(target.refs, graph, ctx, occupants),
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
      occupants,
      items: [
        ...openAgentItems(wt, occupants),
        ...(wt.branch ? [copyItem("copy-branch", "Copy branch name", wt.branch)] : []),
        copyItem("copy-path", "Copy worktree path", wt.path),
        ...(target.uncommitted ? uncommittedWriteActions(wt, graph, catalog) : []),
        ...worktreeWriteActions(wt, graph, catalog),
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
      occupants,
      items: [copyItem("copy-tag", "Copy tag name", ref.name), ...tagWriteActions(ref, graph, catalog)],
    };
  }

  if (ref.kind === "remote") {
    return {
      title: ref.name,
      kind: "remote",
      state: "",
      busy,
      occupants,
      items: [copyItem("copy-branch", "Copy branch name", ref.name), ...remoteWriteActions(ref, graph, catalog)],
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
    occupants,
    items: [
      ...(elsewhere ? openAgentItems(wt, occupants) : []),
      copyItem("copy-branch", "Copy branch name", ref.name),
      ...branchWriteActions(ref, graph, catalog, { checkedOut: !!wt, isCurrent: !!(wt && wt.self) }),
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

// --- Gates (ADR-0096) ------------------------------------------------------

// confirmPhrase is what a reader types before PiCode itself presses Enter on
// a tier C action. The prepare door needs none: there the human reads the
// command in their own prompt and submits it, which *is* the confirmation.
// The run door has no such moment, so it borrows one.
//
// The phrase is always something already on screen — the ref, the folder, or
// the plain word for what is about to be lost — so it can be read off the
// dialog rather than guessed.
export function confirmPhrase(action, { target = "", name = "", branch = "" } = {}) {
  switch (action) {
    case "delete-branch-force":
    case "delete-tag":
    case "delete-remote-branch":
      return target;
    case "worktree-remove":
    case "worktree-remove-force":
      return name;
    case "push":
    case "push-force":
    case "commit-push":
      return branch;
    case "push-tag":
      return target;
    case "reset-hard":
    case "discard":
    case "clean":
      return "discard";
    default:
      return "";
  }
}

// gateFor says what stands between the reader and the delivery.
//
//   door    "prepare" | "run" | "ask"
//   tier    from the server's catalog
//
// Asking an agent is never gated here: the agent is a reader too, it acts in
// its own turn under its own rules, and the prompt says what may be lost.
export function gateFor({ tier = "", door = "prepare", action = "", target = "", name = "", branch = "" } = {}) {
  if (door !== "run" || tier !== "C") return { typed: "" };
  return { typed: confirmPhrase(action, { target, name, branch }) };
}

// --- Undo (ADR-0096 phase 4) ----------------------------------------------

// undoFor names the action that puts the repository back, given what was true
// before. It is offered only where an honest inverse exists.
//
// This is not GitButler's oplog. PiCode does not own the working tree, so it
// records a position, not a snapshot: what comes back is a *command* the
// reader still sends through a door, and only for the branch pointer or the
// ref that moved. Anything that published, or deleted files rather than refs,
// has no inverse here and is not offered one — saying "Undo" over a push or a
// `git clean` would be a lie.
//
//   before: { head, ref: {name, hash} }  what the graph held before the act
//
// Returns {action, target, name, why} or null.
export function undoFor(action, before = {}) {
  const head = before.head || "";
  const ref = before.ref || null;
  const back = (undoAction, why) => (head ? { action: undoAction, target: head, name: "", why } : null);
  const at = head.slice(0, 7);
  switch (action) {
    // `git add -A && git commit` is undone by moving HEAD back and keeping
    // the index: the changes are staged again, exactly as they were. A
    // --hard here would delete the very work that was just committed.
    case "commit":
      return back("reset-soft", `Uncommit, keeping the changes staged, back at ${at}.`);
    // Each reset is undone by the same reset in the other direction: soft
    // touches only HEAD, mixed only HEAD and the index, and neither loses
    // what was in the working tree before.
    case "reset-soft":
      return back("reset-soft", `Put this branch back at ${at}.`);
    case "reset-mixed":
      return back("reset-mixed", `Put this branch back at ${at}.`);
    // A hard reset already discarded the tree; going back hard restores the
    // committed state, which is everything that survived.
    case "reset-hard":
      return back("reset-hard", `Put this branch back at ${at}.`);
    // History that merged, rebased, picked or reverted goes back with
    // --keep: HEAD moves, local changes stay, and git refuses rather than
    // lose one — never --hard over a tree that may have been dirty.
    case "merge":
    case "rebase":
    case "cherry-pick":
    case "revert":
    case "pull":
    case "pull-remote":
      return back("reset-keep", `Move this branch back to ${at}, keeping local changes.`);
    case "delete-branch":
    case "delete-branch-force":
      if (!ref || !ref.name || !ref.hash) return null;
      return { action: "restore-branch", target: ref.hash, name: ref.name, why: `Put ${ref.name} back at ${ref.hash.slice(0, 7)}.` };
    case "delete-tag":
      if (!ref || !ref.name || !ref.hash) return null;
      return { action: "create-tag", target: ref.hash, name: ref.name, why: `Put the tag ${ref.name} back at ${ref.hash.slice(0, 7)}.` };
    default:
      return null;
  }
}

// undoNote is the sentence beside the offer. It never promises more than a
// prepared command: the reader still reads it and sends it.
export function undoNote(undo) {
  if (!undo) return "";
  return undo.why + " It is prepared like any other action — nothing is undone until you send it.";
}
