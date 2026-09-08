import { useEffect, useRef, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import PageFrame from "./PageFrame.jsx";
import ConfigFields from "./ConfigFields.jsx";
import ModeChip from "./ModeChip.jsx";
import ChecklistChip from "./ChecklistChip.jsx";
import { displayAgentName } from "@picode/shared/domain/tree.js";
import { api } from "@picode/shared/client/api.js";
import { toast, toastError } from "../lib/toast.js";
import { catalogBase, PI_TOOLS, resolveLayer } from "@picode/shared/domain/resolveLayer.js";
import { IconX } from "./Icons.jsx";
import PiKeys from "./PiKeys.jsx";

const MODES = ["one-at-a-time", "all"];

export default function PiSettings({ hidden, agent: originalAgent, workspace, catalog, onAgentConfig, agentOnly = false, embedded = false, focus = "" }) {
  const [rep, setRep] = useState(null);
  const [pending, setPending] = useState(null);
  const agentSavingRef = useRef(false);
  const [configError, setConfigError] = useState("");
  const [configStatus, setConfigStatus] = useState("");
  const agent = originalAgent && { ...originalAgent, ...(pending || {}) };
  async function changeAgent(cfg) {
    if (agentSavingRef.current) return;
    agentSavingRef.current = true;
    setPending(cfg);
    setConfigError("");
    setConfigStatus("");
    try {
      await onAgentConfig(cfg);
      setConfigStatus("Saved.");
    } catch (error) { setConfigError(error.message || "Could not save agent settings."); }
    finally { agentSavingRef.current = false; setPending(null); }
  }

  const [loadError, setLoadError] = useState("");
  const [saveError, setSaveError] = useState("");
  const [saving, setSaving] = useState(false);
  const [retry, setRetry] = useState(0);
  const savingRef = useRef(false);
  useEffect(() => {
    if (hidden) return;
    let alive = true;
    setLoadError("");
    setRep(null);
    const q = agent ? "?agentId=" + encodeURIComponent(agent.id) : "";
    api("/api/pi-settings" + q).then(data => {
      if (alive) setRep(data);
    }).catch(error => { if (alive) setLoadError(error.message); });
    return () => { alive = false; };
  }, [hidden, agent && agent.id, retry]);

  useEffect(() => {
    if (hidden || !rep || focus !== "scoped-models") return;
    const frame = requestAnimationFrame(() => document.getElementById("scoped-models")?.scrollIntoView({ block: "center" }));
    return () => cancelAnimationFrame(frame);
  }, [hidden, !!rep, focus]);

  async function save(layer, patch, okMsg) {
    if (savingRef.current) return false;
    savingRef.current = true;
    setSaving(true);
    setSaveError("");
    const previous = rep;
    const key = layer === "project" ? "project" : "global";
    setRep(current => ({ ...current, [key]: { ...current[key], ...patch } }));
    try {
      const next = await api("/api/pi-settings", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ agentId: agent ? agent.id : "", layer, patch }),
      });
      setRep(next);
      toast.ok(okMsg);
      return true;
    } catch (error) {
      setRep(previous);
      setSaveError(error.message);
      return false;
    } finally { savingRef.current = false; setSaving(false); }
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
    <PageFrame id="pi-settings-view" title="Pi settings" embedded={embedded} context={!agentOnly && agent ? displayAgentName(agent, workspace) : ""} hidden={hidden} wide>
      {loadError ? <div className="cli-notice is-error" role="alert"><span>{loadError}</span><button type="button" className="btn btn-ghost btn-sm" onClick={() => setRetry(value => value + 1)}>Try again</button></div> : <>
      {saveError ? <div className="cli-notice is-error" role="alert"><span>{saveError}</span><button type="button" className="btn btn-ghost btn-sm" onClick={() => setSaveError("")}>Dismiss</button></div> : null}
      {saving ? <p role="status">Saving…</p> : null}
      <fieldset className="pi-settings-fields" disabled={saving || !!pending || !rep} aria-busy={saving || !!pending || !rep}>
      {!agentOnly ? <section className="settings-section" data-layer="global">
        <h3>Global</h3>
        <p className="settings-desc">This machine</p>
        <LayerKnobs
          prefix="g"
          values={g}
          catalog={catalog}
          onSave={(patch) => save("global", patch, "Saved for every pi on this machine.")}
        />
      </section> : null}
      {!agentOnly && workspace ? (
        <section className="settings-section" data-layer="workspace">
          <h3>Workspace</h3>
          <p className="settings-desc">{workspace.name}</p>
          {!canProject ? (
            <div className="cli-notice" role="status"><span>This folder is not trusted.</span><a className="btn btn-ghost btn-sm" href={"#/agent/" + encodeURIComponent(agent.id)}>Open agent to trust</a></div>
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
      {agent ? (
        <section className="settings-section" data-layer="agent">
          <h3>Agent</h3>
          <p className="settings-desc">{displayAgentName(agent, workspace)} · all sessions of this pi</p>
          <p className="settings-desc" role="status">{pending ? "Saving…" : configStatus}</p>
          {configError ? <div role="alert"><p>{configError}</p><button type="button" className="btn btn-sm" onClick={() => setConfigError("")}>Dismiss</button></div> : null}
          <fieldset className="m-agent-config-fields" disabled={!!pending} aria-busy={!!pending}>
            <div className="set-rows">
              <div className="set-row set-row-stack">
                <span>Model</span>
                <ConfigFields
                  catalog={catalog}
                  provider={ag.provider}
                  model={ag.model}
                  thinking={ag.thinking}
                  onChange={changeAgent}
                  idPrefix="ag-set"
                  row
                />
              </div>
              <div className="set-row">
                <span>Tools</span>
                <ModeChip cfg={{ opMode: agent.opMode || "full" }} onChange={changeAgent} />
              </div>
              <div className="set-row">
                <span>Checklist</span>
                <ChecklistChip level={agent.checklist || "changes"} readonly={(agent.opMode || "full") === "readonly"} onChange={changeAgent} />
              </div>
            </div>
          </fieldset>
        </section>
      ) : null}
      {!agentOnly ? <PiKeys /> : null}
      </fieldset>
      </>}
    </PageFrame>
  );
}

function LayerKnobs({ prefix, values, catalog, onSave }) {
  if (!values) {
    return (
      <div className="set-rows" aria-busy="true">
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
    <div className="set-rows">
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
        <div className="set-tools">
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
        noValidate
        onSubmit={async (e) => {
          e.preventDefault();
          const s = draft.trim();
          if (!s || list.includes(s)) return;
          if (await onSave([...list, s])) setDraft("");
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
