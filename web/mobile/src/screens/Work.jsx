import { useRef, useState } from "react";
import AgentRow from "../components/AgentRow.jsx";
import TermRow from "../components/TermRow.jsx";
import { freeTerminals } from "../lib/termGroups.js";
import { shortPath } from "@picode/shared/domain/repoLine.js";
import { IconPlus, IconGit, IconFolder, IconSearch, IconX } from "../components/Icons.jsx";
import { WORK_SECTIONS } from "../lib/mobileRoutes.js";
import { agentMatchesSearch, terminalMatchesSearch, searchWorkspaceGroups } from "../lib/mobileListSearch.js";
import PullScreen from "../components/PullScreen.jsx";
import WsFavicon from "../components/WsFavicon.jsx";
import "../styles/mobile-lists.css";

const LABELS = { workspaces: "Workspaces", agents: "Agents", terminals: "Terminals" };
const SEARCH_LABELS = { workspaces: "Search workspaces and their work", agents: "Search free agents", terminals: "Search free terminals" };
let rememberedQuery = "";

// Paseo's workspace grouping, adapted to one focused phone list. Search
// retains the parent folder when it finds an agent or terminal inside it.
export default function Work({ section, onSection, loaded, error, workspaces, freeAgents, terminals, workingIds, busyId, checklists,
  onOpenAgent, onOpenTerm, onStart, onStop, onRemoveTerm, onCreate, onNewTerm, onOpenChanges, onOpenFiles, onOpenGit, onRefresh }) {
  const [query, updateQuery] = useState(() => rememberedQuery);
  const [searchOpen, setSearchOpen] = useState(() => !!rememberedQuery);
  const focusSearch = useRef(false);
  const setQuery = value => { rememberedQuery = value; updateQuery(value); };
  const sec = WORK_SECTIONS.includes(section) ? section : "workspaces";
  const groups = searchWorkspaceGroups(workspaces, terminals, query);
  const agents = (freeAgents || []).filter(agent => agentMatchesSearch(agent, query));
  const terms = freeTerminals(terminals).filter(term => terminalMatchesSearch(term, query));
  const rows = sec === "workspaces" ? groups : sec === "agents" ? agents : terms;
  const searching = Boolean(query.trim());
  const create = () => sec === "terminals" ? onNewTerm(null) : onCreate(sec === "agents" ? "free" : "workspace");
  const addLabel = sec === "workspaces" ? "Add workspace" : sec === "agents" ? "New agent" : "New terminal";
  return (
    <PullScreen scrollKey={"work:" + sec} onRefresh={onRefresh} className="m-v2-lists m-work-v2">
      <div className="m-screen-head m-list-head">
        <div className="m-list-toolbar" data-align-row>
          <select className="m-work-view" aria-label="Work view" value={sec} onChange={event => onSection(event.target.value)}>{WORK_SECTIONS.map(s => <option key={s} value={s}>{LABELS[s]}</option>)}</select>
          <button type="button" className="btn btn-ghost btn-sm" aria-label={searchOpen ? "Close search" : "Search work"} aria-expanded={searchOpen} aria-controls="mobile-work-search" onClick={() => { focusSearch.current = !searchOpen; if (searchOpen) setQuery(""); setSearchOpen(!searchOpen); }}>{searchOpen ? <IconX /> : <IconSearch />}</button>
          <button type="button" className="btn btn-primary btn-sm m-add" aria-label={addLabel} onClick={create}><IconPlus size={14} /> New</button>
        </div>
        {searchOpen ? <input id="mobile-work-search" type="search" className="dlg-input m-list-search" aria-label={SEARCH_LABELS[sec]} placeholder="Search work" value={query} onChange={event => setQuery(event.target.value)} ref={node => { if (node && focusSearch.current) { focusSearch.current = false; node.focus(); } }} /> : null}
      </div>
      {error ? <div className="m-list-notice" role="alert"><p>{loaded ? "Couldn’t refresh your work." : "Couldn’t load your work."}</p><button type="button" className="btn btn-sm" onClick={onRefresh}>Try again</button></div> : null}
      {!loaded ? (error ? null : <div className="m-list-loading" role="status" aria-label="Loading work">{[0, 1, 2].map(i => <div className="m-skel" key={i}><span className="skel-line w-70" /><span className="skel-line w-40" /></div>)}</div>) : rows.length === 0 ? (
        <div className="m-list-empty" role="status">
          <p>{searching ? "No matching work." : sec === "workspaces" ? "No workspaces yet." : sec === "agents" ? "No free agents yet." : "No free terminals yet."}</p>
          <button type="button" className="btn btn-sm" onClick={searching ? () => setQuery("") : create}>{searching ? "Clear search" : addLabel}</button>
        </div>
      ) : sec === "workspaces" ? groups.map(({ workspace: ws, agents: wsAgents, terminals: wsTerms }) => (
        <section key={ws.id} className="m-section m-work-group" aria-label={ws.name}>
          <div className="m-work-group-head">
            <WsFavicon ws={ws} size={17} />
            <div className="m-work-group-title"><h3>{ws.name}</h3><p title={ws.path}>{[shortPath(ws.path), ws.git?.branch].filter(Boolean).join(" · ")}</p></div>
            {ws.git?.dirty ? <button type="button" className="btn btn-ghost btn-sm m-changes-btn" aria-label={ws.git.dirty + " changes in " + ws.name} onClick={() => onOpenChanges("workspace", ws.id, ws.name)}><IconGit size={13} /> {ws.git.dirty}</button> : null}
          </div>
          {wsAgents.length || wsTerms.length ? <ul className="m-list m-group-list">
            {wsAgents.map(agent => <AgentRow key={agent.id} agent={agent} workspace={ws} workingIds={workingIds} checklist={checklists?.[agent.id]} busy={busyId === agent.id} onOpen={onOpenAgent} onStart={onStart} onStop={onStop} />)}
            {wsTerms.map(term => <TermRow key={term.id} term={term} busy={busyId === term.id} onOpen={onOpenTerm} onRemove={onRemoveTerm} />)}
          </ul> : <p className="m-empty-line m-work-empty">No agents or terminals yet.</p>}
          <div className="m-work-group-actions" data-align-row>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => onOpenFiles({ kind: "workspace", id: ws.id })}><IconFolder size={13} /> Files</button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => onOpenGit({ kind: "workspace", id: ws.id })}><IconGit size={13} /> Git</button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => onCreate("agent", ws)}><IconPlus size={13} /> Agent</button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => onNewTerm(ws)}><IconPlus size={13} /> Terminal</button>
          </div>
        </section>
      )) : <ul className="m-list m-group-list" aria-label={LABELS[sec]}>
        {sec === "agents" ? agents.map(agent => <AgentRow key={agent.id} agent={agent} workspace={null} workingIds={workingIds} checklist={checklists?.[agent.id]} busy={busyId === agent.id} onOpen={onOpenAgent} onStart={onStart} onStop={onStop} />)
          : terms.map(term => <TermRow key={term.id} term={term} busy={busyId === term.id} onOpen={onOpenTerm} onRemove={onRemoveTerm} />)}
      </ul>}
    </PullScreen>
  );
}
