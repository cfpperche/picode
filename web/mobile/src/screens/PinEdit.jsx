import { useEffect, useState } from "react";
import ScreenHeader from "../components/ScreenHeader.jsx";
import { api } from "@picode/shared/client/api.js";
import { PIN_LIMITS, bodyLimit, clearDraft, draftToRestore, normalizeTag, readDraft, sameDraft, writeDraft } from "@picode/shared/domain/pinDraft.js";
import { toast, toastError } from "../lib/toast.js";

// Create or edit a pin on the phone: title, tags (comma-separated) and the
// note as plain markdown in a textarea. Files, sketches and the reminder
// stay with the desktop studio; the phone's job is to capture a note in a
// few taps. Same rules as the desk: the draft is retained under the pin's
// key until it matches the server copy, Save sends the version it loaded
// and a 409 keeps the text and says so.
function storage() {
  try { return typeof window !== "undefined" ? window.sessionStorage : null; } catch { return null; }
}

function tagsText(tags) { return (tags || []).join(", "); }
function parseTags(text) {
  const out = [];
  for (const raw of String(text || "").split(/[,\n]/)) {
    const t = normalizeTag(raw);
    if (t && !out.includes(t) && out.length < PIN_LIMITS.tags) out.push(t);
  }
  return out;
}

export default function PinEdit({ pinId, onBack, onSaved }) {
  const [draft, setDraft] = useState({ title: "", tagsText: "", body: "" });
  const [base, setBase] = useState(null);
  const [loaded, setLoaded] = useState(!pinId);
  const [busy, setBusy] = useState(false);
  const [restored, setRestored] = useState(false);
  const key = pinId || "";

  useEffect(() => {
    let stop = false;
    if (!pinId) {
      const kept = draftToRestore(readDraft(storage(), ""), null);
      if (kept) { setDraft({ title: kept.title, tagsText: tagsText(kept.tags), body: kept.body }); setRestored(true); }
      setLoaded(true);
      return undefined;
    }
    api("/api/pins/" + encodeURIComponent(pinId)).then((p) => {
      if (stop) return;
      const server = { title: p.title || "", tags: p.tags || [], body: p.body || "", updatedAt: p.updatedAt || "" };
      const kept = draftToRestore(readDraft(storage(), p.id), server);
      setBase(server);
      const src = kept || server;
      setDraft({ title: src.title, tagsText: tagsText(src.tags), body: src.body });
      setRestored(!!kept);
      setLoaded(true);
    }).catch((e) => { toastError(e); onBack(); });
    return () => { stop = true; };
  }, [pinId]);

  useEffect(() => {
    if (!loaded) return;
    const cur = { title: draft.title, tags: parseTags(draft.tagsText), body: draft.body };
    const same = base ? sameDraft(cur, base) : sameDraft(cur, { title: "", tags: [], body: "" });
    if (same) clearDraft(storage(), key);
    else writeDraft(storage(), key, cur, base ? base.updatedAt : "");
  }, [loaded, draft, base, key]);

  const stand = bodyLimit(draft.body);

  async function save() {
    const title = draft.title.trim();
    if (!title) { toast("Give the pin a title.", "info"); return; }
    if ([...title].length > PIN_LIMITS.title) { toast("Title is too long (max " + PIN_LIMITS.title + " characters).", "warn"); return; }
    if (stand.over) { toast("The note is too long (max 100 KB).", "warn"); return; }
    const body = { title, tags: parseTags(draft.tagsText), body: draft.body };
    setBusy(true);
    try {
      if (pinId) {
        const p = await api("/api/pins/" + encodeURIComponent(pinId), { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ ...body, ifUpdatedAt: base ? base.updatedAt : "" }) });
        clearDraft(storage(), pinId);
        toast.ok("Pin saved.");
        onSaved(p.id);
      } else {
        const p = await api("/api/pins", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
        clearDraft(storage(), "");
        onSaved(p.id);
      }
    } catch (e) {
      if (e && e.status === 409) toast("This pin changed elsewhere. Your text is kept here; go back and open it again to see the latest.", "warn");
      else toastError(e);
    } finally { setBusy(false); }
  }

  return (
    <div className="m-screen m-pin-edit">
      <ScreenHeader title={pinId ? "Edit pin" : "New pin"} onBack={onBack} right={<button type="button" className="m-head-btn m-head-btn-text" disabled={busy || !loaded} onClick={save}>Save</button>} />
      {loaded ? (
        <form className="m-pin-form" onSubmit={(e) => { e.preventDefault(); save(); }}>
          {restored ? <div className="m-pin-restored" role="status">Unsaved changes restored. <button type="button" className="btn btn-sm btn-ghost" onClick={() => { clearDraft(storage(), key); setRestored(false); setDraft(base ? { title: base.title, tagsText: tagsText(base.tags), body: base.body } : { title: "", tagsText: "", body: "" }); }}>Discard</button></div> : null}
          <input className="dlg-input" value={draft.title} maxLength={PIN_LIMITS.title} placeholder="Pin title" aria-label="Pin title" autoFocus={!pinId} onChange={(e) => setDraft({ ...draft, title: e.target.value })} />
          <input className="dlg-input" value={draft.tagsText} placeholder="Tags, separated by commas" aria-label="Tags" onChange={(e) => setDraft({ ...draft, tagsText: e.target.value })} />
          <textarea className="dlg-input m-pin-textarea" value={draft.body} placeholder="Write… (markdown)" aria-label="Note" rows={12} onChange={(e) => setDraft({ ...draft, body: e.target.value })} />
          {stand.near ? <p className={"m-pin-limit" + (stand.over ? " over" : "")}>{(stand.bytes / 1000).toFixed(0)} KB / 100 KB</p> : null}
          <button type="submit" className="btn btn-primary m-pin-save" disabled={busy}>{pinId ? "Save" : "Create pin"}</button>
        </form>
      ) : <p className="m-pin-msg">Loading…</p>}
    </div>
  );
}
