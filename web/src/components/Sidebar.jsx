import { useState } from "react";
import UserMenu from "./UserMenu.jsx";
import ConfigFields from "./ConfigFields.jsx";
import ShareDrawer from "./ShareDrawer.jsx";
import { IconQR, IconChat, IconTerminal, IconPlus, IconFolder, IconAgent, IconPlay, IconStop, IconX, IconCheck } from "./Icons.jsx";
import { agentsOf, displayAgentName } from "../lib/tree.js";

const SIDE_MIN = 180;
const SIDE_MAX = 480;
const SIDE_KEY = "picode-sidebar-w";

export default function Sidebar({
  version, workspaces, selectedId, showForm, formError,
  onNew, onCancel, onSubmit, onSelect, onRun, onStop, onRemove,
  userMenu, catalog, newCfg, onNewCfg, termView, onChat, onTerm,
  freeAgents, onNewFree, onNewAgent, onRemoveAgent, formKind, formWs,
}) {
  const [width, setWidth] = useState(() => {
    const n = parseInt(localStorage.getItem(SIDE_KEY) || "", 10);
    return Number.isFinite(n) ? Math.min(SIDE_MAX, Math.max(SIDE_MIN, n)) : 244;
  });
  const [resizing, setResizing] = useState(false);
  const [shareOpen, setShareOpen] = useState(false);

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

  function agentRow(ag, ws) {
    const mode = ag.mode || "stopped";
    const label = displayAgentName(ag, ws);
    return (
      <li
        key={ag.id}
        className={"ws-item" + (ag.id === selectedId ? " active" : "")}
        onClick={(e) => { if (e.target.closest("button")) return; onSelect(ag.id); }}
      >
        <div className="ws-row1">
          <span className={"ws-dot" + (mode !== "stopped" ? " running" : "")} />
          <span className="ws-name" title={label}>{label}</span>
          <span className="ws-actions">
            {mode === "stopped"
              ? <button type="button" className="ws-icon-btn" title="Run" onClick={() => onRun(ag.id)}><IconPlay /></button>
              : <button type="button" className="ws-icon-btn" title="Stop" onClick={() => onStop(ag.id)}><IconStop size={12} /></button>}
            <button type="button" className="ws-icon-btn danger" title="Remove agent" onClick={() => onRemoveAgent ? onRemoveAgent(ag) : onRemove(ws)}><IconX size={12} /></button>
          </span>
        </div>
        <div className="ws-row2">
          <span className="ws-path" title={ws ? ws.path : (ag.workPath || "")}>{ws ? ws.path : (ag.workPath || "~/.picode/work")}</span>
          <span className="ws-modes" role="radiogroup" aria-label="View" onClick={(e) => e.stopPropagation()}>
            <button type="button" role="radio" className="ws-mode-btn" aria-checked={ag.id === selectedId && !termView} title="Chat" onClick={() => onChat && onChat(ag.id)}><IconChat size={14} /></button>
            <button type="button" role="radio" className="ws-mode-btn" aria-checked={ag.id === selectedId && !!termView} title="Terminal" onClick={() => onTerm && onTerm(ag.id)}><IconTerminal size={14} /></button>
          </span>
        </div>
      </li>
    );
  }

  return (
    <aside id="sidebar" className={resizing ? "resizing" : ""} style={{ width }}>
      <header className="brand">
        <span className="brand-name">PiCode</span>
        <span className="brand-ver" id="ver">{version ? "v" + version : "v—"}</span>
        <button type="button" id="btn-share" className="brand-qr" title="Open on phone" onClick={() => setShareOpen(true)}>
          <IconQR />
        </button>
      </header>

      <div className="side-section">
        <div className="side-head">
          <span className="side-title" title="Agents"><IconAgent /></span>
          <button type="button" className="ws-icon-btn" title="New agent" onClick={onNewFree}><IconPlus /></button>
        </div>
        {showForm && formKind === "free" ? (
          <form className="form-new" onSubmit={onSubmit}>
            <input name="name" type="text" placeholder="Name" autoComplete="off" />
            <input name="path" type="text" placeholder="Folder (optional — ~/.picode/work/name)" autoComplete="off" />
            <ConfigFields catalog={catalog} provider={newCfg.provider} model={newCfg.model} thinking={newCfg.thinking} onChange={onNewCfg} idPrefix="free" />
            <div className="form-actions">
              <button type="submit" className="btn btn-primary btn-sm" title="Add"><IconCheck size={14} /></button>
              <button type="button" className="ws-icon-btn" title="Cancel" onClick={onCancel}><IconX size={12} /></button>
            </div>
            <p className="form-error" hidden={!formError}>{formError}</p>
          </form>
        ) : null}
        <ul className="ws-list">
          {(freeAgents || []).map((ag) => agentRow(ag, null))}
        </ul>

        <div className="side-head" style={{ marginTop: 14 }}>
          <span className="side-title" title="Workspaces"><IconFolder /></span>
          <button id="btn-new" type="button" className="ws-icon-btn" title="New workspace" onClick={onNew}><IconPlus /></button>
        </div>

        {showForm && formKind === "workspace" ? (
        <form id="form-new" className="form-new" onSubmit={onSubmit}>
          <input id="inp-name" name="name" type="text" placeholder="Name (e.g. My App)" autoComplete="off" />
          <input id="inp-path" name="path" type="text" placeholder="Folder path (e.g. ~/code/my-app)" autoComplete="off" />
          <ConfigFields catalog={catalog} provider={newCfg.provider} model={newCfg.model} thinking={newCfg.thinking} onChange={onNewCfg} idPrefix="new" />
          <div className="form-actions">
            <button type="submit" className="btn btn-primary btn-sm" title="Add"><IconCheck size={14} /></button>
            <button type="button" id="btn-cancel" className="ws-icon-btn" title="Cancel" onClick={onCancel}><IconX size={12} /></button>
          </div>
          <p id="form-error" className="form-error" hidden={!formError}>{formError}</p>
        </form>
        ) : null}

        <ul id="ws-list" className="ws-list">
          {workspaces.map((ws) => (
            <li key={ws.id} className="ws-group">
              <div className="ws-group-head">
                <span className="ws-group-name" title={ws.path}>{ws.name}</span>
                <button type="button" className="ws-icon-btn" title="New agent in this folder" onClick={() => onNewAgent && onNewAgent(ws.id)}><IconPlus /></button>
                <button type="button" className="ws-icon-btn danger" title="Remove workspace (files untouched)" onClick={() => onRemove(ws)}><IconX size={12} /></button>
              </div>
              {showForm && formKind === "agent" && formWs === ws.id ? (
                <form className="form-new" onSubmit={onSubmit}>
                  <input name="name" type="text" placeholder="Agent name" autoComplete="off" />
                  <ConfigFields catalog={catalog} provider={newCfg.provider} model={newCfg.model} thinking={newCfg.thinking} onChange={onNewCfg} idPrefix={"ag-" + ws.id} />
                  <div className="form-actions">
                    <button type="submit" className="btn btn-primary btn-sm" title="Add"><IconCheck size={14} /></button>
                    <button type="button" className="ws-icon-btn" title="Cancel" onClick={onCancel}><IconX size={12} /></button>
                  </div>
                  <p className="form-error" hidden={!formError}>{formError}</p>
                </form>
              ) : null}
              <ul className="ws-list nested">
                {agentsOf(ws).map((ag) => agentRow(ag, ws))}
              </ul>
            </li>
          ))}
        </ul>
      </div>

      <footer className="side-foot">
        <UserMenu {...userMenu} />
      </footer>
      <div id="sidebar-sizer" className="sidebar-sizer" title="Drag to resize" onPointerDown={onSizerDown} />
      <ShareDrawer open={shareOpen} onClose={() => setShareOpen(false)} />
    </aside>
  );
}
