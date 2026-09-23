// Inspector rail logic, kept pure so node:test can reach it: which owner the
// rail is anchored to, whether the shell has room for it, and the per-viewer
// preferences it remembers. The rail itself (components/Inspector.jsx) only
// renders what these functions decide. The change-shape logic the rail draws
// (folder grouping, session scope, worktree views) lives in
// @picode/shared/domain/inspector.js — the mobile Inspector screen shares it.
import { locate, displayAgentName } from "@picode/shared/domain/tree.js";
import { repoLine, shortPath, termLine } from "@picode/shared/domain/repoLine.js";
import { isTermTab, tabTermId, isFileTab, parseFileTab, isGitTab, isTreeTab, isAppTab, isInstructionsTab, instructionsTabWorkspace } from "./routes.js";

export const INSPECTOR_MIN = 260;
export const INSPECTOR_MAX = 560;
export const INSPECTOR_DEFAULT = 320;
// The conversation column never drops below this; the rail yields first.
export const CENTER_MIN = 640;
// Two --chat-gutter widths around the centered column.
export const CENTER_GUTTERS = 48;
// A viewer who never toggled the rail gets it open on a wide window only.
export const OPEN_BY_DEFAULT_FROM = 1440;

// anchorFor decides which owner the rail follows. Tabs with a folder (agent,
// terminal, file, git graph, file tree) anchor to their owner. An app tab has
// no folder of its own, so it keeps the previous anchor — unless the app has
// published a **subject**: the agent or terminal it currently has in focus
// (ADR-0109, amendment 2026-09-12). `appSubjects` is the host's map of tab id
// to owner, read here exactly like `gitOwners` and `treeOwners` are: the app
// states a fact about its own content, and the host alone decides that the
// rail follows it. An owner that no longer exists clears the anchor. The same
// anchor is returned by identity when nothing changed, so effects do not
// re-run.
export function anchorFor(selectedId, ctx, last = null) {
  const candidate = ownerOf(selectedId, ctx || {});
  const next = candidate === undefined ? last : candidate;
  if (!next || !ownerExists(next, ctx || {})) return null;
  if (last && last.kind === next.kind && last.id === next.id) return last;
  return { kind: next.kind, id: next.id };
}

// undefined = this tab has no owner of its own (keep the last anchor);
// an object = the owner that authorises reads for this tab.
function ownerOf(id, ctx) {
  if (!id) return undefined;
  if (isTermTab(id)) return { kind: "term", id: tabTermId(id) };
  if (isFileTab(id)) {
    const f = parseFileTab(id);
    return f ? { kind: f.kind, id: f.id } : undefined;
  }
  if (isGitTab(id)) {
    const o = (ctx.gitOwners || {})[id];
    return o ? { kind: o.kind, id: o.id } : undefined;
  }
  if (isTreeTab(id)) {
    const o = (ctx.treeOwners || {})[id];
    return o ? { kind: o.kind, id: o.id } : undefined;
  }
  if (isInstructionsTab(id)) return { kind: "workspace", id: instructionsTabWorkspace(id) };
  if (isAppTab(id)) {
    const o = (ctx.appSubjects || {})[id];
    return o && o.id ? { kind: o.kind, id: o.id } : undefined;
  }
  const loc = locate(ctx.workspaces, ctx.freeAgents, id);
  return loc && loc.agent ? { kind: "agent", id: loc.agent.id } : undefined;
}

export function ownerExists(owner, ctx) {
  if (!owner || !owner.id) return false;
  if (owner.kind === "term") return (ctx.terminals || []).some((t) => t && t.id === owner.id);
  if (owner.kind === "workspace") return (ctx.workspaces || []).some((w) => w && w.id === owner.id);
  const loc = locate(ctx.workspaces, ctx.freeAgents, owner.id);
  return !!(loc && loc.agent);
}

