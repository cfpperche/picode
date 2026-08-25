import { useEffect, useRef, useState } from "react";

export default function SessionBar({ sessions, current, onNew, onResume, onRename, end }) {
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
        <span className="cockpit-chip-label">{label}</span>
        <span className="session-count">{(sessions || []).length}</span>
      </button>
      <button type="button" className="cockpit-chip" onClick={onNew} title="New session">New</button>
      {end ? <div className="session-bar-end">{end}</div> : null}
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
