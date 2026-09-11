import { useEffect, useMemo, useRef, useState } from "react";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { askConfirm } from "../lib/confirm.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { descriptorValuesSchema } from "@picode/shared/contracts/schemas.js";
import PageFrame from "./PageFrame.jsx";
import PiSpinner from "./PiSpinner.jsx";

// Generic configuration form for descriptor-declared packages
// (docs/plans/package-config-manifest.md; extends ADR-0099). The server
// carries the rules in the descriptor; this form renders them and validates
// with the same grammar before sending. It deliberately knows nothing about
// any package: no field is invented here, and a package without a
// descriptor never reaches this page. One file, one scope, no layer
// semantics — layer inheritance is what PackagesConfig (pi-roles) does.

const EMPTY = {};

function valuesOf(view) {
  const out = {};
  for (const f of view?.fields || []) out[f.key] = view?.layer?.values?.[f.key] ?? (f.type === "boolean" ? false : "");
  return out;
}

function typedValues(fields, values) {
  const out = {};
  for (const f of fields || []) {
    const v = values[f.key];
    out[f.key] = f.type === "number" ? (v === "" || v == null ? "" : Number(v)) : v;
  }
  return out;
}

export default function PackageConfigGeneric({ hidden, embedded = false, pkg, backHash = "#/clis/packages/pi" }) {
  const [view, setView] = useState(null);
  const [loadErr, setLoadErr] = useState("");
  const [values, setValues] = useState(EMPTY);
  const [errors, setErrors] = useState({});
  const [saving, setSaving] = useState(false);
  const [conflict, setConflict] = useState("");
  const [note, setNote] = useState("");
  const [noteError, setNoteError] = useState(false);
  const request = useRef(0);
  const current = useRef(null);
  current.current = { view, values, saving };

  const fields = useMemo(() => view?.fields || [], [view]);
  const schema = useMemo(() => descriptorValuesSchema(fields), [fields]);
  const dirty = !!view && !view.layer.invalid && JSON.stringify(values) !== JSON.stringify(valuesOf(view));

  function configURL() {
    return "/api/packages/config?package=" + encodeURIComponent(pkg);
  }

  async function load() {
    if (!pkg) { setView(null); return; }
    if (current.current.saving) return;
    const seq = ++request.current;
    try {
      const next = await api(configURL());
      if (seq !== request.current) return;
      const previous = current.current;
      setView(next);
      setLoadErr("");
      setConflict("");
      // Keep an in-flight draft when the file itself did not move.
      const unchanged = previous.view && previous.view.layer.path === next.layer.path &&
        JSON.stringify(previous.view.layer.values) === JSON.stringify(next.layer.values);
      setValues(unchanged && previous.values ? previous.values : valuesOf(next));
      setErrors({});
    } catch (err) {
      if (seq === request.current) setLoadErr(humanizeError(err.message || String(err)));
    }
  }

  function setConflichReset() {
    setConflict("");
  }

  useEffect(() => { if (!hidden) load(); }, [hidden, pkg]); // eslint-disable-line
  useEffect(() => {
    if (hidden) return;
    return subscribeFeed((ev) => {
      if (ev.type === "feed.open" || ev.type === "feed.reset") { load(); return; }
      if (ev.type === "packages.config") load();
    });
  }, [hidden, pkg]); // eslint-disable-line

  function setField(key, value) {
    setValues((v) => ({ ...v, [key]: value }));
    setNote("");
    setErrors((e) => ({ ...e, [key]: undefined }));
  }

  async function save(force = false) {
    const parsed = schema.safeParse(typedValues(fields, values));
    if (!parsed.success) {
      const next = {};
      for (const issue of parsed.error.issues) next[issue.path[0]] = issue.message;
      setErrors(next);
      return;
    }
    setErrors({});
    setSaving(true);
    try {
      const next = await api(configURL(), {
        method: "PUT",
        body: JSON.stringify({ package: pkg, config: parsed.data, force }),
      });
      setView(next);
      setValues(valuesOf(next));
      setConflict("");
      setNote("Saved.");
      setNoteError(false);
    } catch (err) {
      if (err.status === 409) setConflict(humanizeError(err.message || String(err)));
      else { setNote(humanizeError(err.message || String(err))); setNoteError(true); }
    } finally {
      setSaving(false);
    }
  }

  async function clearFile() {
    if (!view) return;
    if (!(await askConfirm({
      title: "Clear " + (view.title || pkg) + " config",
      message: "Delete " + view.path + "? The package falls back to its built-in behavior.",
      confirmLabel: "Delete file",
    }))) return;
    setSaving(true);
    try {
      const next = await api(configURL(), { method: "DELETE" });
      setView(next);
      setValues(valuesOf(next));
      setConflict("");
      setNote("File deleted.");
      setNoteError(false);
    } catch (err) {
      setNote(humanizeError(err.message || String(err)));
      setNoteError(true);
    } finally {
      setSaving(false);
    }
  }

  return (
    <PageFrame embedded={embedded} id="package-config-generic" title={(view?.title || pkg) + " settings"} hidden={hidden}>
      <div className="pkc-top">
        <a className="pkg-back" href={backHash}>← All packages</a>
        <span className="pkg-foot-spacer" />
      </div>

      {loadErr ? (
        <p className="pkg-notice warn" role="alert">{loadErr} <button type="button" className="btn btn-ghost btn-sm" onClick={load}>Retry</button></p>
      ) : null}
      {!view && !loadErr ? (
        <p className="pkg-loading" role="status"><PiSpinner title="Loading configuration" /> Loading configuration…</p>
      ) : null}

      {view ? (
        <>
          <p className="pkg-fine">{view.application} Writes <code>{view.path}</code> ({view.scope} scope) — the file stays the package's own source of truth.</p>

          {view.layer.invalid ? (
            <div className="pkg-notice err" role="alert">
              <p>{view.layer.invalid}</p>
              <p className="pkg-fine">The file was left untouched. Fix it in the editor, or replace it with this form.</p>
              <div className="pkg-notice-actions">
                <button type="button" className="btn btn-danger btn-sm" onClick={() => save(true)}>Replace file…</button>
              </div>
            </div>
          ) : null}
          {conflict ? (
            <div className="pkg-notice err" role="alert">
              <p>{conflict}</p>
              <div className="pkg-notice-actions">
                <button type="button" className="btn btn-sm" onClick={() => setConflict("")}>Go back</button>
                <button type="button" className="btn btn-danger btn-sm" onClick={() => save(true)}>Replace anyway</button>
              </div>
            </div>
          ) : null}
          {!view.layer.exists && !view.layer.invalid ? (
            <p className="pkg-notice">Not configured yet — saving creates <code>{view.path}</code>.</p>
          ) : null}

          <div className="pkc-section">
            {fields.map((f) => (
              <div className="pkc-row" key={f.key}>
                <label htmlFor={"pkgcfg-" + f.key}>{f.label}{f.required ? <span aria-hidden="true"> *</span> : null}</label>
                {f.type === "enum" ? (
                  <select id={"pkgcfg-" + f.key} className="pkc-select" disabled={saving}
                    value={values[f.key] ?? ""} onChange={(e) => setField(f.key, e.target.value)}>
                    <option value="">{f.required ? "Choose…" : "Not set"}</option>
                    {(f.options || []).map((o) => <option key={o} value={o}>{o}</option>)}
                  </select>
                ) : f.type === "boolean" ? (
                  <input id={"pkgcfg-" + f.key} type="checkbox" disabled={saving}
                    checked={!!values[f.key]} onChange={(e) => setField(f.key, e.target.checked)} />
                ) : (
                  <input id={"pkgcfg-" + f.key} className="pkc-input" disabled={saving}
                    type={f.type === "secret" ? "password" : f.type === "number" ? "number" : "text"}
                    value={values[f.key] ?? ""} onChange={(e) => setField(f.key, e.target.value)} />
                )}
                {f.help ? <p className="pkg-fine">{f.help}</p> : null}
                {errors[f.key] ? <p className="pkg-notice err" role="alert">{errors[f.key]}</p> : null}
              </div>
            ))}
            {fields.length === 0 ? <p className="pkg-fine">This configuration declares no fields.</p> : null}
          </div>

          <div className={dirty ? "pkc-foot dirty" : "pkc-foot"}>
            <span className="pkg-fine pkc-file" title={view.path}>{view.path}{view.layer.exists ? "" : " · not created yet"}</span>
            <div className="pkc-actions" data-align-row data-align-wrap>
              <button type="button" className="btn btn-ghost btn-sm" disabled={!dirty || saving} onClick={() => { setValues(valuesOf(view)); setErrors({}); }}>Discard</button>
              <button type="button" className="btn btn-ghost btn-sm" disabled={saving || !view.layer.exists} onClick={clearFile}>Clear file…</button>
              <button type="button" className="btn btn-primary btn-sm" disabled={saving || (!dirty && !view.layer.invalid)} onClick={() => save(false)}>{saving ? "Saving…" : "Save"}</button>
            </div>
          </div>
          {note ? <p className={"pkg-notice " + (noteError ? "err" : "ok")} role={noteError ? "alert" : "status"}>{note}</p> : null}
        </>
      ) : null}
    </PageFrame>
  );
}
