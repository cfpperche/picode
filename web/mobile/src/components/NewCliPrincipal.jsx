import { useEffect, useState } from "react";
import * as Dialog from "./MobileSheet.jsx";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { managedPrincipalSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { catalogForPrincipal } from "@picode/shared/domain/managedPrincipal.js";

// Workspace catalog picker (ADR-0159 Fatia 3). Same fields as desktop;
// the sheet stays a sheet at every width (ADR-0072).

export default function NewCliPrincipal({ open, workspace, onClose, onCreated }) {
  const [status, setStatus] = useState("loading");
  const [clis, setClis] = useState([]);
  const [loadError, setLoadError] = useState("");
  const [cliId, setCliId] = useState("");
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [retry, setRetry] = useState(0);

  useEffect(() => {
    if (!open) return;
    setStatus("loading");
    setLoadError("");
    setError("");
    setBusy(false);
    setName("");
    setClis([]);
    setCliId("");
    let live = true;
    api("/api/clis").then((d) => {
      if (!live) return;
      const rows = catalogForPrincipal(d.clis || []);
      setClis(rows);
      setCliId(rows[0] ? rows[0].id : "");
      setStatus("ok");
    }).catch((e) => {
      if (!live) return;
      setLoadError(humanizeError(e && e.message ? e.message : String(e)));
      setStatus("err");
    });
    return () => { live = false; };
  }, [open, retry]);

  const selected = clis.find((c) => c.id === cliId);
  const title = "New Agent CLI" + (workspace && workspace.name ? " in " + workspace.name : "");

  async function onSubmit(e) {
    e.preventDefault();
    const parsed = parseForm(managedPrincipalSchema, { cli: cliId, name });
    if (!parsed.ok) { setError(parsed.error); return; }
    if (!workspace || !workspace.id) { setError("Pick a workspace first."); return; }
    setError("");
    setBusy(true);
    try {
      const body = { cli: parsed.value.cli };
      if (parsed.value.name) body.name = parsed.value.name;
      const created = await api("/api/workspaces/" + encodeURIComponent(workspace.id) + "/principals", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      onCreated(created);
    } catch (err) {
      setError(humanizeError(err && err.message ? err.message : String(err)));
    } finally {
      setBusy(false);
    }
  }

  let body = null;
  if (status === "loading") {
    body = (
      <div className="form-new" aria-label="Loading CLIs" role="status">
        <div className="skel-line" style={{ height: "var(--ctl-h)", width: "100%" }} />
        <div className="skel-line" style={{ height: "var(--ctl-h)", width: "100%" }} />
      </div>
    );
  } else if (status === "err") {
    body = (
      <div className="form-new">
        <p className="form-error" role="alert">{loadError || "Couldn’t load Agent CLIs."}</p>
        <div className="dlg-actions">
          <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancel</button>
          <button type="button" className="btn btn-primary btn-sm" onClick={() => setRetry((n) => n + 1)}>Try again</button>
        </div>
      </div>
    );
  } else if (!clis.length) {
    body = (
      <div className="form-new">
        <p>No coding CLIs installed.</p>
        <div className="dlg-actions">
          <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Cancel</button>
          <button type="button" className="btn btn-primary btn-sm" onClick={() => { onClose(); location.hash = "#/clis"; }}>Open Agent CLIs</button>
        </div>
      </div>
    );
  } else {
    body = (
      <form className="form-new create-form" noValidate onSubmit={onSubmit}>
        <select aria-label="CLI" value={cliId} onChange={(e) => setCliId(e.target.value)} disabled={!!busy}>
          {clis.map((c) => (
            <option key={c.id} value={c.id}>{c.name || c.id}</option>
          ))}
        </select>
        <input
          name="name"
          type="text"
          placeholder={selected && selected.name ? selected.name : "Name"}
          autoComplete="off"
          value={name}
          onChange={(e) => setName(e.target.value)}
          disabled={!!busy}
        />
        <p className="form-error" hidden={!error}>{error}</p>
        <div className="dlg-actions">
          <button type="button" className="btn btn-ghost btn-sm" onClick={onClose} disabled={!!busy}>Cancel</button>
          <button type="submit" className="btn btn-primary btn-sm" disabled={!!busy}>{busy ? "Creating…" : "Create"}</button>
        </div>
      </form>
    );
  }

  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-create" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">{title}</Dialog.Title>
          <Dialog.Description className="dlg-body">Pick a coding CLI for this folder.</Dialog.Description>
          {body}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
