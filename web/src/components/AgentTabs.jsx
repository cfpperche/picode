import { locate, displayAgentName } from "../lib/tree.js";
import { isTermTab, tabTermId } from "../lib/routes.js";
import { IconTerminal } from "./Icons.jsx";

export default function AgentTabs({ tabs, workspaces, freeAgents, terminals, selectedId, onSelect, onClose, sessionSlot }) {
  const terms = terminals || [];
  return (
    <>
    <div id="main-tabs" className="main-tabs" hidden={tabs.length === 0}>
      <div id="tab-strip" className="tab-strip">
        {tabs.map((id) => {
          if (isTermTab(id)) {
            const tid = tabTermId(id);
            const term = terms.find((t) => t.id === tid);
            if (!term) return null;
            return (
              <div
                key={id}
                className={"mtab" + (id === selectedId ? " active" : "")}
                onClick={(e) => { if (e.target.closest(".mtab-close")) return; onSelect(id); }}
              >
                <span className="mtab-term"><IconTerminal size={13} /></span>
                <span>{term.name}</span>
                <button className="mtab-close" title="Close tab (terminal keeps running)" onClick={() => onClose(id)}>×</button>
              </div>
            );
          }
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
    {tabs.length > 0 && !isTermTab(selectedId) ? sessionSlot : null}
    </>
  );
}
