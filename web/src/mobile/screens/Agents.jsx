import AgentRow from "../components/AgentRow.jsx";
import { agentsOf } from "../../lib/tree.js";
import { IconPlus } from "../../components/Icons.jsx";

// Every agent on the machine, grouped by workspace. The "+" sheet creates
// a workspace, a free agent or an agent inside a workspace.
export default function Agents({ loaded, workspaces, freeAgents, workingIds, busyId, onOpen, onStart, onStop, onCreate }) {
  const total = (workspaces || []).reduce((n, w) => n + agentsOf(w).length, 0) + (freeAgents || []).length;
  return (
    <div className="m-screen">
      <div className="m-screen-head">
        <h2 className="m-screen-title">Agents</h2>
        <button type="button" className="btn btn-primary btn-sm m-add" onClick={() => onCreate(workspaces && workspaces.length ? "agent" : "workspace")}>
          <IconPlus size={14} /> New
        </button>
      </div>
      {!loaded ? null : total === 0 ? (
        <div className="m-blank">
          <p className="m-blank-title">No agents yet</p>
          <p className="m-blank-sub">Add a project folder to create your first agent.</p>
          <button type="button" className="btn btn-primary" onClick={() => onCreate("workspace")}>Add workspace</button>
        </div>
      ) : (
        <>
          {(workspaces || []).map((ws) => (
            <section key={ws.id} className="m-section">
              <h3 className="m-section-label">{ws.name}<span className="m-section-sub">{ws.path}</span></h3>
              {agentsOf(ws).length === 0 ? (
                <p className="m-empty-line">Empty. <button type="button" className="btn-link" onClick={() => onCreate("agent", ws)}>Add an agent</button></p>
              ) : (
                <ul className="m-list">
                  {agentsOf(ws).map((a) => (
                    <AgentRow key={a.id} agent={a} workspace={ws} workingIds={workingIds} busy={busyId === a.id} onOpen={onOpen} onStart={onStart} onStop={onStop} />
                  ))}
                </ul>
              )}
            </section>
          ))}
          {(freeAgents || []).length ? (
            <section className="m-section">
              <h3 className="m-section-label">Free agents</h3>
              <ul className="m-list">
                {freeAgents.map((a) => (
                  <AgentRow key={a.id} agent={a} workspace={null} workingIds={workingIds} busy={busyId === a.id} onOpen={onOpen} onStart={onStart} onStop={onStop} />
                ))}
              </ul>
            </section>
          ) : null}
        </>
      )}
    </div>
  );
}
