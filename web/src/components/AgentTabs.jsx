import { useState } from "react";
import { locate, displayAgentName } from "../lib/tree.js";
import { isTermTab, tabTermId } from "../lib/routes.js";
import { IconTerminal } from "./Icons.jsx";

export default function AgentTabs({ tabs, workspaces, freeAgents, terminals, selectedId, onSelect, onClose, onReorder, sessionSlot }) {
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
              <Tab key={id} id={id} active={id === selectedId} onSelect={onSelect} onClose={onClose} onReorder={onReorder} closeTitle="Close tab (terminal keeps running)">
                <span className="mtab-term"><IconTerminal size={13} /></span>
                <span>{term.name}</span>
              </Tab>
            );
          }
          const loc = locate(workspaces, freeAgents, id);
          if (!loc || !loc.agent) return null;
          const mode = loc.agent.mode || "stopped";
          return (
            <Tab key={id} id={id} active={id === selectedId} onSelect={onSelect} onClose={onClose} onReorder={onReorder} closeTitle="Close tab (agent keeps running)">
              <span className={"mtab-dot" + (mode !== "stopped" ? " running" : "")} />
              <span>{displayAgentName(loc.agent, loc.workspace)}</span>
            </Tab>
          );
        })}
      </div>
    </div>
    {tabs.length > 0 && !isTermTab(selectedId) ? sessionSlot : null}
    </>
  );
}

function Tab({ id, active, onSelect, onClose, onReorder, closeTitle, children }) {
  const [over, setOver] = useState(false);
  const [dragging, setDragging] = useState(false);
  return (
    <div
      className={"mtab" + (active ? " active" : "") + (over ? " drag-over" : "") + (dragging ? " dragging" : "")}
      draggable="true"
      onDragStart={(e) => {
        if (e.target.closest(".mtab-close")) { e.preventDefault(); return; }
        e.dataTransfer.setData("text/plain", id);
        e.dataTransfer.effectAllowed = "move";
        setDragging(true);
      }}
      onDragEnd={() => { setDragging(false); setOver(false); }}
      onDragOver={(e) => {
        if (!onReorder) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        setOver(true);
      }}
      onDragLeave={() => setOver(false)}
      onDrop={(e) => {
        e.preventDefault();
        setOver(false);
        const from = e.dataTransfer.getData("text/plain");
        if (from && from !== id && onReorder) onReorder(from, id);
      }}
      onClick={(e) => { if (e.target.closest(".mtab-close")) return; onSelect(id); }}
    >
      {children}
      <button type="button" className="mtab-close" draggable="false" title={closeTitle} onClick={() => onClose(id)}>×</button>
    </div>
  );
}
