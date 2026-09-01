import { useEffect, useState } from "react";
import CreateForm from "../../components/CreateForm.jsx";
import { submitCreate, formValues } from "../../lib/createSubmit.js";
import { api } from "../../lib/api.js";

// The desktop's create dialog is already a Vaul bottom sheet below 720px;
// this wrapper owns the kind, the model config and the submit, and hands
// the created agent back so the shell can open it.
export default function CreateSheet({ open, kind: initialKind, workspace, catalog, onClose, onCreated }) {
  const [kind, setKind] = useState(initialKind || "workspace");
  const [cfg, setCfg] = useState({ provider: "", model: "", thinking: "" });
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [sessions, setSessions] = useState(null);

  useEffect(() => {
    if (!open) return;
    setKind(initialKind || "workspace");
    setError("");
    setBusy(false);
  }, [open, initialKind]);

  useEffect(() => {
    if (!open || kind !== "session") return;
    setSessions(null);
    api("/api/pi-sessions").then((d) => setSessions((d && d.sessions) || [])).catch(() => setSessions([]));
  }, [open, kind]);

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

  async function onAdopt(path) {
    setError("");
    setBusy(true);
    try {
      const ag = await api("/api/pi-sessions/adopt", {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ path }),
      });
      onCreated({ kind: "free", created: ag });
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
      sessions={sessions}
      onAdopt={onAdopt}
      onKind={setKind}
      busy={busy}
    />
  );
}
