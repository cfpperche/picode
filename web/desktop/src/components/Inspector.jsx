import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { createRefreshQueue } from "@picode/shared/domain/appRefreshQueue.js";
import { shortPath } from "@picode/shared/domain/repoLine.js";
import { changeKinds, changedDirs, flattenTree, mergeLevel } from "../lib/fileTree.js";
import { ownerFileURL } from "../lib/fileIO.js";
import { useKeptScroll } from "../lib/keepScroll.js";
import { toastError } from "../lib/toast.js";
import { formatChord, primaryChord } from "../lib/appKeys.js";
import { useEdgeResize } from "../lib/resizeEdge.js";
import {
  INSPECTOR_MIN, blockedMessage, changeTotals, defaultOpen, describeAnchor,
  inspectorLayout, maxInspectorWidth, normalizeTouched, prTabLabel, scopeChanges,
  branchChip, gitActionCommand, gitActions, askableAgents, askChannelHint, askGitPrompt,
} from "../lib/inspector.js";
import InspectorChanges from "./InspectorChanges.jsx";
import InspectorFiles from "./InspectorFiles.jsx";
import InspectorPR, { usePullRequest } from "./InspectorPR.jsx";
import InspectorCommitDialog from "./InspectorCommitDialog.jsx";
import { IconCheck, IconChevronRight, IconEllipsis, IconFolders, IconGit, IconPanelRight, IconPanelRightClose, IconReload } from "./Icons.jsx";
import { ProviderFace } from "./ProviderFaces.jsx";
import TerminalCliBadge from "./TerminalCliBadge.jsx";

const REVEAL_STALE_MS = 10_000;
const EMPTY_STATUS = { git: false, changes: [], totals: null, branch: "", worktree: "", upstream: "", ahead: 0, behind: 0, detached: false, repoRoot: "" };

// useInspectorLayout measures the shell and decides whether the rail fits.
// The left sidebar's width is read from the DOM rather than lifted out of
// Sidebar.jsx, so the rail adds nothing to that component.
export function useInspectorLayout({ open, width, narrow }) {
  const [appWidth, setAppWidth] = useState(0);
  const [sidebarWidth, setSidebarWidth] = useState(0);
  useEffect(() => {
    const app = document.getElementById("app");
    const side = document.getElementById("sidebar");
    if (!app || typeof ResizeObserver === "undefined") return undefined;
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        if (entry.target === app) setAppWidth(entry.contentRect.width);
        else setSidebarWidth(entry.contentRect.width);
      }
    });
    observer.observe(app);
    if (side) observer.observe(side);
    return () => observer.disconnect();
  }, []);
  const measured = appWidth || (typeof window !== "undefined" ? window.innerWidth : 0);
  const wantOpen = open === true || open === false ? open : defaultOpen(measured);
  const layout = inspectorLayout({ appWidth, sidebarWidth, inspectorWidth: width, wantOpen, narrow });
  return { ...layout, wantOpen, appWidth, sidebarWidth, maxWidth: Math.max(INSPECTOR_MIN, maxInspectorWidth(appWidth, sidebarWidth)) };
}

// The toggle lives at the end of the main tab strip, where the rail begins.
export function InspectorToggle({ shown, reason, onToggle }) {
  const chord = formatChord(primaryChord("app.inspector.toggle"));
  const squeezed = reason === "squeezed" || reason === "narrow";
  const verb = shown ? "Hide inspector" : "Show inspector";
  const title = squeezed ? "Window too narrow for the inspector" : chord ? `${verb} (${chord})` : verb;
  return (
    <button type="button" className="insp-toggle" aria-pressed={!!shown} aria-controls="inspector" aria-label={verb} title={title} onClick={onToggle}>
      {shown ? <IconPanelRightClose /> : <IconPanelRight />}
    </button>
  );
}

function AnchorFace({ info }) {
  let mark = <IconFolders size={16} />;
  if (info && info.kind === "agent" && info.agent) mark = <ProviderFace agent={info.agent} />;
  else if (info && info.kind === "term" && info.term) mark = <TerminalCliBadge term={info.term} />;
  return <span className="insp-face">{mark}</span>;
}

