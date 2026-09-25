import { useEffect, useRef, useState } from "react";
import PageFrame from "./PageFrame.jsx";
import { useMissions } from "../lib/useMissions.js";
import { MISSION_STATES, missionLocation, missionHash, missionDraft, missionDraftPayload, missionDraftKey, missionActions, latestMissionEvidence } from "@picode/shared/domain/missions.js";
import { missionDraftSchema, missionUpdateSchema, missionEvidenceSchema } from "@picode/shared/contracts/schemas.js";
import "@picode/shared/styles/missions.css";

const LABELS = { relink: "Restore workspace", assign: "Assign agent", dispatch: "Send mission", acknowledge: "Confirm receipt", report: "Add update", decision: "Record decision", evidence: "Add evidence", "request-review": "Request review", accept: "Accept result", changes: "Request changes", pause: "Pause mission", resume: "Resume mission", cancel: "Cancel mission", release: "Release agent", reopen: "Reopen mission", archive: "Archive mission", edit: "Edit objective" };
function readDraft(key, fallback) { try { return JSON.parse(sessionStorage.getItem(key)) || fallback; } catch { return fallback; } }
function saveDraft(key, value) { try { sessionStorage.setItem(key, JSON.stringify(value)); } catch { /* storage is optional */ } }
function clearDraft(key) { try { sessionStorage.removeItem(key); } catch { /* storage is optional */ } }

export default function Missions() {
  const [loc, setLoc] = useState(() => missionLocation(location.hash) || { id: "", workspace: "" });
  useEffect(() => { const on = () => setLoc(missionLocation(location.hash) || { id: "", workspace: "" }); window.addEventListener("hashchange", on); return () => window.removeEventListener("hashchange", on); }, []);
  return <MissionPage key={loc.id + ":" + !!loc.create} loc={loc} />;
}

