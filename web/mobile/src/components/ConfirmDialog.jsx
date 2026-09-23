import { useEffect, useRef, useState } from "react";
import { Alert as AlertDialog } from "./MobileSheet.jsx";
import { emptyExitDraft, pickOutcome, takesReasons, toggleReason } from "@picode/shared/domain/agentExit.js";

export default function ConfirmDialog() {
  const [req, setReq] = useState(null);
  const [picked, setPicked] = useState({});
  const [typed, setTyped] = useState({});
  // The removal dialog's question (ADR-0194): optional, never blocks Remove.
  const [draft, setDraft] = useState(emptyExitDraft);
  const [stopAsking, setStopAsking] = useState(false);
  const reqRef = useRef(null);

  useEffect(() => {
    function onAsk(e) {
      const next = e.detail || {};
      const init = {};
      (next.choices || []).forEach((c) => { if (c && c.id) init[c.id] = !!c.checked; });
      reqRef.current = next;
      setPicked(init);
      setTyped({});
      setDraft(emptyExitDraft());
      setStopAsking(false);
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
    if (r.feedback) {
      r.resolve({ ...picked, feedback: { draft, shown: !stopAsking, stopAsking } });
      return;
    }
    if (!choices.length) {
      r.resolve(true);
      return;
    }
    r.resolve({ ...picked });
  }

  const choices = req && req.choices ? req.choices : [];
  const fb = req && req.feedback ? req.feedback : null;
  const tax = fb ? fb.taxonomy : null;
  // A checked choice that demands a typed confirmation (GitHub-style
  // "type the name to delete") blocks the confirm button until the
  // typed text matches exactly.
  const blocked = choices.some((c) => c.typed && picked[c.id] && (typed[c.id] || "").trim() !== c.typed.expected);

  return (
    <AlertDialog.Root open={!!req} onOpenChange={(o) => { if (!o) finish(false); }}>
      <AlertDialog.Portal>
        <AlertDialog.Overlay className="dlg-overlay" />
        <AlertDialog.Content className={"dlg" + (fb ? " dlg-exit" : "")} onCloseAutoFocus={(e) => e.preventDefault()}>
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
                  {c.checkedHint && picked[c.id] ? <p className="dlg-choice-hint">{c.checkedHint}</p> : null}
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
          {tax && !stopAsking ? (
            <div className="exit-ask" role="group" aria-labelledby="exit-ask-title">
              <div className="exit-ask-head">
                <span id="exit-ask-title" className="exit-ask-title">
                  How did it go? <span className="exit-ask-optional">Optional</span>
                </span>
                <button type="button" className="btn-link exit-ask-stop" onClick={() => setStopAsking(true)}>Stop asking</button>
              </div>
              <div className="exit-chips" role="group" aria-label="Outcome">
                {tax.outcomes.map((o) => (
                  <button
                    key={o.id}
                    type="button"
                    className="exit-chip"
                    aria-pressed={draft.outcome === o.id}
                    onClick={() => setDraft((d) => pickOutcome(tax, d, o.id))}
                  >
                    {o.label}
                  </button>
                ))}
              </div>
              {takesReasons(tax, draft.outcome) ? (
                <>
                  <div className="exit-ask-sub" id="exit-ask-why">What got in the way?</div>
                  <div className="exit-chips" role="group" aria-labelledby="exit-ask-why">
                    {tax.reasons.map((r) => (
                      <button
                        key={r.id}
                        type="button"
                        className="exit-chip"
                        aria-pressed={draft.reasons.includes(r.id)}
                        title={r.hint || undefined}
                        onClick={() => setDraft((d) => toggleReason(d, r.id))}
                      >
                        {r.label}
                      </button>
                    ))}
                  </div>
                </>
              ) : null}
              {draft.outcome ? (
                <textarea
                  className="dlg-input exit-note"
                  rows={2}
                  maxLength={1000}
                  aria-label="Note"
                  placeholder="Anything else worth remembering? (optional)"
                  value={draft.note}
                  onChange={(e) => { const note = e.target.value; setDraft((d) => ({ ...d, note })); }}
                />
              ) : null}
            </div>
          ) : null}
          {tax && stopAsking ? (
            <p className="exit-ask-stopped">
              Removing agents won't ask again. You can turn it back on in Outcomes.{" "}
              <button type="button" className="btn-link" onClick={() => setStopAsking(false)}>Keep asking</button>
            </p>
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
