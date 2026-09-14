import { useEffect, useState } from "react";
import ScreenHeader from "../components/ScreenHeader.jsx";
import { IconArchive, IconArchiveRestore, IconPencil, IconTrash } from "../components/Icons.jsx";
import { askConfirm } from "../lib/confirm.js";
import { toast, toastError } from "../lib/toast.js";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";

// A snippet on the phone, read-only: kind, slug, tags and the body with
// its placeholders. It is where More → Snippets lands; Edit opens the form.
export default function Snippet({ snipId, onBack, onEdit }) {
  const [snip, setSnip] = useState(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

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

  async function archive() {
    if (!snip || busy) return;
    setBusy(true);
    try {
      await api("/api/snips/" + encodeURIComponent(snip.id) + "/archived", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ archived: !snip.archivedAt }),
      });
      toast.ok(snip.archivedAt ? "Snippet unarchived." : "Snippet archived.");
    } catch (e) { toastError(e); } finally { setBusy(false); }
  }

  async function remove() {
    if (!snip || busy) return;
    const ok = await askConfirm({
      title: "Delete snippet",
      message: 'Delete "' + (snip.title || "this snippet") + '"? This cannot be undone.',
      confirmLabel: "Delete",
      danger: true,
    });
    if (!ok) return;
    setBusy(true);
    try {
      await api("/api/snips/" + encodeURIComponent(snip.id), { method: "DELETE" });
      toast.ok("Snippet deleted.");
      onBack();
    } catch (e) { toastError(e); setBusy(false); }
  }

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
          <div className="m-snip-actions">
            <button type="button" className="btn btn-sm" disabled={busy} onClick={archive}>
              {snip.archivedAt ? <IconArchiveRestore size={14} /> : <IconArchive size={14} />}
              {snip.archivedAt ? "Unarchive" : "Archive"}
            </button>
            <button type="button" className="btn btn-sm btn-danger" disabled={busy} onClick={remove}><IconTrash size={14} /> Delete</button>
          </div>
          {snip.archivedAt ? <p className="m-pin-hint">Archived — it is out of the list and the picker until you unarchive it.</p> : null}
        </div>
      ) : null}
    </div>
  );
}
