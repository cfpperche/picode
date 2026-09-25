import { useEffect, useRef, useState } from "react";
import ScreenHeader from "../components/ScreenHeader.jsx";
import { api } from "@picode/shared/client/api.js";
import { SNIP_RESERVED, bodyLimit, clearDraft, draftToRestore, expandSnip, formFromSnip, parseSnip, readDraft, sameDraft, setDefaultInBody, snipSlug, tagsFromInput, writeDraft } from "@picode/shared/domain/snipDraft.js";
import { toast, toastError } from "../lib/toast.js";

// Create or edit a snippet on the phone: title, slug (auto from the
// title until edited), kind, tags and the body. Same rules as the desk:
// the draft is retained under the snippet's key until it matches the
// server copy, Save sends the version it loaded and a 409 keeps the text.
// Drafts live in localStorage (snippets v2, F4): same keys, same
// `draftToRestore` base rule, but a closed tab no longer loses the text.
// v2 also brings the desk's placeholder table (default, optional, enum),
// live validation and the address check — the phone is where a snippet is
// most likely to be written, so the rules cannot live on the desk alone.
function storage() {
  try {
    if (typeof window !== "undefined" && window.localStorage) return window.localStorage;
  } catch { /* blocked storage falls through */ }
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
  const [enums, setEnums] = useState({}); // name -> enum[]; rides the draft, sent on save
  const [slugState, setSlugState] = useState(""); // "" | "checking" | "free" | "taken"
  const lastGood = useRef("");
  const key = snipId || "";

  useEffect(() => {
    let stop = false;
    if (!snipId) {
      const kept = draftToRestore(readDraft(storage(), ""), null);
      // A capture or an import is not a crash: the text is here because the
      // reader asked for it, so no "restore" banner and no Discard offer.
      if (kept) { setF({ ...formFromSnip(null), ...kept }); setRestored(!kept.origin); }
      if (kept && kept.slugLocked) setSlugLocked(true);
      const stored = readDraft(storage(), "");
      if (stored && stored.enums) setEnums(stored.enums);
      setLoaded(true);
      return undefined;
    }
    api("/api/snips/" + encodeURIComponent(snipId)).then((p) => {
      if (stop) return;
      const server = formFromSnip(p);
      const stored = readDraft(storage(), p.id);
      const kept = draftToRestore(stored, server);
      setBase(server);
      setF(kept ? { ...server, ...kept } : server);
      setRestored(!!kept);
      if (stored && stored.enums) setEnums(stored.enums);
      setSlugLocked(true);
      setLoaded(true);
    }).catch((e) => { toastError(e); onBack(); });
    return () => { stop = true; };
  }, [snipId]);

  useEffect(() => {
    if (!loaded) return;
    const same = base ? sameDraft(f, base) : sameDraft(f, formFromSnip(null));
    if (same) clearDraft(storage(), key);
    else writeDraft(storage(), key, { ...f, enums, slugLocked }, base ? base.updatedAt : "");
  }, [loaded, f, base, key, enums, slugLocked]);

  // The same address check the desk runs (F8), so the phone does not send a
  // save the server is going to refuse: debounced, and the answer a save
  // would give. A failed check is no check — it never blocks a save alone.
  const wanted = (!slugLocked && !f.slug ? snipSlug(f.title) : f.slug) || snipSlug(f.title);
  useEffect(() => {
    if (!wanted) { setSlugState(""); return undefined; }
    let live = true;
    setSlugState("checking");
    const t = setTimeout(() => {
      const q = snipId ? "?except=" + encodeURIComponent(snipId) : "";
      api("/api/snips/slug/" + encodeURIComponent(wanted) + q)
        .then(() => { if (live) setSlugState("free"); })
        .catch((e) => { if (live) setSlugState(e && e.status === 409 ? "taken" : ""); });
    }, 350);
    return () => { live = false; clearTimeout(t); };
  }, [wanted, snipId]);

  const parsed = parseSnip(f.body);
  // While the body is invalid the table falls back to the last good parse:
  // the reader never loses sight of the placeholders they had.
  const live = parsed.ok ? parsed : parseSnip(lastGood.current || "");
  const invalid = !parsed.ok;
  if (parsed.ok) lastGood.current = f.body;
  const preview = parsed.ok ? expandSnip(f.body, {}, {}) : { text: "" };
  const cap = bodyLimit(f.body);
  const validationErr = invalid ? (
    parsed.error === "unclosed placeholder"
      ? "Close every placeholder — check around “" + (parsed.excerpt || f.body.slice(-24)) + "”."
      : parsed.error === "invalid placeholder name"
        ? "Invalid placeholder name near “" + (parsed.excerpt || "") + "”."
        : parsed.error
  ) : "";

  function set(patch) {
    setF((cur) => {
      const next = { ...cur, ...patch };
      if (!slugLocked && patch.title != null) next.slug = snipSlug(patch.title);
      return next;
    });
  }

  async function save() {
    if (!f.title.trim()) { toast("Give the snippet a title.", "info"); return; }
    if (invalid) { toast(validationErr, "warn"); return; }
    if (slugState === "taken") { toast("That address is already used. Pick another one.", "warn"); return; }
    if (cap.over) { toast("The body is too long (max 100 KB).", "warn"); return; }
    const body = {
      title: f.title.trim(),
      slug: f.slug || snipSlug(f.title),
      description: f.description,
      kind: f.kind === "shell" ? "shell" : "prompt",
      body: f.body,
      tags: tagsFromInput(f.tags),
      placeholders: (parsed.placeholders || []).map((ph) => ({ ...ph, enum: enums[ph.name] })),
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
      {/* One save action: the form's own button (a header Save duplicated it). */}
      <ScreenHeader title={snipId ? "Edit snippet" : "New snippet"} onBack={onBack} />
      {loaded ? (
        <form noValidate className="m-pin-form" onSubmit={(e) => { e.preventDefault(); save(); }}>
          {restored ? <div className="m-pin-restored" role="status">Unsaved changes restored. <button type="button" className="btn btn-sm btn-ghost" onClick={() => { clearDraft(storage(), key); setRestored(false); setF(base ? { ...base } : formFromSnip(null)); }}>Discard</button></div> : null}
          {err ? <p className="form-error" role="alert">{err}</p> : null}
          <select className="dlg-input" value={f.kind || "prompt"} aria-label="Kind" onChange={(e) => set({ kind: e.target.value })}>
            <option value="prompt">Prompt</option>
            <option value="shell">Command</option>
          </select>
          <input className="dlg-input" value={f.title} maxLength={200} placeholder="Snippet title" aria-label="Snippet title" onChange={(e) => set({ title: e.target.value })} />
          <input className="dlg-input" value={f.slug} maxLength={64} placeholder={snipSlug(f.title) || "slug"} aria-label="Slug" onChange={(e) => { setSlugLocked(true); set({ slug: e.target.value }); }} />
          {/* One line, under the field it is about: taken means the save is off. */}
          <p className={"m-pin-hint" + (slugState === "taken" ? " bad" : "")} role={slugState === "taken" ? "alert" : undefined}>
            {slugState === "taken"
              ? "Another snippet already uses /snip:" + wanted + " — pick another address."
              : "/snip:" + (wanted || "…") + (f.kind === "shell" ? " · pasted into a terminal after you confirm" : " · sent to an agent")}
          </p>
          <input className="dlg-input" value={f.description} maxLength={500} placeholder="Description" aria-label="Description" onChange={(e) => set({ description: e.target.value })} />
          <input className="dlg-input" value={f.tags} placeholder="Tags, separated by commas" aria-label="Tags" onChange={(e) => set({ tags: e.target.value })} />
          <textarea className="dlg-input m-pin-textarea m-snip-textarea" value={f.body} placeholder={"Write… {{name}} placeholders"} aria-label="Body" rows={12} onChange={(e) => set({ body: e.target.value })} />
          {invalid ? <p className="form-error" role="alert">{validationErr}</p> : null}
          {live && live.placeholders.length ? (
            <div className={"m-snip-ph" + (invalid ? " is-stale" : "")}>
              <span className="m-pin-label">Placeholders</span>
              {live.placeholders.map((ph) => {
                const reserved = SNIP_RESERVED.includes(ph.name);
                return (
                  <div className="m-snip-ph-row" key={ph.name}>
                    <div className="m-snip-ph-head">
                      <code className="m-snip-ph-name">{"{{"}{ph.name}{"}}"}</code>
                      {reserved ? <span className="m-pin-hint">from the target</span> : (
                        <label className="m-snip-ph-opt">
                          <input
                            type="checkbox"
                            checked={ph.optional}
                            aria-label={"Optional for " + ph.name}
                            onChange={(e) => set({ body: e.target.checked
                              ? setDefaultInBody(f.body, ph.name, ph.default, true)
                              : setDefaultInBody(f.body, ph.name, "", false) })}
                          />
                          <span>Optional</span>
                        </label>
                      )}
                    </div>
                    {reserved ? <p className="m-pin-hint">Filled from the target — nothing to set here.</p> : (
                      <>
                        <input
                          className="dlg-input"
                          value={ph.optional ? ph.default : ""}
                          placeholder={ph.optional ? "Default when skipped" : "Required — the agent asks"}
                          disabled={!ph.optional}
                          aria-label={"Default for " + ph.name}
                          onChange={(e) => set({ body: setDefaultInBody(f.body, ph.name, e.target.value, true) })}
                        />
                        <input
                          className="dlg-input"
                          value={(enums[ph.name] || []).join(", ")}
                          placeholder="Offer choices, e.g. dev, prod"
                          aria-label={"Enum values for " + ph.name}
                          onChange={(e) => setEnums((cur) => ({ ...cur, [ph.name]: e.target.value.split(",").map((x) => x.trim()).filter(Boolean) }))}
                        />
                      </>
                    )}
                  </div>
                );
              })}
            </div>
          ) : null}
          {!invalid && parsed.ok && !live.placeholders.length && preview.text ? (
            <label className="m-snip-preview">
              <span className="m-pin-label">Preview</span>
              <pre className="m-pin-md">{preview.text}</pre>
            </label>
          ) : null}
          {cap.near ? <p className={"m-pin-limit" + (cap.over ? " over" : "")}>{(cap.bytes / 1000).toFixed(0)} KB / 100 KB</p> : null}
          <button type="submit" className="btn btn-primary m-pin-save" disabled={busy || !loaded || invalid || slugState === "taken"}>{snipId ? "Save" : "Create snippet"}</button>
        </form>
      ) : <p className="m-pin-msg">Loading…</p>}
    </div>
  );
}