// describeAnchor: the identity line the rail's header shows — who, where,
// and on what branch — read from the live lists so a rename shows at once.
export function describeAnchor(anchor, ctx) {
  if (!anchor) return null;
  const c = ctx || {};
  if (anchor.kind === "term") {
    const t = (c.terminals || []).find((x) => x && x.id === anchor.id);
    if (!t) return null;
    const line = termLine(t);
    return { kind: "term", name: t.name || "Terminal", path: t.cwd || "", git: line.git, term: t, agent: null, workspace: null };
  }
  if (anchor.kind === "workspace") {
    const ws = (c.workspaces || []).find((x) => x && x.id === anchor.id);
    if (!ws) return null;
    return { kind: "workspace", name: ws.name || "Workspace", path: ws.path || "", git: ws.git || null, term: null, agent: null, workspace: ws };
  }
  const loc = locate(c.workspaces, c.freeAgents, anchor.id);
  if (!loc || !loc.agent) return null;
  const line = repoLine(loc.agent, loc.workspace);
  const path = loc.agent.workPath || (loc.workspace && loc.workspace.path) || "";
  return { kind: "agent", name: displayAgentName(loc.agent, loc.workspace), path, git: line.git, term: null, agent: loc.agent, workspace: loc.workspace };
}

export function clampInspectorWidth(n) {
  const v = Math.round(Number(n));
  if (!Number.isFinite(v)) return INSPECTOR_DEFAULT;
  return Math.min(INSPECTOR_MAX, Math.max(INSPECTOR_MIN, v));
}

// maxInspectorWidth: the widest the rail may be before the conversation
// column would fall under CENTER_MIN. Unknown app width = no constraint yet.
export function maxInspectorWidth(appWidth, sidebarWidth) {
  if (!(appWidth > 0)) return INSPECTOR_MAX;
  const room = appWidth - (sidebarWidth || 0) - CENTER_MIN - CENTER_GUTTERS;
  return Math.min(INSPECTOR_MAX, Math.max(0, Math.round(room)));
}

// inspectorLayout: whether the rail is shown and at what width. A narrow
// layout (the ≤767px column shell) never shows it; a closed rail stays
// closed; otherwise the rail shrinks to what the window allows and hides
// only when even its minimum width would squeeze the conversation. wantOpen
// is preserved through a squeeze, so widening the window brings it back.
export function inspectorLayout({ appWidth = 0, sidebarWidth = 0, inspectorWidth = INSPECTOR_DEFAULT, wantOpen = true, narrow = false } = {}) {
  const width = clampInspectorWidth(inspectorWidth);
  if (narrow) return { shown: false, reason: "narrow", width };
  if (!wantOpen) return { shown: false, reason: "closed", width };
  const max = maxInspectorWidth(appWidth, sidebarWidth);
  if (max < INSPECTOR_MIN) return { shown: false, reason: "squeezed", width };
  return { shown: true, reason: "", width: Math.min(width, max) };
}

export function defaultOpen(appWidth) {
  return appWidth >= OPEN_BY_DEFAULT_FROM;
}

// filterRows keeps the rows whose name or path contains the query, plus the
// folders above them so the hierarchy still reads. Empty query = every row.
export function filterRows(rows, query) {
  const q = String(query || "").trim().toLowerCase();
  if (!q) return rows || [];
  const keep = new Set();
  for (const row of rows || []) {
    const path = String(row.path || "");
    if (String(row.name || "").toLowerCase().includes(q) || path.toLowerCase().includes(q)) {
      keep.add(path);
      const parts = path.split("/");
      for (let i = 1; i < parts.length; i++) keep.add(parts.slice(0, i).join("/"));
    }
  }
  return (rows || []).filter((r) => keep.has(String(r.path || "")));
}

// blockedMessage: the one line shown when a background read met a 409 —
// the owner's folder is not the pinned root any more (ADR-0074). Only a
// terminal can move; for anything else the honest word is "changed".
export function blockedMessage(kind, cwd) {
  if (kind === "term" && cwd) return `This terminal moved to ${shortPath(cwd)}.`;
  return "This folder changed.";
}

const OPEN_KEY = "picode-inspector-open";
const WIDTH_KEY = "picode-inspector-w";
const TAB_KEY = "picode-inspector-tab";
// "Run when no agent is working here": the one preference that lets PiCode
// press Enter on a prepared command (ADR-0078 stage 2). Off by default.
const RUN_KEY = "picode-inspector-run";

function safeStorage() {
  try { return typeof localStorage !== "undefined" ? localStorage : null; } catch { return null; }
}

