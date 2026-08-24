import { useEffect, useRef, useState } from "react";
import { IconKind } from "./Icons.jsx";

const KINDS = [
  { id: "prompt", label: "Prompt" },
  { id: "steer", label: "Steer" },
  { id: "follow_up", label: "Follow-up" },
];

export default function KindChip({ value, onChange }) {
  const [open, setOpen] = useState(false);
  const wrap = useRef(null);
  const cur = KINDS.find((k) => k.id === value) || KINDS[0];

  useEffect(() => {
    if (!open) return;
    const onDoc = (e) => {
      if (wrap.current && !wrap.current.contains(e.target)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [open]);

  return (
    <div className="cockpit-chip-wrap" ref={wrap}>
      <button
        type="button"
        id="task-kind"
        className="cockpit-chip"
        title="How the message is delivered"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
      >
        <span className="cockpit-chip-icon"><IconKind /></span>
        <span className="cockpit-chip-label">{cur.label}</span>
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" aria-hidden="true">
          <path d="m6 9 6 6 6-6" />
        </svg>
      </button>
      {open && (
        <div className="cockpit-pop cockpit-pop-sm" role="listbox">
          {KINDS.map((k) => (
            <button
              key={k.id}
              type="button"
              className={"cockpit-opt" + (k.id === cur.id ? " active" : "")}
              onClick={() => { onChange(k.id); setOpen(false); }}
            >
              {k.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
