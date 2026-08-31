import { useEffect, useState } from "react";
import { parseRoute } from "../lib/routes.js";
import UserMenu from "./UserMenu.jsx";
import ShareDrawer from "./ShareDrawer.jsx";
import { IconChat, IconTerminal, IconPlus, IconFolder, IconAgent, IconPlay, IconStop, IconX, IconGit, IconChevronRight, IconPin, IconSession, IconSettings } from "./Icons.jsx";
import Pins from "./Pins.jsx";
import { agentsOf, displayAgentName } from "../lib/tree.js";
import { shortModel } from "../lib/chip.js";
import { repoLine, termLine } from "../lib/repoLine.js";
import { freeTerminals, workspaceTerminals } from "../lib/termGroups.js";
import ProviderFaces, { ProviderFace } from "./ProviderFaces.jsx";
import PiSpinner from "./PiSpinner.jsx";

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
  terminals, onNewTerm, onSelectTerm, onRemoveTerm, onRenameTerm, onSessions,
  onGitGraph,
}) {
  const [width, setWidth] = useState(() => {
    const n = parseInt(localStorage.getItem(SIDE_KEY) || "", 10);
    return Number.isFinite(n) ? Math.min(SIDE_MAX, Math.max(SIDE_MIN, n)) : 244;
  });
  const [resizing, setResizing] = useState(false);
  const [shareOpen, setShareOpen] = useState(false);
  const [tab, setTab] = useState(() => {
    try {
      const v = localStorage.getItem(TAB_KEY);
      if (v === "pins" || v === "terms" || v === "workspaces") return v;
      return "agents";
    } catch { return "agents"; }
  });
  function selectTab(next) {
    setTab(next);
    try { localStorage.setItem(TAB_KEY, next); } catch { /* ignore */ }
  }
  useEffect(() => {
    const sync = () => { if (parseRoute() === "pins") selectTab("pins"); };
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
    const mode = ag.mode || "stopped";
    const label = displayAgentName(ag, ws);
    const model = shortModel(ag.model || "");
    const title = model ? label + " - " + model : label;
    const repo = repoLine(ag, ws);
    return (
      <li
        key={ag.id}
        className={"ws-item" + (ag.id === selectedId ? " active" : "")}
        onClick={(e) => { if (e.target.closest("button")) return; onSelect(ag.id); }}
      >
        <div className="ws-row1 tree-row">
          <span className="tree-spc" aria-hidden="true" />
          {ag.id === workingId || (workingIds || []).includes(ag.id) ? <PiSpinner /> : <ProviderFace agent={ag} />}
          <span className="ws-name" title={title}>
            <button type="button" className="ws-name-btn" title="Rename" onClick={() => onRenameAgent && onRenameAgent(ag, label)}>{label}</button>
            {model ? <span className="ws-model"> - {model}</span> : null}
          </span>
          {ag.id === waitingId ? <span className="ws-wait">Waiting</span> : null}
        </div>
        <div className="ws-row2 tree-row">
          <span className="tree-spc" aria-hidden="true" />
          {repo.git ? (
            <button
              type="button"
              className="tree-icon tree-icon-btn"
              title={"Git graph" + (repo.git.branch ? " — " + repo.git.branch : "")}
              onClick={(e) => { e.stopPropagation(); onGitGraph && onGitGraph("agent", ag.id, label); }}
            >
              <IconGit size={14} />
            </button>
          ) : (
            <span className="tree-icon"><IconFolder size={14} /></span>
          )}
          <span className="ws-path" title={ws ? ws.path : (ag.workPath || "")}>{repo.text}</span>
        </div>
        <span className="ws-actions">
          {mode === "stopped"
            ? <button type="button" className="ws-icon-btn" title="Run" onClick={() => onRun(ag.id)}><IconPlay /></button>
            : <button type="button" className="ws-icon-btn" title="Stop" onClick={() => onStop(ag.id)}><IconStop size={12} /></button>}
          <button type="button" className="ws-icon-btn danger" title="Remove agent" onClick={() => onRemoveAgent ? onRemoveAgent(ag) : onRemove(ws)}><IconX size={12} /></button>
          <button type="button" className="ws-icon-btn" title="Chat" aria-pressed={ag.id === selectedId && !termView} onClick={(e) => { e.stopPropagation(); onChat && onChat(ag.id); }}><IconChat size={14} /></button>
          <button type="button" className="ws-icon-btn" title="Terminal" aria-pressed={ag.id === selectedId && !!termView} onClick={(e) => { e.stopPropagation(); onTerm && onTerm(ag.id); }}><IconTerminal size={14} /></button>
        </span>
      </li>
    );
  }

  function termRow(t) {
    return (
      <li
        key={t.id}
        className={"ws-item" + (selectedId === "t:" + t.id ? " active" : "")}
        onClick={(e) => { if (e.target.closest("button")) return; onSelectTerm && onSelectTerm(t.id); }}
      >
        <div className="ws-row1">
          <span className="tree-icon"><IconTerminal size={14} /></span>
          <button type="button" className="ws-name ws-name-btn" title="Rename" onClick={() => onRenameTerm && onRenameTerm(t)}>{t.name}</button>
        </div>
        {(() => { const line = termLine(t); return (
        <div className="ws-row2">
          {line.git ? (
            <button type="button" className="tree-icon tree-icon-btn" title={"Git graph" + (line.git.branch ? " — " + line.git.branch : "")} onClick={(e) => { e.stopPropagation(); onGitGraph && onGitGraph("term", t.id, t.name); }}><IconGit size={14} /></button>
          ) : (
            <span className="tree-icon"><IconFolder size={14} /></span>
          )}
          <span className="ws-path" title={t.cwd}>{line.text}</span>
        </div>
        ); })()}
        <span className="ws-actions">
          <button type="button" className="ws-icon-btn danger" title="Remove terminal" onClick={() => onRemoveTerm && onRemoveTerm(t)}><IconX size={12} /></button>
          <button type="button" className="ws-icon-btn" title="Settings" onClick={() => { location.hash = "#/termset/" + encodeURIComponent(t.id); }}><IconSettings /></button>
        </span>
      </li>
    );
  }

  const sortedFreeAgents = [...(freeAgents || [])].sort((a, b) =>
    displayAgentName(a, null).localeCompare(displayAgentName(b, null), undefined, { sensitivity: "base" }));

  return (
    <aside id="sidebar" className={resizing ? "resizing" : ""} style={{ width }}>
      <header className="brand">
        <span className="brand-title">
          <span className="brand-name">PiCode</span>
          {/* Four tabs eat the header; below ~254px the version would push
              the name into ellipsis, so it yields (it lives in the user
              menu too). The name never truncates. */}
          {width >= 254 ? <span className="brand-ver" id="ver">{version ? "v" + version : "v—"}</span> : null}
        </span>
        <nav className="brand-tabs" role="tablist" aria-label="Sidebar">
          <button type="button" role="tab" className="brand-tab" aria-selected={tab === "agents"} title="Agents" aria-label="Agents" onClick={() => selectTab("agents")}><IconAgent size={16} /></button>
          <button type="button" role="tab" className="brand-tab" aria-selected={tab === "workspaces"} title="Workspaces" aria-label="Workspaces" onClick={() => selectTab("workspaces")}><IconFolder size={16} /></button>
          <button type="button" role="tab" className="brand-tab" aria-selected={tab === "terms"} title="Terminals" aria-label="Terminals" onClick={() => selectTab("terms")}><IconTerminal size={16} /></button>
          <button type="button" role="tab" className="brand-tab" aria-selected={tab === "pins"} title="Pins" aria-label="Pins" onClick={() => selectTab("pins")}><IconPin size={16} /></button>
        </nav>
      </header>

      {tab === "pins" ? (
        <Pins />
      ) : tab === "terms" ? (
      <div className="side-section side-terms">
        <div className="pins-head">
          <span className="pins-title">Terminals</span>
          <button type="button" className="ws-icon-btn" title="Terminal defaults" onClick={() => { location.hash = "#/termset"; }}><IconSettings /></button>
          <button type="button" className="ws-icon-btn" title="New terminal" onClick={() => onNewTerm && onNewTerm()}><IconPlus /></button>
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
              <div className="ws-group-head tree-row" onClick={() => toggleWs(ws.id)}>
                <span className={"ws-chev" + (isOpen(ws.id) ? " open" : "")}><IconChevronRight /></span>
                <span className="tree-icon"><IconFolder size={16} /></span>
                <span className="ws-group-name" title={ws.path}>{ws.name}</span>
                <span className="tree-meta">{!isOpen(ws.id) ? collapsedMark(agentsOf(ws)) : null}</span>
                <span className="ws-group-actions" onClick={(e) => e.stopPropagation()}>
                  <button type="button" className="ws-icon-btn" title="New agent in this folder" onClick={() => onNewAgent && onNewAgent(ws.id)}><IconPlus /></button>
                  <button type="button" className="ws-icon-btn" title="New terminal in this folder" onClick={() => onNewTerm && onNewTerm(ws.id)}><IconTerminal size={12} /></button>
                  <button type="button" className="ws-icon-btn" title="Sessions — every Pi session in this folder" aria-label={"Sessions for " + ws.name} onClick={() => onSessions && onSessions(ws.id)}><IconSession /></button>
                  <button type="button" className="ws-icon-btn danger" title="Remove workspace (files untouched)" onClick={() => onRemove(ws)}><IconX size={12} /></button>
                </span>
              </div>
              {isOpen(ws.id) ? (
                (agentsOf(ws).length || wsTerms.length) ? (
                  <>
                    {agentsOf(ws).length ? <ul className="ws-list tree-children">{agentsOf(ws).map((ag) => agentRow(ag, ws))}</ul> : null}
                    {wsTerms.length ? <ul className="ws-list tree-children side-terms">{wsTerms.map(termRow)}</ul> : null}
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
