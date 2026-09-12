import { useState } from "react";

// Presentation is app-owned (ADR-0072); capture validation stays headless.
export default function ToolCapture({ preview, error, onPreview, onDetails }) {
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);
  if (!preview && !error) return null;
  if (error || failed) return (
    <div className="tp-preview tp-preview-unavailable">
      <span>Capture unavailable</span>
      <button type="button" className="btn btn-ghost btn-sm" onClick={failed
        ? () => { setFailed(false); setAttempt((n) => n + 1); } : onDetails}>
        {failed ? "Retry" : "View result"}
      </button>
    </div>
  );
  return (
    <div className="tp-preview">
      <div className="tp-preview-meta">
        <span>Last capture</span>
        {preview.ts ? <time dateTime={new Date(preview.ts).toISOString()} title={new Date(preview.ts).toLocaleString()}>
          {new Date(preview.ts).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
        </time> : null}
        {preview.source ? <span className="tp-preview-source" title={preview.source}>{preview.source}</span> : null}
      </div>
      <button type="button" className="tp-preview-pic" onClick={() => onPreview?.(preview.image)} title="View capture">
        <img key={attempt} src={preview.image} alt={preview.title || "Last capture"} loading="lazy" onError={() => setFailed(true)} />
      </button>
      {preview.title || preview.url ? (
        <div className="tp-preview-cap">
          {preview.title ? <span className="tp-preview-title" title={preview.title}>{preview.title}</span> : null}
          {preview.url ? <span className="tp-preview-url" title={preview.url}>{preview.url}</span> : null}
        </div>
      ) : null}
    </div>
  );
}
