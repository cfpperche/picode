import { useState } from "react";
import UserMenu from "./UserMenu.jsx";
import ConfigFields from "./ConfigFields.jsx";

const SIDE_MIN = 180;
const SIDE_MAX = 480;
const SIDE_KEY = "picode-sidebar-w";

export default function Sidebar({
  version, workspaces, selectedId, showForm, formError,
  onNew, onCancel, onSubmit, onSelect, onRun, onStop, onRemove,
  userMenu, catalog, newCfg, onNewCfg,
}) {
  const [width, setWidth] = useState(() => {
    const n = parseInt(localStorage.getItem(SIDE_KEY) || "", 10);
    return Number.isFinite(n) ? Math.min(SIDE_MAX, Math.max(SIDE_MIN, n)) : 244;
  });
  const [resizing, setResizing] = useState(false);

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

  return (
    <aside id="sidebar" className={resizing ? "resizing" : ""} style={{ width }}>
      <header className="brand">
        <span className="brand-name">PiCode</span>
        <span className="brand-ver" id="ver">{version ? "v" + version : "v—"}</span>
      </header>

      <div className="side-section">
        <div className="side-head">
          <span className="side-title">Agents</span>
          <button id="btn-new" className="btn btn-ghost btn-sm" title="Add a workspace" onClick={onNew}>+ New</button>
        </div>

        <form id="form-new" className="form-new" hidden={!showForm} onSubmit={onSubmit}>
          <input id="inp-name" name="name" type="text" placeholder="Name (e.g. My App)" autoComplete="off" />
          <input id="inp-path" name="path" type="text" placeholder="Folder path (e.g. ~/code/my-app)" autoComplete="off" />
          <ConfigFields catalog={catalog} provider={newCfg.provider} model={newCfg.model} thinking={newCfg.thinking} onChange={onNewCfg} idPrefix="new" />
          <div className="form-actions">
            <button type="submit" className="btn btn-primary btn-sm">Add</button>
            <button type="button" id="btn-cancel" className="btn btn-ghost btn-sm" onClick={onCancel}>Cancel</button>
          </div>
          <p id="form-error" className="form-error" hidden={!formError}>{formError}</p>
        </form>

        <ul id="ws-list" className="ws-list">
          {workspaces.map((ws) => {
            const mode = ws.agent ? ws.agent.mode : "stopped";
            return (
              <li
                key={ws.id}
                className={"ws-item" + (ws.id === selectedId ? " active" : "")}
                onClick={(e) => { if (e.target.closest("button")) return; onSelect(ws.id); }}
              >
                <div className="ws-row1">
                  <span className={"ws-dot" + (mode !== "stopped" ? " running" : "")} />
                  <span className="ws-name" title={ws.name}>{ws.name}</span>
                  <span className="ws-actions">
                    {mode === "stopped"
                      ? <button className="btn btn-ghost btn-sm btn-managed" title="Run with the task panel" onClick={() => onRun(ws.id)}>Run</button>
                      : <button className="btn btn-ghost btn-sm btn-stop" title="Stop the agent" onClick={() => onStop(ws.id)}>Stop</button>}
                    <button className="btn btn-ghost btn-sm btn-danger btn-remove" title="Remove workspace (files untouched)" onClick={() => onRemove(ws)}>×</button>
                  </span>
                </div>
                <div className="ws-row2">
                  <span className="ws-path" title={ws.path}>{ws.path}</span>
                  <span className="ws-mode">{mode}</span>
                </div>
              </li>
            );
          })}
        </ul>
      </div>

      <footer className="side-foot">
        <UserMenu {...userMenu} />
      </footer>
      <div id="sidebar-sizer" className="sidebar-sizer" title="Drag to resize" onPointerDown={onSizerDown} />
    </aside>
  );
}
