import PageFrame from "./PageFrame.jsx";
import { displayAgentName } from "../lib/tree.js";

export default function PiSettings({ hidden, agent, workspace }) {
  return (
    <PageFrame id="pi-settings-view" title="Settings" hidden={hidden} wide>
      {!agent ? (
        <p className="settings-desc">Select an agent to edit pi settings.</p>
      ) : (
        <>
          <section className="settings-section" data-layer="global">
            <h3>Global</h3>
            <p className="settings-desc">This machine · ~/.pi/agent/settings.json</p>
          </section>
          {workspace ? (
            <section className="settings-section" data-layer="workspace">
              <h3>Workspace</h3>
              <p className="settings-desc">{workspace.name} · {workspace.path}</p>
            </section>
          ) : null}
          <section className="settings-section" data-layer="agent">
            <h3>Agent</h3>
            <p className="settings-desc">{displayAgentName(agent, workspace)} · all sessions of this pi</p>
          </section>
        </>
      )}
    </PageFrame>
  );
}
