import { useState } from "react";
import { usePeerCommunication, peerOwnerKey } from "../lib/usePeerCommunication.js";
import { principalLabel } from "@picode/shared/domain/managedPrincipal.js";
import { askConfirm } from "../lib/confirm.js";
import "./peer-messages.css";

export default function PeerConnectionDetails({ hidden, ownerKey = "" }) {
 return <Mailbox key={ownerKey} hidden={hidden} ownerKey={ownerKey} />;
}
function Mailbox({ hidden, ownerKey }) {
  const m = usePeerCommunication(hidden, ownerKey), [copied, setCopied] = useState(false), [copyError, setCopyError] = useState("");
  const liveState = m.active && m.data?.live?.[m.active.id];
  const liveLabel = ({ working: "Working", "needs-you": "Needs your input", idle: "Open", open: "Open" })[liveState];
  const problem = m.error || m.loadError || m.historyError;
  const locked = m.busy || !!m.loadError || !!m.historyError;
  const name = id => m.data?.connections.find(p => p.id === id)?.label || "Removed connection";
  async function change(action) {
    if (m.active) {
      const ok = await askConfirm({ title: action === "enable" ? "Replace connection?" : "Disable messages?", message: action === "enable" ? "The current connection will stop working. Resume this conversation to use its replacement." : "This conversation will no longer send or receive new messages. Its history stays available.", confirmLabel: action === "enable" ? "Replace" : "Disable", danger: true });
      if (!ok) return;
    }
    setCopied(false); setCopyError(""); await m.mutate(action);
  }
  async function copySetup() {
    try {
      const setup = { mcpServers: { "picode-communication": { url: new URL(m.secret.endpoint, location.origin).href, headers: { Authorization: `Bearer ${m.secret.token}` } } } };
      await navigator.clipboard.writeText(JSON.stringify(setup, null, 2)); setCopied(true); setCopyError("");
    } catch { setCopyError("Copy failed. Select the credential below to copy it manually."); }
  }
  return <section className="peer-body" aria-label="Session messages">
    <div className="peer-heading"><div><h3>Messages</h3><p>Direct messages between opted-in conversations.</p></div><button className="btn btn-ghost" disabled={m.loading || m.busy} onClick={() => { m.refresh(true); m.readHistory(); }}>{m.loading ? "Refreshing…" : "Refresh"}</button></div>
    {problem ? <div className="cli-notice is-error" role="alert"><span>{problem}</span>{m.errorCode === "adapter_missing" ? <a className="btn btn-ghost" href="#/clis/pi/packages">Open Packages</a> : <button className="btn btn-ghost" disabled={m.loading} onClick={() => { m.refresh(true); m.readHistory(); }}>Try again</button>}</div> : null}
    {!m.data && !problem ? <div className="cli-loading" aria-label="Loading messages"><div /><div /><div /></div> : null}
    {m.data?.owners.length === 0 ? <div className="cli-notice"><span>No agents yet.</span><a className="btn btn-primary" href="#/clis/new/pi">New agent</a></div> : null}
    {!!m.data?.owners.length && <>
      <label className="peer-picker">Conversation<select aria-label="Conversation" value={m.owner ? peerOwnerKey(m.owner) : ""} disabled={m.busy} onChange={e => { location.hash = `#/clis/messages/${encodeURIComponent(e.target.value)}`; }}>
        {!m.owner ? <option value="">Choose a conversation</option> : null}
        {m.data.owners.map(o => <option key={peerOwnerKey(o)} value={peerOwnerKey(o)}>{o.label} · {o.kind === "agent" ? principalLabel(o) : (o.cli || "Shell")}</option>)}
      </select></label>
      {!m.owner ? <div className="cli-notice"><span>This conversation is no longer available.</span><a className="btn btn-ghost" href="#/clis/messages">Choose another</a></div> : <>
        {!m.owner.sessionKey || !m.owner.cli ? <div className="cli-notice"><span>Open a conversation first so PiCode can identify it.</span><a className="btn btn-ghost" href={m.owner.kind === "agent" ? `#/agent/${m.owner.ownerId}` : "#/clis"}>Open {m.owner.kind === "agent" ? "agent" : "Agent CLIs"}</a></div> : <div className="peer-connection">
          <div><strong>{m.busy ? "Updating connection…" : m.active ? (m.data?.launches?.[m.active.id] ? "Configuration saved" : "Enabled · client setup required") : "Messages disabled"}</strong><p>Only opted-in conversations in this workspace can contact each other. New messages can notify an open conversation.</p></div>
          <div className="peer-actions" data-align-row data-align-wrap><button className="btn btn-primary" disabled={locked} onClick={() => change("enable")}>{m.busy ? "Updating…" : m.active ? "Replace connection" : "Enable messages"}</button>{m.active ? <button className="btn btn-ghost" disabled={locked} onClick={() => change("disable")}>Disable</button> : null}</div>
          {m.active && m.data?.launches?.[m.active.id] ? <div className="cli-notice"><span>Check Participants above for connection status and the next action.</span><a className="btn btn-ghost" href={m.owner.kind === "agent" ? `#/agent/${m.owner.ownerId}` : "#/clis"}>Open {m.owner.kind === "agent" ? "agent" : "Agent CLIs"}</a></div> : !m.data?.launchCLIs?.includes(m.owner.cli) ? <p>Automatic setup is not available for this CLI. Follow the <a href="https://cfpperche.github.io/picode/guide/communication" target="_blank" rel="noreferrer">connection guide</a>.</p> : null}
          {liveLabel ? <p role="status">{liveLabel}</p> : null}
          <details><summary>Recorded conversation</summary><code>{m.owner.sessionKey}</code></details>
        </div>}
        {m.secret ? <section className="peer-setup" aria-label="Connect this conversation"><h4>Connect this conversation</h4><p>Use this credential only for this conversation. It is shown once.</p>
          <div className="peer-actions" data-align-row data-align-wrap><button className="btn btn-primary" onClick={copySetup}>{copied ? "Copied" : "Copy MCP configuration"}</button><button className="btn btn-ghost" onClick={() => m.setSecret(null)}>Dismiss</button></div>
          {copyError ? <p role="alert">{copyError}</p> : null}
          <details><summary>Connection details</summary><label>Server URL<input readOnly value={new URL(m.secret.endpoint, location.origin).href} /></label><label>Authorization header<textarea readOnly aria-label="Authorization header" value={`Bearer ${m.secret.token}`} rows={3} /></label></details>
        </section> : null}
        <div className="peer-history-heading"><h4>History</h4>{m.connections.length > 1 ? <select aria-label="Connection history" value={m.selected?.id || ""} onChange={e => m.setHistoryId(e.target.value)}>{m.connections.map(p => <option key={p.id} value={p.id}>{p.active ? "Current" : "Previous"} · {new Date(p.createdAt).toLocaleString()}</option>)}</select> : null}</div>
        {!m.selected || m.history?.length === 0 ? <div className="cli-notice"><span>No messages in this connection yet.</span><a className="btn btn-ghost" href="https://cfpperche.github.io/picode/guide/communication" target="_blank" rel="noreferrer">Connection guide</a></div> : !m.history && !m.historyError ? <div className="cli-loading" aria-label="Loading history"><div /><div /></div> : null}
        {!!m.history?.length && <><ol className="peer-history">{m.history.map(message => <li key={message.id}>
          <div className="peer-message-meta"><strong>{name(message.senderId)} → {name(message.recipientId)}</strong><time dateTime={message.createdAt}>{new Date(message.createdAt).toLocaleString()}</time><span>{message.ackedAt ? "Acknowledged" : message.attention === "notified" ? "Notified · awaiting acknowledgement" : ["attempted", "uncertain"].includes(message.attention) ? "Stored · notification unconfirmed" : "Stored · notification pending"}</span></div>
          <p>{message.body}</p>{message.replyTo ? <small>Reply to {message.replyTo}</small> : null}
        </li>)}</ol>{m.hasOlder ? <button className="btn btn-ghost" disabled={m.reading} onClick={() => m.readHistory(true)}>{m.reading ? "Loading…" : "Older messages"}</button> : null}</>}
      </>}
    </>}
  </section>;
}
