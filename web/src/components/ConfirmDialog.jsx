import { useEffect, useRef, useState } from "react";
import * as AlertDialog from "@radix-ui/react-alert-dialog";

export default function ConfirmDialog() {
  const [req, setReq] = useState(null);
  const [picked, setPicked] = useState({});
  const reqRef = useRef(null);

  useEffect(() => {
    function onAsk(e) {
      const next = e.detail || {};
      const init = {};
      (next.choices || []).forEach((c) => { if (c && c.id) init[c.id] = !!c.checked; });
      reqRef.current = next;
      setPicked(init);
      setReq(next);
    }
    window.addEventListener("picode-confirm", onAsk);
    return () => window.removeEventListener("picode-confirm", onAsk);
  }, []);

  function finish(ok) {
    const r = reqRef.current;
    if (!r) return;
    reqRef.current = null;
    setReq(null);
    const choices = r.choices || [];
    if (!ok) {
      r.resolve(false);
      return;
    }
    if (!choices.length) {
      r.resolve(true);
      return;
    }
    r.resolve({ ...picked });
  }

  const choices = req && req.choices ? req.choices : [];

  return (
    <AlertDialog.Root open={!!req} onOpenChange={(o) => { if (!o) finish(false); }}>
      <AlertDialog.Portal>
        <AlertDialog.Overlay className="dlg-overlay" />
        <AlertDialog.Content className="dlg" onCloseAutoFocus={(e) => e.preventDefault()}>
          <AlertDialog.Title className="dlg-title">{req ? req.title : ""}</AlertDialog.Title>
          <AlertDialog.Description className="dlg-body">{req ? req.message : ""}</AlertDialog.Description>
          {choices.length ? (
            <div className="dlg-choices">
              {choices.map((c) => (
                <label key={c.id} className="dlg-choice">
                  <input
                    type="checkbox"
                    checked={!!picked[c.id]}
                    onChange={(e) => setPicked((cur) => ({ ...cur, [c.id]: e.target.checked }))}
                  />
                  <span>{c.label}</span>
                </label>
              ))}
            </div>
          ) : null}
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
