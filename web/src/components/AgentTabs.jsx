import { locate, displayAgentName } from "../lib/tree.js";

export default function AgentTabs({ tabs, workspaces, freeAgents, selectedId, onSelect, onClose, sessionSlot }) {
  return (
    <>
    <div id="main-tabs" className="main-tabs" hidden={tabs.length === 0}>
      <div id="tab-strip" className="tab-strip">
        {tabs.map((id) => {
          const loc = locate(workspaces, freeAgents, id);
          if (!loc || !loc.agent) return null;
          const mode = loc.agent.mode || "stopped";
          return (
            <div
              key={id}
              className={"mtab" + (id === selectedId ? " active" : "")}
              onClick={(e) => { if (e.target.closest(".mtab-close")) return; onSelect(id); }}
            >
              <span className={"mtab-dot" + (mode !== "stopped" ? " running" : "")} />
              <span>{displayAgentName(loc.agent, loc.workspace)}</span>
              <button className="mtab-close" title="Close tab (agent keeps running)" onClick={() => onClose(id)}>×</button>
            </div>
          );
        })}
      </div>
    </div>
    {tabs.length > 0 ? sessionSlot : null}
    </>
  );
}
