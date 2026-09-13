import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import ScreenHeader from "../components/ScreenHeader.jsx";
import GitActionsSheet from "../components/GitActionsSheet.jsx";
import GitGraphActionSheet from "../components/GitGraphActionSheet.jsx";
import { targetFor } from "../lib/git/graphSheet.js";
import GitReadState from "../components/GitReadState.jsx";
import GitChanges, { GitWorktreeChanges } from "../components/GitChanges.jsx";
import GitHistory from "../components/GitHistory.jsx";
import GitCommit from "../components/GitCommit.jsx";
import GitDiff from "../components/GitDiff.jsx";
import GitPullRequest from "../components/GitPullRequest.jsx";
import GitWorkspaceSheet from "../components/GitWorkspaceSheet.jsx";
import { IconChevronDown, IconGit } from "../components/Icons.jsx";
import { pickerOptions, triggerLabel, workspaceForOwner } from "@picode/shared/domain/workspacePicker.js";
import { askableAgents } from "../lib/git/actions.js";
import { branchSummary, gitURL, isMoved, rootProblem, shortHash, worktreeName } from "../lib/git/model.js";
import "../styles/mobile-git.css";

// Mobile owns the presentation. The canonical root is pinned by /browse;
// only Follow adopts another root. Git execution stays in host callbacks.
export default function Git(props) {
  const key = `${props.owner?.kind}:${props.owner?.id}:${props.root || ""}`;
  return <GitView key={key} {...props} />;
}

