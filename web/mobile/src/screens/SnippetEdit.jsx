import { useEffect, useState } from "react";
import ScreenHeader from "../components/ScreenHeader.jsx";
import { api } from "@picode/shared/client/api.js";
import { bodyLimit, clearDraft, draftToRestore, formFromSnip, parseSnip, readDraft, sameDraft, snipSlug, tagsFromInput, writeDraft } from "@picode/shared/domain/snipDraft.js";
import { toast, toastError } from "../lib/toast.js";

// Create or edit a snippet on the phone: title, slug (auto from the
// title until edited), kind, tags and the body. Same rules as the desk:
// the draft is retained under the snippet's key until it matches the
// server copy, Save sends the version it loaded and a 409 keeps the text.
function storage() {
  try { return typeof window !== "undefined" ? window.sessionStorage : null; } catch { return null; }
}

export default function SnippetEdit({ snipId, onBack, onSaved }) {
  const [f, setF] = useState(() => formFromSnip(null));
  const [base, setBase] = useState(null);
  const [loaded, setLoaded] = useState(!snipId);
  const [busy, setBusy] = useState(false);
  const [restored, setRestored] = useState(false);
  const [slugLocked, setSlugLocked] = useState(!!snipId);
  const [err, setErr] = useState("");
  const key = snipId || "";

  useEffect(() => {
    let stop = false;
    if (!snipId) {
      const kept = draftToRestore(readDraft(storage(), ""), null);
      // A capture or an import is not a crash: the text is here because the
      // reader asked for it, so no "restore" banner and no Discard offer.
      if (kept) { setF({ ...formFromSnip(null), ...kept }); setRestored(!kept.origin); }
      setLoaded(true);
      return undefined;
    }
    api("/api/snips/" + encodeURIComponent(snipId)).then((p) => {
      if (stop) return;
      const server = formFromSnip(p);
      const kept = draftToRestore(readDraft(storage(), p.id), server);
      setBase(server);
      setF(kept ? { ...server, ...kept } : server);
      setRestored(!!kept);
      setSlugLocked(true);
      setLoaded(true);
    }).catch((e) => { toastError(e); onBack(); });
    return () => { stop = true; };
  }, [snipId]);

  useEffect(() => {
    if (!loaded) return;
    const same = base ? sameDraft(f, base) : sameDraft(f, formFromSnip(null));
    if (same) clearDraft(storage(), key);
    else writeDraft(storage(), key, f, base ? base.updatedAt : "");
  }, [loaded, f, base, key]);

  const parsed = parseSnip(f.body);
  const cap = bodyLimit(f.body);

  function set(patch) {
    setF((cur) => {
      const next = { ...cur, ...patch };
      if (!slugLocked && patch.title != null) next.slug = snipSlug(patch.title);
      return next;
    });
  }

  async function save() {
    if (!f.title.trim()) { toast("Give the snippet a title.", "info"); return; }
    if (!parsed.ok) { toast("Close every {{placeholder}} in the body.", "warn"); return; }
    if (cap.over) { toast("The body is too long (max 100 KB).", "warn"); return; }
    const body = {
      title: f.title.trim(),
      slug: f.slug || snipSlug(f.title),
      description: f.description,
      kind: f.kind === "shell" ? "shell" : "prompt",
      body: f.body,
      tags: tagsFromInput(f.tags),
    };
    setBusy(true);
    try {
      if (snipId) {
        const p = await api("/api/snips/" + encodeURIComponent(snipId), { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ ...body, ifUpdatedAt: base ? base.updatedAt : "" }) });
        clearDraft(storage(), snipId);
        toast.ok("Snippet saved.");
        onSaved(p.id);
      } else {
        const p = await api("/api/snips", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
        clearDraft(storage(), "");
        toast.ok("Snippet saved.");
        onSaved(p.id);
      }
    } catch (e) {
      if (e && e.status === 409) toast("This snippet changed elsewhere. Your text is kept here; go back and open it again to see the latest.", "warn");
      else toastError(e);
    } finally { setBusy(false); }
  }

  return (
    <div className="m-screen m-pin-edit">
      <ScreenHeader title={snipId ? "Edit snippet" : "New snippet"} onBack={onBack} right={<button type="button" className="m-head-btn m-head-btn-text" disabled={busy || !loaded} onClick={save}>Save</button>} />
      {loaded ? (
        <form className="m-pin-form" onSubmit={(e) => { e.preventDefault(); save(); }}>
          {restored ? <div className="m-pin-restored" role="status">Unsaved changes restored. <button type="button" className="btn btn-sm btn-ghost" onClick={() => { clearDraft(storage(), key); setRestored(false); setF(base ? { ...base } : formFromSnip(null)); }}>Discard</button></div> : null}
          {err ? <p className="form-error" role="alert">{err}</p> : null}
          <select className="dlg-input" value={f.kind || "prompt"} aria-label="Kind" onChange={(e) => set({ kind: e.target.value })}>
            <option value="prompt">Prompt</option>
            <option value="shell">Command</option>
          </select>
          <input className="dlg-input" value={f.title} maxLength={200} placeholder="Snippet title" aria-label="Snippet title" onChange={(e) => set({ title: e.target.value })} />
          <input className="dlg-input" value={f.slug} maxLength={64} placeholder={snipSlug(f.title) || "slug"} aria-label="Slug" onChange={(e) => { setSlugLocked(true); set({ slug: e.target.value }); }} />
          <input className="dlg-input" value={f.description} maxLength={500} placeholder="Description" aria-label="Description" onChange={(e) => set({ description: e.target.value })} />
          <input className="dlg-input" value={f.tags} placeholder="Tags, separated by commas" aria-label="Tags" onChange={(e) => set({ tags: e.target.value })} />
          <textarea className="dlg-input m-pin-textarea m-snip-textarea" value={f.body} placeholder={"Write… {{name}} placeholders"} aria-label="Body" rows={12} onChange={(e) => set({ body: e.target.value })} />
          {cap.near ? <p className={"m-pin-limit" + (cap.over ? " over" : "")}>{(cap.bytes / 1000).toFixed(0)} KB / 100 KB</p> : null}
          <button type="submit" className="btn btn-primary m-pin-save" disabled={busy}>{snipId ? "Save" : "Create snippet"}</button>
        </form>
      ) : <p className="m-pin-msg">Loading…</p>}
    </div>
  );
}
