import { useCallback, useEffect, useRef, useState } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { pinHash } from "../../lib/routes.js";

// NotePanel — the loaded body of a `note` panel (docs/plans/matrix-canvas.md
// §4.2): one pin's markdown, read-only, rendered the way every other surface
// in the app renders markdown (react-markdown + remarkGfm over the shared
// `.md` styles — AppSurface and FilePreview do the same). Writing stays in
// Pin Studio, which the panel's header opens.
//
// The list the surface holds is summaries only — a 100 KB note has no
// business riding every feed frame (docs/architecture/pins.md) — so the body
// reads the whole pin itself, once per mount, and again when the feed says
// that pin changed. No timer: `pin.updated` is the trigger, exactly as the
// Inspector follows `git.updated`.
//
// This body has no `term.buffer`, so the canvas's still never applies to it
// (loadPolicy's `pane` row); the band still unmounts it away from the
// viewport, which costs a fetch and no socket.
export default function NotePanel({ pinId, title }) {
  const [pin, setPin] = useState(null);
  const [error, setError] = useState("");
  const [nonce, setNonce] = useState(0);
  const idRef = useRef(pinId);
  idRef.current = pinId;

  const reload = useCallback(() => setNonce((n) => n + 1), []);
  useEffect(() => {
    if (!pinId) return undefined;
    let stop = false;
    api("/api/pins/" + encodeURIComponent(pinId))
      .then((p) => { if (!stop) { setPin(p); setError(""); } })
      .catch((e) => { if (!stop) setError(humanizeError(e && e.message ? e.message : String(e))); });
    return () => { stop = true; };
  }, [pinId, nonce]);

  // The feed carries the pin's summary; the body it does not carry is
  // exactly why this refetches instead of patching.
  useEffect(() => subscribeFeed((ev) => {
    const hit = ev.type === "pin.updated" && ev.data && ev.data.id === idRef.current;
    if (hit || ev.type === "feed.open" || ev.type === "feed.reset") reload();
  }), [reload]);

  const studio = pinHash(pinId);
  if (error) {
    return (
      <div className="mx-placeholder mx-state" role="status">
        <span>{error}</span>
        <button type="button" className="btn btn-sm" onClick={reload}>Try again</button>
      </div>
    );
  }
  if (!pin) {
    return (
      <div className="mx-md" aria-busy="true">
        <div className="file-skel" aria-hidden="true">
          <span className="skel-line w-80" /><span className="skel-line w-90" /><span className="skel-line w-50" />
        </div>
      </div>
    );
  }
  if (!String(pin.body || "").trim()) {
    return (
      <div className="mx-placeholder mx-state" role="status">
        <span>This note is empty.</span>
        <a className="btn btn-sm" href={studio}>Write it</a>
      </div>
    );
  }
  return (
    <div className="mx-md md" aria-label={(pin.title || title || "Note") + " — note"}>
      <Markdown remarkPlugins={[remarkGfm]}>{pin.body}</Markdown>
    </div>
  );
}
