import { useState } from "react";
import { toast as sonner } from "sonner";
import { terminalCliFaviconUrls, terminalCliLabel, terminalCliMark } from "@picode/shared/domain/terminalCli.js";
import { IconError, IconGit, IconInfo, IconOk, IconPin, IconWarn, IconX } from "./Icons.jsx";

// The notice card on the phone (study:
// docs/benchmarks/2026-09-07-superset-notifications.md). Same three zones
// and the same shared model as desktop, drawn for a thumb: the card is
// full-width above the tab bar, and every control in the footer is a
// touch target rather than a 26px pill.
//
// ADR-0072: the model is shared (@picode/shared/domain/notice.js), the
// card is not.

const LEVEL_ICON = { ok: IconOk, info: IconInfo, warn: IconWarn, error: IconError, busy: IconInfo };

function ActorFace({ actor }) {
  // A pin speaks with the pin glyph (ADR-0100), never a CLI face.
  if (actor.kind === "pin") {
    return <span className="notice-face is-mark is-pin" title={actor.name} aria-hidden="true"><IconPin size={12} /></span>;
  }
  const urls = terminalCliFaviconUrls(actor.cli);
  const [failed, setFailed] = useState(0);
  const src = failed < urls.length ? urls[failed] : "";
  const label = terminalCliLabel(actor.cli) || actor.name;
  if (src) {
    return <img className="notice-face" src={src} alt="" title={label} onError={() => setFailed((n) => n + 1)} />;
  }
  return <span className="notice-face is-mark" title={label} aria-hidden="true">{terminalCliMark(actor.cli) || "·"}</span>;
}

function LevelGlyph({ level }) {
  const Icon = LEVEL_ICON[level] || IconInfo;
  return <Icon className="notice-glyph" size={14} />;
}

function runAction(action, id) {
  if (action.run) action.run();
  else if (action.hash) location.hash = action.hash;
  sonner.dismiss(id);
}

export default function Notice({ n, id, prefs }) {
  const p = prefs || {};
  const hasFoot = n.meta.length > 0 || n.actions.length > 0;
  const cls = ["notice", "notice-" + n.level, n.actor ? "has-head" : "no-head"];
  if (p.closeButton) cls.push("has-x");
  if (hasFoot) cls.push("has-foot");

  return (
    <div className={cls.join(" ")} role="status" aria-live={n.level === "error" ? "assertive" : "polite"}>
      {p.closeButton ? (
        <button type="button" className="notice-x" aria-label="Dismiss" onClick={() => { if (n.onClose) n.onClose(); sonner.dismiss(id); }}>
          <IconX size={15} />
        </button>
      ) : null}

      {n.actor ? (
        <div className="notice-head">
          <ActorFace actor={n.actor} />
          <span className="notice-who">{n.actor.name}</span>
          {n.status ? <span className="notice-status">{n.status}</span> : null}
        </div>
      ) : null}

      <div className="notice-body">
        {n.actor ? null : <LevelGlyph level={n.level} />}
        <span className="notice-text">{n.title}</span>
      </div>

      {n.body ? <p className="notice-detail">{n.body}</p> : null}

      {hasFoot ? (
        <div className="notice-foot">
          <div className="notice-meta">
            {n.meta.length ? <IconGit size={12} className="notice-meta-icon" /> : null}
            {n.meta.map((chip, i) => (
              <span key={i} className={"notice-chip tone-" + chip.tone}>{chip.text}</span>
            ))}
          </div>
          <div className="notice-actions">
            {n.actions.map((a, i) => (
              <button
                key={i}
                type="button"
                className={"btn btn-sm" + (a.primary ? " btn-primary" : " btn-ghost")}
                onClick={() => runAction(a, id)}
              >
                {a.label}
              </button>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  );
}

export function renderNotice(n, id, prefs) {
  return <Notice n={n} id={id} prefs={prefs} />;
}
