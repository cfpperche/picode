import { useEffect, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import PageFrame from "./PageFrame.jsx";
import ConfigFields from "./ConfigFields.jsx";
import ModeChip from "./ModeChip.jsx";
import { displayAgentName } from "../lib/tree.js";
import { api } from "../lib/api.js";
import { toast, toastError } from "../lib/toast.js";
import { catalogBase, PI_TOOLS, resolveLayer } from "../lib/resolveLayer.js";
import { IconX } from "./Icons.jsx";

const MODES = ["one-at-a-time", "all"];

export default function PiSettings({ hidden, agent, workspace, catalog, onAgentConfig }) {
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

  async function save(layer, patch, okMsg) {
    try {
      const next = await api("/api/pi-settings", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ agentId: agent ? agent.id : "", layer, patch }),
      });
      setRep(next);
      toast.ok(okMsg);
    } catch (e) { toastError(e); }
  }

  const floor = catalogBase(catalog);
  const g = rep && rep.global ? resolveLayer(rep.global, floor) : null;
  const p = rep && rep.project ? resolveLayer(rep.project, g || floor) : null;
  const parent = p || g || floor;
  const ag = agent ? {
    provider: agent.provider || parent.defaultProvider || "",
    model: agent.model || parent.defaultModel || "",
    thinking: agent.thinking || parent.defaultThinkingLevel || "",
  } : null;
  const canProject = !!(rep && rep.writable && rep.writable.project);

  return (
    <PageFrame id="pi-settings-view" title="Settings" hidden={hidden} wide>
      {!agent ? (
        <p className="settings-desc">Select an agent to edit pi settings.</p>
      ) : (
        <>
          <section className="settings-section" data-layer="global">
            <h3>Global</h3>
            <p className="settings-desc">This machine · ~/.pi/agent/settings.json</p>
            <LayerKnobs
              prefix="g"
              values={g}
              catalog={catalog}
              onSave={(patch) => save("global", patch, "Saved for every pi on this machine.")}
            />
          </section>
          {workspace ? (
            <section className="settings-section" data-layer="workspace">
              <h3>Workspace</h3>
              <p className="settings-desc">{workspace.name} · {workspace.path}</p>
              {!canProject ? (
                <p className="settings-desc">This folder is not trusted. Run /trust in the terminal.</p>
              ) : (
                <LayerKnobs
                  prefix="w"
                  values={p}
                  catalog={catalog}
                  onSave={(patch) => save("project", patch, "Saved for this folder.")}
                />
              )}
            </section>
          ) : null}
          <section className="settings-section" data-layer="agent">
            <h3>Agent</h3>
            <p className="settings-desc">{displayAgentName(agent, workspace)} · all sessions of this pi</p>
            <div className="set-rows" data-align-row>
              <div className="set-row set-row-stack">
                <span>Model</span>
                <ConfigFields
                  catalog={catalog}
                  provider={ag.provider}
                  model={ag.model}
                  thinking={ag.thinking}
                  onChange={(cfg) => onAgentConfig && onAgentConfig(cfg)}
                  idPrefix="ag-set"
                  row
                />
              </div>
              <div className="set-row">
                <span>Tools</span>
                <ModeChip cfg={{ opMode: agent.opMode || "full" }} onChange={(cfg) => onAgentConfig && onAgentConfig(cfg)} />
              </div>
            </div>
          </section>
        </>
      )}
    </PageFrame>
  );
}

function LayerKnobs({ prefix, values, catalog, onSave }) {
  if (!values) {
    return (
      <div className="set-rows" data-align-row aria-busy="true">
        {[0, 1, 2, 3].map((i) => (
          <div key={i} className="set-row" aria-hidden="true">
            <span className="skel-line w-50" />
            <span className="skel-line w-40" />
          </div>
        ))}
      </div>
    );
  }
  return (
    <div className="set-rows" data-align-row>
      <div className="set-row">
        <label htmlFor={prefix + "-compact"}>Auto-compact</label>
        <Switch.Root
          id={prefix + "-compact"}
          className="rx-switch"
          checked={!!values.compactionEnabled}
          onCheckedChange={(v) => onSave({ compactionEnabled: v })}
        >
          <Switch.Thumb className="rx-switch-thumb" />
        </Switch.Root>
      </div>
      <div className="set-row">
        <label htmlFor={prefix + "-steer"}>Steering</label>
        <select id={prefix + "-steer"} value={values.steeringMode || "one-at-a-time"} onChange={(e) => onSave({ steeringMode: e.target.value })}>
          {MODES.map((m) => <option key={m} value={m}>{m}</option>)}
        </select>
      </div>
      <div className="set-row">
        <label htmlFor={prefix + "-follow"}>Follow-up</label>
        <select id={prefix + "-follow"} value={values.followUpMode || "one-at-a-time"} onChange={(e) => onSave({ followUpMode: e.target.value })}>
          {MODES.map((m) => <option key={m} value={m}>{m}</option>)}
        </select>
      </div>
      <div className="set-row set-row-stack">
        <span>Defaults</span>
        <ConfigFields
          catalog={catalog}
          provider={values.defaultProvider || ""}
          model={values.defaultModel || ""}
          thinking={values.defaultThinkingLevel || ""}
          onChange={(cfg) => onSave({
            defaultProvider: cfg.provider,
            defaultModel: cfg.model,
            defaultThinkingLevel: cfg.thinking,
          })}
          idPrefix={prefix + "-def"}
          row
        />
      </div>
      <div className="set-row set-row-stack" id={prefix === "g" ? "scoped-models" : undefined}>
        <span>Scoped models</span>
        <PatternField list={values.enabledModels || []} onSave={(enabledModels) => onSave({ enabledModels })} />
      </div>
      <div className="set-row set-row-stack">
        <span>Tools</span>
        <div className="set-tools" data-align-row>
          {PI_TOOLS.map((t) => {
            const on = (values.defaultTools || []).includes(t);
            return (
              <label key={t} className="set-check">
                <input
                  type="checkbox"
                  checked={on}
                  onChange={() => {
                    const cur = new Set(values.defaultTools || []);
                    if (on) cur.delete(t); else cur.add(t);
                    onSave({ defaultTools: PI_TOOLS.filter((x) => cur.has(x)) });
                  }}
                />
                {t}
              </label>
            );
          })}
        </div>
      </div>
    </div>
  );
}

function PatternField({ list, onSave }) {
  const [draft, setDraft] = useState("");
  return (
    <div className="set-pats">
      <form
        className="set-pat-add"
        onSubmit={(e) => {
          e.preventDefault();
          const s = draft.trim();
          if (!s || list.includes(s)) return;
          onSave([...list, s]);
          setDraft("");
        }}
      >
        <input value={draft} onChange={(e) => setDraft(e.target.value)} placeholder="claude-* · gpt-4o" aria-label="Model pattern" />
        <button type="submit" className="btn btn-ghost btn-sm">Add</button>
      </form>
      {list.length === 0 ? (
        <p className="side-empty">All models</p>
      ) : (
        <ul className="set-pat-list">
          {list.map((pat) => (
            <li key={pat}>
              <code>{pat}</code>
              <button type="button" className="ws-icon-btn" title="Remove" onClick={() => onSave(list.filter((x) => x !== pat))}><IconX size={12} /></button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
