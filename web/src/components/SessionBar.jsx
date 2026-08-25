import { useEffect, useRef, useState } from "react";
import { IconSession, IconPlus, IconChat } from "./Icons.jsx";

export default function SessionBar({ sessions, current, onNew, onResume, onRename, onChat, inline }) {
  const [open, setOpen] = useState(false);
  const [q, setQ] = useState("");
  const wrap = useRef(null);
  const searchRef = useRef(null);
  const cur = (sessions || []).find((s) => s.path === current);
  const label = cur ? (cur.name || cur.preview || "Current session") : "Sessions";

  useEffect(() => {
    if (!open) return;
    setQ("");
    const onDoc = (e) => {
      if (wrap.current && !wrap.current.contains(e.target)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    requestAnimationFrame(() => searchRef.current && searchRef.current.focus());
    return () => document.removeEventListener("mousedown", onDoc);
  }, [open]);

  const needle = q.trim().toLowerCase();
  const filtered = (sessions || []).filter((s) => {
    if (!needle) return true;
    const hay = ((s.name || "") + " " + (s.preview || "") + " " + (s.updatedAt || "")).toLowerCase();
    return hay.includes(needle);
  });

  return (
    <div className={"session-bar" + (inline ? " inline" : "")} ref={wrap}>
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
          <input
            ref={searchRef}
            className="combo-input"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Search sessions"
            aria-label="Search sessions"
            onKeyDown={(e) => e.stopPropagation()}
          />
          <div className="session-list">
            {filtered.length === 0 && <div className="combo-empty">{(sessions || []).length === 0 ? "No sessions yet" : "No matches"}</div>}
            {filtered.map((s) => (
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
