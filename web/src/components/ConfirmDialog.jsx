import { useEffect, useRef, useState } from "react";
import * as AlertDialog from "@radix-ui/react-alert-dialog";

export default function ConfirmDialog() {
  const [req, setReq] = useState(null);
  const [picked, setPicked] = useState({});
  const [typed, setTyped] = useState({});
  const reqRef = useRef(null);

  useEffect(() => {
    function onAsk(e) {
      const next = e.detail || {};
      const init = {};
      (next.choices || []).forEach((c) => { if (c && c.id) init[c.id] = !!c.checked; });
      reqRef.current = next;
      setPicked(init);
      setTyped({});
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
  // A checked choice that demands a typed confirmation (GitHub-style
  // "type the name to delete") blocks the confirm button until the
  // typed text matches exactly.
  const blocked = choices.some((c) => c.typed && picked[c.id] && (typed[c.id] || "").trim() !== c.typed.expected);

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
                <div key={c.id}>
                  <label className="dlg-choice">
                    <input
                      type="checkbox"
                      checked={!!picked[c.id]}
                      onChange={(e) => setPicked((cur) => ({ ...cur, [c.id]: e.target.checked }))}
                    />
                    <span>{c.label}</span>
                  </label>
                  {c.typed && picked[c.id] ? (
                    <div className="dlg-typed">
                      <p className="dlg-typed-hint">{c.typed.hint}</p>
                      <input
                        type="text"
                        className="dlg-input"
                        autoFocus
                        autoComplete="off"
                        spellCheck={false}
                        placeholder={c.typed.expected}
                        value={typed[c.id] || ""}
                        onChange={(e) => setTyped((cur) => ({ ...cur, [c.id]: e.target.value }))}
                      />
                    </div>
                  ) : null}
                </div>
              ))}
            </div>
          ) : null}
          <div className="dlg-actions">
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => finish(false)}>Cancel</button>
            <button
              type="button"
              className={"btn btn-sm " + (req && req.danger ? "btn-danger" : "btn-primary")}
              disabled={blocked}
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