// The Inspector is a navigation and review rail beside the center: Changes
// (the working tree as a folder tree with counts) and Files (the project
// tree). It hosts no editor — a click opens the existing file tab in the
// center, in Editor or Diff view. It follows the selected tab's owner and
// pins that owner's folder as the root precondition (ADR-0074): a background
// read that meets a 409 becomes a blocked line with Follow, never a silent
// retarget. Data flow, guards and refresh idioms mirror FileTreeSurface.
export default function Inspector({
  hidden, anchor, workspaces, freeAgents, terminals, touchedPaths,
  tab, onTab, width, maxWidth, onWidth, onToggle, activePath,
  onOpenFile, onOpenDiff, onOpenGraph, onOpenTree, onOpenTerminal, onChanges,
  runMode, onRunMode, onAskAgent,
}) {
  const anchorKind = anchor ? anchor.kind : "";
  const anchorId = anchor ? anchor.id : "";
  const owner = useMemo(() => (anchorKind && anchorId ? { kind: anchorKind, id: anchorId } : null), [anchorKind, anchorId]);
  const info = describeAnchor(anchor, { workspaces, freeAgents, terminals });
  const [levels, setLevels] = useState(null);
  const [expanded, setExpanded] = useState(() => new Set());
  const [status, setStatus] = useState(EMPTY_STATUS);
  const [root, setRoot] = useState("");
  const [gone, setGone] = useState(false);
  const [blocked, setBlocked] = useState(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [scope, setScope] = useState("all");
  const [liveWidth, setLiveWidth] = useState(null);
  const [nonce, setNonce] = useState(0);
  const [commitDialog, setCommitDialog] = useState(null);
  const rootRef = useRef("");
  const lifetimeRef = useRef(0);
  const busyRef = useRef(false);
  const loadRef = useRef(() => {});
  const hiddenRef = useRef(hidden);
  const expandedRef = useRef(expanded);
  const lastLoadRef = useRef(0);
  const queueRef = useRef(null);
  const onChangesRef = useRef(onChanges);
  hiddenRef.current = hidden;
  expandedRef.current = expanded;
  onChangesRef.current = onChanges;
  const bodyRef = useKeptScroll(hidden, [".insp-body"]);

  // A 409 says the owner's folder is no longer the pinned root. The message
  // names where the owner went, read live — the fleet lists may still carry
  // the old cwd — and falls back to "changed" when nothing better is known.
  const markBlocked = useCallback(async (who, generation) => {
    let cwd = "";
    try {
      const page = await api(ownerFileURL(who, "cwd"));
      cwd = (page && page.cwd) || "";
    } catch { /* workspaces have no cwd route; the generic line suffices */ }
    if (generation !== lifetimeRef.current) return;
    setBlocked({ cwd: cwd && cwd !== rootRef.current ? cwd : "" });
  }, []);

  const load = useCallback(async (manual = false) => {
    if (!owner || busyRef.current) return;
    const generation = lifetimeRef.current;
    busyRef.current = true;
    setBusy(true);
    try {
      // Only the first read and an explicit Refresh/Follow go without the
      // root: everything else asserts the pinned folder and accepts a 409.
      const page = await api(ownerFileURL(owner, "browse", "", manual ? "" : rootRef.current));
      if (generation !== lifetimeRef.current) return;
      if (!page || page.cwdOk === false) {
        setGone(true);
        setBlocked(null);
        setError("");
        return;
      }
      const moved = !!rootRef.current && page.root !== rootRef.current;
      if (moved) {
        expandedRef.current = new Set();
        setExpanded(new Set());
      }
      rootRef.current = page.root;
      setRoot(page.root);
      setGone(false);
      setBlocked(null);
      setError("");
      let next = mergeLevel({}, page);
      const pages = await Promise.all([...expandedRef.current].map(async (dir) => {
        try { return [dir, await api(ownerFileURL(owner, "browse", dir, page.root))]; } catch { return [dir, null]; }
      }));
      if (generation !== lifetimeRef.current) return;
      for (const [, sub] of pages) if (sub) next = mergeLevel(next, sub);
      const failed = new Set(pages.filter(([, sub]) => !sub).map(([dir]) => dir));
      if (failed.size) setExpanded((prev) => new Set([...prev].filter((dir) => !failed.has(dir))));
      setLevels((prev) => {
        const merged = moved || !prev ? next : { ...prev, ...next };
        for (const dir of failed) delete merged[dir];
        return merged;
      });
      try {
        const st = await api(ownerFileURL(owner, "gitstatus", "", page.root));
        if (generation !== lifetimeRef.current) return;
        setStatus(st && st.git
          ? {
            git: true, changes: st.changes || [], totals: st.totals || changeTotals(st.changes),
            branch: st.branch || "", worktree: st.worktree || "", upstream: st.upstream || "",
            ahead: Number(st.ahead) || 0, behind: Number(st.behind) || 0, detached: !!st.detached,
            repoRoot: st.repoRoot || "",
          }
          : EMPTY_STATUS);
      } catch (e) {
        if (generation === lifetimeRef.current) setError(e.message || "Could not read changes.");
      }
    } catch (e) {
      if (generation !== lifetimeRef.current) return;
      if (/folder changed/i.test(e.message || "")) await markBlocked(owner, generation);
      else setError(e.message || "Could not read this folder.");
    } finally {
      if (generation === lifetimeRef.current) {
        busyRef.current = false;
        setBusy(false);
        lastLoadRef.current = Date.now();
      }
    }
  }, [owner]);
  loadRef.current = load;

  // A new anchor forgets everything the previous one had: root, tree, status.
  useEffect(() => {
    lifetimeRef.current++;
    busyRef.current = false;
    rootRef.current = "";
    expandedRef.current = new Set();
    setLevels(null);
    setExpanded(new Set());
    setStatus(EMPTY_STATUS);
    setRoot("");
    setGone(false);
    setBlocked(null);
    setError("");
    setBusy(false);
    setScope("all");
    if (queueRef.current) queueRef.current.stop();
    queueRef.current = createRefreshQueue(() => loadRef.current(false));
    if (owner && !hiddenRef.current) void loadRef.current(false);
    return () => { if (queueRef.current) queueRef.current.stop(); };
  }, [owner]);

  // Live: the fleet git watcher names the folder it inspected; the feed's
  // open/reset reconcile like every other list (ADR-0048). Bursts coalesce.
  useEffect(() => subscribeFeed((ev) => {
    if (hiddenRef.current || !rootRef.current || !queueRef.current) return;
    const hit = ev.type === "git.updated" && ev.data && ev.data.path === rootRef.current;
    const reconcile = ev.type === "feed.open" || ev.type === "feed.reset";
    if (hit || reconcile) void queueRef.current.request();
  }), []);

  useEffect(() => {
    const kick = () => { if (!document.hidden && !hiddenRef.current && rootRef.current && queueRef.current) void queueRef.current.request(); };
    document.addEventListener("visibilitychange", kick);
    window.addEventListener("focus", kick);
    return () => { document.removeEventListener("visibilitychange", kick); window.removeEventListener("focus", kick); };
  }, []);

  // Revealing the rail refetches only when its last read is stale; a rail
  // that was hidden when its anchor changed loads now.
  useEffect(() => {
    if (hidden || !owner) return;
    if (!rootRef.current) { void loadRef.current(false); return; }
    if (Date.now() - lastLoadRef.current > REVEAL_STALE_MS && queueRef.current) void queueRef.current.request();
  }, [hidden, owner]);

  // Terminals are not in the fleet git watcher; their rows carry live cwd and
  // dirty facts instead, so a change there asks for a fresh read (which is
  // where a moved terminal turns into the blocked line).
  const termDirty = anchorKind === "term" && info && info.git ? Number(info.git.dirty || 0) : -1;
  const termCwd = anchorKind === "term" && info ? info.path : "";
  useEffect(() => {
    if (anchorKind !== "term" || hiddenRef.current || !rootRef.current || !queueRef.current) return;
    void queueRef.current.request();
  }, [termDirty, termCwd, anchorKind]);

  useEffect(() => {
    if (!onChangesRef.current) return;
    onChangesRef.current({ owner, root, paths: new Set((status.changes || []).map((c) => c.path)) });
  }, [status, root, owner]);

  async function toggle(path) {
    if (!owner) return;
    const open = new Set(expandedRef.current);
    if (open.has(path)) open.delete(path);
    else open.add(path);
    expandedRef.current = open;
    setExpanded(open);
    if (!open.has(path) || (levels && levels[path])) return;
    const generation = lifetimeRef.current;
    const at = rootRef.current;
    try {
      const page = await api(ownerFileURL(owner, "browse", path, at));
      if (generation === lifetimeRef.current && at === rootRef.current) setLevels((prev) => mergeLevel(prev || {}, page));
    } catch (e) {
      if (generation !== lifetimeRef.current) return;
      setExpanded((prev) => { const next = new Set(prev); next.delete(path); return next; });
      if (/folder changed/i.test(e.message || "")) await markBlocked(owner, generation);
      else setError(e.message || "Could not read that folder.");
    }
  }

  async function reveal() {
    if (!owner) return;
    try {
      await api(ownerFileURL(owner, "reveal", "", rootRef.current), {
        method: "POST", headers: { "Content-Type": "application/json" }, body: "{}",
      });
    } catch (e) { toastError(e); }
  }

  const sizer = useEdgeResize({
    width, min: INSPECTOR_MIN, max: maxWidth, edge: "left",
    onChange: setLiveWidth,
    onCommit: (next) => { setLiveWidth(null); if (onWidth) onWidth(next); },
  });

  // The PR read is eager once the folder is a repository, so the tab can
  // carry the number; it stays cached server-side and never polls.
  const pull = usePullRequest({ owner, root, branch: status.branch, enabled: !!status.git && !hidden, nonce });
  const changes = status.git ? status.changes : [];
  const scopable = anchorKind === "agent" && Array.isArray(touchedPaths);
  const touched = scopable && scope === "agent" ? normalizeTouched(touchedPaths, root) : null;
  const scoped = scopeChanges(changes, touched);
  const totals = touched ? changeTotals(scoped) : status.totals || changeTotals(changes);
  const kinds = useMemo(() => changeKinds(changes), [changes]);
  const dirtyDirs = useMemo(() => changedDirs(changes), [changes]);
  const rows = useMemo(() => flattenTree(levels || {}, expanded), [levels, expanded]);
  const panel = status.git && (tab === "changes" || tab === "pr") ? tab : "files";
  const order = status.git ? ["changes", "files", "pr"] : ["files"];
  const name = info ? info.name : "Inspector";
  const shownPath = root || (info && info.path) || "";
  const shownWidth = liveWidth == null ? width : liveWidth;
  const chord = formatChord(primaryChord("app.inspector.toggle"));
  const refresh = () => { setNonce((n) => n + 1); return loadRef.current(true); };
  const chip = branchChip(status);
  const actions = owner ? gitActions(status) : [];
  // Git actions prepare the exact command in the owner's terminal; the human
  // submits it there (ADR-0078). Nothing here runs git.
  const runGit = (action) => {
    if (!onOpenTerminal || !owner) return;
    onOpenTerminal(owner, root, gitActionCommand(action, { branch: status.branch, upstream: status.upstream }), { run: !!runMode });
  };
  // Stage 3 (ADR-0078): agents running in this repository can be asked
  // instead — a prompt through their own channel. Nothing here runs git.
  const askable = useMemo(
    () => (onAskAgent && status.git ? askableAgents({ workspaces, freeAgents }, { root, repoRoot: status.repoRoot, anchor }) : []),
    [onAskAgent, status.git, status.repoRoot, workspaces, freeAgents, root, anchor],
  );
  const askGit = (who, action) => {
    if (!onAskAgent) return;
    onAskAgent(who, askGitPrompt(action, { root, branch: status.branch, upstream: status.upstream }), root, action);
  };
  function onTabKey(e) {
    if (order.length < 2 || !["ArrowLeft", "ArrowRight", "Home", "End"].includes(e.key)) return;
    e.preventDefault();
    const at = Math.max(0, order.indexOf(panel));
    const next = e.key === "Home" ? order[0] : e.key === "End" ? order[order.length - 1]
      : order[(at + (e.key === "ArrowRight" ? 1 : order.length - 1)) % order.length];
    onTab(next);
    e.currentTarget.querySelector(`[data-panel="${next}"]`)?.focus();
  }

  return (
    <aside id="inspector" className={"inspector" + (sizer.resizing ? " resizing" : "")} style={{ width: shownWidth }} hidden={!!hidden} aria-label="Inspector">
      <div
        className="insp-sizer" role="separator" aria-orientation="vertical" aria-label="Inspector width"
        aria-valuemin={INSPECTOR_MIN} aria-valuemax={maxWidth} aria-valuenow={shownWidth} tabIndex={0} title="Drag to resize"
        onPointerDown={sizer.onPointerDown} onKeyDown={sizer.onKeyDown}
      />
      <header className="insp-head">
        <AnchorFace info={info} />
        <span className="insp-id" title={shownPath}>
          <span className="insp-name">{name}</span>
          {shownPath ? <span className="insp-path">{shortPath(shownPath)}</span> : null}
        </span>
        <div className="insp-actions" data-align-row>
          {owner ? (
            <button type="button" className="insp-btn" title={busy ? "Refreshing…" : "Refresh"} aria-label="Refresh" onClick={refresh} disabled={busy}>
              <IconReload size={14} className={busy ? "insp-spin" : undefined} />
            </button>
          ) : null}
          {owner && actions.length && onOpenTerminal ? (
            <DropdownMenu.Root>
              <DropdownMenu.Trigger asChild>
                <button type="button" className="insp-btn" aria-label="Git actions" title="Git actions"><IconGit size={14} /></button>
              </DropdownMenu.Trigger>
              <DropdownMenu.Portal>
                <DropdownMenu.Content className="composer-more-pop insp-git-pop" side="bottom" align="end" sideOffset={6} collisionPadding={8}>
                  <DropdownMenu.Item className="composer-more-item" onSelect={() => runGit("fetch")}><span>Fetch</span><span className="insp-menu-cmd">git fetch --prune</span></DropdownMenu.Item>
                  {actions.includes("pull") ? <DropdownMenu.Item className="composer-more-item" onSelect={() => runGit("pull")}><span>Pull</span><span className="insp-menu-cmd">git pull --ff-only</span></DropdownMenu.Item> : null}
                  {actions.includes("push") ? <DropdownMenu.Item className="composer-more-item" onSelect={() => runGit("push")}><span>Push</span><span className="insp-menu-cmd">{status.upstream ? "git push" : "git push -u origin …"}</span></DropdownMenu.Item> : null}
                  <DropdownMenu.Separator className="um-divider" />
                  <DropdownMenu.Item className="composer-more-item" onSelect={() => setCommitDialog({ push: false })}><span>Commit…</span><span className="insp-menu-cmd">git add -A && git commit</span></DropdownMenu.Item>
                  {actions.includes("commit-push") ? <DropdownMenu.Item className="composer-more-item" onSelect={() => setCommitDialog({ push: true })}><span>Commit and push…</span><span className="insp-menu-cmd">… && git push</span></DropdownMenu.Item> : null}
                  {pull.page && pull.page.status !== "ok" ? <>
                    <DropdownMenu.Separator className="um-divider" />
                    <DropdownMenu.Item className="composer-more-item" onSelect={() => runGit("pr")}><span>Create pull request</span><span className="insp-menu-cmd">gh pr create --fill</span></DropdownMenu.Item>
                  </> : null}
                  {onRunMode ? <>
                    <DropdownMenu.Separator className="um-divider" />
                    {/* Stage 2 (ADR-0078): PiCode presses Enter only when the
                        interlock finds nobody else writing this repository;
                        otherwise the command is prepared as before. */}
                    <DropdownMenu.CheckboxItem className="composer-more-item insp-menu-check" checked={!!runMode} onCheckedChange={(v) => onRunMode(!!v)}>
                      <span className="insp-menu-tick" aria-hidden="true"><DropdownMenu.ItemIndicator><IconCheck size={12} /></DropdownMenu.ItemIndicator></span>
                      <span>Run when no agent is working here</span>
                    </DropdownMenu.CheckboxItem>
                  </> : null}
                  {askable.length ? <>
                    <DropdownMenu.Separator className="um-divider" />
                    {askable.map((who) => (
                      <DropdownMenu.Sub key={who.id}>
                        <DropdownMenu.SubTrigger className="composer-more-item insp-menu-sub">
                          <span>Ask {who.name}</span>
                          <span className="insp-menu-hint">{askChannelHint(who)}</span>
                          <IconChevronRight className="insp-menu-chev" />
                        </DropdownMenu.SubTrigger>
                        <DropdownMenu.Portal>
                          <DropdownMenu.SubContent className="composer-more-pop insp-git-pop" sideOffset={4} alignOffset={-4} collisionPadding={8}>
                            <DropdownMenu.Item className="composer-more-item" onSelect={() => askGit(who, "fetch")}>Fetch</DropdownMenu.Item>
                            {actions.includes("pull") ? <DropdownMenu.Item className="composer-more-item" onSelect={() => askGit(who, "pull")}>Pull</DropdownMenu.Item> : null}
                            {actions.includes("push") ? <DropdownMenu.Item className="composer-more-item" onSelect={() => askGit(who, "push")}>Push</DropdownMenu.Item> : null}
                            <DropdownMenu.Separator className="um-divider" />
                            <DropdownMenu.Item className="composer-more-item" onSelect={() => setCommitDialog({ push: false, ask: who })}>Commit…</DropdownMenu.Item>
                            {actions.includes("commit-push") ? <DropdownMenu.Item className="composer-more-item" onSelect={() => setCommitDialog({ push: true, ask: who })}>Commit and push…</DropdownMenu.Item> : null}
                            {pull.page && pull.page.status !== "ok" ? <>
                              <DropdownMenu.Separator className="um-divider" />
                              <DropdownMenu.Item className="composer-more-item" onSelect={() => askGit(who, "pr")}>Create pull request</DropdownMenu.Item>
                            </> : null}
                          </DropdownMenu.SubContent>
                        </DropdownMenu.Portal>
                      </DropdownMenu.Sub>
                    ))}
                  </> : null}
                </DropdownMenu.Content>
              </DropdownMenu.Portal>
            </DropdownMenu.Root>
          ) : null}
          {owner ? (
            <DropdownMenu.Root>
              <DropdownMenu.Trigger asChild>
                <button type="button" className="insp-btn" aria-label="More" title="More"><IconEllipsis /></button>
              </DropdownMenu.Trigger>
              <DropdownMenu.Portal>
                <DropdownMenu.Content className="composer-more-pop" side="bottom" align="end" sideOffset={6} collisionPadding={8}>
                  <DropdownMenu.Item className="composer-more-item" onSelect={reveal}><IconFolders size={14} /><span>Reveal folder</span></DropdownMenu.Item>
                  {status.git && onOpenGraph && owner.kind !== "workspace" ? (
                    <DropdownMenu.Item className="composer-more-item" onSelect={() => onOpenGraph(owner, name)}><IconGit size={14} /><span>Open git graph</span></DropdownMenu.Item>
                  ) : null}
                  {onOpenTree ? (
                    <DropdownMenu.Item className="composer-more-item" onSelect={() => onOpenTree(owner, name)}><IconFolders size={14} /><span>Open as tab</span></DropdownMenu.Item>
                  ) : null}
                </DropdownMenu.Content>
              </DropdownMenu.Portal>
            </DropdownMenu.Root>
          ) : null}
          <button type="button" className="insp-btn" title={chord ? `Hide inspector (${chord})` : "Hide inspector"} aria-label="Hide inspector" onClick={onToggle}>
            <IconPanelRightClose />
          </button>
        </div>
      </header>
      {owner ? (
        <div className="insp-tabs-row">
          <nav className="insp-tabs" role="tablist" aria-label="Inspector view" onKeyDown={onTabKey}>
            {status.git ? (
              <button type="button" role="tab" className="ft-tab" data-panel="changes" tabIndex={panel === "changes" ? 0 : -1} aria-selected={panel === "changes"} onClick={() => onTab("changes")}>
                Changes{changes.length > 0 ? <span className="ft-tab-badge">{changes.length}</span> : null}
              </button>
            ) : null}
            <button type="button" role="tab" className="ft-tab" data-panel="files" tabIndex={panel === "files" ? 0 : -1} aria-selected={panel === "files"} onClick={() => onTab("files")}>Files</button>
            {status.git ? (
              <button type="button" role="tab" className="ft-tab" data-panel="pr" tabIndex={panel === "pr" ? 0 : -1} aria-selected={panel === "pr"} onClick={() => onTab("pr")}>{prTabLabel(pull.page)}</button>
            ) : null}
          </nav>
          {chip ? (
            <span className={"insp-branch" + (chip.unpublished || chip.detached ? " insp-branch-noted" : "")} title={chip.title}>
              <IconGit size={12} />
              <span className="insp-branch-name">{chip.name}</span>
              {chip.worktree ? <span className="insp-branch-wt">{chip.worktree}</span> : null}
              {chip.ahead ? <span className="insp-branch-ab" aria-label={`${chip.ahead} ahead`}>↑{chip.ahead}</span> : null}
              {chip.behind ? <span className="insp-branch-ab" aria-label={`${chip.behind} behind`}>↓{chip.behind}</span> : null}
              {chip.unpublished ? <span className="insp-branch-note">unpublished</span> : chip.detached ? <span className="insp-branch-note">detached</span> : null}
            </span>
          ) : root && !status.git && !gone ? <span className="insp-branch insp-branch-none">Not a git repository</span> : null}
        </div>
      ) : null}
      {blocked ? (
        <p className="insp-notice" role="status">
          <span>{blockedMessage(anchorKind, blocked.cwd)}</span>
          <button type="button" className="btn btn-sm" onClick={refresh} disabled={busy}>Follow</button>
        </p>
      ) : null}
      {error ? (
        <p className="insp-notice" role="status">
          <span>{error}</span>
          <button type="button" className="btn btn-sm" onClick={refresh} disabled={busy}>Try again</button>
        </p>
      ) : null}
      <div className="insp-body" ref={bodyRef}>
        {!owner ? (
          <p className="insp-msg">Open an agent or terminal to inspect its files.</p>
        ) : gone ? (
          <p className="insp-msg">That folder is gone. <button type="button" className="btn btn-sm" onClick={refresh} disabled={busy}>Refresh</button></p>
        ) : levels === null ? (
          error || blocked ? null : (
            <div className="ft-skeleton" aria-label="Loading files" aria-busy="true">
              {Array.from({ length: 10 }, (_, i) => <div key={i} className="skel-line" style={{ width: 30 + ((i * 19) % 50) + "%" }} />)}
            </div>
          )
        ) : panel === "pr" ? (
          <InspectorPR
            page={pull.page} error={pull.error} busy={pull.busy} branch={status.branch}
            onRetry={() => setNonce((n) => n + 1)}
            onTerminal={onOpenTerminal ? (command) => onOpenTerminal(owner, root, command) : undefined}
          />
        ) : panel === "changes" ? (
          <InspectorChanges
            changes={scoped} totals={totals} scope={scope} onScope={setScope} scopable={scopable}
            activePath={activePath} onOpen={(path) => onOpenDiff(owner, path)} onViewFiles={() => onTab("files")}
          />
        ) : (
          <InspectorFiles
            rows={rows} kinds={kinds} dirtyDirs={dirtyDirs} activePath={activePath}
            onToggle={toggle} onOpen={(path) => onOpenFile(owner, path)} onRefresh={refresh}
          />
        )}
      </div>
      <InspectorCommitDialog
        open={!!commitDialog} push={!!(commitDialog && commitDialog.push)} status={status} run={!!runMode}
        ask={(commitDialog && commitDialog.ask) || null} root={root}
        onClose={() => setCommitDialog(null)}
        onPrepare={(command) => { setCommitDialog(null); if (onOpenTerminal && owner) onOpenTerminal(owner, root, command, { run: !!runMode }); }}
        onAsk={(text, action) => { const who = commitDialog && commitDialog.ask; setCommitDialog(null); if (who && onAskAgent) onAskAgent(who, text, root, action); }}
      />
    </aside>
  );
}
