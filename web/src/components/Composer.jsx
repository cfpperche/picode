import { useEffect, useRef, useState } from "react";
import ConfigFields from "./ConfigFields.jsx";
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
          <ConfigFields
            catalog={catalog}
            provider={(cfg && cfg.provider) || ""}
            model={(cfg && cfg.model) || ""}
            thinking={(cfg && cfg.thinking) || ""}
            onChange={onConfig || (() => {})}
            idPrefix="agent"
            row
          />
          <select id="task-kind" className="kind-chip" title="How the message is delivered" value={kind} onChange={(e) => onKind(e.target.value)}>
            <option value="prompt">Prompt</option>
            <option value="steer">Steer</option>
            <option value="follow_up">Follow-up</option>
          </select>
          <span className="composer-status" id="composer-status" hidden={stopped}>
            <span className={"dot" + (streaming ? " streaming" : "")} id="chat-dot" />
            <span id="chat-status-text">{status}</span>
          </span>
          <span className="composer-hint">Enter ↵</span>
          <button id="btn-dock" className="btn btn-ghost btn-sm" title="Show the terminal" hidden={stopped} onClick={onToggleDock}>Terminal</button>
          <button id="btn-stop-agent" className="btn btn-ghost btn-sm btn-danger" title="Stop this agent" hidden={stopped} onClick={onStop}>Stop</button>
          <button id="task-send" className="btn btn-primary btn-sm" title="Send" onClick={onSend}>Send</button>
        </div>
      </div>
    </div>
  );
}
