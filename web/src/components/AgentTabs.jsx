export default function AgentTabs({ tabs, workspaces, selectedId, onSelect, onClose, sessionSlot }) {
  return (
    <>
    <div id="main-tabs" className="main-tabs" hidden={tabs.length === 0}>
      <div id="tab-strip" className="tab-strip">
        {tabs.map((id) => {
          const ws = workspaces.find((w) => w.id === id);
          if (!ws) return null;
          const mode = ws.agent ? ws.agent.mode : "stopped";
          return (
            <div
              key={id}
              className={"mtab" + (id === selectedId ? " active" : "")}
              onClick={(e) => { if (e.target.closest(".mtab-close")) return; onSelect(id); }}
            >
              <span className={"mtab-dot" + (mode !== "stopped" ? " running" : "")} />
              <span>{ws.name}</span>
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
