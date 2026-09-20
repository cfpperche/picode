import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { shortPath } from "@picode/shared/domain/repoLine.js";
import {
  sessionGroups, resolveSessionView, changeTotals, normalizeTouched, scopeChanges,
  compactCount,
} from "@picode/shared/domain/inspector.js";
import ScreenHeader from "../components/ScreenHeader.jsx";
import GitReadState from "../components/GitReadState.jsx";
import GitActionsSheet from "../components/GitActionsSheet.jsx";
import GitDiff from "../components/GitDiff.jsx";
import GitPullRequest from "../components/GitPullRequest.jsx";
import { GitFileRow } from "../components/GitChanges.jsx";
import { askableAgents } from "../lib/git/actions.js";
import { branchSummary, gitURL, isMoved, rootProblem } from "../lib/git/model.js";
import { folderSections } from "../lib/git/folders.js";
import { ownerFileURL } from "../lib/fileIO.js";
import { useGitRead } from "../lib/git/useGitRead.js";
import { useAgentTouched } from "../lib/agentTouched.js";
import { IconBack, IconChevronRight } from "../components/Icons.jsx";
import "../styles/mobile-git.css";

// The mobile Inspector (ADR-0078's rail, re-shaped for the phone): one
// pushed screen anchored to an owner — agent, terminal or workspace — with
// the review surface the desktop rail keeps beside the conversation:
// Changes (folder-grouped, session-scoped, multi-worktree), Files, PR.
// It hosts no editor: a file opens in the existing tools. Reads go through
// the same owner-scoped endpoints as the Git tool, with the same root
// precondition; live updates ride the shared feed. Never an interval.
export default function Inspector(props) {
  const key = `${props.owner?.kind}:${props.owner?.id}:${props.root || ""}`;
  return <InspectorView key={key} {...props} />;
}

