import { useEffect, useRef } from "react";

export default function Composer({
  kind, onKind, value, onChange, onSend, status, streaming,
  stopped, onToggleDock, onStop,
}) {
  const ta = useRef(null);
  useEffect(() => {
    const el = ta.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 160) + "px";
  }, [value]);

  return (
    <div className="composer-wrap">
      <div className="composer">
        <textarea
          id="task-input"
          ref={ta}
          rows={1}
          placeholder="Send a task to this agent…"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); onSend(); }
          }}
        />
        <div className="composer-controls">
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
