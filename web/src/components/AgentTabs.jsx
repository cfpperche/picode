import { useState } from "react";
import { locate, displayAgentName } from "../lib/tree.js";
import { isTermTab, tabTermId, isFileTab, parseFileTab, isGitTab, gitTabKey, isTreeTab, treeTabRoot, isAppTab, tabAppId } from "../lib/routes.js";
import { repoNameFromKey } from "../lib/gitgraph.js";
import { IconTerminal, IconFile, IconGit, IconFolders } from "./Icons.jsx";
import AppIcon from "./AppIcon.jsx";

export default function AgentTabs({ tabs, workspaces, freeAgents, terminals, apps, selectedId, onSelect, onClose, onReorder, sessionSlot }) {
  const terms = terminals || [];
  const appList = apps || [];
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
          if (isFileTab(id)) {
            const f = parseFileTab(id);
            if (!f) return null;
            const name = (f.path || "").split("/").pop() || f.path || "File";
            return (
              <Tab key={id} id={id} active={id === selectedId} onSelect={onSelect} onClose={onClose} onReorder={onReorder} closeTitle="Close tab">
                <span className="mtab-term"><IconFile size={13} /></span>
                <span title={f.path}>{name}</span>
              </Tab>
            );
          }
          if (isGitTab(id)) {
            // The tab is the repository (ADR-0022), so its name comes from the
            // key rather than from whichever owner happened to open it.
            const name = repoNameFromKey(gitTabKey(id)) || "Git";
            return (
              <Tab key={id} id={id} active={id === selectedId} onSelect={onSelect} onClose={onClose} onReorder={onReorder} closeTitle="Close tab">
                <span className="mtab-term"><IconGit size={13} /></span>
                <span title={gitTabKey(id)}>{name}</span>
              </Tab>
            );
          }
          if (isTreeTab(id)) {
            // The tab is the folder (ADR-0030); its name is the folder's own.
            const root = treeTabRoot(id);
            const name = root.startsWith("@") ? "Files" : root.split("/").filter(Boolean).pop() || root;
            return (
              <Tab key={id} id={id} active={id === selectedId} onSelect={onSelect} onClose={onClose} onReorder={onReorder} closeTitle="Close tab">
                <span className="mtab-term"><IconFolders size={13} /></span>
                <span title={root}>{name}</span>
              </Tab>
            );
          }
          if (isAppTab(id)) {
            // The manifest may not have arrived yet — render the raw id
            // rather than null (a null tab silently vanishes, ADR-0036).
            const aid = tabAppId(id);
            const m = appList.find((a) => a.id === aid);
            return (
              <Tab key={id} id={id} active={id === selectedId} onSelect={onSelect} onClose={onClose} onReorder={onReorder} closeTitle="Close tab">
                <span className="mtab-term"><AppIcon name={m ? m.icon : ""} label={m ? m.name : aid} size={13} /></span>
                <span>{m ? m.name : aid}</span>
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
    {tabs.length > 0 && !isTermTab(selectedId) && !isFileTab(selectedId) && !isGitTab(selectedId) && !isTreeTab(selectedId) && !isAppTab(selectedId) ? sessionSlot : null}
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
