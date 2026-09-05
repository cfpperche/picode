import { useEffect, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { parseRoute, appRoute } from "../lib/routes.js";
import UserMenu from "./UserMenu.jsx";
import ShareDrawer, { OPEN_EVENT } from "./ShareDrawer.jsx";
import { IconTerminal, IconPlus, IconFolder, IconFolders, IconAgent, IconX, IconChevronRight, IconPin, IconSession, IconSettings, IconGrid } from "./Icons.jsx";
import Pins from "./Pins.jsx";
import AppsGrid from "./AppsGrid.jsx";
import { aggregateBadge } from "../lib/appPrimitives.js";
import { agentsOf, displayAgentName } from "../lib/tree.js";
import { freeTerminals, workspaceTerminals } from "../lib/termGroups.js";
import ProviderFaces from "./ProviderFaces.jsx";
import { AgentRow, TermRow } from "./WorkspaceRows.jsx";

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
const TAB_KEY = "picode-side-tab";

export default function Sidebar({
  version, workspaces, selectedId,
  onNew, onSelect, onRun, onStop, onRemove,
  userMenu, termView, onChat, onTerm,
  freeAgents, onNewFree, onNewAgent, onRemoveAgent, onRenameAgent,
  workingId,
  workingIds,
  waitingId,
  checklists,
  terminals, onNewTerm, onSelectTerm, onRemoveTerm, onRenameTerm, onSessions,
  onGitGraph,
  onFileTree,
  onOpenDashboard,
  apps, onOpenApp,
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
  const [tab, setTab] = useState(() => {
    try {
      const v = localStorage.getItem(TAB_KEY);
      if (v === "pins" || v === "terms" || v === "agents" || v === "apps") return v;
      return "workspaces";
    } catch { return "workspaces"; }
  });
  function selectTab(next) {
    setTab(next);
    try { localStorage.setItem(TAB_KEY, next); } catch { /* ignore */ }
  }
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

  function collapsedMark(agents) {
    return <ProviderFaces agents={agents} />;
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
      />
    );
  }

  const sortedFreeAgents = [...(freeAgents || [])].sort((a, b) =>
    displayAgentName(a, null).localeCompare(displayAgentName(b, null), undefined, { sensitivity: "base" }));

  const appBadge = aggregateBadge(apps);

  return (
    <aside id="sidebar" className={resizing ? "resizing" : ""} style={{ width }}>
      <header className="brand">
        <span className="brand-title">
          <button type="button" className="brand-name" title="Dashboard" onClick={() => onOpenDashboard && onOpenDashboard()}>PiCode</button>
          {/* Five tabs eat the header (ADR-0036); below ~286px the version
              would push the name into ellipsis, so it yields (it lives in
              the user menu too). The name never truncates. */}
          {width >= 286 ? <span className="brand-ver" id="ver" title={version ? "v" + version : ""}>{version ? "v" + version : "v—"}</span> : null}
        </span>
        <nav className={"brand-tabs" + (width < 240 ? " brand-tabs-tight" : "")} role="tablist" aria-label="Sidebar">
          <button type="button" role="tab" className="brand-tab" aria-selected={tab === "workspaces"} title="Workspaces" aria-label="Workspaces" onClick={() => selectTab("workspaces")}><IconFolders size={16} /></button>
          <button type="button" role="tab" className="brand-tab" aria-selected={tab === "agents"} title="Agents" aria-label="Agents" onClick={() => selectTab("agents")}><IconAgent size={16} /></button>
          <button type="button" role="tab" className="brand-tab" aria-selected={tab === "terms"} title="Terminals" aria-label="Terminals" onClick={() => selectTab("terms")}><IconTerminal size={16} /></button>
          <button type="button" role="tab" className="brand-tab" aria-selected={tab === "apps"} title="Apps" aria-label="Apps" onClick={() => selectTab("apps")}>
            <IconGrid size={16} />
            {appBadge.count > 0 ? <span className="brand-tab-badge">{appBadge.count > 99 ? "99+" : appBadge.count}</span> : appBadge.dot ? <span className="brand-tab-dot" /> : null}
          </button>
          <button type="button" role="tab" className="brand-tab" aria-selected={tab === "pins"} title="Pins" aria-label="Pins" onClick={() => selectTab("pins")}><IconPin size={16} /></button>
        </nav>
      </header>

      {tab === "pins" ? (
        <Pins />
      ) : tab === "apps" ? (
        <AppsGrid apps={apps} onOpen={onOpenApp} />
      ) : tab === "terms" ? (
      <div className="side-section">
        <div className="pins-head">
          <span className="pins-title">Terminals</span>
          <button type="button" className="ws-icon-btn" title="New terminal" onClick={() => onNewTerm && onNewTerm()}><IconPlus /></button>
          <button type="button" className="ws-icon-btn" title="Terminal defaults" onClick={() => { location.hash = "#/termset"; }}><IconSettings /></button>
        </div>
        {freeTerminals(terminals).length === 0 ? (
          <p className="side-empty pins-empty">No terminals yet. <button type="button" className="side-empty-act" onClick={() => onNewTerm && onNewTerm()}>New terminal</button></p>
        ) : (
          <ul className="ws-list">{freeTerminals(terminals).map(termRow)}</ul>
        )}
      </div>
      ) : tab === "agents" ? (
      <div className="side-section">
        <div className="pins-head">
          <span className="pins-title">Agents</span>
          <button type="button" className="ws-icon-btn" title="New agent" onClick={() => onNewFree()}><IconPlus /></button>
        </div>
        {sortedFreeAgents.length === 0 ? (
          <p className="side-empty pins-empty">No free agents yet. <button type="button" className="side-empty-act" onClick={() => onNewFree()}>New agent</button></p>
        ) : (
          <ul className="ws-list">{sortedFreeAgents.map((ag) => agentRow(ag, null))}</ul>
        )}
      </div>
      ) : (
      <div className="side-section">
        <div className="pins-head">
          <span className="pins-title">Workspaces</span>
          <button id="btn-new" type="button" className="ws-icon-btn" title="New workspace" onClick={() => onNew()}><IconPlus /></button>
        </div>
        {workspaces.length === 0 ? (
          <p className="side-empty pins-empty">No workspaces yet. <button type="button" className="side-empty-act" onClick={() => onNew()}>Add workspace</button></p>
        ) : (
        <ul id="ws-list" className="ws-list">
          {workspaces.map((ws) => {
            const wsTerms = workspaceTerminals(terminals, ws.id);
            return (
            <li key={ws.id} className="ws-group">
              <div className="ws-group-head" onClick={() => toggleWs(ws.id)}>
                <span className={"ws-chev" + (isOpen(ws.id) ? " open" : "")}><IconChevronRight /></span>
                <span className="tree-icon"><WsFavicon ws={ws} /></span>
                <span className="ws-group-name" title={ws.path}>{ws.name}</span>
                <span className="tree-meta">{!isOpen(ws.id) ? collapsedMark(agentsOf(ws)) : null}</span>
                <span className="ws-group-actions" onClick={(e) => e.stopPropagation()}>
                  <button type="button" className="ws-icon-btn" title="Files in this folder" onClick={() => onFileTree && onFileTree("workspace", ws.id, ws.name)}><IconFolder size={12} /></button>
                  <button type="button" className="ws-icon-btn" title="New agent in this folder" onClick={() => onNewAgent && onNewAgent(ws.id)}><IconPlus /></button>
                  <DropdownMenu.Root><DropdownMenu.Trigger asChild><button type="button" className="ws-icon-btn" title="New terminal in this folder" aria-label={"New terminal in " + ws.name}><IconTerminal size={12} /></button></DropdownMenu.Trigger><DropdownMenu.Portal><DropdownMenu.Content className="um-popover" side="bottom" align="start" sideOffset={6} collisionPadding={12}>
                    <DropdownMenu.Item className="um-item" onSelect={() => onNewTerm?.(ws.id)}>Shell terminal</DropdownMenu.Item>
                    <DropdownMenu.Item className="um-item" onSelect={() => { location.hash = "#/clis/new/pi?workspace=" + encodeURIComponent(ws.id); }}>Agent CLI terminal</DropdownMenu.Item>
                  </DropdownMenu.Content></DropdownMenu.Portal></DropdownMenu.Root>
                  <button type="button" className="ws-icon-btn" title="Sessions — every Pi session in this folder" aria-label={"Sessions for " + ws.name} onClick={() => onSessions && onSessions(ws.id)}><IconSession /></button>
                  <button type="button" className="ws-icon-btn danger" title="Remove workspace" onClick={() => onRemove(ws)}><IconX size={12} /></button>
                </span>
              </div>
              {isOpen(ws.id) ? (
                (agentsOf(ws).length || wsTerms.length) ? (
                  <>
                    {agentsOf(ws).length ? <ul className="ws-list tree-children">{agentsOf(ws).map((ag) => agentRow(ag, ws))}</ul> : null}
                    {wsTerms.length ? <ul className="ws-list tree-children">{wsTerms.map(termRow)}</ul> : null}
                  </>
                ) : <p className="side-empty">Empty — add an agent or a terminal.</p>
              ) : null}
            </li>
            );
          })}
        </ul>
        )}
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
