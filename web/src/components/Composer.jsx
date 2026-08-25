import { useEffect, useRef, useState } from "react";
import ProviderChip from "./ProviderChip.jsx";
import ModelChip from "./ModelChip.jsx";
import ThinkingChip from "./ThinkingChip.jsx";
import ModeChip from "./ModeChip.jsx";
import KindChip from "./KindChip.jsx";
import { IconSend, IconStop } from "./Icons.jsx";
import ComposerStatus from "./ComposerStatus.jsx";
import { filterSlash } from "../lib/slash.js";
import { newHist, histPush, histUp, histDown, histTyped, caretFirstLine, caretLastLine } from "../lib/composerHist.js";

export default function Composer({
  kind, onKind, value, onChange, onSend, status, streaming,
  stopped, onToggleDock, onStop, onAbort, catalog, cfg, onConfig, onSlash, statusBar, onCompact, sessionBar,
}) {
  const ta = useRef(null);
  const hist = useRef(newHist());
  const [slashIdx, setSlashIdx] = useState(0);
  const hits = filterSlash(value);

  useEffect(() => {
    const el = ta.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.max(52, Math.min(el.scrollHeight, 160)) + "px";
  }, [value]);

  useEffect(() => { setSlashIdx(0); }, [value]);

  function pickSlash(cmd) {
    onChange("");
    if (!cmd) return;
    if (onSlash) onSlash(cmd);
  }

  return (
    <div className="composer-wrap">
      <div className="composer" onClick={(e) => { if (e.target === e.currentTarget) ta.current?.focus(); }}>
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
        {sessionBar ? <div className="composer-tools">{sessionBar}</div> : null}
        <textarea
          id="task-input"
          ref={ta}
          rows={2}
          placeholder="Message the agent, or / for commands"
          value={value}
          onChange={(e) => { histTyped(hist.current); onChange(e.target.value); }}
          onKeyDown={(e) => {
            if (hits.length) {
              if (e.key === "ArrowDown") { e.preventDefault(); setSlashIdx((i) => Math.min(hits.length - 1, i + 1)); return; }
              if (e.key === "ArrowUp") { e.preventDefault(); setSlashIdx((i) => Math.max(0, i - 1)); return; }
              if (e.key === "Tab" || (e.key === "Enter" && !e.shiftKey)) { e.preventDefault(); pickSlash(hits[slashIdx]); return; }
              if (e.key === "Escape") { e.preventDefault(); onChange(""); return; }
            }
            if (e.key === "ArrowUp" && caretFirstLine(ta.current)) {
              e.preventDefault();
              onChange(histUp(hist.current, value || ""));
              requestAnimationFrame(() => { const el = ta.current; if (el) el.setSelectionRange(el.value.length, el.value.length); });
              return;
            }
            if (e.key === "ArrowDown" && caretLastLine(ta.current)) {
              e.preventDefault();
              onChange(histDown(hist.current, value || ""));
              requestAnimationFrame(() => { const el = ta.current; if (el) el.setSelectionRange(el.value.length, el.value.length); });
              return;
            }
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              histPush(hist.current, value);
              onSend();
            }
          }}
        />
        <div className="composer-controls">
          <div className="composer-left">
            <ProviderChip catalog={catalog} cfg={cfg} onChange={onConfig || (() => {})} />
            <ModelChip catalog={catalog} cfg={cfg} onChange={onConfig || (() => {})} />
            <ThinkingChip catalog={catalog} cfg={cfg} onChange={onConfig || (() => {})} />
            <ModeChip cfg={cfg} onChange={onConfig || (() => {})} />
            <KindChip value={kind} onChange={onKind} />
          </div>
          <div className="composer-right">
            <span id="chat-status-text" className="sr-only">{status}{streaming ? " streaming" : ""}</span>
            {streaming ? (
              <button id="task-abort" type="button" className="icon-btn icon-btn-stop" title="Stop" onClick={onAbort}>
                <IconStop size={16} />
              </button>
            ) : (
              <button id="task-send" type="button" className="icon-btn icon-btn-send" title="Send" disabled={!value || !value.trim()} onClick={() => { histPush(hist.current, value); onSend(); }}>
                <IconSend size={16} />
              </button>
            )}
          </div>
        </div>
        <ComposerStatus bar={statusBar} onCompact={onCompact} />
      </div>
    </div>
  );
}
