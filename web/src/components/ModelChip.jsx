import { useEffect, useRef, useState } from "react";
import { chipLabel } from "../lib/chip.js";

export default function ModelChip({ catalog, cfg, onChange, open: forceOpen }) {
  const [open, setOpen] = useState(false);
  const wrap = useRef(null);
  const levels = (catalog && catalog.thinking) || ["off", "minimal", "low", "medium", "high", "xhigh", "max"];
  const providers = (catalog && catalog.providers) || [];
  const provider = (cfg && cfg.provider) || "";
  const model = (cfg && cfg.model) || "";
  const thinking = (cfg && cfg.thinking) || "";

  useEffect(() => {
    if (forceOpen) setOpen(true);
  }, [forceOpen]);

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
        id="agent-model"
        className="cockpit-chip"
        title="Model"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
      >
        <span className="cockpit-chip-label">{chipLabel(cfg)}</span>
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" aria-hidden="true">
          <path d="m6 9 6 6 6-6" />
        </svg>
      </button>
      <button type="button" id="agent-thinking" className="sr-only" tabIndex={-1} onFocus={() => setOpen(true)}>thinking</button>
      <button type="button" id="agent-provider" className="sr-only" tabIndex={-1} onFocus={() => setOpen(true)}>provider</button>
      {open && (
        <div className="cockpit-pop" role="listbox">
          <button
            type="button"
            className={"cockpit-opt" + (!model && !provider ? " active" : "")}
            onClick={() => { onChange({ provider: "", model: "", thinking: "" }); setOpen(false); }}
          >
            Default model
          </button>
          {providers.map((p) => (
            <div key={p.id} className="cockpit-group">
              <div className="cockpit-group-label">{p.id}</div>
              {(p.models || []).map((m) => {
                const selected = provider === p.id && model === m.id;
                return (
                  <div key={m.id}>
                    <button
                      type="button"
                      className={"cockpit-opt" + (selected ? " active" : "")}
                      onClick={() => onChange({ provider: p.id, model: m.id, thinking })}
                    >
                      {m.id}
                    </button>
                    {selected && m.thinking && (
                      <div className="cockpit-think">
                        <button
                          type="button"
                          className={"think-pill" + (!thinking ? " on" : "")}
                          onClick={() => onChange({ provider: p.id, model: m.id, thinking: "" })}
                        >default</button>
                        {levels.map((l) => (
                          <button
                            key={l}
                            type="button"
                            className={"think-pill" + (thinking === l ? " on" : "")}
                            onClick={() => onChange({ provider: p.id, model: m.id, thinking: l })}
                          >{l}</button>
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