function MissionPage({ loc }) {
  const vm = useMissions(loc.id, loc.workspace);
  const { mission: v, data, workspaces, error, busy, reload, connected } = vm;
  const [action, setAction] = useState("");
  const [archived, setArchived] = useState(false);
  const evidenceRef = useRef(null);
  const actions = missionActions(v).filter(x => x !== "dispatch" || data?.observation?.canDispatch);
  const evidenceRevision = ["in-review", "completed"].includes(v?.state) && v.reviewRevision ? v.reviewRevision : data?.observation?.revision;
  const reportedCount = v?.criteria.filter(c => latestMissionEvidence(v, c.id, evidenceRevision)).length || 0;
  const showEvidence = () => { evidenceRef.current?.scrollIntoView({ behavior: "smooth", block: "start" }); evidenceRef.current?.focus({ preventScroll: true }); };
  const workspace = workspaces.find(w => w.id === (v?.workspaceId || loc.workspace));
  const act = async (name, fields = {}) => {
    const result = await vm.mutate(name, fields);
    if (result) {
      setAction("");
      if (result.submission?.error) vm.setError(result.submission.error + " Check the agent before continuing.");
      if (result.submission && !result.submission.error) vm.setError("");
    }
    return result;
  };
  return <PageFrame title={v?.title || (loc.create ? "New mission" : "Missions")} className="missions">
    {!connected && data ? <p className="mission-notice" role="status">Connection lost. Showing the last update. <button className="btn btn-sm" onClick={reload}>Reconnect</button></p> : null}
    {error ? <div className="mission-notice" role="alert">{error} <button className="btn btn-sm" onClick={reload}>Refresh</button></div> : null}
    {busy ? <p className="mission-muted" role="status">{LABELS[busy] || "Saving"}…</p> : null}
    {!data && !error ? <div className="mission-loading" aria-label="Loading missions"><span /><span /><span /></div> : null}
    {loc.create && data ? <MissionEditor vm={vm} workspace={loc.workspace} onCancel={() => { location.hash = missionHash(); }} /> : null}
    {!loc.id && !loc.create && data ? <>
      <div className="mission-toolbar" data-align-row><select aria-label="Filter by workspace" value={loc.workspace} onChange={e => { location.hash = "#/missions" + (e.target.value ? "?workspace=" + encodeURIComponent(e.target.value) : ""); }}><option value="">All workspaces</option>{workspaces.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}</select><a className="btn btn-primary" href={"#/missions/new" + (loc.workspace ? "?workspace=" + encodeURIComponent(loc.workspace) : "")}>Create mission</a></div>
      <label className="mission-muted"><input type="checkbox" checked={archived} onChange={e => setArchived(e.target.checked)} /> Include archived</label>
      <ul className="mission-list">{(data.missions || []).filter(m => archived || !m.archived).map(m => <li key={m.id}><a className="mission-row" href={missionHash(m.id)}><span><strong>{m.title}</strong><span className="mission-muted">{m.workspaceName}{m.assignment?.reserved ? " · " + m.assignment.agentName : " · No agent assigned"}{m.nextAction ? " · " + m.nextAction : ""}</span></span><span className="mission-state" data-state={m.state}>{MISSION_STATES[m.state]}</span></a></li>)}</ul>
      {!(data.missions || []).some(m => archived || !m.archived) ? <div className="mission-empty">No missions in this view. <a href="#/missions/new">Create mission</a></div> : null}
      {vm.hasMore ? <button className="btn" disabled={vm.moreBusy} onClick={vm.loadMore}>{vm.moreBusy ? "Loading…" : "Load more"}</button> : null}
    </> : null}
    {v ? <div className="mission-detail">
      <div className="mission-toolbar"><a href={"#/missions?workspace=" + encodeURIComponent(v.workspaceId)}>All missions</a><button className="btn btn-ghost" onClick={reload}>Refresh</button><span className="mission-state" data-state={v.state}>{MISSION_STATES[v.state]}</span></div>
      <section>{!data.observation?.available ? <p className="mission-notice">Working folder unavailable. Restore its location to continue this mission. <button className="btn" onClick={reload}>Refresh</button></p> : null}<p>{v.objective}</p><div className="mission-meta mission-muted"><span>{v.workspaceName}</span><span aria-hidden="true">·</span>{v.assignment && data.observation?.agentAvailable ? <a href={"#/agent/" + encodeURIComponent(v.assignment.agentId)}>{v.assignment.agentName} · {v.assignment.cli}</a> : <span>{v.assignment ? v.assignment.agentName + " · Unavailable" : "No agent assigned"}</span>}{v.assignment ? <span>{v.assignment.reserved ? "Assigned" : "Released"}</span> : null}</div></section>
      <section className="mission-primary"><h3>{v.blocker ? "Needs your attention" : "Next action"}</h3><p>{v.blocker || v.nextAction || (v.state === "completed" ? "Result accepted." : "Choose the next action for this mission.")}</p>
        {v.assignment?.reserved && ["paused", "cancelled"].includes(v.state) ? <p className="mission-muted">The agent may still be running. Open it before releasing responsibility.</p> : null}
        {v.assignment?.delivery === "unconfirmed" && v.assignment.reserved ? <p className="mission-muted">Receipt is not confirmed. Check the agent before continuing.</p> : null}
        {v.assignment?.reserved && v.assignment.delivery === "prepared" && !data.observation?.canDispatch ? <p className="mission-muted">Copy the mission context and continue in the agent. Direct sending is unavailable for this CLI.</p> : null}
        <div className="mission-actions">{v.criteria.length ? <button type="button" className="btn" onClick={showEvidence}>Read evidence ({reportedCount}/{v.criteria.length})</button> : null}{actions.filter(x => !["release", "archive", "cancel", "edit", "reopen", "relink"].includes(x)).map(x => <button type="button" className={"btn " + ((["dispatch", "accept", "decision", "acknowledge"].includes(x) || x === "assign" && !v.assignment?.reserved) ? "btn-primary" : "btn-ghost")} disabled={!!busy} key={x} onClick={() => setAction(x)}>{x === "assign" && v.assignment?.reserved ? "Transfer mission" : LABELS[x]}</button>)}</div>
      </section>
      {action === "edit" ? <MissionEditor vm={vm} base={v} onCancel={() => setAction("")} onSaved={() => setAction("")} /> : action ? <MissionAction key={action} action={action} vm={vm} workspace={workspace} onSubmit={act} onCancel={() => setAction("")} /> : null}
      <section className="mission-section mission-evidence-review" ref={evidenceRef} tabIndex={-1}><h3>Evidence for review</h3><p className="mission-muted">{reportedCount} of {v.criteria.length} criteria have reports for this version. These are reports; files named in notes are not attached.</p><ul className="mission-criteria">{v.criteria.map(c => { const e = latestMissionEvidence(v, c.id, evidenceRevision); return <li key={c.id}><strong>{c.text}</strong><div className="mission-evidence">{e ? <><span className="mission-evidence-status" data-outcome={e.outcome}>{e.actor === "owner" ? "Owner" : "Agent"} reported {e.outcome === "pass" ? "pass" : "fail"}</span><span className="mission-evidence-type">{e.kind === "file" ? "File reference · content not attached" : e.kind === "delivery" ? "Delivery ID" : "Observation"}</span><p>{e.value}</p><span>{new Date(e.at).toLocaleString()}</span>{e.revision ? <code>Revision {e.revision.slice(0, 12)}</code> : null}</> : "No evidence for this revision."}</div></li>; })}</ul>
        {v.reviewRevision ? <p className="mission-muted">Review revision: <code>{v.reviewRevision.slice(0, 12)}</code>{v.state !== "completed" && (data.observation?.revision !== v.reviewRevision || !data.observation?.clean) ? " · Candidate changed; refresh the review." : ""}</p> : null}
        {v.acceptedAt ? <p>Accepted {new Date(v.acceptedAt).toLocaleString()}.</p> : null}
        {v.inboxId ? <a href={"#/inbox/" + encodeURIComponent(v.inboxId)}>Open Inbox notification</a> : null}
      </section>
      <details className="mission-section"><summary>Context and decisions</summary><p>{v.context || "No additional context recorded."}</p><pre className="mission-packet">{data.context}</pre><button type="button" className="btn" onClick={async () => { try { await navigator.clipboard.writeText(data.context); } catch { vm.setError("Copy was unavailable. Select the context text and copy it."); } }}>Copy context</button></details>
      <section className="mission-section"><h3>History</h3><ul className="mission-history">{(data.history || []).map(h => { const entry = h.action === "evidence" ? h.snapshot?.evidence?.at(-1) : null; const criterion = entry && h.snapshot?.criteria?.find(c => c.id === entry.criterionId); return <li key={h.sequence}><strong>{LABELS[h.action] || h.action}</strong><time dateTime={h.at}>{new Date(h.at).toLocaleString()} · {h.actor === "owner" ? "Owner" : "Agent"}</time>{h.note ? <p>{h.note}</p> : null}{entry ? <details className="mission-history-evidence"><summary>Read this evidence</summary><p className="mission-muted">{criterion?.text || "Criterion unavailable"} · {entry.outcome === "pass" ? "Reported pass" : "Reported fail"} · {entry.kind === "file" ? "File reference" : entry.kind === "delivery" ? "Delivery ID" : "Observation"}</p><p>{entry.value}</p>{entry.revision ? <code>Revision {entry.revision.slice(0, 12)}</code> : null}</details> : null}{h.action === "assign" ? <p className="mission-muted">{h.snapshot.assignment?.agentName} · {h.snapshot.assignment?.cli}</p> : null}</li>; })}</ul>{vm.hasMore ? <button className="btn" disabled={vm.moreBusy} onClick={vm.loadMore}>{vm.moreBusy ? "Loading…" : "Earlier history"}</button> : null}</section>
      <div className="mission-actions">{actions.filter(x => ["release", "archive", "cancel", "edit", "reopen", "relink"].includes(x)).map(x => <button className="btn btn-ghost" disabled={!!busy} key={x} onClick={() => setAction(x)}>{x === "archive" && v.archived ? "Restore from archive" : LABELS[x]}</button>)}</div>
    </div> : null}
  </PageFrame>;
}

