import { useEffect, useState } from "react";
import ScreenHeader from "../components/ScreenHeader.jsx";
import { IconPencil } from "../components/Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";

// A snippet on the phone, read-only: kind, slug, tags and the body with
// its placeholders. It is where More → Snippets lands; Edit opens the form.
export default function Snippet({ snipId, onBack, onEdit }) {
  const [snip, setSnip] = useState(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let stop = false;
    function load() {
      api("/api/snips/" + encodeURIComponent(snipId)).then((p) => { if (!stop) { setSnip(p); setError(""); } })
        .catch((e) => { if (!stop) setError(e && e.message ? e.message : "Could not open this snippet."); });
    }
    load();
    const unsub = subscribeFeed((ev) => {
      if (ev.type === "feed.open" || ev.type === "feed.reset") load();
      else if (ev.type === "snip.updated" && ev.data && ev.data.id === snipId) load();
      else if (ev.type === "snip.deleted" && ev.data && ev.data.id === snipId) setError("This snippet was deleted.");
    });
    return () => { stop = true; unsub(); };
  }, [snipId]);

  return (
    <div className="m-screen m-pin">
      <ScreenHeader title={snip ? snip.title : "Snippet"} onBack={onBack} right={snip && onEdit ? <button type="button" className="m-head-btn" aria-label="Edit snippet" onClick={() => onEdit(snip.id)}><IconPencil size={16} /></button> : null} />
      {error ? <p className="m-pin-msg">{error}</p> : null}
      {!snip && !error ? <p className="m-pin-msg">Loading…</p> : null}
      {snip ? (
        <div className="m-pin-body">
          <div className="m-pin-tags">
            <span className="m-pin-tag m-pin-flag">{snip.kind === "shell" ? "Command" : "Prompt"}</span>
            <span className="m-pin-tag">/snip:{snip.slug}</span>
            {(snip.tags || []).map((t) => <span key={t} className="m-pin-tag">#{t}</span>)}
          </div>
          {snip.description ? <p className="m-pin-remind">{snip.description}</p> : null}
          {snip.body ? <pre className="m-pin-md m-snip-body">{snip.body}</pre> : null}
          {snip.placeholders && snip.placeholders.length ? (
            <div className="m-pin-tags">
              {snip.placeholders.map((p) => (
                <span key={p.name} className="m-pin-tag">{p.name}{p.optional ? "=" + (p.default || "…") : ""}</span>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
