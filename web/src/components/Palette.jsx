import { useEffect, useMemo, useRef, useState } from "react";

export default function Palette({ open, workspaces, onClose, onRun }) {
  const [q, setQ] = useState("");
  const [idx, setIdx] = useState(0);
  const inputRef = useRef(null);

  const actions = useMemo(() => buildActions(workspaces), [workspaces]);
  const filtered = useMemo(() => {
    const s = q.trim().toLowerCase();
    if (!s) return actions;
    return actions.filter((a) => a.label.toLowerCase().includes(s) || a.group.toLowerCase().includes(s));
  }, [actions, q]);

  useEffect(() => {
    if (open) {
      setQ("");
      setIdx(0);
      requestAnimationFrame(() => inputRef.current && inputRef.current.focus());
    }
  }, [open]);

  useEffect(() => { setIdx(0); }, [q]);

  if (!open) return null;

  function run(i) {
    const a = filtered[i];
    if (!a) return;
    onClose();
    onRun(a);
  }

  return (
    <div id="palette-root" className="palette-root" onMouseDown={(e) => { if (e.target.id === "palette-root") onClose(); }}>
      <div id="palette" className="palette" role="dialog" aria-label="Command palette">
        <input
          id="palette-input"
          ref={inputRef}
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Switch agent, run, stop…"
          onKeyDown={(e) => {
            if (e.key === "Escape") { e.preventDefault(); onClose(); }
            if (e.key === "ArrowDown") { e.preventDefault(); setIdx((i) => Math.min(filtered.length - 1, i + 1)); }
            if (e.key === "ArrowUp") { e.preventDefault(); setIdx((i) => Math.max(0, i - 1)); }
            if (e.key === "Enter") { e.preventDefault(); run(idx); }
          }}
        />
        <ul className="palette-list">
          {filtered.length === 0 ? <li className="palette-empty">No matches</li> : null}
          {filtered.map((a, i) => (
            <li
              key={a.id}
              className={"palette-item" + (i === idx ? " active" : "")}
              onMouseEnter={() => setIdx(i)}
              onClick={() => run(i)}
            >
              <span className="palette-label">{a.label}</span>
              <span className="palette-group">{a.group}</span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

function buildActions(workspaces) {
  const out = [
    { id: "settings", label: "Settings", group: "app", kind: "settings" },
    { id: "providers", label: "Providers", group: "app", kind: "providers" },
    { id: "mcps", label: "MCPs", group: "app", kind: "mcps" },
  ];
  for (const ws of workspaces) {
    const mode = ws.agent ? ws.agent.mode : "stopped";
    out.push({ id: "open-" + ws.id, label: "Open " + ws.name, group: ws.name, kind: "open", wsId: ws.id });
    if (mode === "stopped") {
      out.push({ id: "run-" + ws.id, label: "Run " + ws.name, group: ws.name, kind: "run", wsId: ws.id });
      out.push({ id: "term-" + ws.id, label: "Open terminal · " + ws.name, group: ws.name, kind: "term", wsId: ws.id });
    } else {
      out.push({ id: "stop-" + ws.id, label: "Stop " + ws.name, group: ws.name, kind: "stop", wsId: ws.id });
    }
  }
  return out;
}