function GitView({ owner, title, root: initialRoot = "", initialCommit = "", onBack, onOpenFile, onOpenTerminal, onAskAgent, onOpen, onPickWorkspace, workspaces = [], freeAgents = [], terminals = [], runMode = false }) {
  const [root, setRoot] = useState(initialRoot);
  const [status, setStatus] = useState(null);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState("");
  const [blocked, setBlocked] = useState("");
  const [section, setSection] = useState(initialCommit ? "history" : "changes");
  const [trail, setTrail] = useState(initialCommit ? [{ type: "commit", hash: initialCommit }] : []);
  const [actions, setActions] = useState(false);
  const [wsOpen, setWsOpen] = useState(false);
  const [nonce, setNonce] = useState(0);
  const [visited, setVisited] = useState({ history: !!initialCommit, pr: false });
  // The graph's write vocabulary (ADR-0096): the last history payload, the
  // server's action catalog (tiers come from it; with none, tier 0 only),
  // and the target a long-press or the header handed to the sheet.
  const [graph, setGraph] = useState(null);
  const [catalog, setCatalog] = useState(null);
  const [sheet, setSheet] = useState(null);
  useEffect(() => {
    let live = true;
    api("/api/git/actions").then((page) => {
      if (!live) return;
      const byId = {};
      for (const a of (page && page.actions) || []) if (a && a.id) byId[a.id] = a;
      setCatalog(byId);
    }).catch(() => {});
    return () => { live = false; };
  }, []);
  // An action reports back by watching (ADR-0096): after a delivery the
  // screen polls the cheap token for up to 30s, only while visible, and
  // reloads when the repository changed. No standing timer.
  const watch = useRef(0);
  const afterAction = useCallback(async () => {
    const mine = ++watch.current;
    let started = "";
    try { started = (await api(gitURL(owner, "git/head"))).token || ""; } catch { /* the deadline still applies */ }
    for (let i = 0; i < 30 && watch.current === mine; i += 1) {
      await new Promise((r) => setTimeout(r, 1000));
      if (document.hidden) { i -= 1; continue; }
      try {
        const head = await api(gitURL(owner, "git/head"));
        if (head && head.token && head.token !== started) { loadRef.current?.(); return; }
      } catch { /* not an outcome */ }
    }
    if (watch.current === mine) loadRef.current?.();
    // owner's own fields, not ownerKey: that const is declared further down
    // and a dependency array is read at render time.
  }, [owner?.kind, owner?.id]);
  const main = useRef(null);
  const scroll = useRef({});
  const rootRef = useRef(initialRoot);
  const request = useRef(null);
  const blockedRef = useRef("");
  const loadRef = useRef(null);
  const ownerKey = `${owner?.kind}:${owner?.id}`;
  const markMoved = useCallback(e => { const message = rootProblem(e); blockedRef.current = message; setBlocked(message); }, []);

  const load = useCallback(async (follow = false) => {
    if (blockedRef.current && !follow) return;
    if (!owner?.id) { setError("This project is unavailable."); setBusy(false); return; }
    request.current?.abort();
    const controller = new AbortController();
    request.current = controller;
    setBusy(true); setError("");
    try {
      const page = await api(gitURL(owner, "browse", { root: follow ? "" : rootRef.current }), { signal: controller.signal });
      const next = await api(gitURL(owner, "gitstatus", { root: page.root }), { signal: controller.signal });
      if (controller.signal.aborted) return;
      if (rootRef.current && page.root !== rootRef.current) { setTrail([]); setSection("changes"); }
      rootRef.current = page.root; setRoot(page.root); setStatus(next);
      blockedRef.current = ""; setBlocked("");
      setNonce(n => n + 1);
    } catch (e) {
      if (controller.signal.aborted) return;
      if (isMoved(e)) markMoved(e);
      else setError(humanizeError(e.message || "Could not read this repository."));
    } finally {
      if (!controller.signal.aborted) setBusy(false);
    }
  }, [ownerKey, markMoved]);
  loadRef.current = load;
  useEffect(() => { load(); return () => request.current?.abort(); }, [load]);

  useEffect(() => {
    let timer;
    const refresh = () => {
      if (!rootRef.current || blockedRef.current || document.hidden) return;
      clearTimeout(timer);
      timer = setTimeout(() => loadRef.current?.(), 150);
    };
    const unsub = subscribeFeed(event => {
      if (event.type === "feed.open" || event.type === "feed.reset" || (event.type === "git.updated" && event.data?.path === rootRef.current)) refresh();
    });
    window.addEventListener("focus", refresh);
    document.addEventListener("visibilitychange", refresh);
    return () => { clearTimeout(timer); unsub(); window.removeEventListener("focus", refresh); document.removeEventListener("visibilitychange", refresh); };
  }, []);

  const agents = useMemo(() => askableAgents({ workspaces, freeAgents }, { root, repoRoot: status?.repoRoot, anchor: owner }), [workspaces, freeAgents, root, status?.repoRoot, ownerKey]);
  // Which workspace's folder this screen reads through, and which ones it could
  // be pointed at (the rule lives in shared domain — the desktop asks it too).
  const wsOptions = useMemo(() => pickerOptions(workspaces, { reposOnly: true }), [workspaces]);
  const ownerWorkspace = workspaceForOwner(owner, { workspaces, freeAgents, terminals });
  const wsLabel = triggerLabel(ownerWorkspace, title);
  const current = trail.at(-1);
  const viewKey = current ? `${current.type}:${current.hash || current.file?.path || current.wt?.path}:${trail.length}` : section;
  const saveScroll = () => { scroll.current[viewKey] = main.current?.scrollTop || 0; };
  const goSection = next => { saveScroll(); setVisited(v => ({ ...v, [next]: true })); setSection(next); };
  const push = item => { if (!blockedRef.current) { saveScroll(); setTrail(items => [...items, item]); } };
  const back = () => { saveScroll(); if (trail.length) setTrail(items => items.slice(0, -1)); else onBack?.(); };
  useLayoutEffect(() => { if (main.current) main.current.scrollTop = scroll.current[viewKey] || 0; }, [viewKey]);
  const refresh = () => load();
  const headerTitle = current?.type === "commit" ? `Commit ${shortHash(current.hash)}` : current?.type === "working-file" || current?.type === "commit-file" ? "Diff" : current?.type === "worktree" ? "Worktree changes" : "Git";
  const openFile = file => { if (!blockedRef.current) onOpenFile?.(file); };
  let content;
  if (current?.type === "commit") content = <GitCommit key={current.hash} owner={owner} root={root} hash={current.hash} onMoved={markMoved} onBack={back} onCommit={hash => push({ type: "commit", hash })} onFile={(file, commit) => push({ type: "commit-file", file, commit })} />;
  else if (current?.type === "commit-file") content = <GitDiff owner={owner} root={root} file={current.file} commit={current.commit} onMoved={markMoved} />;
  else if (current?.type === "working-file") content = <GitDiff owner={owner} root={current.root || root} file={current.file} worktree={current.worktree} nonce={nonce} blocked={!!blocked} onMoved={markMoved} onOpenFile={openFile} />;
  else if (current?.type === "worktree") content = <GitWorktreeChanges owner={owner} wt={current.wt} nonce={nonce} blocked={!!blocked} onMoved={markMoved} onBack={back} onSelect={file => push({ type: "working-file", file, root: current.wt.path, worktree: current.wt.branch || current.wt.head })} />;


  return <div className="m-screen m-git">
    <ScreenHeader title={headerTitle} sub={current?.type === "worktree" ? worktreeName(current.wt) : title} onBack={back} right={status?.git && !blocked ? (
      current?.type === "commit"
        ? <button type="button" className="btn m-git-header-action" aria-label="Actions on this commit" onClick={() => setSheet(targetFor({ commit: (graph?.commits || []).find((c) => c.hash === current.hash) || { hash: current.hash }, refs: graph?.refs || [] }))}>Actions</button>
        : current?.type === "worktree"
          ? <button type="button" className="btn m-git-header-action" aria-label="Actions on this worktree" onClick={() => setSheet(targetFor({ worktree: current.wt, uncommitted: true }))}>Actions</button>
          : !current ? <button type="button" className="btn m-git-header-action" aria-label="Git actions" onClick={() => setActions(true)}>Actions</button> : null
    ) : null} />
    {!current && status?.git ? <nav className="m-git-tabs" aria-label="Git views">{[["changes", "Changes"], ["history", "History"], ["pr", "Pull request"]].map(([id, label]) => <button type="button" key={id} aria-current={section === id ? "page" : undefined} onClick={() => goSection(id)}>{label}</button>)}</nav> : null}
    <main className="m-git-main" ref={main}>
      {blocked ? <div className="m-git-state m-git-warning" role="alert"><p>{blocked}</p><button type="button" className="btn" disabled={busy} onClick={() => { setActions(false); load(true); }}>Follow folder</button></div> : null}
      <GitReadState error={error} loading={!status && busy} onRetry={() => load()} />
      {status?.git ? <>
        {!current ? <div className="m-git-repository">
          <div className="m-git-repo-facts">
            {/* Fewer than two choices is not a control: a dropdown that can only
                pick what is already picked is chrome, so the folder keeps its
                plain line. */}
            {wsOptions.length > 1 ? (
              <button type="button" className="m-git-ws" aria-haspopup="dialog" aria-expanded={wsOpen} onClick={() => setWsOpen(true)}>
                <IconGit size={12} />
                <span className="m-git-ws-label">{wsLabel}</span>
                <IconChevronDown />
              </button>
            ) : (
              <span className="m-git-ws-static"><IconGit size={12} /> {wsLabel}</span>
            )}
            <p>{branchSummary(status)}</p>
          </div>
          <button type="button" className="btn" disabled={busy || !!blocked} onClick={refresh}>{busy ? "Refreshing…" : "Refresh"}</button>
        </div> : null}
        {content}
        <div className="m-git-view" hidden={!!current || section !== "changes"}><GitChanges status={status} onHistory={() => goSection("history")} onSelect={file => push({ type: "working-file", file })} /></div>
        {visited.history ? <div className="m-git-view" hidden={!!current || section !== "history"}><GitHistory key={root} owner={owner} root={root} nonce={nonce} blocked={!!blocked} onMoved={markMoved} onCommit={hash => push({ type: "commit", hash })} onWorktree={wt => push({ type: "worktree", wt })} onActions={(payload) => setSheet(targetFor(payload))} onGraph={setGraph} /></div> : null}
        {visited.pr ? <div className="m-git-view" hidden={!!current || section !== "pr"}><GitPullRequest key={root} owner={owner} root={root} nonce={nonce} onMoved={markMoved} onOpenTerminal={onOpenTerminal} blocked={!!blocked} /></div> : null}
      </> : status ? <div className="m-git-state"><p>This folder is not a Git repository.</p><button className="btn" type="button" onClick={onBack}>Back</button></div> : null}
    </main>
    <GitWorkspaceSheet open={wsOpen} workspaces={workspaces} currentId={ownerWorkspace ? ownerWorkspace.id : ""} onPick={wsId => { setWsOpen(false); onPickWorkspace?.(wsId); }} onClose={() => setWsOpen(false)} />
    <GitActionsSheet owner={owner} root={root} status={status} agents={agents} open={actions} blocked={!!blocked} onClose={() => setActions(false)} onOpenTerminal={onOpenTerminal} onAskAgent={onAskAgent} />
    <GitGraphActionSheet owner={owner} root={root} target={sheet} graph={graph} ctx={{ workspaces, freeAgents, terminals, catalog }} run={runMode} open={!!sheet} blocked={!!blocked} onClose={() => setSheet(null)} onOpenTerminal={onOpenTerminal} onAskAgent={onAskAgent} onOpen={onOpen} onDelivered={afterAction} />
  </div>;
}
