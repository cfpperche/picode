import { useState } from "react";
import { verbLabel } from "../../lib/needsYou.js";
import { relTime, absTime } from "../../lib/relTime.js";

const INLINE_OPTIONS = 6;

// One decision, answerable in place. A live dialog offers the agent's own
// options; a blocking inbox item offers the verbs the item allows. Free
// text opens a textarea under the card, never a modal.
export default function NeedsYouCard({ entry, onAnswer, onRespond, onOpen }) {
  const [text, setText] = useState(entry.prefill || "");
  const [mode, setMode] = useState(""); // "" | "text" | verb needing text
  const [busy, setBusy] = useState(false);

  async function fire(fn) {
    setBusy(true);
    try { await fn(); } finally { setBusy(false); }
  }

  const isAsk = entry.kind === "ask";
  const where = isAsk ? [entry.agentName, entry.where && entry.where !== entry.agentName ? entry.where : ""].filter(Boolean).join(" · ") : (entry.reason || "");

  return (
    <article className={"m-card m-need" + (isAsk ? " is-ask" : " is-inbox")}>
      <div className="m-need-head">
        <span className="m-need-kind">{isAsk ? "Agent is waiting" : entry.inboxKind === "question" ? "Question" : "Approval"}</span>
        {entry.at ? <span className="m-need-when" title={absTime(entry.at)}>{relTime(entry.at)}</span> : null}
      </div>
      <button type="button" className="m-need-body" onClick={() => onOpen && onOpen(entry)}>
        <span className="m-need-title">{entry.title}</span>
        {where ? <span className="m-need-where">{where}</span> : null}
        {entry.message ? <span className="m-need-msg">{entry.message}</span> : null}
      </button>

      {isAsk ? (
        entry.method === "confirm" ? (
          <div className="m-need-actions">
            <button type="button" className="btn btn-primary" disabled={busy} onClick={() => fire(() => onAnswer(entry, { confirmed: true }))}>Yes</button>
            <button type="button" className="btn" disabled={busy} onClick={() => fire(() => onAnswer(entry, { confirmed: false }))}>No</button>
            <button type="button" className="btn btn-ghost" disabled={busy} onClick={() => fire(() => onAnswer(entry, { cancelled: true }))}>Cancel</button>
          </div>
        ) : entry.method === "select" && entry.options.length <= INLINE_OPTIONS ? (
          <div className="m-need-actions m-need-options">
            {entry.options.map((o) => (
              <button key={o} type="button" className="btn" disabled={busy} onClick={() => fire(() => onAnswer(entry, { value: o }))}>{o}</button>
            ))}
            <button type="button" className="btn btn-ghost" disabled={busy} onClick={() => fire(() => onAnswer(entry, { cancelled: true }))}>Cancel</button>
          </div>
        ) : entry.method === "select" ? (
          <div className="m-need-actions">
            <select className="dlg-input" value={text || entry.options[0] || ""} onChange={(e) => setText(e.target.value)} aria-label={entry.title}>
              {entry.options.map((o) => <option key={o} value={o}>{o}</option>)}
            </select>
            <button type="button" className="btn btn-primary" disabled={busy} onClick={() => fire(() => onAnswer(entry, { value: text || entry.options[0] || "" }))}>Send</button>
            <button type="button" className="btn btn-ghost" disabled={busy} onClick={() => fire(() => onAnswer(entry, { cancelled: true }))}>Cancel</button>
          </div>
        ) : (
          <div className="m-need-actions m-need-text">
            <textarea className="dlg-input" rows={entry.method === "editor" ? 4 : 2} placeholder={entry.placeholder || "Type your answer"} value={text} onChange={(e) => setText(e.target.value)} aria-label={entry.title} />
            <div className="m-need-actions">
              <button type="button" className="btn btn-primary" disabled={busy || !text.trim()} onClick={() => fire(() => onAnswer(entry, { value: text }))}>Send</button>
              <button type="button" className="btn btn-ghost" disabled={busy} onClick={() => fire(() => onAnswer(entry, { cancelled: true }))}>Cancel</button>
            </div>
          </div>
        )
      ) : (
        <div className="m-need-actions m-need-text">
          {mode ? (
            <>
              <textarea className="dlg-input" rows={3} placeholder="Your reply" value={text} onChange={(e) => setText(e.target.value)} aria-label="Reply" />
              <div className="m-need-actions">
                <button type="button" className="btn btn-primary" disabled={busy || !text.trim()} onClick={() => fire(() => onRespond(entry, mode, text))}>Send</button>
                <button type="button" className="btn btn-ghost" disabled={busy} onClick={() => setMode("")}>Back</button>
              </div>
            </>
          ) : (
            <div className="m-need-actions">
              {entry.verbs.map((v) => (
                <button
                  key={v}
                  type="button"
                  className={"btn" + (v === "accept" ? " btn-primary" : v === "ignore" ? " btn-ghost" : "")}
                  disabled={busy}
                  onClick={() => (v === "respond" || v === "edit" ? setMode(v) : fire(() => onRespond(entry, v, "")))}
                >
                  {verbLabel(v)}
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </article>
  );
}
