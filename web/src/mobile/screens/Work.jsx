import AgentRow from "../components/AgentRow.jsx";
import TermRow from "../components/TermRow.jsx";
import { agentsOf } from "../../lib/tree.js";
import { freeTerminals, workspaceTerminals } from "../../lib/termGroups.js";
import { shortPath } from "../../lib/repoLine.js";
import { IconPlus } from "../../components/Icons.jsx";
import { WORK_SECTIONS } from "../../lib/mobileRoutes.js";
import PullScreen from "../components/PullScreen.jsx";
import { IconGit } from "../../components/Icons.jsx";

const LABELS = { workspaces: "Workspaces", agents: "Agents", terminals: "Terminals" };

// Work mirrors the desktop sidebar's rail: the same three views, one
// segmented control. Workspaces = one card per folder with its agents and
// terminals; Agents = the free ones (no folder); Terminals = the free ones
// too — a workspace's terminal is listed once, on its card.
export default function Work({ section, onSection, loaded, workspaces, freeAgents, terminals, workingIds, busyId, checklists,
  onOpenAgent, onOpenTerm, onStart, onStop, onRemoveTerm, onCreate, onNewTerm, onOpenChanges, onRefresh }) {
  const sec = WORK_SECTIONS.includes(section) ? section : "workspaces";
  return (
    <PullScreen onRefresh={onRefresh}>
      <div className="m-screen-head">
        <div className="dash-range m-seg m-work-seg" role="radiogroup" aria-label="Work view">
          {WORK_SECTIONS.map((s) => (
            <label key={s} className="dash-range-opt">
              <input type="radio" name="m-work" value={s} checked={sec === s} onChange={() => onSection(s)} />
              <span className="dash-range-face">{LABELS[s]}</span>
            </label>
          ))}
        </div>
        {sec === "terminals" ? (
          <button type="button" className="btn btn-primary btn-sm m-add" onClick={() => onNewTerm(null)}><IconPlus size={14} /> New</button>
        ) : (
          <button type="button" className="btn btn-primary btn-sm m-add" onClick={() => onCreate(sec === "agents" ? "free" : "workspace")}><IconPlus size={14} /> New</button>
        )}
      </div>
      {!loaded ? null : sec === "workspaces" ? (
        <Workspaces workspaces={workspaces} terminals={terminals} workingIds={workingIds} busyId={busyId} checklists={checklists}
          onOpenAgent={onOpenAgent} onOpenTerm={onOpenTerm} onStart={onStart} onStop={onStop} onRemoveTerm={onRemoveTerm} onCreate={onCreate} onNewTerm={onNewTerm} onOpenChanges={onOpenChanges} />
      ) : sec === "agents" ? (
        <FreeAgents freeAgents={freeAgents} workingIds={workingIds} busyId={busyId} checklists={checklists} onOpenAgent={onOpenAgent} onStart={onStart} onStop={onStop} onCreate={onCreate} />
      ) : (
        <Terminals terminals={terminals} busyId={busyId} onOpenTerm={onOpenTerm} onRemoveTerm={onRemoveTerm} onNewTerm={onNewTerm} />
      )}
    </PullScreen>
  );
}

function Workspaces({ workspaces, terminals, workingIds, busyId, checklists, onOpenAgent, onOpenTerm, onStart, onStop, onRemoveTerm, onCreate, onNewTerm, onOpenChanges }) {
  if (!workspaces || workspaces.length === 0) {
    return (
      <div className="m-blank">
        <p className="m-blank-title">No workspaces yet</p>
        <p className="m-blank-sub">A workspace is a project folder. Agents and terminals live inside it.</p>
        <button type="button" className="btn btn-primary" onClick={() => onCreate("workspace")}>Add workspace</button>
      </div>
    );
  }
  return workspaces.map((ws) => {
    const agents = agentsOf(ws);
    const terms = workspaceTerminals(terminals, ws.id);
    return (
      <section key={ws.id} className="m-section m-ws-card">
        <h3 className="m-section-label">{ws.name}<span className="m-section-sub">{shortPath(ws.path)}{ws.git && ws.git.branch ? " · " + ws.git.branch : ""}</span></h3>
        {agents.length === 0 && terms.length === 0 ? (
          <p className="m-empty-line">Empty.</p>
        ) : (
          <ul className="m-list">
            {agents.map((a) => (
              <AgentRow key={a.id} agent={a} workspace={ws} workingIds={workingIds} checklist={checklists && checklists[a.id]} busy={busyId === a.id} onOpen={onOpenAgent} onStart={onStart} onStop={onStop} />
            ))}
            {terms.map((t) => (
              <TermRow key={t.id} term={t} busy={busyId === t.id} onOpen={onOpenTerm} onRemove={onRemoveTerm} />
            ))}
          </ul>
        )}
        <div className="m-ws-actions">
          <button type="button" className="btn btn-sm" onClick={() => onCreate("agent", ws)}><IconPlus size={13} /> Agent</button>
          <button type="button" className="btn btn-sm" onClick={() => onNewTerm(ws)}><IconPlus size={13} /> Terminal</button>
          {ws.git && ws.git.dirty ? (
            <button type="button" className="btn btn-sm m-changes-btn" title="Uncommitted changes" onClick={() => onOpenChanges("workspace", ws.id, ws.name)}><IconGit size={13} /> {ws.git.dirty}</button>
          ) : null}
        </div>
      </section>
    );
  });
}

function FreeAgents({ freeAgents, workingIds, busyId, checklists, onOpenAgent, onStart, onStop, onCreate }) {
  if (!freeAgents || freeAgents.length === 0) {
    return (
      <div className="m-blank">
        <p className="m-blank-title">No free agents</p>
        <p className="m-blank-sub">A free agent works in its own folder, outside any workspace.</p>
        <button type="button" className="btn btn-primary" onClick={() => onCreate("free")}>New agent</button>
      </div>
    );
  }
  return (
    <ul className="m-list">
      {freeAgents.map((a) => (
        <AgentRow key={a.id} agent={a} workspace={null} workingIds={workingIds} checklist={checklists && checklists[a.id]} busy={busyId === a.id} onOpen={onOpenAgent} onStart={onStart} onStop={onStop} />
      ))}
    </ul>
  );
}

function Terminals({ terminals, busyId, onOpenTerm, onRemoveTerm, onNewTerm }) {
  const free = freeTerminals(terminals);
  if (free.length === 0) {
    return (
      <div className="m-blank">
        <p className="m-blank-title">No free terminals</p>
        <p className="m-blank-sub">A free terminal lives outside any workspace; a workspace's terminals are on its card.</p>
        <button type="button" className="btn btn-primary" onClick={() => onNewTerm(null)}>New terminal</button>
      </div>
    );
  }
  return (
    <ul className="m-list">
      {free.map((t) => (
        <TermRow key={t.id} term={t} busy={busyId === t.id} onOpen={onOpenTerm} onRemove={onRemoveTerm} />
      ))}
    </ul>
  );
}
