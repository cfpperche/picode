import { useEffect, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { api } from "../lib/api.js";

export default function Changelog({ open, onClose }) {
  const [text, setText] = useState("");
  const [err, setErr] = useState("");
  useEffect(() => {
    if (!open) return;
    setErr("");
    setText("");
    api("/api/changelog/pi").then((d) => setText(d.text || "")).catch((e) => setErr(e.message || "Not found"));
  }, [open]);
  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-create dlg-changelog" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">pi changelog</Dialog.Title>
          <Dialog.Description className="dlg-body">Installed pi package on this machine.</Dialog.Description>
          {err ? <p className="form-error">{err}</p> : <pre className="changelog-pre">{text || "…"}</pre>}
          <div className="dlg-actions">
            <button type="button" className="btn btn-primary btn-sm" onClick={onClose}>Close</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
