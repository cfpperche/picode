import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import PageFrame from "./PageFrame.jsx";
import StatTile from "./StatTile.jsx";
import RankedBars from "./RankedBars.jsx";
import DateRangePicker from "./DateRangePicker.jsx";
import { SwitchCtl } from "./settingsControls.jsx";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { relTime, absTime } from "@picode/shared/domain/relTime.js";
import {
  choiceLabel, emptyExitDraft, exitHeadline, fmtLifetime, fmtTurns, outcomeRows, pickOutcome, reasonRows, takesReasons, toggleReason,
} from "@picode/shared/domain/agentExit.js";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";
import { DOCS_BASE } from "../lib/commandDocs.js";
import "./outcomes.css";

const FREE = "ws_free";
const OUTCOME_FILTERS = ["resolved", "partial", "unresolved", "trial", "unanswered"];

function query(params) {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v) q.set(k, v);
  const s = q.toString();
  return s ? "?" + s : "";
}

// Outcomes (ADR-0194): how each removed agent ended — the person's answer
// beside what PiCode saw. A catalog first; the insights that read it come
// next (option B of the 2026-09-23 study), so nothing here guesses.
export default function Outcomes({ hidden, workspaces = [] }) {
  const [range, setRange] = useState("all");
  const [ws, setWs] = useState("");
  const [cli, setCli] = useState("");
  const [outcome, setOutcome] = useState("");
  const [data, setData] = useState(null); // { exits, next, ask, taxonomy }
  const [summary, setSummary] = useState(null);
  const [loadErr, setLoadErr] = useState("");
  const [open, setOpen] = useState("");
  const [editing, setEditing] = useState(null); // { id, draft }
  const [more, setMore] = useState(false);
  const seq = useRef(0);
  // A hidden page stays mounted (routes keep their state); it must not
  // refetch on every feed event nobody is looking at.
  const hiddenRef = useRef(hidden);
  hiddenRef.current = hidden;

  function load() {
    const n = ++seq.current;
    const base = { range, workspace: ws, cli };
    Promise.all([
      api("/api/agent-exits" + query({ ...base, outcome })),
      api("/api/agent-exits/summary" + query(base)),
    ]).then(([list, sum]) => {
      if (n !== seq.current) return;
      setData(list);
      setSummary(sum.summary);
      setLoadErr("");
    }).catch(() => {
      if (n !== seq.current) return;
      setLoadErr("Couldn't load outcomes.");
    });
  }

  useEffect(() => { if (!hidden) load(); }, [hidden, range, ws, cli, outcome]);

  // Every removal, label, undo or delete announces itself (ADR-0048); the
  // page refetches rather than patching, since a count moves with each.
  useEffect(() => subscribeFeed((ev) => {
    if (hiddenRef.current) return;
    if (ev.type === "feed.open" || ev.type === "feed.reset" || String(ev.type || "").startsWith("agent_exit.")) load();
  }), [range, ws, cli, outcome]);

  const tax = data ? data.taxonomy : null;
  const exits = data ? data.exits : null;
  const head = exitHeadline(summary);
  const filtered = !!(ws || cli || outcome || range !== "all");

  const wsOptions = useMemo(() => {
    const seen = new Map([[FREE, "Free agents"]]);
    for (const w of workspaces) seen.set(w.id, w.name);
    for (const ex of exits || []) if (ex.workspaceId && !seen.has(ex.workspaceId)) seen.set(ex.workspaceId, ex.workspaceName || ex.workspaceId);
    return [...seen.entries()];
  }, [workspaces, exits]);
  const cliOptions = useMemo(() => ((summary && summary.byCli) || []).map((r) => r.cli), [summary]);

  async function setAsk(on) {
    setData((d) => (d ? { ...d, ask: on } : d));
    try {
      await api("/api/agent-exits/prefs", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ ask: on }) });
    } catch (err) { toastError(err); load(); }
  }

  async function loadMore() {
    if (!data || !data.next || more) return;
    setMore(true);
    try {
      const page = await api("/api/agent-exits" + query({ range, workspace: ws, cli, outcome, before: data.next }));
      setData((d) => ({ ...d, exits: [...d.exits, ...page.exits], next: page.next }));
    } catch (err) { toastError(err); } finally { setMore(false); }
  }

  async function saveLabel() {
    if (!editing) return;
    const { id, draft } = editing;
    try {
      await api("/api/agent-exits/" + encodeURIComponent(id), {
        method: "PATCH", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ outcome: draft.outcome, reasons: takesReasons(tax, draft.outcome) ? draft.reasons : [], note: draft.outcome ? draft.note : "" }),
      });
      setEditing(null);
      toast.ok("Outcome saved.");
      load();
    } catch (err) { toastError(err); }
  }

  async function removeRecord(ex) {
    const ok = await askConfirm({
      title: "Delete this record?",
      message: `The record of "${ex.agentName}" leaves Outcomes and its numbers. The agent is already gone; nothing else changes.`,
      confirmLabel: "Delete", danger: true,
    });
    if (!ok) return;
    try {
      await api("/api/agent-exits/" + encodeURIComponent(ex.id), { method: "DELETE" });
      if (open === ex.id) setOpen("");
      load();
    } catch (err) { toastError(err); }
  }

  const firstLoad = !data && !loadErr;
  const empty = data && exits.length === 0;

  return (
    <PageFrame id="outcomes-view" title="Outcomes" hidden={hidden}>
      <div className="outc-top">
        <DateRangePicker value={range} onChange={setRange} name="outcomes-range" />
        <label className="outc-ask">
          <span>Ask when removing agents</span>
          <SwitchCtl checked={data ? data.ask : true} onChange={setAsk} label="Ask how it went when removing an agent" />
        </label>
      </div>

      {loadErr ? (
        <p className="outc-err" role="alert">{loadErr} <button type="button" className="btn-link" onClick={load}>Retry</button></p>
      ) : null}

      <div className="dash-kpi-row outc-kpis">
        <StatTile label="Removed" value={String(head.total)} loading={firstLoad} compareLabel={range === "all" ? "all time" : undefined} />
        <StatTile label="Resolved" value={head.resolvedShare} loading={firstLoad} compareLabel={head.answered ? `${head.resolved} of ${head.answered} answered` : "no answers yet"} />
        <StatTile label="Answered when asked" value={head.answerRate} loading={firstLoad} compareLabel={summary && summary.asked ? `${summary.askedAnswered} of ${summary.asked} asked` : "not asked yet"} />
        <StatTile
          label="Median life"
          value={summary && summary.total ? fmtLifetime(summary.medianLifetimeS) : "—"}
          loading={firstLoad}
          compareLabel={summary && summary.medianTurns != null ? `median ${summary.medianTurns} turns` : "turns not measured"}
        />
      </div>

      {!empty || filtered ? (
        <div className="dash-grid-2 outc-grid">
          <div className="dashboard-section">
            <div className="dash-section-label">How they ended</div>
            {firstLoad ? <span className="skel-line w-70" /> : <RankedBars items={outcomeRows(summary, tax)} empty="No agents removed in this period." />}
          </div>
          <div className="dashboard-section">
            <div className="dash-section-label">What got in the way</div>
            {firstLoad ? <span className="skel-line w-70" /> : <RankedBars items={reasonRows(summary, tax)} empty="No reasons given in this period." />}
          </div>
        </div>
      ) : null}

      <div className="outc-filters" role="group" aria-label="Filter removed agents">
        <select className="set-select" value={ws} onChange={(e) => setWs(e.target.value)} aria-label="Workspace">
          <option value="">All workspaces</option>
          {wsOptions.map(([id, name]) => <option key={id} value={id}>{name}</option>)}
        </select>
        <select className="set-select" value={cli} onChange={(e) => setCli(e.target.value)} aria-label="CLI">
          <option value="">All CLIs</option>
          {cliOptions.map((c) => <option key={c} value={c}>{c}</option>)}
        </select>
        <select className="set-select" value={outcome} onChange={(e) => setOutcome(e.target.value)} aria-label="Outcome">
          <option value="">Every outcome</option>
          {OUTCOME_FILTERS.map((o) => <option key={o} value={o}>{o === "unanswered" ? "No answer" : choiceLabel(tax && tax.outcomes, o)}</option>)}
        </select>
        <a className="btn btn-ghost btn-sm outc-export" href="/api/agent-exits/export" download>Export</a>
      </div>

      {firstLoad ? (
        <ul className="outc-list" aria-hidden="true">
          {[0, 1, 2].map((i) => <li key={i} className="outc-row outc-skel"><span className="skel-line w-70" /></li>)}
        </ul>
      ) : empty ? (
        filtered ? (
          <p className="outc-empty">Nothing matches these filters. <button type="button" className="btn-link" onClick={() => { setWs(""); setCli(""); setOutcome(""); setRange("all"); }}>Clear filters</button></p>
        ) : (
          <p className="outc-empty">No agents removed yet — each one you remove lands here with how it went. <a className="btn-link" href={DOCS_BASE + "/guide/outcomes"} target="_blank" rel="noreferrer">How outcomes work</a></p>
        )
      ) : exits ? (
        <ul className="outc-list">
          {exits.map((ex) => {
            const isOpen = open === ex.id;
            const edit = editing && editing.id === ex.id ? editing : null;
            return (
              <li key={ex.id} className={"outc-row" + (isOpen ? " is-open" : "")}>
                <button type="button" className="outc-row-head" aria-expanded={isOpen} onClick={() => { setOpen(isOpen ? "" : ex.id); setEditing(null); }}>
                  <span className="outc-name" title={ex.agentName}>{ex.agentName}</span>
                  <span className="outc-cli" title={[ex.cli, ex.model].filter(Boolean).join(" · ")}>
                    {ex.cli}{ex.model ? <span className="outc-sub"> · {ex.model}</span> : null}
                  </span>
                  <span className="outc-ws" title={ex.workspaceName}>{ex.workspaceId === FREE ? "Free" : ex.workspaceName || "—"}</span>
                  <span className="outc-num" title="How long it lived">{fmtLifetime(ex.lifetimeS)}</span>
                  <span className="outc-num" title={ex.turns == null ? "Born before PiCode counted turns" : "Turns"}>{fmtTurns(ex.turns)}</span>
                  <span className="outc-answer">
                    {ex.outcome
                      ? <span className="exit-outcome" data-outcome={ex.outcome}>{choiceLabel(tax && tax.outcomes, ex.outcome)}</span>
                      : <span className="outc-noanswer">{ex.askSkip === "workspace" ? "With its workspace" : "No answer"}</span>}
                  </span>
                  <span className="outc-when" title={absTime(ex.removedAt)}>{relTime(ex.removedAt)}</span>
                </button>
                {isOpen ? (
                  <div className="outc-detail">
                    {edit ? (
                      <LabelEditor tax={tax} edit={edit} onChange={(draft) => setEditing({ id: ex.id, draft })} onSave={saveLabel} onCancel={() => setEditing(null)} />
                    ) : (
                      <ExitDetail ex={ex} tax={tax} onLabel={() => setEditing({ id: ex.id, draft: { outcome: ex.outcome, reasons: ex.reasons || [], note: ex.note || "" } })} onDelete={() => removeRecord(ex)} />
                    )}
                  </div>
                ) : null}
              </li>
            );
          })}
        </ul>
      ) : null}
      {data && data.next ? (
        <div className="outc-more">
          <button type="button" className="btn btn-ghost btn-sm" disabled={more} onClick={loadMore}>{more ? "Loading…" : "Show more"}</button>
        </div>
      ) : null}
    </PageFrame>
  );
}

