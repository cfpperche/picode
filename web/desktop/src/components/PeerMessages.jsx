import { useEffect, useState } from "react";
import { participantKey, participantState, participantOptions, selectedWorkspace } from "@picode/shared/domain/peerParticipants.js";
import { peerParticipantsSchema } from "@picode/shared/contracts/schemas.js";
import { useWorkspaceCommunication } from "../lib/useWorkspaceCommunication.js";
import { askConfirm } from "../lib/confirm.js";
import AgentClisFrame from "./AgentClisFrame.jsx";
import CliTabs from "./CliTabs.jsx";
import PeerConnectionDetails from "./PeerConnectionDetails.jsx";
import GrantedContacts from "./GrantedContacts.jsx";
import "./peer-messages.css";

export default function PeerMessages({ hidden, ownerKey = "" }) {
  // Granted contacts are not workspace-scoped (ADR-0116: a grant is per pair,
  // and half of what it is for is pairing two folders), so the audit list is a
  // sibling of the workspace section and never remounts with the picker. It is
  // Messages' own section — the grant lives in `peer_connections`, the canvas
  // is only where it was drawn (ADR-0109, amendment 2026-09-11).
  return <AgentClisFrame id="peer-messages-view" hidden={hidden}><CliTabs view="messages" /><WorkspaceMessages key={ownerKey} hidden={hidden} route={ownerKey} /><GrantedContacts hidden={hidden} /></AgentClisFrame>;
}
function WorkspaceMessages({ hidden, route }) {
  const [workspace, setWorkspace] = useState(route.startsWith("workspace:") ? route.slice(10) : "");
  const m = useWorkspaceCommunication(hidden, workspace);
  const resolved = selectedWorkspace(m.data, route);
  useEffect(() => { if (!workspace && resolved) setWorkspace(resolved); }, [workspace, resolved]);
  const [edits, setEdits] = useState({}), [advanced, setAdvanced] = useState(false), [details, setDetails] = useState("");
  const [from, setFrom] = useState(""), [to, setTo] = useState("");
  const ws = m.data?.workspaces.find(w => w.id === workspace);
  const owners = (m.data?.owners || []).filter(o => o.workspaceId === workspace && o.cli);
  const peers = (m.data?.connections || []).filter(c => c.workspaceId === workspace);
  const checks = (m.data?.checks || []).filter(c => c.workspaceId === workspace);
  const preference = o => m.data?.participants.find(p => participantKey(p) === participantKey(o));
  const enabled = o => { const p = preference(o); return !!p?.enabled && p.workspaceId === workspace && p.cli === o.cli; };
  const current = o => peers.find(p => p.active && participantKey(p) === participantKey(o));
  const selected = o => enabled(o) || !!current(o);
  const checked = o => edits[participantKey(o)] ?? selected(o);
  const changed = owners.filter(o => checked(o) !== selected(o) || (checked(o) && !enabled(o)));
  const status = o => participantState(o, preference(o), current(o), m.data?.live?.[participantKey(o)], checks, m.data?.identity?.[participantKey(o)], m.data?.recovery?.[participantKey(o)]);
  const participants = owners.filter(enabled);
  const connected = participants.filter(o => status(o).ready);
  const { sender, receivers, recipient } = participantOptions(participants, from, to);
  const canTest = sender && recipient && status(sender).ready && status(recipient).ready;
  const optionLabel = o => status(o).ready ? o.label : `${o.label} — ${status(o).connection}`;
  const pendingTest = checks.find(c => ["pending", "attempted", "running"].includes(c.phase));
  const latest = checks[0];
  const label = id => peers.find(p => p.id === id)?.label || "Removed participant";
  const openHash = o => o.kind === "agent" ? `#/agent/${o.ownerId}` : `#/term/${o.ownerId}`;
  async function apply(rows = changed) {
    const parsed = peerParticipantsSchema.safeParse({ participants: rows.map(o => ({ kind: o.kind, ownerId: o.ownerId, enabled: checked(o), revision: preference(o)?.revision || 0 })) });
    if (!parsed.success) { m.setError(parsed.error.issues[0].message); return; }
    const ok = await askConfirm({ title: "Apply communication?", message: "Selected participants can contact each other in this workspace, including future conversations. PiCode will prepare connections and resume supported open conversations when idle. Drafts or pending approvals pause setup.", confirmLabel: "Apply and connect" });
    if (ok && await m.mutate("participants", parsed.data)) setEdits({});
  }
  async function test() {
    if (!canTest) return;
    if (!await askConfirm({ title: "Test communication?", message: `${sender.label} and ${recipient.label} will exchange a test message in their existing conversations. This uses their configured models and may require tool approval.`, confirmLabel: "Run test" })) return;
    await m.mutate("test", { from: current(sender).id, to: current(recipient).id });
  }
  async function open(o) {
    if (!enabled(o) || m.data?.live?.[participantKey(o)]) { location.hash = openHash(o); return; }
    if (await m.mutate("open", { kind: o.kind, ownerId: o.ownerId, revision: preference(o)?.revision })) location.hash = openHash(o);
  }
  async function reconnect(o) {
    if (!await askConfirm({title:"Reconnect this conversation?",message:"PiCode will resume this same conversation after it becomes idle. Your history and browser draft stay available.",confirmLabel:"Reconnect"})) return;
    await m.mutate("open",{kind:o.kind,ownerId:o.ownerId,revision:preference(o)?.revision,reconnect:true});
  }
  return <section className="peer-body peer-workspace" aria-label="Workspace communication">
    <div className="peer-heading"><div><h3>Communication</h3><p>Choose who can talk to each other in this workspace.</p></div><button className="btn btn-ghost" disabled={!!m.busy || m.loading} onClick={() => { m.refresh(true); m.readHistory(); }}>Refresh</button></div>
    {m.error && <div className="cli-notice is-error" role="alert"><span>{m.error}</span><button className="btn btn-ghost" onClick={() => m.refresh(true)}>Try again</button></div>}
    {!m.data && m.loading ? <div className="cli-loading" aria-label="Loading participants"><div /><div /><div /></div> : null}
    {m.data && <label className="peer-picker">Workspace<select aria-label="Workspace" disabled={!!m.busy} value={workspace} onChange={e => { location.hash = `#/clis/messages/${encodeURIComponent(`workspace:${e.target.value}`)}`; }}><option value="">Choose a workspace</option>{m.data.workspaces.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}</select></label>}
    {m.data && !ws ? <div className="cli-notice"><span>{workspace ? "This workspace is no longer available." : "Choose a workspace to connect its agents."}</span>{!m.data.workspaces.length && <a className="btn btn-primary" href="#/">Open workspaces</a>}</div> : null}
    {ws && <>
      <form noValidate onSubmit={e => { e.preventDefault(); apply(); }} className="peer-participants">
        <div className="peer-history-heading"><h4>Participants</h4><button className="btn btn-primary" type="submit" disabled={!changed.length || !!m.busy}>{m.busy === "participants" ? "Applying…" : "Apply and connect"}</button></div>
        {!owners.length ? <div className="cli-notice"><span>Add an agent or Agent CLI to start a conversation.</span><a className="btn btn-primary" href={`#/clis/new/pi?workspace=${encodeURIComponent(workspace)}`}>New participant</a></div> : <ul className="peer-participant-list">{owners.map(o => {
          const p = preference(o), c = current(o), state = status(o);
          const picked = checked(o), on = enabled(o), editing = picked !== selected(o);
          return <li key={participantKey(o)} className={`peer-participant is-${state.kind}`}>
            <label className="peer-participant-choice"><input type="checkbox" checked={picked} disabled={!!m.busy} onChange={e => setEdits(v => ({ ...v, [participantKey(o)]: e.target.checked }))} /><span><strong>{o.label}</strong><small>{o.kind === "agent" ? "Pi agent" : o.cli}</small></span></label>
            <span className="peer-participant-state" role="status">{editing ? (picked ? "Will connect" : "Will disconnect") : <><span>{state.label}</span>{state.connection && <small>{state.connection}{state.verified ? " · Test passed" : ""}</small>}</>}</span>
            {(on || c) && <button className="btn btn-ghost" type="button" disabled={!!m.busy} onClick={() => open(o)}>{m.busy === "open" ? "Opening…" : m.data.live?.[participantKey(o)] ? "Open" : "Open and connect"}</button>}
            {on && (state.reason === "restart-required" || state.reason === "unobserved" || state.reason === "no-conversation" || (p?.problem && !["stopped","connected"].includes(p.phase))) ? <div className="peer-participant-problem"><span>{state.reason === "restart-required" ? "State recording failed. Restore storage access, then restart this CLI using its terminal controls." : state.reason === "unobserved" ? "Confirm the conversation in the terminal. It reconnects when its identity is available." : state.reason === "no-conversation" ? "Start a conversation in the terminal to connect." : p.problem}</span>{state.action === "terminal-controls" ? <a className="btn btn-ghost" href={`#/clis/${encodeURIComponent(o.cli)}/terminals`}>Terminal controls</a> : state.action === "packages" ? <a href="#/clis/pi/packages">Open Packages</a> : state.action === "reconnect" ? <button type="button" className="btn btn-ghost" disabled={!!m.busy} onClick={() => reconnect(o)}>Reconnect</button> : state.action === "retry" ? <button type="button" className="btn btn-ghost" disabled={!!m.busy} onClick={() => apply([o])}>Retry setup</button> : null}</div> : null}
          </li>;
        })}</ul>}
      </form>
      <section className="peer-test" aria-label="Connection test"><div><h4>Test communication</h4><p>{connected.length < 2 ? "Select two participants and apply. Unavailable participants stay listed with their connection status." : "Send a test message through the agents themselves."}</p></div>
        {participants.length > 0 && <div className="peer-test-controls"><label>From<select aria-label="From" value={sender ? participantKey(sender) : ""} disabled={!!pendingTest || !!m.busy} onChange={e => setFrom(e.target.value)}>{participants.map(o => <option key={participantKey(o)} value={participantKey(o)} disabled={!status(o).ready}>{optionLabel(o)}</option>)}</select></label><label>To<select aria-label="To" value={recipient ? participantKey(recipient) : ""} disabled={!!pendingTest || !!m.busy} onChange={e => setTo(e.target.value)}>{!receivers.length && <option value="">Select another participant</option>}{receivers.map(o => <option key={participantKey(o)} value={participantKey(o)} disabled={!status(o).ready}>{optionLabel(o)}</option>)}</select></label><button className="btn btn-primary" onClick={test} disabled={!canTest || !!pendingTest || !!m.busy}>{pendingTest || m.busy === "test" ? "Testing…" : "Run test"}</button></div>}
        {latest && <p className={`peer-test-result is-${latest.phase}`} role="status">{({pending:"Waiting for the sender to become available.",attempted:"Submitting the test…",running:"Waiting for the agents to exchange and acknowledge messages. Check their conversations for approvals.",passed:"Message, reply and both acknowledgments verified.",expired:"The test did not finish. Open the conversations to check permissions or model limits, then try again.",uncertain:"Test submission could not be confirmed. Check the sender before trying again.",cancelled:"The connection changed. Run a new test after reconnecting."})[latest.phase]}</p>}
      </section>
      <section aria-label="Communication activity"><div className="peer-history-heading"><h4>Activity</h4></div>{!m.history.length ? <p className="peer-empty">No messages yet. Once connected, ask an agent to talk to another participant.</p> : <ol className="peer-history">{m.history.map(v => <li key={v.id}><div className="peer-message-meta"><strong>{label(v.senderId)} → {label(v.recipientId)}</strong><time dateTime={v.createdAt}>{new Date(v.createdAt).toLocaleString()}</time><span>{v.ackedAt ? "Acknowledged" : v.attention === "notified" ? "Notified" : ["uncertain","attempted"].includes(v.attention) ? "Notification unconfirmed" : "Waiting to notify"}</span></div><p>{v.body}</p></li>)}</ol>}{m.hasOlder && <button className="btn btn-ghost" onClick={() => m.readHistory(m.history.at(-1)?.seq)}>Older messages</button>}</section>
      <details className="peer-advanced" open={advanced} onToggle={e => setAdvanced(e.currentTarget.open)}><summary>Advanced</summary>{advanced && <><label className="peer-picker">Participant<select value={details || (owners[0] && participantKey(owners[0])) || ""} onChange={e => setDetails(e.target.value)}>{owners.map(o => <option key={participantKey(o)} value={participantKey(o)}>{o.label}</option>)}</select></label>{owners.length > 0 && <PeerConnectionDetails hidden={hidden} ownerKey={details || participantKey(owners[0])} />}</>}</details>
    </>}
  </section>;
}
