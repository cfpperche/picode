import { useEffect, useState } from "react";
import CreateForm from "./CreateForm.jsx";
import { submitCreate, formValues } from "../lib/createSubmit.js";

// The desktop's create dialog is already a Vaul bottom sheet below 720px;
// this wrapper owns the kind, the model config and the submit, and hands
// the created agent back so the shell can open it.
export default function CreateSheet({ open, kind: initialKind, workspace, catalog, onClose, onCreated }) {
  const [kind, setKind] = useState(initialKind || "workspace");
  const [cfg, setCfg] = useState({ provider: "", model: "", thinking: "" });
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open) return;
    setKind(initialKind || "workspace");
    setError("");
    setBusy(false);
  }, [open, initialKind]);

  async function onSubmit(e) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      const res = await submitCreate(kind, formValues(e.target), cfg, workspace ? workspace.id : "");
      e.target.reset();
      setCfg({ provider: "", model: "", thinking: "" });
      onCreated(res);
    } catch (err) {
      setError(err.message || String(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <CreateForm
      open={open}
      kind={kind}
      workspaceName={workspace ? workspace.name : ""}
      catalog={catalog}
      cfg={cfg}
      onCfg={setCfg}
      error={error}
      onSubmit={onSubmit}
      onClose={onClose}
      busy={busy}
    />
  );
}
