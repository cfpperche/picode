import { useEffect, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { api } from "../lib/api.js";
import { activeAccountLine, barTone, formatMoney, formatReset, usageCopy } from "../lib/providerUsage.js";

export default function UsageDialog({ provider, onClose, onSignIn }) {
  const open = !!provider;
  const [data, setData] = useState(null);
  const [loadErr, setLoadErr] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open) {
      setData(null);
      setLoadErr("");
      setBusy(false);
      return;
    }
    return load(false);
  }, [open, provider && provider.id]);

  function load(keep) {
    let cancelled = false;
    if (!keep) {
      setData(null);
      setLoadErr("");
    }
    setBusy(true);
    api("/api/providers/" + encodeURIComponent(provider.id) + "/usage")
      .then((rep) => {
        if (cancelled) return;
        setData(rep);
        setLoadErr("");
      })
      .catch(() => {
        if (cancelled) return;
        setLoadErr("Couldn't load usage.");
        if (!keep) setData(null);
      })
      .finally(() => {
        if (!cancelled) setBusy(false);
      });
    return () => { cancelled = true; };
  }

  const copy = loadErr ? { line: loadErr, action: "retry" } : usageCopy(data);
  const windows = (data && data.windows) || [];
  const showBars = data && data.status === "ok" && windows.length > 0;
  const showSkel = busy && !data && !loadErr;

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-usage" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">{provider ? provider.id : "Usage"}</Dialog.Title>
          <Dialog.Description className="dlg-body">
            {activeAccountLine(provider, data)}
            {data && data.plan ? " · " + data.plan : ""}
          </Dialog.Description>

          {showSkel ? (
            <div className="usage-windows" aria-hidden="true">
              <div className="usage-skel" />
              <div className="usage-skel" />
            </div>
          ) : null}

          {showBars ? (
            <div className="usage-windows">
              {windows.map((w) => (
                <UsageWindow key={w.id || w.label} w={w} />
              ))}
            </div>
          ) : null}

          {copy.line && !showSkel ? <p className="usage-empty">{copy.line}</p> : null}

          <div className="dlg-actions">
            {copy.action === "signin" ? (
              <button type="button" className="btn btn-primary btn-sm" onClick={onSignIn}>Sign in</button>
            ) : (
              <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={() => load(true)}>
                {busy && data ? "Refreshing…" : "Refresh"}
              </button>
            )}
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>Close</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function UsageWindow({ w }) {
  if (w.remaining != null && w.usedPercent == null) {
    return (
      <div className="usage-win">
        <div className="usage-win-head">
          <span className="usage-win-label">{w.label}</span>
          <span className="usage-win-meta usage-money">{formatMoney(w.remaining, w.unit)}</span>
        </div>
      </div>
    );
  }
  const used = w.usedPercent == null ? 0 : Number(w.usedPercent);
  const tone = barTone(used);
  return (
    <div className={"usage-win " + tone}>
      <div className="usage-win-head">
        <span className="usage-win-label">{w.label}</span>
        <span className="usage-win-meta">{Math.round(used)}%{w.resetsAt ? " · " + formatReset(w.resetsAt) : ""}</span>
      </div>
      <div className="usage-track">
        <span className="usage-fill" style={{ width: Math.max(0, Math.min(100, used)) + "%" }} />
      </div>
    </div>
  );
}
