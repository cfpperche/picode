import { useEffect, useRef, useState } from "react";
import ModelChip from "./ModelChip.jsx";
import KindChip from "./KindChip.jsx";
import { IconSend, IconStop, IconTerminal } from "./Icons.jsx";
import { filterSlash } from "../lib/slash.js";

export default function Composer({
  kind, onKind, value, onChange, onSend, status, streaming,
  stopped, onToggleDock, onStop, catalog, cfg, onConfig, onSlash,
}) {
  const ta = useRef(null);
  const [slashIdx, setSlashIdx] = useState(0);
  const hits = filterSlash(value);

  useEffect(() => {
    const el = ta.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 160) + "px";
  }, [value]);

  useEffect(() => { setSlashIdx(0); }, [value]);

  function pickSlash(cmd) {
    onChange("");
    if (!cmd) return;
    if (onSlash) onSlash(cmd);
  }

  return (
    <div className="composer-wrap">
      <div className="composer">
        {hits.length > 0 && (
          <ul className="slash-menu" role="listbox">
            {hits.map((c, i) => (
              <li key={c.id}>
                <button
                  type="button"
                  className={"slash-item" + (i === slashIdx ? " active" : "")}
                  onMouseEnter={() => setSlashIdx(i)}
                  onClick={() => pickSlash(c)}
                >
                  <span className="slash-label">{c.label}</span>
                  <span className="slash-hint">{c.hint}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
        <textarea
          id="task-input"
          ref={ta}
          rows={1}
          placeholder="Message, or / for pi commands"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={(e) => {
            if (hits.length) {
              if (e.key === "ArrowDown") { e.preventDefault(); setSlashIdx((i) => Math.min(hits.length - 1, i + 1)); return; }
              if (e.key === "ArrowUp") { e.preventDefault(); setSlashIdx((i) => Math.max(0, i - 1)); return; }
              if (e.key === "Tab" || (e.key === "Enter" && !e.shiftKey)) { e.preventDefault(); pickSlash(hits[slashIdx]); return; }
              if (e.key === "Escape") { e.preventDefault(); onChange(""); return; }
            }
            if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); onSend(); }
          }}
        />
        <div className="composer-controls">
          <div className="composer-left">
            <ModelChip catalog={catalog} cfg={cfg} onChange={onConfig || (() => {})} />
            <KindChip value={kind} onChange={onKind} />
          </div>
          <div className="composer-right">
            <span className={"dot" + (streaming ? " streaming" : "")} id="chat-dot" hidden={stopped} title={status} />
            <span id="chat-status-text" className="sr-only">{status}</span>
            <button id="btn-dock" className="icon-btn" title="Terminal" hidden={stopped} onClick={onToggleDock}>
              <IconTerminal />
            </button>
            <button id="btn-stop-agent" className="icon-btn icon-btn-danger" title="Stop" hidden={stopped} onClick={onStop}>
              <IconStop />
            </button>
            <button id="task-send" className="icon-btn icon-btn-send" title="Send" disabled={!value || !value.trim()} onClick={onSend}>
              <IconSend />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
