import { useEffect, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { parseRoute, appRoute } from "../lib/routes.js";
import UserMenu from "./UserMenu.jsx";
import RailTabs from "./RailTabs.jsx";
import ShareDrawer, { OPEN_EVENT } from "./ShareDrawer.jsx";
import { IconTerminal, IconPlus, IconFolder, IconFolders, IconAgent, IconGit, IconX, IconChevronRight, IconPin, IconSession, IconSettings, IconGrid, IconCli } from "./Icons.jsx";
import Pins from "./Pins.jsx";
import AppsGrid from "./AppsGrid.jsx";
import { agentsOf, displayAgentName } from "@picode/shared/domain/tree.js";
import { wsLine } from "@picode/shared/domain/repoLine.js";
import { freeTerminals, workspaceTerminals } from "../lib/termGroups.js";
import ProviderFaces from "./ProviderFaces.jsx";
import { AgentRow, RowMenu, RowMenuItem, TermRow } from "./WorkspaceRows.jsx";

// Workspace cards wear the project's favicon when it has one (ADR-0027).
// The list advertises whether one exists, so a normal workspace without an
// icon never creates an expected 404. Failures are still remembered per
// page-load when a file disappears between the list and image request.
const faviconFailed = new Set();
export function WsFavicon({ ws }) {
  const [failed, setFailed] = useState(faviconFailed.has(ws.id));
  if (failed || ws.hasFavicon === false) return <IconFolder size={16} />;
  return (
    <img
      className="ws-favicon" width={16} height={16} alt="" loading="lazy"
      src={"/api/workspaces/" + encodeURIComponent(ws.id) + "/favicon"}
      onError={() => { faviconFailed.add(ws.id); setFailed(true); }}
    />
  );
}

const SIDE_MIN = 180;
const SIDE_MAX = 480;
const SIDE_KEY = "picode-sidebar-w";

export default function Sidebar({
  inShell = false,
  tab, selectTab,
  workspaces, selectedId,
  onNew, onSelect, onRun, onStop, onRemove,
  userMenu, termView, onChat, onTerm,
  freeAgents, onNewFree, onNewAgent, onNewCliPrincipal, onRemoveAgent, onRenameAgent,
  workingId,
  workingIds,
  waitingId,
  checklists,
  terminals, onNewTerm, onSelectTerm, onRemoveTerm, onRenameTerm, onLaunchAction, onContinueTerm, clis, onSessions,
  onGitGraph,
  onFileTree,
  onOpenDashboard,
  onOpenClis,
  apps, nativeApps, onOpenApp, webapps, webappsErr, webappsLoaded, onRetryWebapps, onOpenWebapp, onSavedWebapp, onRemoveWebapp, onRefreshWebapp, onClearWebappData, desktop,
}) {
  const [width, setWidth] = useState(() => {
    const n = parseInt(localStorage.getItem(SIDE_KEY) || "", 10);
    return Number.isFinite(n) ? Math.min(SIDE_MAX, Math.max(SIDE_MIN, n)) : 244;
  });
  const [resizing, setResizing] = useState(false);
  const [shareOpen, setShareOpen] = useState(false);
  useEffect(() => {
    const on = () => setShareOpen(true);
    window.addEventListener(OPEN_EVENT, on);
    return () => window.removeEventListener(OPEN_EVENT, on);
  }, []);
  useEffect(() => {
    const sync = () => {
      if (parseRoute() === "pins") selectTab("pins");
      else if (appRoute()) selectTab("apps");
    };
    sync();
    window.addEventListener("hashchange", sync);
    return () => window.removeEventListener("hashchange", sync);
  }, []);
  const [openWs, setOpenWs] = useState(() => {
    try { return JSON.parse(localStorage.getItem("picode-ws-open") || "{}"); }
    catch { return {}; }
  });

  function onSizerDown(e) {
    e.preventDefault();
    const startX = e.clientX;
    const startW = width;
    let latest = startW;
    setResizing(true);
    const move = (ev) => {
      latest = Math.min(SIDE_MAX, Math.max(SIDE_MIN, Math.round(startW + (ev.clientX - startX))));
      setWidth(latest);
    };
    const up = () => {
      setResizing(false);
      localStorage.setItem(SIDE_KEY, String(latest));
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  }

  // Collapsed header advertises everyone inside: managed agents first, then
  // terminals (agent CLI favicons, shell marks) — never a fake "empty".
  function collapsedMark(agents, terms) {
    return <ProviderFaces agents={agents} terms={terms} />;
  }

  function isOpen(id) { return openWs[id] !== false; }
  function toggleWs(id) {
    setOpenWs((s) => {
      const n = { ...s, [id]: !isOpen(id) };
      try { localStorage.setItem("picode-ws-open", JSON.stringify(n)); } catch { /* ignore */ }
      return n;
    });
  }

  function agentRow(ag, ws) {
    return (
      <AgentRow
        key={ag.id}
        agent={ag} ws={ws}
        selectedId={selectedId} onSelect={onSelect}
        workingId={workingId} workingIds={workingIds} waitingId={waitingId} checklists={checklists}
        onFileTree={onFileTree} onGitGraph={onGitGraph}
        onRenameAgent={onRenameAgent}
        onRun={onRun} onStop={onStop}
        onRemoveAgent={onRemoveAgent} onRemove={onRemove}
        onChat={onChat} onTerm={onTerm} termView={termView}
      />
    );
  }

  function termRow(t) {
    return (
      <TermRow
        key={t.id}
        term={t}
        selectedId={selectedId} onSelectTerm={onSelectTerm}
        onFileTree={onFileTree} onGitGraph={onGitGraph}
        onRenameTerm={onRenameTerm} onRemoveTerm={onRemoveTerm}
        onLaunchAction={onLaunchAction} onContinueTerm={onContinueTerm} clis={clis}
      />
    );
  }

  const sortedFreeAgents = [...(freeAgents || [])].sort((a, b) =>
    displayAgentName(a, null).localeCompare(displayAgentName(b, null), undefined, { sensitivity: "base" }));

  // The shell's merged row sizes its brand+rail cluster to this sidebar's
  // live width, so the rail tabs sit exactly above the column they control.
  useEffect(() => {
    if (!inShell) return;
    document.documentElement.style.setProperty("--shell-nav-w", width + "px");
  }, [inShell, width]);

  return (
    <aside id="sidebar" className={resizing ? "resizing" : ""} style={{ width }}>
      {/* In the desktop shell the brand row lives in the window's merged
          top row (ADR-0122); the sidebar starts at its own content. */}
      {!inShell && <header className="brand">
        <span className="brand-title">
          <button type="button" className="brand-name" title="Dashboard" onClick={() => onOpenDashboard && onOpenDashboard()}>PiCode</button>
        </span>
        <RailTabs tab={tab} selectTab={selectTab} apps={apps} pkgUpdates={userMenu?.pkgUpdates} tight={width < 260} onOpenClis={onOpenClis} />
      </header>
      }

      {tab === "pins" ? (
        <Pins />
      ) : tab === "apps" ? (
        <AppsGrid apps={apps} nativeApps={nativeApps} onOpen={onOpenApp} webapps={webapps} webappsErr={webappsErr} webappsLoaded={webappsLoaded} onRetryWebapps={onRetryWebapps} onOpenWebapp={onOpenWebapp} onSavedWebapp={onSavedWebapp} onRemoveWebapp={onRemoveWebapp} onRefreshWebapp={onRefreshWebapp} onClearWebappData={onClearWebappData} desktop={desktop} />
      ) : tab === "terms" ? (
      <div className="side-section">
        <div className="pins-head">
          <span className="pins-title">Terminals</span>
          <button type="button" className="ws-icon-btn" title="New terminal" onClick={() => onNewTerm && onNewTerm()}><IconPlus /></button>
          <button type="button" className="ws-icon-btn" title="Terminal defaults" onClick={() => { location.hash = "#/termset"; }}><IconSettings /></button>
        </div>
        <div className="side-scroll">
        {freeTerminals(terminals).length === 0 ? (
          <p className="side-empty pins-empty">No terminals yet. <button type="button" className="side-empty-act" onClick={() => onNewTerm && onNewTerm()}>New terminal</button></p>
        ) : (
          <ul className="ws-list">{freeTerminals(terminals).map(termRow)}</ul>
        )}
        </div>
      </div>
      ) : tab === "agents" ? (
      <div className="side-section">
        <div className="pins-head">
          <span className="pins-title">Agents</span>
          <button type="button" className="ws-icon-btn" title="New agent" onClick={() => onNewFree()}><IconPlus /></button>
        </div>
        <div className="side-scroll">
        {sortedFreeAgents.length === 0 ? (
          <p className="side-empty pins-empty">No free agents yet. <button type="button" className="side-empty-act" onClick={() => onNewFree()}>New agent</button></p>
        ) : (
          <ul className="ws-list">{sortedFreeAgents.map((ag) => agentRow(ag, null))}</ul>
        )}
        </div>
      </div>
      ) : (
      <div className="side-section">
        <div className="pins-head">
          <span className="pins-title">Workspaces</span>
          <button id="btn-new" type="button" className="ws-icon-btn" title="New workspace" onClick={() => onNew()}><IconPlus /></button>
        </div>
        <div className="side-scroll">
        {workspaces.length === 0 ? (
          <p className="side-empty pins-empty">No workspaces yet. <button type="button" className="side-empty-act" onClick={() => onNew()}>Add workspace</button></p>
        ) : (
        <ul id="ws-list" className="ws-list">
          {workspaces.map((ws) => {
            const wsTerms = workspaceTerminals(terminals, ws.id);
            // Not a line on the card any more (the rows below already carry
            // path and branch); the menu still asks whether this folder is a
            // repository before offering its history.
            const wsRepo = wsLine(ws);
            const wsAgents = agentsOf(ws);
            return (
            <li key={ws.id} className="ws-group">
              <div className="ws-group-head" onClick={() => toggleWs(ws.id)}>
                <span className={"ws-chev" + (isOpen(ws.id) ? " open" : "")}><IconChevronRight /></span>
                <span className="tree-icon"><WsFavicon ws={ws} /></span>
                <span className="ws-group-name" title={ws.path}>{ws.name}</span>
                <span className="tree-meta">{!isOpen(ws.id) ? collapsedMark(wsAgents, wsTerms) : null}</span>
                {/* Two controls, not five: one creates, one holds the rest
                    (VS Code caps a row at three; the child rows already read
                    this way). Both stay visible — a control hidden until
                    hover is unreachable by touch (WCAG 1.4.13). */}
                <span className="ws-group-actions" onClick={(e) => e.stopPropagation()}>
                  <DropdownMenu.Root><DropdownMenu.Trigger asChild><button type="button" className="ws-icon-btn" title="New in this folder" aria-label={"New in " + ws.name}><IconPlus /></button></DropdownMenu.Trigger><DropdownMenu.Portal><DropdownMenu.Content className="um-popover" side="bottom" align="end" sideOffset={6} collisionPadding={12}>
                    <DropdownMenu.Item className="um-item" onSelect={() => onNewAgent && onNewAgent(ws.id)}>Agent</DropdownMenu.Item>
                    <DropdownMenu.Item className="um-item" onSelect={() => onNewTerm?.(ws.id)}>Shell terminal</DropdownMenu.Item>
                    <DropdownMenu.Item className="um-item" onSelect={() => onNewCliPrincipal && onNewCliPrincipal(ws)}>Agent CLI</DropdownMenu.Item>
                  </DropdownMenu.Content></DropdownMenu.Portal></DropdownMenu.Root>
                  <RowMenu label={ws.name}>
                    <RowMenuItem onSelect={() => { location.hash = "#/clis/messages/" + encodeURIComponent("workspace:" + ws.id); }}><IconSession size={13} /> Communication</RowMenuItem>
                    <RowMenuItem onSelect={() => onFileTree && onFileTree("workspace", ws.id, ws.name)}><IconFolder size={13} /> Files</RowMenuItem>
                    {wsRepo.git ? <RowMenuItem onSelect={() => onGitGraph && onGitGraph("workspace", ws.id, ws.name)}><IconGit size={13} /> Git graph</RowMenuItem> : null}
                    {/* Sessions read through an agent — an empty workspace
                        answers 409, so it does not offer the item (ADR-0027). */}
                    {wsAgents.length ? <RowMenuItem onSelect={() => onSessions && onSessions(ws.id)}><IconSession size={13} /> Sessions</RowMenuItem> : null}
                    <RowMenuItem danger onSelect={() => onRemove(ws)}><IconX size={13} /> Remove workspace</RowMenuItem>
                  </RowMenu>
                </span>
              </div>
              {isOpen(ws.id) ? (
                (wsAgents.length || wsTerms.length) ? (
                  <>
                    {wsAgents.length ? <ul className="ws-list tree-children">{wsAgents.map((ag) => agentRow(ag, ws))}</ul> : null}
                    {wsTerms.length ? <ul className="ws-list tree-children">{wsTerms.map(termRow)}</ul> : null}
                  </>
                ) : (
                  <p className="side-empty">
                    Empty — <button type="button" className="side-empty-act" onClick={() => onNewAgent && onNewAgent(ws.id)}>add an agent</button>
                    {" or "}<button type="button" className="side-empty-act" onClick={() => onNewTerm && onNewTerm(ws.id)}>a terminal</button>.
                  </p>
                )
              ) : null}
            </li>
            );
          })}
        </ul>
        )}
        </div>
      </div>
      )}

      <footer className="side-foot">
        <UserMenu {...userMenu} onShare={() => setShareOpen(true)} />
      </footer>
      <div id="sidebar-sizer" className="sidebar-sizer" title="Drag to resize" onPointerDown={onSizerDown} />
      <ShareDrawer open={shareOpen} onClose={() => setShareOpen(false)} />
    </aside>
  );
}
