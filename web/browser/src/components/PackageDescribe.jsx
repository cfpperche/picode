import { useEffect, useRef, useState } from "react";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { askConfirm } from "../lib/confirm.js";
import { packageDescribeSchema } from "@picode/shared/contracts/schemas.js";
import PageFrame from "./PageFrame.jsx";
import PiSpinner from "./PiSpinner.jsx";

// Describe config (ADR-0119 C5): the owner describes a package's config
// file and typed fields once; from then on the package is GUI-configurable
// through the generic form. Create, edit and delete all happen here —
// deleting the description honestly removes the package's Configure button.
// This page knows nothing about the package itself; the fields are whatever
// its documentation says the config file holds.

const TYPES = ["string", "enum", "boolean", "number", "secret"];
const EMPTY_FIELD = { key: "", label: "", type: "string", required: false, options: "", min: "", max: "", help: "" };

function templateToDraft(d) {
  return {
    id: d.id || "",
    title: d.title || "",
    application: d.application || "",
    match: d.match || "",
    files: (d.files || [{ scope: "agent", path: "", format: "json" }]).map((f) => ({ ...f })),
    fields: (d.fields || []).map((f) => ({
      key: f.key || "", label: f.label || "", type: f.type || "string", required: !!f.required,
      options: (f.options || []).join(", "), min: f.min ?? "", max: f.max ?? "", help: f.help || "",
    })),
  };
}

function draftToDescriptor(draft) {
  const fields = draft.fields
    .filter((f) => f.key.trim() !== "")
    .map((f) => {
      const out = { key: f.key.trim(), label: f.label.trim(), type: f.type };
      if (f.required) out.required = true;
      if (f.type === "enum") out.options = f.options.split(",").map((s) => s.trim()).filter(Boolean);
      if (f.type === "number") {
        if (f.min !== "") out.min = Number(f.min);
        if (f.max !== "") out.max = Number(f.max);
      }
      if (f.help) out.help = f.help;
      return out;
    });
  return {
    id: draft.id,
    title: draft.title,
    application: draft.application,
    match: draft.match,
    files: draft.files.map((f) => ({ ...f })),
    fields,
  };
}

