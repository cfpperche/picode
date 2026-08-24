import { statusSegments } from "../lib/statusbar.js";

export default function ComposerStatus({ bar }) {
  const parts = statusSegments(bar);
  if (!parts.length) return null;
  return (
    <div className="composer-statusbar" aria-label="Session status">
      {parts.map((p, i) => (
        <span key={p.key} className={"sb-seg" + (p.tone ? " sb-" + p.tone : "")}>
          {i > 0 ? <span className="sb-dot">·</span> : null}
          {p.text}
        </span>
      ))}
    </div>
  );
}