function MissionEditor({ vm, base, workspace, onCancel, onSaved }) {
  const key = missionDraftKey(base?.id);
  const [saved] = useState(() => readDraft(key, { draft: missionDraft(base, workspace), version: base?.version || 0 }));
  const [draft, setDraft] = useState(saved.draft), [error, setError] = useState("");
  const [version, setVersion] = useState(saved.version);
  useEffect(() => saveDraft(key, { draft, version }), [key, draft, version]);
  const field = name => ({ value: draft[name], onChange: e => setDraft(d => ({ ...d, [name]: e.target.value })) });
  const submit = async e => {
    e.preventDefault(); const parsed = missionDraftSchema.safeParse(draft);
    if (!parsed.success) { setError(parsed.error.issues[0].message); return; }
    const result = await vm.mutate(base ? "edit" : "create", { ...missionDraftPayload(parsed.data, base), ...(base ? { expectedVersion: version } : {}) });
    if (result) { clearDraft(key); if (onSaved) onSaved(); else location.hash = missionHash(result.mission.id); }
  };
  return <form className="mission-form mission-section" noValidate onSubmit={submit}>
    {error ? <p role="alert">{error}</p> : null}
    {base && version !== base.version ? <div className="mission-notice" role="alert">This mission changed. Review the latest objective before applying your draft.<details><summary>Latest saved objective</summary><p>{base.title}</p><p>{base.objective}</p><ul>{base.criteria.map(c => <li key={c.id}>{c.text}</li>)}</ul><p>{base.context}</p></details><button type="button" className="btn" onClick={() => setVersion(base.version)}>Apply draft to latest version</button></div> : null}
    {!base ? <label>Workspace<select {...field("workspaceId")}><option value="">Choose workspace</option>{vm.workspaces.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}</select></label> : null}
    <label>Title<input autoFocus {...field("title")} placeholder="Add password recovery" /></label>
    <label>Objective<textarea {...field("objective")} placeholder="Describe the result you want." /></label>
    <label>Acceptance criteria<textarea {...field("criteriaText")} placeholder={"A valid link lets the user set a new password.\nThe flow works on a phone."} /></label>
    <label>Next action<input {...field("nextAction")} placeholder="Inspect the existing sign-in flow" /></label>
    <details><summary>Context and decisions</summary><textarea aria-label="Context and decisions" {...field("context")} placeholder="Relevant files, decisions and constraints" /></details>
    <div className="mission-actions"><button className="btn btn-primary" disabled={!!vm.busy} type="submit">{vm.busy ? "Saving…" : base ? "Save objective" : "Create mission"}</button><button className="btn" type="button" onClick={onCancel}>Cancel</button></div>
  </form>;
}

