import { useState } from "react";
import { toast as sonner } from "sonner";
import { terminalCliFaviconUrls, terminalCliLabel, terminalCliMark } from "@picode/shared/domain/terminalCli.js";
import { IconError, IconGit, IconInfo, IconOk, IconWarn, IconX } from "./Icons.jsx";

// The notice card (study: docs/benchmarks/2026-09-07-superset-notifications.md).
// Three zones, and only the zones the notice actually filled: identity
// (who spoke, and for how long they worked), the sentence, and a footer
// carrying the metadata left and at most two ways out right. A notice with
// no actor is what the 317 legacy toast() calls produce — one glyph, one
// line, no footer.
//
// The model and its policies live in @picode/shared/domain/notice.js;
// ADR-0072 keeps the card itself here, so the phone can draw its own.

const LEVEL_ICON = { ok: IconOk, info: IconInfo, warn: IconWarn, error: IconError, busy: IconInfo };

// The agent's own mark, the way TerminalCliBadge does it: walk the
// vendor's asset list, fall back to the two-letter mark, never invent a
// glyph of our own.
function ActorFace({ actor }) {
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

  return (
    <div className={cls.join(" ")} role="status" aria-live={n.level === "error" ? "assertive" : "polite"}>
      {p.closeButton ? (
        <button type="button" className="notice-x" aria-label="Dismiss" onClick={() => sonner.dismiss(id)}>
          <IconX size={13} />
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
            {n.meta.length ? <IconGit size={11} className="notice-meta-icon" /> : null}
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

// The renderer handed to sonner.custom() by lib/toast.js.
export function renderNotice(n, id, prefs) {
  return <Notice n={n} id={id} prefs={prefs} />;
}
