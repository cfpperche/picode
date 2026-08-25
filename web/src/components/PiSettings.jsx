import { useEffect, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import PageFrame from "./PageFrame.jsx";
import ConfigFields from "./ConfigFields.jsx";
import { displayAgentName } from "../lib/tree.js";
import { api } from "../lib/api.js";
import { toast, toastError } from "../lib/toast.js";

const MODES = ["one-at-a-time", "all"];

export default function PiSettings({ hidden, agent, workspace, catalog }) {
  const [rep, setRep] = useState(null);

  useEffect(() => {
    if (hidden) return;
    let alive = true;
    const q = agent ? "?agentId=" + encodeURIComponent(agent.id) : "";
    api("/api/pi-settings" + q).then((data) => {
      if (alive) setRep(data);
    }).catch((e) => { if (alive) toastError(e); });
    return () => { alive = false; };
  }, [hidden, agent && agent.id]);

  async function save(patch) {
    try {
      const next = await api("/api/pi-settings", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ agentId: agent ? agent.id : "", layer: "global", patch }),
      });
      setRep(next);
      toast.ok("Saved for every pi on this machine.");
    } catch (e) { toastError(e); }
  }

  const g = rep && rep.global;

  return (
    <PageFrame id="pi-settings-view" title="Settings" hidden={hidden} wide>
      {!agent ? (
        <p className="settings-desc">Select an agent to edit pi settings.</p>
      ) : (
        <>
          <section className="settings-section" data-layer="global">
            <h3>Global</h3>
            <p className="settings-desc">This machine · ~/.pi/agent/settings.json</p>
            {!g ? (
              <div className="set-rows" data-align-row aria-busy="true">
                {[0, 1, 2, 3].map((i) => (
                  <div key={i} className="set-row" aria-hidden="true">
                    <span className="skel-line w-50" />
                    <span className="skel-line w-40" />
                  </div>
                ))}
              </div>
            ) : (
              <div className="set-rows" data-align-row>
                <div className="set-row">
                  <label htmlFor="g-compact">Auto-compact</label>
                  <Switch.Root
                    id="g-compact"
                    className="rx-switch"
                    checked={!!g.compactionEnabled}
                    onCheckedChange={(v) => save({ compactionEnabled: v })}
                  >
                    <Switch.Thumb className="rx-switch-thumb" />
                  </Switch.Root>
                </div>
                <div className="set-row">
                  <label htmlFor="g-steer">Steering</label>
                  <select id="g-steer" value={g.steeringMode || "one-at-a-time"} onChange={(e) => save({ steeringMode: e.target.value })}>
                    {MODES.map((m) => <option key={m} value={m}>{m}</option>)}
                  </select>
                </div>
                <div className="set-row">
                  <label htmlFor="g-follow">Follow-up</label>
                  <select id="g-follow" value={g.followUpMode || "one-at-a-time"} onChange={(e) => save({ followUpMode: e.target.value })}>
                    {MODES.map((m) => <option key={m} value={m}>{m}</option>)}
                  </select>
                </div>
                <div className="set-row set-row-stack">
                  <span>Defaults</span>
                  <ConfigFields
                    allowEmpty
                    catalog={catalog}
                    provider={g.defaultProvider || ""}
                    model={g.defaultModel || ""}
                    thinking={g.defaultThinkingLevel || ""}
                    onChange={(cfg) => save({
                      defaultProvider: cfg.provider,
                      defaultModel: cfg.model,
                      defaultThinkingLevel: cfg.thinking,
                    })}
                    idPrefix="g-def"
                    row
                  />
                </div>
              </div>
            )}
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