// Per-viewer, not navigable: open state, width and tab live in localStorage
// like the sidebar's (no hash route — same reasoning as termView).
export function readInspectorPrefs(storage = safeStorage()) {
  let open = null, width = INSPECTOR_DEFAULT, tab = "changes", run = false;
  try {
    const o = storage && storage.getItem(OPEN_KEY);
    open = o === "1" ? true : o === "0" ? false : null;
    const w = parseInt((storage && storage.getItem(WIDTH_KEY)) || "", 10);
    if (Number.isFinite(w)) width = clampInspectorWidth(w);
    const t = storage && storage.getItem(TAB_KEY);
    tab = t === "files" || t === "pr" ? t : "changes";
    run = (storage && storage.getItem(RUN_KEY)) === "1";
  } catch { /* private mode, quota — defaults are fine */ }
  return { open, width, tab, run };
}

export function writeInspectorPrefs(prefs, storage = safeStorage()) {
  if (!prefs || !storage) return;
  try {
    if (prefs.open === true || prefs.open === false) storage.setItem(OPEN_KEY, prefs.open ? "1" : "0");
    if (Number.isFinite(Number(prefs.width))) storage.setItem(WIDTH_KEY, String(clampInspectorWidth(prefs.width)));
    if (prefs.tab === "files" || prefs.tab === "changes" || prefs.tab === "pr") storage.setItem(TAB_KEY, prefs.tab);
    if (prefs.run === true || prefs.run === false) storage.setItem(RUN_KEY, prefs.run ? "1" : "0");
  } catch { /* preference is optional */ }
}

// runFallbackNote: the toast when a run was refused and the command was
// prepared instead — the server's reason, then what to do.
export function runFallbackNote(reason) {
  const why = String(reason || "").trim().replace(/\.?$/, "");
  return (why ? why + ". " : "") + "The command is ready in the terminal; press Enter to run it.";
}

// --- Pull request tab (ADR-0078, phase 2) --------------------------------

// prTabLabel: the tab names the number once gh has answered (Paseo's
// "PR #3981"); every other state stays a plain "PR".
export function prTabLabel(page) {
  const n = page && page.status === "ok" && page.pr && page.pr.number;
  return n ? `PR #${n}` : "PR";
}

// prStateLabel: one word for the pill — Draft wins over Open.
export function prStateLabel(pr) {
  if (!pr) return "";
  if (pr.draft) return "Draft";
  const s = String(pr.state || "").toLowerCase();
  return s === "open" ? "Open" : s === "merged" ? "Merged" : s === "closed" ? "Closed" : s ? s[0].toUpperCase() + s.slice(1) : "";
}

// prChecksLabel: "2 passed · 1 failed · 1 pending" — zero groups are silent,
// and no checks at all says so instead of printing a row of zeros.
export function prChecksLabel(checks) {
  if (!checks || !Number(checks.total)) return "No checks";
  const parts = [];
  if (checks.failed) parts.push(`${checks.failed} failed`);
  if (checks.pending) parts.push(`${checks.pending} pending`);
  if (checks.passed) parts.push(`${checks.passed} passed`);
  if (checks.skipped) parts.push(`${checks.skipped} skipped`);
  return parts.join(" · ");
}

// prReviewLabel: GitHub's reviewDecision in the words a person uses.
export function prReviewLabel(decision) {
  switch (String(decision || "").toUpperCase()) {
    case "APPROVED": return "Approved";
    case "CHANGES_REQUESTED": return "Changes requested";
    case "REVIEW_REQUIRED": return "Review required";
    default: return "No review yet";
  }
}

// prBlockedAction: which single action a blocked state offers. gh missing →
// the install page; not logged in → a terminal with the login typed; a
// folder without a GitHub remote has nothing PiCode can do; anything else
// retries.
export function prBlockedAction(reason) {
  switch (reason) {
    case "gh-missing": return "install";
    case "gh-unauth": return "login";
    case "no-remote":
    case "no-git": return "";
    default: return "retry";
  }
}

// --- Git actions ----------------------------------------------------------

// The composer moved to @picode/shared/domain/gitCommands.js when the git
// graph became a third caller (ADR-0096); it was duplicated verbatim in
// mobile before that. Re-exported here so every existing import keeps
// working and there is still one definition.
export {
  shellQuote,
  gitActionCommand,
  gitActions,
  branchChip,
  askableAgents,
  askChannelHint,
  askGitPrompt,
  askedNote,
} from "@picode/shared/domain/gitCommands.js";