function Fact({ label, children }) {
  return (
    <>
      <dt>{label}</dt>
      <dd>{children}</dd>
    </>
  );
}

function ExitDetail({ ex, tax, onLabel, onDelete }) {
  const c = ex.config || {};
  const launch = c.launch || null;
  const sig = ex.signals || {};
  const setup = [
    ex.provider ? ["Provider", ex.provider] : null,
    ex.model ? ["Model", ex.model] : null,
    c.thinking ? ["Thinking", c.thinking] : null,
    c.opMode ? ["Tools", c.opMode] : null,
    c.checklist ? ["Checklist", c.checklist] : null,
    c.packages && c.packages.length ? ["Packages", c.packages.join(", ")] : null,
    launch && launch.args && launch.args.length ? ["Launch arguments", launch.args.join(" ")] : null,
    launch && launch.envKeys && launch.envKeys.length ? ["Environment set", launch.envKeys.join(", ")] : null,
    c.workPath ? ["Folder", c.workPath] : null,
  ].filter(Boolean);
  const answer = ex.outcome ? choiceLabel(tax && tax.outcomes, ex.outcome) : "No answer";
  return (
    <>
      <dl className="outc-facts">
        <Fact label="Outcome">
          {answer}
          {ex.reasons && ex.reasons.length ? <span className="outc-sub"> — {ex.reasons.map((r) => choiceLabel(tax && tax.reasons, r)).join(", ")}</span> : null}
        </Fact>
        {ex.note ? <Fact label="Note"><span className="outc-note">{ex.note}</span></Fact> : null}
        <Fact label="Lived">{fmtLifetime(ex.lifetimeS)}, {fmtTurns(ex.turns)} turns{ex.lastWorkedAt ? <span className="outc-sub"> · last worked {relTime(ex.lastWorkedAt)}</span> : null}</Fact>
        <Fact label="Asked you">{sig.inboxBlocking || 0} of {sig.inboxItems || 0} Inbox items needed you</Fact>
        {sig.checklistTotal ? <Fact label="Checklist">{sig.checklistDone} of {sig.checklistTotal} steps done</Fact> : null}
        {setup.map(([k, v]) => <Fact key={k} label={k}><span className="outc-mono">{v}</span></Fact>)}
        {c.extraPrompt ? <Fact label="Extra prompt"><span className="outc-note">{c.extraPrompt}</span></Fact> : null}
        <Fact label="Sessions">{ex.sessionsPurged ? "Deleted with the agent" : ex.sessions && (ex.sessions.piSessionPath || ex.sessions.cliSessionPath) ? <span className="outc-mono">{ex.sessions.piSessionPath || ex.sessions.cliSessionPath}</span> : "Not recorded"}</Fact>
      </dl>
      <div className="outc-actions">
        <button type="button" className="btn btn-sm" onClick={onLabel}>{ex.outcome ? "Change answer" : "Answer"}</button>
        <button type="button" className="btn btn-ghost btn-sm btn-danger" onClick={onDelete}>Delete record</button>
      </div>
    </>
  );
}

