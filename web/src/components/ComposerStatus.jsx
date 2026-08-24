import { statusSegments } from "../lib/statusbar.js";

export default function ComposerStatus({ bar }) {
  const parts = statusSegments(bar);
  if (!parts.length) return null;
  return (
    <div className="composer-statusbar" aria-label="Session status">
      {parts.map((p, i) => (
        <span key={p.key} className={"sb-seg" + (p.tone ? " sb-" + p.tone : "")}>
          {i > 0 ? <span className="sb-dot" aria-hidden="true">/</span> : null}
          {p.kind === "bar" ? <CtxBar p={p} /> : p.text}
        </span>
      ))}
    </div>
  );
}

function CtxBar({ p }) {
  return (
    <span className="sb-ctx" title={p.text}>
      <span className="sb-track" aria-hidden="true">
        <span className="sb-fill" style={{ width: (p.pct || 0) + "%" }} />
      </span>
      <span className="sb-ctx-lab">{p.text}</span>
    </span>
  );
}