function InspectorView({ owner, title, root: initialRoot = "", initialView = "", onBack, onOpenFile, onOpenTerminal, onAskAgent, workspaces = [], freeAgents = [], terminals = [] }) {
  const sessionTouched = useAgentTouched(owner?.kind === "agent" ? owner.id : "");
  const [root, setRoot] = useState(initialRoot);
  const [status, setStatus] = useState(null);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState("");
  const [blocked, setBlocked] = useState("");
  const [section, setSection] = useState(initialView === "files" || initialView === "pr" ? initialView : "changes");
  const [actions, setActions] = useState(false);
  const [nonce, setNonce] = useState(0);
  const [prNonce, setPrNonce] = useState(0);
  const [trail, setTrail] = useState([]);
  // The session scope starts where the desktop starts: everything, with the
  // "This agent" slice one tap away once the session has touched files.
  const [scope, setScope] = useState("all");
  const [followedRef, setFollowedRef] = useState("");
  const [dismissed, setDismissed] = useState(false);

  const main = useRef(null);
  const rootRef = useRef(initialRoot);
  const request = useRef(null);
  const blockedRef = useRef("");
  const loadRef = useRef(null);
  const watch = useRef(0);
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
      if (rootRef.current && page.root !== rootRef.current) setTrail([]);
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

  // The pull request of the anchor's branch, read once the folder is known.
  // The server caches per folder and branch for a minute; nothing polls.
  const prRead = useGitRead(root && status?.git ? gitURL(owner, "pr", { root, refresh: prNonce > 0 }) : "", prNonce, markMoved, !!blocked);

  const agents = useMemo(() => askableAgents({ workspaces, freeAgents }, { root, repoRoot: status?.repoRoot, anchor: owner }), [workspaces, freeAgents, root, status?.repoRoot, ownerKey]);
  const groups = useMemo(() => (status?.git ? sessionGroups(status, root) : []), [status, root]);
  const view = useMemo(() => resolveSessionView({ groups, followedRef, dismissed }), [groups, followedRef, dismissed]);
  // Scope chips exist only where a session exists: an agent anchor whose
  // conversation named files this browser session (deep links read All).
  const touchedPaths = owner?.kind === "agent" ? sessionTouched : null;
  const scopable = owner?.kind === "agent" && Array.isArray(touchedPaths);
  const shown = useMemo(() => (view.shown || []).map((g) => {
    if (!scopable || scope === "all") return { group: g, changes: g.changes || [], totals: g.totals || changeTotals(g.changes) };
    const set = normalizeTouched(touchedPaths, g.path);
    const changes = scopeChanges(g.changes, set);
    return { group: g, changes, totals: changeTotals(changes) };
  }), [view.shown, scopable, scope, touchedPaths]);
  const visible = scopable && scope === "agent" ? shown.filter((s) => s.changes.length > 0) : shown;
  const sessionFiles = (status?.git ? (status.changes || []).length : 0) + (status?.worktrees || []).reduce((n, wt) => n + ((wt.changes || []).length), 0);
  const prPage = prRead.data;
  const prLabel = prPage?.status === "ok" && prPage.pr?.number ? `PR #${prPage.pr.number}` : "PR";
  const actionRoot = view.pill ? view.pill.path : root;

  const current = trail.at(-1);
  const scroll = useRef({});
  const viewKey = current ? `diff:${current.file?.path}` : section;
  const saveScroll = () => { scroll.current[viewKey] = main.current?.scrollTop || 0; };
  const goSection = next => { saveScroll(); setSection(next); };
  const back = () => { saveScroll(); if (trail.length) setTrail(items => items.slice(0, -1)); else onBack?.(); };
  useLayoutEffect(() => { if (main.current) main.current.scrollTop = scroll.current[viewKey] || 0; }, [viewKey]);
  const refresh = () => { load(); setPrNonce(n => n + 1); };
  const openFile = file => { if (!blockedRef.current && file) onOpenFile?.(file); };

  let content;
  if (current) content = <GitDiff owner={owner} root={current.root || root} file={current.file} worktree={current.worktree} nonce={nonce} blocked={!!blocked} onMoved={markMoved} onOpenFile={openFile} />;

  const tabs = !current && status?.git ? (
    <nav className="m-git-tabs" aria-label="Inspector views">
      <button type="button" aria-current={section === "changes" ? "page" : undefined} onClick={() => goSection("changes")}>Changes{sessionFiles > 0 ? <span className="m-insp-badge">{sessionFiles}</span> : null}</button>
      <button type="button" aria-current={section === "files" ? "page" : undefined} onClick={() => goSection("files")}>Files</button>
      <button type="button" aria-current={section === "pr" ? "page" : undefined} onClick={() => goSection("pr")}>{prLabel}</button>
    </nav>
  ) : null;

  return <div className="m-screen m-git m-inspector">
    <ScreenHeader title={current ? "Diff" : "Inspector"} sub={title} onBack={back} right={!current && status?.git && !blocked ? (
      <button type="button" className="btn m-git-header-action" aria-label="Git actions" onClick={() => setActions(true)}>Actions</button>
    ) : null} />
    {tabs}
    <main className="m-git-main" ref={main}>
      {blocked ? <div className="m-git-state m-git-warning" role="alert"><p>{blocked}</p><button type="button" className="btn" disabled={busy} onClick={() => load(true)}>Follow folder</button></div> : null}
      <GitReadState error={error} loading={!status && busy} onRetry={() => load()} />
      {status?.git ? <>
        {!current ? <div className="m-git-repository">
          <div className="m-git-repo-facts">
            <p>{branchSummary(status)}{view.pill ? <span className="m-insp-following"> · reviewing {shortPath(view.pill.path)}</span> : null}</p>
          </div>
          <button type="button" className="btn" disabled={busy || !!blocked} onClick={refresh}>{busy ? "Refreshing…" : "Refresh"}</button>
        </div> : null}
        {content}
        <div className="m-git-view" hidden={!!current || section !== "changes"}>
          <InspectorChanges shown={visible} view={view} scopable={scopable} scope={scope} onScope={setScope}
            onDismiss={() => setDismissed(true)} onRefollow={() => setDismissed(false)}
            onSelect={(file, group) => setTrail(items => [...items, { type: "working-file", file, root: group.path, worktree: group.isRoot ? "" : group.ref }])}
            onViewFiles={() => goSection("files")} />
        </div>
        <div className="m-git-view" hidden={!!current || section !== "files"}>
          <InspectorFiles owner={owner} root={root} blocked={!!blocked} enabled={section === "files"} onOpen={openFile} />
        </div>
        <div className="m-git-view" hidden={!!current || section !== "pr"}>
          <GitPullRequest owner={owner} root={root} nonce={prNonce} blocked={!!blocked} onOpenTerminal={onOpenTerminal}
            read={{ ...prRead, onRetry: () => setPrNonce(n => n + 1) }} />
        </div>
      </> : status ? <>
        <div className="m-git-state"><p>This folder is not a Git repository.</p></div>
        <InspectorFiles owner={owner} root={root} blocked={!!blocked} enabled onOpen={openFile} onMoved={markMoved} />
      </> : null}
    </main>
    <GitActionsSheet owner={owner} root={actionRoot} status={status} agents={agents} open={actions} blocked={!!blocked} onClose={() => setActions(false)} onOpenTerminal={onOpenTerminal} onAskAgent={onAskAgent} />
  </div>;
}

