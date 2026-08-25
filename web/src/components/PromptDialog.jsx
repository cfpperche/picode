import { useEffect, useRef, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";

export default function PromptDialog() {
  const [req, setReq] = useState(null);
  const [val, setVal] = useState("");
  const reqRef = useRef(null);
  const inputRef = useRef(null);

  useEffect(() => {
    function onAsk(e) {
      reqRef.current = e.detail;
      setVal(e.detail.defaultValue || "");
      setReq(e.detail);
    }
    window.addEventListener("picode-prompt", onAsk);
    return () => window.removeEventListener("picode-prompt", onAsk);
  }, []);

  function finish(v) {
    const r = reqRef.current;
    if (!r) return;
    reqRef.current = null;
    setReq(null);
    r.resolve(v);
  }

  return (
    <Dialog.Root open={!!req} onOpenChange={(o) => { if (!o) finish(null); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg" onOpenAutoFocus={(e) => { e.preventDefault(); inputRef.current?.focus(); }}>
          <Dialog.Title className="dlg-title">{req ? req.title : ""}</Dialog.Title>
          {req && req.message ? <Dialog.Description className="dlg-body">{req.message}</Dialog.Description> : <Dialog.Description className="sr-only">Enter a name</Dialog.Description>}
          <input
            ref={inputRef}
            className="dlg-input"
            value={val}
            onChange={(e) => setVal(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") { e.preventDefault(); finish(val.trim() || null); }
            }}
          />
          <div className="dlg-actions">
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => finish(null)}>Cancel</button>
            <button type="button" className="btn btn-primary btn-sm" onClick={() => finish(val.trim() || null)}>
              {req ? req.confirmLabel : "Save"}
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
