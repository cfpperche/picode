import { useEffect, useRef, useState } from "react";
import * as AlertDialog from "@radix-ui/react-alert-dialog";

export default function ConfirmDialog() {
  const [req, setReq] = useState(null);
  const reqRef = useRef(null);

  useEffect(() => {
    function onAsk(e) {
      reqRef.current = e.detail;
      setReq(e.detail);
    }
    window.addEventListener("picode-confirm", onAsk);
    return () => window.removeEventListener("picode-confirm", onAsk);
  }, []);

  function finish(ok) {
    const r = reqRef.current;
    if (!r) return;
    reqRef.current = null;
    setReq(null);
    r.resolve(ok);
  }

  return (
    <AlertDialog.Root open={!!req} onOpenChange={(o) => { if (!o) finish(false); }}>
      <AlertDialog.Portal>
        <AlertDialog.Overlay className="dlg-overlay" />
        <AlertDialog.Content className="dlg" onCloseAutoFocus={(e) => e.preventDefault()}>
          <AlertDialog.Title className="dlg-title">{req ? req.title : ""}</AlertDialog.Title>
          <AlertDialog.Description className="dlg-body">{req ? req.message : ""}</AlertDialog.Description>
          <div className="dlg-actions">
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => finish(false)}>Cancel</button>
            <button
              type="button"
              className={"btn btn-sm " + (req && req.danger ? "btn-danger" : "btn-primary")}
              onClick={() => finish(true)}
            >
              {req ? req.confirmLabel : "Continue"}
            </button>
          </div>
        </AlertDialog.Content>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  );
}
