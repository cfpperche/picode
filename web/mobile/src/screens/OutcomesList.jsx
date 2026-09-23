import { useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { relTime } from "@picode/shared/domain/relTime.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import {
  choiceLabel, emptyExitDraft, exitHeadline, fmtExitCost, fmtLifetime, fmtTurns, pickOutcome, takesReasons, toggleReason,
} from "@picode/shared/domain/agentExit.js";
import { IconChevronRight } from "../components/Icons.jsx";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";
import { DOCS_BASE } from "../lib/commandDocs.js";

function cliName(id) {
  const label = terminalCliLabel(id);
  return label === "Terminal" ? id : label;
}

// Outcomes on the phone (ADR-0194): the same catalog the desktop page reads —
// how each removed agent ended, what it cost — with an answer given later
// and a record deleted. Numbers and filters stay on the desktop page; the
// phone shows the list and the one switch.
export default function OutcomesList() {
  const [data, setData] = useState(null); // { exits, next, ask, taxonomy }
  const [summary, setSummary] = useState(null);
  const [err, setErr] = useState("");
  const [open, setOpen] = useState("");
  const [edit, setEdit] = useState(null); // { id, draft }
  const seq = useRef(0);

  function load() {
    const n = ++seq.current;
    Promise.all([api("/api/agent-exits"), api("/api/agent-exits/summary")])
      .then(([list, sum]) => {
        if (n !== seq.current) return;
        setData(list);
        setSummary(sum.summary);
        setErr("");
      })
      .catch(() => { if (n === seq.current) setErr("Couldn't load outcomes."); });
  }

  useEffect(load, []);
  useEffect(() => subscribeFeed((ev) => {
    if (ev.type === "feed.open" || ev.type === "feed.reset" || String(ev.type || "").startsWith("agent_exit.")) load();
  }), []);

  const tax = data ? data.taxonomy : null;

  async function setAsk(on) {
    setData((d) => (d ? { ...d, ask: on } : d));
    try {
      await api("/api/agent-exits/prefs", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ ask: on }) });
    } catch (e) { toastError(e); load(); }
  }

  async function save() {
    const { id, draft } = edit;
    try {
      await api("/api/agent-exits/" + encodeURIComponent(id), {
        method: "PATCH", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ outcome: draft.outcome, reasons: takesReasons(tax, draft.outcome) ? draft.reasons : [], note: draft.outcome ? draft.note : "" }),
      });
      setEdit(null);
      toast("Outcome saved.", "ok");
      load();
    } catch (e) { toastError(e); }
  }

  async function remove(ex) {
    const ok = await askConfirm({
      title: "Delete this record?",
      message: `The record of "${ex.agentName}" leaves Outcomes and its numbers. The agent is already gone; nothing else changes.`,
      confirmLabel: "Delete", danger: true,
    });
    if (!ok) return;
    try {
      await api("/api/agent-exits/" + encodeURIComponent(ex.id), { method: "DELETE" });
      setOpen("");
      load();
    } catch (e) { toastError(e); }
  }

  if (err) {
    return (
      <div className="m-v2-lists m-outcomes">
        <div className="m-list-empty" role="alert"><p>{err}</p><button type="button" className="btn btn-sm" onClick={load}>Retry</button></div>
      </div>
    );
  }
  if (!data) return <div className="m-v2-lists m-outcomes"><p className="m-pin-msg">Loading…</p></div>;

  const head = exitHeadline(summary);
  const exits = data.exits;
  return (
    <div className="m-v2-lists m-outcomes" aria-label="Outcomes">
      <section className="m-section">
        <ul className="m-list m-menu m-group-list">
          <li className="m-row">
            <label className="m-row-main m-outc-ask">
              <span className="m-row-text">
                <span className="m-row-title">Ask when removing agents</span>
                <span className="m-row-sub">How did it go? — optional, never blocks Remove</span>
              </span>
              <input type="checkbox" role="switch" checked={!!data.ask} onChange={(e) => setAsk(e.target.checked)} />
            </label>
          </li>
        </ul>
      </section>
      {exits.length ? (
        <p className="m-outc-facts">
          {head.total} removed · {head.resolvedShare} resolved{head.attempts ? ` (${head.resolved} of ${head.attempts} tasks)` : ""}
          {summary && summary.costMeasured ? ` · ${fmtExitCost({ cost: summary.cost, estimated: summary.estimated })} measured` : ""}
        </p>
      ) : null}
      {exits.length === 0 ? (
        <div className="m-list-empty" role="status">
          <p>No agents removed yet — each one you remove lands here with how it went.</p>
          <a className="btn btn-sm" href={DOCS_BASE + "/guide/outcomes"} target="_blank" rel="noreferrer">How outcomes work</a>
        </div>
      ) : (
        <section className="m-section">
          <ul className="m-list m-menu m-group-list">
            {exits.map((ex) => {
              const isOpen = open === ex.id;
              const answer = ex.outcome ? choiceLabel(tax && tax.outcomes, ex.outcome) : ex.askSkip === "workspace" ? "With its workspace" : "No answer";
              return (
                <li key={ex.id} className={"m-row m-outc-row" + (isOpen ? " is-open" : "")}>
                  <button type="button" className="m-row-main" aria-expanded={isOpen} onClick={() => { setOpen(isOpen ? "" : ex.id); setEdit(null); }}>
                    <span className="m-row-text">
                      <span className="m-row-title">
                        {ex.agentName}{" "}
                        {ex.outcome ? <span className="exit-outcome" data-outcome={ex.outcome}>{answer}</span> : <span className="m-outc-none">{answer}</span>}
                      </span>
                      <span className="m-row-sub">
                        {[cliName(ex.cli), fmtLifetime(ex.lifetimeS), fmtExitCost(ex.cost), relTime(ex.removedAt)].join(" · ")}
                      </span>
                    </span>
                    <IconChevronRight size={16} className="m-row-chev m-outc-chev" />
                  </button>
                  {isOpen ? (
                    <div className="m-outc-detail">
                      {edit && edit.id === ex.id ? (
                        <Editor tax={tax} draft={edit.draft} onChange={(draft) => setEdit({ id: ex.id, draft })} onSave={save} onCancel={() => setEdit(null)} />
                      ) : (
                        <>
                          <dl className="m-outc-facts-list">
                            <dt>Outcome</dt>
                            <dd>{answer}{ex.reasons && ex.reasons.length ? " — " + ex.reasons.map((r) => choiceLabel(tax && tax.reasons, r)).join(", ") : ""}</dd>
                            {ex.note ? <><dt>Note</dt><dd className="m-outc-note">{ex.note}</dd></> : null}
                            <dt>Lived</dt><dd>{fmtLifetime(ex.lifetimeS)}, {fmtTurns(ex.turns)} turns</dd>
                            <dt>Cost</dt><dd>{fmtExitCost(ex.cost)}{ex.cost ? ` · ${ex.cost.scope === "agent" ? "every session" : "last session"}` : " — not measured"}</dd>
                            <dt>Setup</dt><dd>{[cliName(ex.cli), ex.model, ex.config && ex.config.thinking].filter(Boolean).join(" · ") || "—"}</dd>
                            <dt>Workspace</dt><dd>{ex.workspaceId === "ws_free" ? "Free" : ex.workspaceName || "—"}</dd>
                          </dl>
                          <div className="m-outc-actions">
                            <button type="button" className="btn btn-sm" onClick={() => setEdit({ id: ex.id, draft: { outcome: ex.outcome, reasons: ex.reasons || [], note: ex.note || "" } })}>{ex.outcome ? "Change answer" : "Answer"}</button>
                            <button type="button" className="btn btn-sm btn-danger" onClick={() => remove(ex)}>Delete record</button>
                          </div>
                        </>
                      )}
                    </div>
                  ) : null}
                </li>
              );
            })}
          </ul>
        </section>
      )}
    </div>
  );
}

