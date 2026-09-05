import { useEffect, useRef, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { webhookSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { destinationLabel, webhookPresets } from "@picode/shared/domain/integrations.js";
import { askConfirm } from "../lib/confirm.js";

const request = (path, method, body) => api(path, { method, headers: { "Content-Type": "application/json" }, ...(body === undefined ? {} : { body: JSON.stringify(body) }) });

export default function Webhooks({ hidden }) {
  const [rows, setRows] = useState(null);
  const [error, setError] = useState("");
  const [loadError, setLoadError] = useState("");
  const [busy, setBusy] = useState("");
  const [editor, setEditor] = useState(null);
  const [secret, setSecret] = useState("");
  const [result, setResult] = useState(null);
  const generation = useRef(0);
  async function load() {
    const gen = ++generation.current;
    try { const next = await api("/api/webhooks"); if (gen === generation.current) { setRows(next); setLoadError(""); } }
    catch (err) { if (gen === generation.current) setLoadError(err.message); }
  }
  useEffect(() => {
    if (hidden) return;
    load();
    const off = subscribeFeed(ev => { if (ev.type.startsWith("webhook.") || ["feed.open", "feed.reset"].includes(ev.type)) load(); });
    return () => { off(); generation.current++; };
  }, [hidden]);

  async function mutate(row, action) {
    if (busy) return;
    if (action === "remove" || action === "rotate") {
      const ok = await askConfirm({ title: action === "remove" ? "Remove webhook?" : "Replace signing secret?", message: action === "remove" ? "Stop future deliveries to " + destinationLabel(row.URL || row.url) + ". An in-flight request may still arrive." : "The receiver must use the new secret immediately. An in-flight request may still use the old one.", confirmLabel: action === "remove" ? "Remove" : "Replace secret", danger: true });
      if (!ok) return;
    }
    setBusy(row.id); setError(""); setResult(null);
    if (action === "toggle") setRows(old => old.map(w => w.id === row.id ? { ...w, enabled: !w.enabled } : w));
    try {
      const path = "/api/webhooks/" + encodeURIComponent(row.id);
      if (action === "toggle") await request(path, "PATCH", { revision: row.revision, enabled: !row.enabled });
      if (action === "remove") { await request(path, "DELETE"); setRows(old => old.filter(w => w.id !== row.id)); }
      if (action === "test") {
        const next = await request(path + "/test", "POST");
        setResult({ id: row.id, ok: next.delivered, message: next.delivered ? "Test received successfully." : next.error });
      }
      if (action === "rotate") { const next = await request(path + "/secret", "POST", { revision: row.revision }); setSecret(next.secret); }
    } catch (err) { setError(err.message); }
    finally { setBusy(""); load(); }
  }

  return <section hidden={hidden} className="integrations-webhooks" aria-label="Webhooks">
    <div className="integrations-heading"><p>Send PiCode events to your services.</p>{rows?.length ? <button className="btn btn-primary" onClick={() => setEditor({})}>Add webhook</button> : null}</div>
    {(error || loadError) ? <div className="integrations-notice" role="alert"><span>{error || loadError}</span><button className="btn btn-ghost" onClick={() => { setError(""); load(); }}>Refresh</button></div> : null}
    {rows === null && !loadError ? <div className="mcp-skel" aria-label="Loading webhooks"><div className="skel-line w-70" /><div className="skel-line w-40" /></div> : null}
    {rows?.length === 0 ? <div className="integrations-empty"><p>No webhooks yet.</p><button className="btn btn-primary" onClick={() => setEditor({})}>Add webhook</button></div> : null}
    {!!rows?.length && <ul className="integrations-list">{rows.map(row => <li key={row.id} className="integration-row" aria-busy={busy === row.id}>
      <div className="integration-main"><strong>{destinationLabel(row.url)}</strong><span className="pkg-fine">{row.types.join(" · ")}</span>
        <span className={"integration-status " + (row.lastStatus === "retrying" || row.lastStatus === "missed" ? "is-error" : "")}>{busy === row.id ? "Working…" : !row.enabled ? "Paused" : row.lastStatus === "delivered" ? "Last event delivered" : row.lastStatus === "retrying" ? "Delivery failed · retry scheduled" : row.lastStatus === "missed" ? "Some events expired" : "Waiting for an event"}</span>
        {row.lastError && <span className="pkg-fine">{row.lastError}</span>}
        {row.lastAttemptAt && <span className="pkg-fine">Last attempt: {new Date(row.lastAttemptAt).toLocaleString()}</span>}
        {result?.id === row.id && <span className={"integration-status " + (result.ok ? "" : "is-error")} role="status">Last test: {result.message}</span>}
      </div>
      <div className="integration-actions">
        <div className="integration-actions" data-align-row><button className="btn btn-ghost" role="switch" aria-label={"Enable " + destinationLabel(row.url)} aria-checked={row.enabled} disabled={!!busy} onClick={() => mutate(row, "toggle")}>{row.enabled ? "On" : "Off"}</button>
        <button className="btn btn-ghost" disabled={!!busy} onClick={() => mutate(row, "test")}>{busy === row.id ? "Working…" : "Send test"}</button>
        <button className="btn btn-ghost" disabled={!!busy} onClick={() => setEditor(row)}>Edit</button></div>
        <div className="integration-actions" data-align-row><button className="btn btn-ghost" disabled={!!busy} onClick={() => mutate(row, "rotate")}>Replace secret</button>
        <button className="btn btn-ghost" disabled={!!busy} onClick={() => mutate(row, "remove")}>Remove</button></div>
      </div>
    </li>)}</ul>}
    {editor && <WebhookEditor row={editor} onClose={() => setEditor(null)} onSaved={next => { setEditor(null); setResult(null); if (next.secret) setSecret(next.secret); load(); }} />}
    {secret && <SecretDialog secret={secret} onClose={() => setSecret("")} />}
  </section>;
}

function WebhookEditor({ row, onClose, onSaved }) {
  const [form, setForm] = useState({ url: row.url || "", types: row.types?.join(", ") || webhookPresets[0].types });
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  async function save(e) {
    e.preventDefault(); if (busy) return;
    const parsed = parseForm(webhookSchema, form);
    if (!parsed.ok) { setError(parsed.error); return; }
    setBusy(true); setError("");
    try {
      const path = "/api/webhooks" + (row.id ? "/" + encodeURIComponent(row.id) : "");
      const next = await request(path, row.id ? "PATCH" : "POST", { ...parsed.value, ...(row.id ? { revision: row.revision } : {}) });
      onSaved(next);
    } catch (err) { setError(err.message); setBusy(false); }
  }
  return <Dialog.Root open onOpenChange={open => { if (!open && !busy) onClose(); }}><Dialog.Portal><Dialog.Overlay className="dlg-overlay" /><Dialog.Content className="dlg integration-dialog">
    <Dialog.Title className="dlg-title">{row.id ? "Edit webhook" : "Add webhook"}</Dialog.Title>
    <Dialog.Description className="dlg-desc">Choose where to send events and what to include.</Dialog.Description>
    <form noValidate onSubmit={save} className="integration-form">
      <label>Receiver URL<input className="dlg-input" aria-label="Receiver URL" value={form.url} onChange={e => setForm({ ...form, url: e.target.value })} placeholder="https://your-service.example/webhook" autoComplete="off" disabled={busy} /></label>
      {form.url.startsWith("http://") && <p className="integration-status is-error">HTTP sends event data without encryption. Use it only on a trusted network.</p>}
      <label>Events<select className="dlg-input" aria-label="Events" disabled={busy} value={webhookPresets.find(p => p.types === form.types)?.types || "custom"} onChange={e => setForm({ ...form, types: e.target.value === "custom" ? "" : e.target.value })}>{webhookPresets.map(p => <option key={p.label} value={p.types}>{p.label}</option>)}<option value="custom">Custom events</option></select></label>
      <label>Event prefixes<input className="dlg-input" aria-label="Event prefixes" value={form.types} onChange={e => setForm({ ...form, types: e.target.value })} placeholder="agent., inbox." disabled={busy} /></label>
      <p className="pkg-fine">Matching events may include messages and project details. Only send them to services you trust.</p>
      {error && <p className="form-error" role="alert">{error}</p>}
      <div className="integration-actions" data-align-row><button type="button" className="btn btn-ghost" disabled={busy} onClick={onClose}>Cancel</button><button type="submit" className="btn btn-primary" disabled={busy}>{busy ? "Saving…" : row.id ? "Save changes" : "Create webhook"}</button></div>
    </form>
  </Dialog.Content></Dialog.Portal></Dialog.Root>;
}

function SecretDialog({ secret, onClose }) {
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState("");
  async function copy() { try { await navigator.clipboard.writeText(secret); setCopied(true); } catch { setError("Select the secret and copy it manually."); } }
  return <Dialog.Root open onOpenChange={open => { if (!open) onClose(); }}><Dialog.Portal><Dialog.Overlay className="dlg-overlay" /><Dialog.Content className="dlg integration-dialog">
    <Dialog.Title className="dlg-title">Save your signing secret</Dialog.Title><Dialog.Description className="dlg-desc">Copy it to your receiver now. PiCode will not show it again.</Dialog.Description>
    <input className="dlg-input" aria-label="Signing secret" value={secret} readOnly onFocus={e => e.target.select()} />
    {error && <p className="form-error" role="alert">{error}</p>}
    <div className="integration-actions" data-align-row><button className="btn btn-ghost" onClick={copy}>{copied ? "Copied" : "Copy secret"}</button><button className="btn btn-primary" onClick={onClose}>Done</button></div>
  </Dialog.Content></Dialog.Portal></Dialog.Root>;
}