function LabelEditor({ tax, edit, onChange, onSave, onCancel }) {
  const d = edit.draft || emptyExitDraft();
  if (!tax) return null;
  return (
    <div className="exit-ask outc-edit">
      <div className="exit-chips" role="group" aria-label="Outcome">
        {tax.outcomes.map((o) => (
          <button key={o.id} type="button" className="exit-chip" aria-pressed={d.outcome === o.id} onClick={() => onChange(pickOutcome(tax, d, o.id))}>{o.label}</button>
        ))}
      </div>
      {takesReasons(tax, d.outcome) ? (
        <Fragment>
          <div className="exit-ask-sub">What got in the way?</div>
          <div className="exit-chips" role="group" aria-label="What got in the way">
            {tax.reasons.map((r) => (
              <button key={r.id} type="button" className="exit-chip" aria-pressed={d.reasons.includes(r.id)} title={r.hint || undefined} onClick={() => onChange(toggleReason(d, r.id))}>{r.label}</button>
            ))}
          </div>
        </Fragment>
      ) : null}
      {d.outcome ? (
        <textarea className="dlg-input exit-note" rows={2} maxLength={1000} aria-label="Note" placeholder="Anything else worth remembering? (optional)" value={d.note} onChange={(e) => onChange({ ...d, note: e.target.value })} />
      ) : null}
      <div className="outc-actions">
        <button type="button" className="btn btn-primary btn-sm" onClick={onSave}>Save</button>
        <button type="button" className="btn btn-ghost btn-sm" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  );
}
