import { useEffect, useRef, useState } from "react";
import { IconSession, IconPlus, IconChat } from "./Icons.jsx";

export default function SessionBar({ sessions, current, onNew, onResume, onRename, onChat }) {
  const [open, setOpen] = useState(false);
  const wrap = useRef(null);
  const cur = (sessions || []).find((s) => s.path === current);
  const label = cur ? (cur.name || cur.preview || "Current session") : "Sessions";

  useEffect(() => {
    if (!open) return;
    const onDoc = (e) => {
      if (wrap.current && !wrap.current.contains(e.target)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [open]);

  return (
    <div className="session-bar" ref={wrap}>
      <button
        type="button"
        id="session-picker"
        className="cockpit-chip"
        aria-expanded={open}
        title="Sessions"
        onClick={() => setOpen((o) => !o)}
      >
        <span className="cockpit-chip-icon"><IconSession /></span>
        <span className="cockpit-chip-label">{label}</span>
        <span className="session-count">{(sessions || []).length}</span>
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" aria-hidden="true">
          <path d="m6 9 6 6 6-6" />
        </svg>
      </button>
      <button type="button" className="cockpit-chip" onClick={onNew} title="New session">
        <span className="cockpit-chip-icon"><IconPlus /></span>
        <span className="cockpit-chip-label">New</span>
      </button>
      {onChat ? (
        <div className="session-bar-end">
          <button type="button" className="cockpit-chip" onClick={onChat} title="Chat">
            <span className="cockpit-chip-icon"><IconChat /></span>
            <span className="cockpit-chip-label">Chat</span>
          </button>
        </div>
      ) : null}
      {open && (
        <div className="session-pop" role="listbox">
          <div className="session-list">
            {(sessions || []).length === 0 && <div className="combo-empty">No sessions yet</div>}
            {(sessions || []).map((s) => (
              <button
                key={s.path}
                type="button"
                className={"cockpit-opt" + (s.path === current ? " selected" : "")}
                onClick={() => { if (s.path !== current) onResume(s.path); setOpen(false); }}
              >
                <span className="session-name">{s.name || s.preview || "Untitled"}</span>
                <span className="combo-hint">{shortDate(s.updatedAt || s.createdAt)}</span>
                {onRename ? (
                  <span
                    className="session-rename"
                    onClick={(e) => { e.stopPropagation(); onRename(s); }}
                  >Rename</span>
                ) : null}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function shortDate(iso) {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
}