function InspectorChanges({ shown, view, scopable, scope, onScope, onDismiss, onRefollow, onSelect, onViewFiles }) {
  const dirty = shown.some((s) => s.changes.length > 0);
  if (!shown.length || !dirty) {
    return <div className="m-git-state">
      {scopable && scope === "agent"
        ? <><p>No files from this agent yet.</p><button type="button" className="btn" onClick={() => onScope("all")}>Show all</button></>
        : <><p>No uncommitted changes.</p><button type="button" className="btn" onClick={onViewFiles}>View files</button></>}
    </div>;
  }
  const pill = view.pill;
  return <section aria-label="Changes">
    {pill ? <div className="m-insp-pill">
      <span>Following <strong>{pill.branch || shortPath(pill.path)}</strong></span>
      <button type="button" className="btn btn-sm" onClick={onDismiss}>Back</button>
    </div> : null}
    {!pill && view.switcher?.length ? <div className="m-insp-pill">
      <span>{view.switcher.length} {view.switcher.length === 1 ? "worktree has" : "worktrees have"} changes</span>
      <button type="button" className="btn btn-sm" onClick={onRefollow}>View</button>
    </div> : null}
    {scopable ? <div className="m-insp-scope" role="group" aria-label="Changes scope">
      {[["all", "All"], ["agent", "This agent"]].map(([id, label]) => (
        <button type="button" key={id} className="m-insp-chip" aria-pressed={scope === id} onClick={() => onScope(id)}>{label}</button>
      ))}
    </div> : null}
    {shown.map(({ group, changes, totals }) => <InspectorGroup key={group.key} group={group} changes={changes} totals={totals} onSelect={onSelect} />)}
  </section>;
}

function InspectorGroup({ group, changes, totals, onSelect }) {
  const sections = useMemo(() => folderSections(changes), [changes]);
  return <div className="m-insp-group">
    {!group.isRoot ? <p className="m-git-hint">{group.branch || "Detached"} · {shortPath(group.path)}</p> : null}
    <div className="m-git-section-head"><h2>{totals.files ? `${totals.files} changed ${totals.files === 1 ? "file" : "files"}` : "Working tree"}</h2>{totals.files ? <span><span className="m-git-add">+{compactCount(totals.add)}</span> <span className="m-git-del">−{compactCount(totals.del)}</span></span> : null}</div>
    {sections.map((section) => <div key={section.dir || "."} className="m-insp-folder">
      {section.dir ? <p className="m-insp-dir"><span>{section.dir}</span>{section.add || section.del ? <span className="m-insp-dir-counts">{section.add ? <span className="m-git-add">+{compactCount(section.add)}</span> : null}{section.del ? <span className="m-git-del">−{compactCount(section.del)}</span> : null}</span> : null}</p> : null}
      <div className="m-git-files">
        {section.files.map(file => <GitFileRow key={file.path} file={file} onSelect={() => onSelect(file, group)} />)}
      </div>
    </div>)}
  </div>;
}

// The project tree, one level at a time — a doorway into the Files tool,
// not an editor: tapping a file hands it to the tool this phone already has.
function InspectorFiles({ owner, root, blocked, enabled, onOpen, onMoved }) {
  const [dir, setDir] = useState("");
  const [retry, setRetry] = useState(0);
  useEffect(() => { setDir(""); }, [root]);
  const { data, error, loading } = useGitRead(enabled && !blocked && owner?.id ? ownerFileURL(owner, "browse", dir, root) : "", `${dir}:${retry}`, onMoved, blocked);
  const page = data;
  if (!enabled) return null;
  return <section aria-label="Files">
    {dir ? <button type="button" className="m-insp-up" onClick={() => setDir(dir.includes("/") ? dir.slice(0, dir.lastIndexOf("/")) : "")}>
      <IconBack size={13} /><span>{dir.split("/").pop()}</span>
    </button> : null}
    <GitReadState error={error} loading={!page && loading} onRetry={() => setRetry(n => n + 1)} />
    {page ? <div className="m-git-files m-insp-browse">
      {(page.dirs || []).map(row => <button type="button" key={row.path} className="m-git-file-row m-insp-dir-row" onClick={() => setDir(row.path)}>
        <span className="m-git-file-name"><strong>{row.name}</strong></span>
        <IconChevronRight size={13} />
      </button>)}
      {(page.files || []).map(row => <button type="button" key={row.path} className="m-git-file-row" onClick={() => onOpen({ owner, path: row.path, root })}>
        <span className="m-git-file-name"><strong>{row.name}</strong></span>
      </button>)}
      {!(page.dirs || []).length && !(page.files || []).length ? <p className="m-empty-line">This folder is empty.</p> : null}
    </div> : null}
  </section>;
}