function MissionAction({ action, vm, workspace, onSubmit, onCancel }) {
  const v = vm.mission, key = missionDraftKey(v.id + ":" + action);
  const [baseVersion] = useState(v.version);
  const [fields, setFields] = useState(() => readDraft(key, { workspaceId: v.workspaceId, filesReady: false, agentId: "", note: "", nextAction: v.nextAction || "", criterionId: v.criteria[0]?.id || "", kind: "note", value: "", outcome: "pass", sourceStopped: false, targetReady: false }));
  const [error, setError] = useState(""); const form = useRef(null);
  useEffect(() => { form.current?.querySelector("input:not([type=checkbox]),textarea,select,button")?.focus(); }, []);
  useEffect(() => saveDraft(key, { ...fields, sourceStopped: false, targetReady: false, filesReady: false }), [key, fields]);
  const field = name => ({ value: fields[name], onChange: e => setFields(f => ({ ...f, [name]: e.target.value })) });
  const check = name => ({ checked: fields[name], onChange: e => setFields(f => ({ ...f, [name]: e.target.checked })) });
  const noteAction = ["report", "decision", "changes", "block"].includes(action);
  const submit = async e => {
    e.preventDefault(); setError(""); let payload = {};
    if (noteAction) { const p = missionUpdateSchema.safeParse(fields); if (!p.success) { setError(p.error.issues[0].message); return; } payload = p.data; }
    if (action === "evidence") { const p = missionEvidenceSchema.safeParse(fields); if (!p.success) { setError(p.error.issues[0].message); return; } payload.evidence = p.data; }
    if (action === "assign") payload = { agentId: fields.agentId, targetReady: fields.targetReady, sourceStopped: fields.sourceStopped, filesReady: fields.filesReady };
    if (["accept", "release"].includes(action)) payload.sourceStopped = fields.sourceStopped;
    if (action === "relink") payload = { workspaceId: fields.workspaceId, filesReady: fields.filesReady };
    if (action === "acknowledge") payload.targetReady = fields.targetReady;
    if (action === "archive") payload.archived = !v.archived;
    if (await onSubmit(action, { ...payload, expectedVersion: baseVersion })) clearDraft(key);
  };
  return <form ref={form} className="mission-form mission-section" noValidate onSubmit={submit}>
    <h3>{action === "assign" && v.assignment?.reserved ? "Transfer mission" : LABELS[action]}</h3>{error ? <p role="alert">{error}</p> : null}
    {action === "relink" ? <><label>Workspace<select {...field("workspaceId")}>{vm.workspaces.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}</select></label><label><input type="checkbox" {...check("filesReady")} /> I restored the required files in this working folder.</label></> : null}
    {action === "assign" ? <><label>Responsible agent<select {...field("agentId")}><option value="">Choose agent</option>{(workspace?.agents || []).map(a => <option key={a.id} value={a.id}>{a.name} · {a.cli || "pi"}</option>)}</select></label>{!workspace?.agents?.length ? <p>No agents in this workspace. <a href="#/">Create an agent</a></p> : null}<details open><summary>Context to carry</summary><pre className="mission-packet">{vm.data.context}</pre></details><label><input type="checkbox" {...check("targetReady")} /> I checked that the target is ready and has no unrelated work or draft.</label>{v.assignment ? <label><input type="checkbox" {...check("filesReady")} /> Required files are already present in the target working folder.</label> : null}</> : null}
    {(["accept", "release"].includes(action) || action === "assign" && v.assignment?.reserved) ? <label><input type="checkbox" {...check("sourceStopped")} /> I confirmed that the current agent and its child processes have stopped writing.</label> : null}
    {action === "acknowledge" ? <label><input type="checkbox" {...check("targetReady")} /> I confirmed receipt in the assigned agent. Record this as my confirmation.</label> : null}
    {action === "dispatch" ? <><p>Send this context to {v.assignment?.agentName}.</p><pre className="mission-packet">{vm.data.context}</pre></> : null}
    {noteAction ? <><label>{action === "decision" ? "Your decision" : "Update"}<textarea {...field("note")} /></label><label>Next action<input {...field("nextAction")} /></label></> : null}
    {action === "evidence" ? <><label>Criterion<select {...field("criterionId")}>{v.criteria.map(c => <option key={c.id} value={c.id}>{c.text}</option>)}</select></label><label>Evidence type<select {...field("kind")}><option value="note">Observation or test result</option><option value="file">File in working folder</option><option value="delivery">Delivery ID</option></select></label><label>{fields.kind === "file" ? "Relative file path" : fields.kind === "delivery" ? "Delivery ID" : "Evidence"}<textarea {...field("value")} /></label><label>Reported outcome<select {...field("outcome")}><option value="pass">Pass</option><option value="fail">Fail</option></select></label></> : null}
    {action === "pause" || action === "cancel" ? <p>Future mission sends will stop. The agent may still be running; open it to stop its current work. Files will be kept.</p> : null}
    {action === "accept" ? <p>Accept the result against the current criteria and evidence. Integration remains a separate action in Delivery.</p> : null}
    <div className="mission-actions"><button className="btn btn-primary" disabled={!!vm.busy} type="submit">{vm.busy ? "Saving…" : LABELS[action]}</button><button className="btn" type="button" onClick={onCancel}>Back</button></div>
  </form>;
}
