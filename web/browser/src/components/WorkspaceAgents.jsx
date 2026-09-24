import { useState } from "react";
import { displayAgentName } from "@picode/shared/domain/tree.js";
import { agentStatusLabel } from "@picode/shared/domain/agentStatus.js";
import { terminalStatus, terminalStatusLabel } from "@picode/shared/domain/terminalCli.js";
import { relTime } from "@picode/shared/domain/relTime.js";
import { workspaceAgentsHash, workspaceOverviewHash } from "../lib/routes.js";

const summaryOrder = ["needs-you", "working", "compacting", "interactive", "open", "ready", "stopped"];

function statusSummary(rows) {
  const counts = new Map();
  for (const row of rows) counts.set(row.status, (counts.get(row.status) || 0) + 1);
  return summaryOrder.filter((status) => counts.has(status)).map((status) => {
    const count = counts.get(status);
    return `${count} ${agentStatusLabel(status).toLowerCase()}`;
  }).join(" · ");
}

function AgentSummaryRow({ row, workspace, onOpenAgent }) {
  const { agent, action, detail, status, stamp } = row;
  const age = stamp ? relTime(stamp) : "";
  return <li className="wo-agent-row">
    <div className="wo-agent-info">
      <div className="wo-agent-title"><strong>{displayAgentName(agent, workspace)}</strong><span className={`wo-agent-state is-${status}`}>{agentStatusLabel(status)}{age ? ` · ${age}` : ""}</span></div>
      <small>{agent.cli || "Pi"}{detail ? ` · ${detail}` : ""}</small>
    </div>
    {action.href ? <a className="wo-agent-action" href={action.href}>{action.label}</a>
      : <button className="wo-agent-action" type="button" onClick={() => onOpenAgent(agent.id)}>{action.label}</button>}
  </li>;
}

export default function WorkspaceAgents({ workspace, rows, shells, full = false, missionsError = false, retry, onOpenAgent, onOpenTerm, onNewAgent }) {
  const [query, setQuery] = useState("");
  const q = query.trim().toLocaleLowerCase();
  const visible = full ? rows.filter(({ agent, detail }) => !q || `${displayAgentName(agent, workspace)} ${agent.cli || "Pi"} ${detail}`.toLocaleLowerCase().includes(q)) : rows.slice(0, 4);
  const visibleShells = full ? shells.filter((term) => !q || `${term.name || "Terminal"} ${terminalStatusLabel(term) || terminalStatus(term)}`.toLocaleLowerCase().includes(q)) : shells.slice(0, 2);
  return <section className="wo-block wo-agents" aria-label="Agents">
    <div className="wo-block-head"><h3>Agents</h3>{full ? <a href={workspaceOverviewHash(workspace.id)}>Back to overview</a> : (rows.length > 4 || shells.length > 2) ? <a href={workspaceAgentsHash(workspace.id)}>View all ({rows.length + shells.length})</a> : null}</div>
    {rows.length ? <p className="wo-agent-summary">{statusSummary(rows)}</p> : null}
    {missionsError ? <p className="wo-note">Mission details unavailable. <button type="button" onClick={retry}>Retry</button></p> : null}
    {full && rows.length + shells.length > 0 ? <label className="wo-agent-search"><span>Find an agent or terminal</span><input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search agents" /></label> : null}
    {visible.length ? <ul className="wo-agent-list">{visible.map((row) => <AgentSummaryRow key={row.agent.id} row={row} workspace={workspace} onOpenAgent={onOpenAgent} />)}</ul> : null}
    {visibleShells.length ? <div className="wo-agent-terminals"><h4>Terminals</h4><ul className="wo-agent-list">{visibleShells.map((term) => <li className="wo-agent-row" key={term.id}><div className="wo-agent-info"><strong>{term.name || "Terminal"}</strong><small>{terminalStatusLabel(term) || terminalStatus(term)}</small></div><button className="wo-agent-action" type="button" onClick={() => onOpenTerm(term.id)}>Open terminal</button></li>)}</ul></div> : null}
    {!rows.length && !shells.length ? <p className="wo-empty">No agents here yet. <button type="button" onClick={() => onNewAgent(workspace.id)}>Create agent</button></p> : null}
    {full && q && !visible.length && !visibleShells.length ? <p className="wo-empty">No matching agents. <button type="button" onClick={() => setQuery("")}>Clear search</button></p> : null}
  </section>;
}
