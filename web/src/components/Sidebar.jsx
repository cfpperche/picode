import { useEffect, useState } from "react";
import { parseRoute } from "../lib/routes.js";
import UserMenu from "./UserMenu.jsx";
import ShareDrawer from "./ShareDrawer.jsx";
import { IconChat, IconTerminal, IconPlus, IconFolder, IconAgent, IconPlay, IconStop, IconX, IconGit, IconChevronRight, IconPin, IconPencil, IconSession } from "./Icons.jsx";
import Pins from "./Pins.jsx";
import { agentsOf, displayAgentName } from "../lib/tree.js";
import { shortModel } from "../lib/chip.js";
import { repoLine } from "../lib/repoLine.js";
import { workspaceAgents } from "../lib/providerIcon.js";
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
  freeAgents, onNewFree, onNewAgent, onRemoveAgent,
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
      if (v === "pins" || v === "terms") return v;
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
          <span className="ws-name" title={title}>{label}{model ? <span className="ws-model"> - {model}</span> : null}</span>
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

  return (
    <aside id="sidebar" className={resizing ? "resizing" : ""} style={{ width }}>
      <header className="brand">
        <span className="brand-title">
          <span className="brand-name">PiCode</span>
          <span className="brand-ver" id="ver">{version ? "v" + version : "v—"}</span>
        </span>
        <nav className="brand-tabs" role="tablist" aria-label="Sidebar">
          <button type="button" role="tab" className="brand-tab" aria-selected={tab === "agents"} title="Agents" aria-label="Agents" onClick={() => selectTab("agents")}><IconAgent size={16} /></button>
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
          <button type="button" className="ws-icon-btn" title="New terminal" onClick={() => onNewTerm && onNewTerm()}><IconPlus /></button>
        </div>
        {(terminals || []).length === 0 ? (
          <p className="side-empty pins-empty">No terminals yet</p>
        ) : (
          <ul className="ws-list">
            {[...(terminals || [])].sort((a, b) => String(a.name || "").localeCompare(String(b.name || ""), undefined, { sensitivity: "base" })).map((t) => (
              <li
                key={t.id}
                className={"ws-item" + (selectedId === "t:" + t.id ? " active" : "")}
                onClick={(e) => { if (e.target.closest("button")) return; onSelectTerm && onSelectTerm(t.id); }}
              >
                <div className="ws-row1">
                  <span className="tree-icon"><IconTerminal size={14} /></span>
                  <span className="ws-name" title={t.cwd} onDoubleClick={(e) => { e.stopPropagation(); onRenameTerm && onRenameTerm(t); }}>{t.name}</span>
                </div>
                <div className="ws-row2">
                  <span className="ws-path" title={t.cwd}>{t.cwd}</span>
                </div>
                <span className="ws-actions">
                  {t.git ? (
                    <button type="button" className="ws-icon-btn" title={"Git graph" + (t.git.branch ? " — " + t.git.branch : "")} onClick={() => onGitGraph && onGitGraph("term", t.id, t.name)}><IconGit size={14} /></button>
                  ) : null}
                  <button type="button" className="ws-icon-btn" title="Rename" onClick={() => onRenameTerm && onRenameTerm(t)}><IconPencil /></button>
                  <button type="button" className="ws-icon-btn danger" title="Remove terminal" onClick={() => onRemoveTerm && onRemoveTerm(t)}><IconX size={12} /></button>
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
      ) : (
      <div className="side-section">
        <div className="side-head tree-row" onClick={() => toggleWs("sec-agents")}>
          <span className={"ws-chev" + (isOpen("sec-agents") ? " open" : "")}><IconChevronRight /></span>
          <span className="tree-icon"><IconAgent size={16} /></span>
          <span className="side-title">Agents</span>
          <span className="tree-meta">{!isOpen("sec-agents") ? collapsedMark(freeAgents) : null}</span>
          <button type="button" className="ws-icon-btn" title="New agent" onClick={(e) => { e.stopPropagation(); onNewFree(); }}><IconPlus /></button>
        </div>
        {isOpen("sec-agents") ? (
          (freeAgents || []).length
            ? <ul className="ws-list tree-children">{(freeAgents || []).map((ag) => agentRow(ag, null))}</ul>
            : <p className="side-empty">No agents</p>
        ) : null}

        <div className="side-head tree-row" style={{ marginTop: 14 }} onClick={() => toggleWs("sec-workspaces")}>
          <span className={"ws-chev" + (isOpen("sec-workspaces") ? " open" : "")}><IconChevronRight /></span>
          <span className="tree-icon"><IconFolder size={16} /></span>
          <span className="side-title">Workspaces</span>
          <span className="tree-meta">{!isOpen("sec-workspaces") ? collapsedMark(workspaceAgents(workspaces)) : null}</span>
          <button id="btn-new" type="button" className="ws-icon-btn" title="New workspace" onClick={(e) => { e.stopPropagation(); onNew(); }}><IconPlus /></button>
        </div>

        {isOpen("sec-workspaces") ? (
        workspaces.length === 0 ? (
          <p className="side-empty">No workspaces</p>
        ) : (
        <ul id="ws-list" className="ws-list tree-children">
          {workspaces.map((ws) => (
            <li key={ws.id} className="ws-group">
              <div className="ws-group-head tree-row" onClick={() => toggleWs(ws.id)}>
                <span className={"ws-chev" + (isOpen(ws.id) ? " open" : "")}><IconChevronRight /></span>
                <span className="tree-icon"><IconFolder size={16} /></span>
                <span className="ws-group-name" title={ws.path}>{ws.name}</span>
                <span className="tree-meta">{!isOpen(ws.id) ? collapsedMark(agentsOf(ws)) : null}</span>
                <span className="ws-group-actions" onClick={(e) => e.stopPropagation()}>
                  <button type="button" className="ws-icon-btn" title="New agent in this folder" onClick={() => onNewAgent && onNewAgent(ws.id)}><IconPlus /></button>
                  <button type="button" className="ws-icon-btn" title="Sessions — every Pi session in this folder" aria-label={"Sessions for " + ws.name} onClick={() => onSessions && onSessions(ws.id)}><IconSession /></button>
                  <button type="button" className="ws-icon-btn danger" title="Remove workspace (files untouched)" onClick={() => onRemove(ws)}><IconX size={12} /></button>
                </span>
              </div>
              {isOpen(ws.id) ? (
                agentsOf(ws).length
                  ? <ul className="ws-list tree-children">{agentsOf(ws).map((ag) => agentRow(ag, ws))}</ul>
                  : <p className="side-empty">No agents</p>
              ) : null}
            </li>
          ))}
        </ul>
        )
        ) : null}
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