export default function PackageDescribe({ hidden, embedded = false, pkg, backHash = "#/clis/pi/packages", configHashFor }) {
  const [draft, setDraft] = useState(null);
  const [exists, setExists] = useState(false);
  const [loadErr, setLoadErr] = useState("");
  const [errors, setErrors] = useState({});
  const [saving, setSaving] = useState(false);
  const [note, setNote] = useState("");
  const [noteError, setNoteError] = useState(false);
  const loaded = useRef(false);

  function configURL() {
    return "/api/packages/describe?package=" + encodeURIComponent(pkg);
  }

  async function load() {
    if (!pkg) { setDraft(null); return; }
    try {
      const next = await api(configURL());
      setDraft(templateToDraft(next.descriptor));
      setExists(!!next.exists);
      setLoadErr("");
      setNote(""); setNoteError(false); setErrors({});
    } catch (err) {
      setLoadErr(humanizeError(err.message || String(err)));
    }
  }

  useEffect(() => { if (!hidden && pkg && !loaded.current) { loaded.current = true; load(); } }, [hidden, pkg]); // eslint-disable-line

  function set(patch) {
    setDraft((d) => ({ ...d, ...patch }));
    setNote("");
  }

  function setFieldRow(i, patch) {
    setDraft((d) => {
      const fields = d.fields.map((f, j) => (j === i ? { ...f, ...patch } : f));
      return { ...d, fields };
    });
    setNote("");
  }

  async function save() {
    const parsed = packageDescribeSchema.safeParse(draft);
    if (!parsed.success) {
      const next = {};
      for (const issue of parsed.error.issues) next[issue.path.join(".")] = issue.message;
      setErrors(next);
      setNote("Fix the highlighted fields.");
      setNoteError(true);
      return;
    }
    setSaving(true);
    try {
      // Send the converted descriptor, not the editor draft: empty strings
      // would arrive as strings where Go expects numbers or nothing.
      const res = await api(configURL(), { method: "PUT", body: JSON.stringify(draftToDescriptor(draft)) });
      setNote("Description saved — Configure is available for this package.");
      setNoteError(false);
      setExists(true);
      if (!exists && configHashFor) {
        // First description: land on the value form right away.
        setTimeout(() => { if (configHashFor) location.hash = configHashFor(pkg); }, 500);
      }
      void res;
    } catch (err) {
      setNote(humanizeError(err.message || String(err)));
      setNoteError(true);
    } finally {
      setSaving(false);
    }
  }

  async function removeDescription() {
    if (!(await askConfirm({
      title: "Delete description",
      message: `Delete your description of ${pkg}? The package loses its Configure button until you describe it again. The config file itself is not touched.`,
      confirmLabel: "Delete description",
    }))) return;
    try {
      await api(configURL(), { method: "DELETE" });
      setNote(""); setNoteError(false);
      if (backHash) location.hash = backHash;
    } catch (err) {
      setNote(humanizeError(err.message || String(err)));
      setNoteError(true);
    }
  }

  const err = (path) => errors[path];

  return (
    <PageFrame embedded={embedded} id="package-describe" title={(pkg || "package") + " — describe config"} hidden={hidden}>
      <div className="pkc-top">
        <a className="pkg-back" href={backHash}>← All packages</a>
        <span className="pkg-foot-spacer" />
      </div>

      {loadErr ? (
        <p className="pkg-notice warn" role="alert">{loadErr} <button type="button" className="btn btn-ghost btn-sm" onClick={load}>Retry</button></p>
      ) : null}
      {!draft && !loadErr ? (
        <p className="pkg-loading" role="status"><PiSpinner title="Loading" /> Loading…</p>
      ) : null}

      {draft ? (
        <>
          <p className="pkg-fine">Describe the config file this package reads — file, scope and typed fields. Saving makes the package GUI-configurable; you can edit or delete this description at any time. The file itself is only written when you configure values.</p>
          {exists ? <p className="pkg-notice ok" role="status">You have described this package already — editing replaces it.</p> : null}

          <div className="pkc-section">
            <div className="pkc-row">
              <label htmlFor="pd-title">Title *</label>
              <input id="pd-title" className="pkc-input" value={draft.title} disabled={saving} onChange={(e) => set({ title: e.target.value })} />
              {errors.title ? <p className="pkg-notice err" role="alert">{errors.title}</p> : null}
            </div>
            <div className="pkc-row">
              <label htmlFor="pd-application">Applied when</label>
              <input id="pd-application" className="pkc-input" value={draft.application} disabled={saving} onChange={(e) => set({ application: e.target.value })} placeholder="Applies the next time the package reads its config." />
            </div>
          </div>

          <div className="pkc-section">
            <h3>Config file</h3>
            {draft.files.map((f, i) => (
              <div className="pkc-row" key={i}>
                <label htmlFor="pd-scope">Scope</label>
                <select id="pd-scope" className="pkc-select" value={f.scope} disabled={saving} onChange={(e) => set({ files: [{ ...f, scope: e.target.value }] })}>
                  <option value="agent">agent — machine-global (~/.pi/agent)</option>
                  <option value="workspace">workspace — the workspace folder</option>
                </select>
                <label htmlFor="pd-path">File path *</label>
                <input id="pd-path" className="pkc-input" value={f.path} disabled={saving} onChange={(e) => set({ files: [{ ...f, path: e.target.value }] })} placeholder="web-search.json" />
                {errors["files.0.path"] ? <p className="pkg-notice err" role="alert">{errors["files.0.path"]}</p> : null}
              </div>
            ))}
          </div>

          <div className="pkc-section">
            <h3>Fields</h3>
            {draft.fields.map((f, i) => (
              <div className="pkc-row" key={i}>
                <div className="pkg-notice-actions" data-align-row data-align-wrap>
                  <input className="pkc-input" style={{ maxWidth: "10rem" }} aria-label={`Field ${i + 1} key`} placeholder="key" value={f.key} disabled={saving} onChange={(e) => setFieldRow(i, { key: e.target.value })} />
                  <input className="pkc-input" aria-label={`Field ${i + 1} label`} placeholder="Label" value={f.label} disabled={saving} onChange={(e) => setFieldRow(i, { label: e.target.value })} />
                  <select className="pkc-select" aria-label={`Field ${i + 1} type`} value={f.type} disabled={saving} onChange={(e) => setFieldRow(i, { type: e.target.value })}>
                    {TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
                  </select>
                  <label className="pkg-fine"><input type="checkbox" checked={f.required} disabled={saving} onChange={(e) => setFieldRow(i, { required: e.target.checked })} /> required</label>
                  {f.type === "enum" ? (
                    <input className="pkc-input" style={{ maxWidth: "18rem" }} aria-label={`Field ${i + 1} options`} placeholder="option, option, …" value={f.options} disabled={saving} onChange={(e) => setFieldRow(i, { options: e.target.value })} />
                  ) : null}
                  {f.type === "number" ? (
                    <>
                      <input className="pkc-input" style={{ maxWidth: "6rem" }} type="number" aria-label={`Field ${i + 1} min`} placeholder="min" value={f.min} disabled={saving} onChange={(e) => setFieldRow(i, { min: e.target.value })} />
                      <input className="pkc-input" style={{ maxWidth: "6rem" }} type="number" aria-label={`Field ${i + 1} max`} placeholder="max" value={f.max} disabled={saving} onChange={(e) => setFieldRow(i, { max: e.target.value })} />
                    </>
                  ) : null}
                  <input className="pkc-input" aria-label={`Field ${i + 1} help`} placeholder="help" value={f.help} disabled={saving} onChange={(e) => setFieldRow(i, { help: e.target.value })} />
                  <button type="button" className="btn btn-ghost btn-sm" disabled={saving} onClick={() => set({ fields: draft.fields.filter((_, j) => j !== i) })}>Remove</button>
                </div>
              </div>
            ))}
            <button type="button" className="btn btn-sm" disabled={saving} onClick={() => set({ fields: [...draft.fields, { ...EMPTY_FIELD }] })}>Add field</button>
            {errors.fields ? <p className="pkg-notice err" role="alert">{errors.fields}</p> : null}
          </div>

          <div className="pkc-foot">
            <span className="pkg-fine pkc-file">Stored in PiCode's data dir — deleting it returns the package to "no settings known".</span>
            <div className="pkc-actions" data-align-row data-align-wrap>
              {exists ? <button type="button" className="btn btn-ghost btn-sm" disabled={saving} onClick={removeDescription}>Delete description…</button> : null}
              <button type="button" className="btn btn-primary btn-sm" disabled={saving} onClick={save}>{saving ? "Saving…" : exists ? "Save description" : "Save description"}</button>
            </div>
          </div>
          {note ? <p className={"pkg-notice " + (noteError ? "err" : "ok")} role={noteError ? "alert" : "status"}>{note}</p> : null}
        </>
      ) : null}
    </PageFrame>
  );
}