function Editor({ tax, draft, onChange, onSave, onCancel }) {
  const d = draft || emptyExitDraft();
  if (!tax) return null;
  return (
    <div className="exit-ask m-outc-edit">
      <div className="exit-ask-head"><span className="exit-ask-title" id="m-outc-edit-title">How did it go?</span></div>
      <div className="exit-chips" role="group" aria-labelledby="m-outc-edit-title">
        {tax.outcomes.map((o) => (
          <button key={o.id} type="button" className="exit-chip" aria-pressed={d.outcome === o.id} onClick={() => onChange(pickOutcome(tax, d, o.id))}>{o.label}</button>
        ))}
      </div>
      {takesReasons(tax, d.outcome) ? (
        <>
          <div className="exit-ask-sub" id="m-outc-why">What got in the way?</div>
          <div className="exit-chips" role="group" aria-labelledby="m-outc-why">
            {tax.reasons.map((r) => (
              <button key={r.id} type="button" className="exit-chip" aria-pressed={d.reasons.includes(r.id)} onClick={() => onChange(toggleReason(d, r.id))}>{r.label}</button>
            ))}
          </div>
        </>
      ) : null}
      {d.outcome ? (
        <textarea className="dlg-input exit-note" rows={2} maxLength={1000} aria-label="Note" placeholder="Anything else worth remembering? (optional)" value={d.note} onChange={(e) => onChange({ ...d, note: e.target.value })} />
      ) : null}
      <div className="m-outc-actions m-outc-actions-end">
        <button type="button" className="btn btn-sm" onClick={onCancel}>Cancel</button>
        <button type="button" className="btn btn-sm btn-primary" onClick={onSave}>Save</button>
      </div>
    </div>
  );
}
