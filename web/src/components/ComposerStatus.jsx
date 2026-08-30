import { statusSegments } from "../lib/statusbar.js";
import PiSpinner from "./PiSpinner.jsx";

export default function ComposerStatus({ bar, onCompact }) {
  const parts = statusSegments(bar);
  if (!parts.length) return null;
  return (
    <div className="composer-statusbar" aria-label="Session status">
      {parts.map((p, i) => (
        <span key={p.key} className={"sb-seg" + (p.tone ? " sb-" + p.tone : "")}>
          {i > 0 ? <span className="sb-sep" aria-hidden="true">/</span> : null}
          {p.kind === "bar" ? <CtxBar p={p} onCompact={onCompact} /> : p.kind === "compact" ? <CompactSeg p={p} /> : p.text}
        </span>
      ))}
    </div>
  );
}

function CompactSeg({ p }) {
  return (
    <span className="sb-compact" title="Older turns are being summarized into a compact. Large sessions can take minutes.">
      <PiSpinner title="Compacting" />
      {p.text}
    </span>
  );
}

function CtxBar({ p, onCompact }) {
  const inner = (
    <>
      <span className="sb-track" aria-hidden="true">
        <span className="sb-fill" style={{ width: (p.pct || 0) + "%" }} />
      </span>
      <span className="sb-ctx-lab">{p.text}</span>
    </>
  );
  if (!onCompact) return <span className="sb-ctx" title={p.text}>{inner}</span>;
  return (
    <button type="button" className="sb-ctx sb-ctx-btn" title="Compact session" onClick={onCompact}>
      {inner}
    </button>
  );
}
