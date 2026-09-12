import { useEffect, useRef, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import PageFrame from "./PageFrame.jsx";
import ConfigFields from "./ConfigFields.jsx";
import ModeChip from "./ModeChip.jsx";
import ChecklistChip from "./ChecklistChip.jsx";
import { displayAgentName } from "@picode/shared/domain/tree.js";
import { api } from "@picode/shared/client/api.js";
import { toast } from "../lib/toast.js";
import { catalogBase, PI_TOOLS, resolveLayer } from "@picode/shared/domain/resolveLayer.js";
import { IconX } from "./Icons.jsx";
import PiKeys from "./PiKeys.jsx";

const MODES = ["one-at-a-time", "all"];

export default function PiSettings({ hidden, agent: originalAgent, workspace, catalog, onAgentConfig, embedded = false, focus = "", disabled = false, layer = "", tab = "settings", onLayerChange = () => {}, onTabChange = () => {} }) {
  const [rep, setRep] = useState(null);
  const [pending, setPending] = useState(null);
  const agentSavingRef = useRef(false);
  const [configError, setConfigError] = useState("");
  const [configStatus, setConfigStatus] = useState("");
  const agent = originalAgent && { ...originalAgent, ...(pending || {}) };
  async function changeAgent(cfg) {
    if (disabled || agentSavingRef.current) return;
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
  const [drafts, setDrafts] = useState({});
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

  async function save(layerID, patch, okMsg) {
    if (disabled || savingRef.current) return false;
    savingRef.current = true;
    setSaving(true);
    setSaveError("");
    const previous = rep;
    const key = layerID === "project" ? "project" : "global";
    // A reset is not merged optimistically: the keys that left the file — and
    // the has flags the rows read — can only come from the server's report.
    if (!patch.reset) setRep(current => ({ ...current, [key]: { ...current[key], ...patch } }));
    try {
      const next = await api("/api/pi-settings", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ agentId: agent ? agent.id : "", layer: layerID, patch }),
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
  // Unreadable defaults must not become inferred agent overrides on save.
  const parent = rep ? (p || g || floor) : {};
  const ag = agent ? {
    provider: agent.provider || parent.defaultProvider || "",
    model: agent.model || parent.defaultModel || "",
    thinking: agent.thinking || parent.defaultThinkingLevel || "",
  } : null;
  const canProject = !!(rep && rep.writable && rep.writable.project);
  const agSet = !!(agent && (agent.provider || agent.model || agent.thinking));

  // One layer at a time (docs/plans/cli-settings-ux.md). The route's layer
  // wins; a link that only names an agent opens that agent's layer, and the
  // Palette's scoped-models shortcut opens the machine layer, where that row
  // lives. A layer that cannot exist for this context is never offered.
  const layers = [
    { id: "global", label: "This machine", file: (rep && rep.global && rep.global.path) || "" },
    ...(workspace ? [{ id: "project", label: workspace.name || "This folder", file: (rep && rep.project && rep.project.path) || "" }] : []),
    ...(agent ? [{ id: "agent", label: displayAgentName(agent, workspace), file: "" }] : []),
  ];
  const wanted = focus === "scoped-models" ? "global" : layer;
  const active = layers.find((l) => l.id === wanted) || layers[layers.length - 1];
  const pane = tab === "keys" ? "keys" : "settings";
  const own = active.id === "global" ? rep && rep.global : active.id === "project" ? rep && rep.project : null;

  return (
    <PageFrame id="pi-settings-view" title="Pi settings" embedded={embedded} hidden={hidden}>
      {loadError ? <div className="pi-settings-notice" role="alert"><span title={loadError}>Could not load Pi defaults.</span><button type="button" className="btn btn-ghost btn-sm" onClick={() => setRetry(value => value + 1)}>Try again</button></div> : null}
      {saveError ? <div className="pi-settings-notice" role="alert"><span>{saveError}</span><button type="button" className="btn btn-ghost btn-sm" onClick={() => setSaveError("")}>Dismiss</button></div> : null}
      {saving ? <p role="status">Saving…</p> : null}
      <div className="pkg-tabs" role="tablist" aria-label="Pi settings sections">
        <button type="button" role="tab" className="pkg-tab" aria-selected={pane === "settings"} onClick={() => onTabChange("settings")}>Settings</button>
        <button type="button" role="tab" className="pkg-tab" aria-selected={pane === "keys"} onClick={() => onTabChange("keys")}>Keys</button>
      </div>
      {pane === "keys" ? (
        <PiKeys disabled={disabled || saving || !!pending} />
      ) : (
        <>
          <div className="settings-layer">
            <span className="settings-layer-label">Edit</span>
            <div className="pkg-scope" role="radiogroup" aria-label="Which layer to edit">
              {layers.map((l) => (
                <button
                  key={l.id}
                  type="button"
                  role="radio"
                  className="pkg-scope-btn"
                  aria-checked={active.id === l.id}
                  disabled={saving || !!pending}
                  onClick={() => onLayerChange(l.id)}
                >{l.label}</button>
              ))}
            </div>
          </div>
          {active.file ? <p className="settings-file" title={active.file}>{active.file}</p> : null}
          <fieldset className="pi-settings-fields" disabled={disabled || saving || !!pending} aria-busy={saving || !!pending}>
          {/* The agent layer is PiCode's own fields, not the native file: a
              malformed settings.json must not lock it (ADR-0101 recovery). */}
          {active.id === "agent" ? (
          <section className="settings-section" data-layer="agent">
            <>
                <p className="settings-desc">{displayAgentName(agent, workspace)} · all sessions of this pi</p>
                <p className="settings-desc" role="status">{pending ? "Saving…" : configStatus}</p>
                {configError ? <div role="alert"><p>{configError}</p><button type="button" className="btn btn-sm" onClick={() => setConfigError("")}>Dismiss</button></div> : null}
                <div className="set-rows">
                  <div className={"set-row set-row-stack" + (agSet ? " is-set" : "")}>
                    <span className="set-label">Model<span className="set-src">{agSet ? "Set here" : "From " + inheritedLabel(workspace)}</span></span>
                    <div className="set-ctl">
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
                  </div>
                  <div className="set-row">
                    <span className="set-label">Tools</span>
                    <div className="set-ctl"><ModeChip cfg={{ opMode: agent.opMode || "full" }} onChange={changeAgent} /></div>
                  </div>
                  <div className="set-row">
                    <span className="set-label">Checklist</span>
                    <div className="set-ctl"><ChecklistChip level={agent.checklist || "changes"} readonly={(agent.opMode || "full") === "readonly"} onChange={changeAgent} /></div>
                  </div>
                </div>
              </>
          </section>
          ) : (
          <fieldset className="pi-settings-fields" disabled={!rep || !!loadError} hidden={!rep && !!loadError}>
          <section className="settings-section" data-layer={active.id}>
            {!canProject && active.id === "project" ? (
              <div className="cli-notice" role="status"><span>This folder is not trusted.</span><a className="btn btn-ghost btn-sm" href={"#/agent/" + encodeURIComponent(agent.id)}>Open agent to trust</a></div>
            ) : (
              <LayerKnobs
                key={active.id}
                prefix={active.id === "global" ? "g" : "w"}
                draft={drafts[active.id] || ""}
                onDraft={(value) => setDrafts((current) => ({ ...current, [active.id]: value }))}
                values={active.id === "global" ? g : p}
                own={own}
                parentLabel={active.id === "project" ? "From This machine" : "Pi default"}
                catalog={catalog}
                saving={saving}
                onSave={(patch) => save(active.id, patch, active.id === "global" ? "Saved for every pi on this machine." : "Saved for this folder.")}
                onReset={(keys) => save(active.id, { reset: keys }, "Back to inherited.")}
              />
            )}
          </section>
          </fieldset>
          )}
          </fieldset>
        </>
      )}
    </PageFrame>
  );
}

function inheritedLabel(workspace) {
  return workspace ? "this folder" : "This machine";
}

function LayerKnobs({ prefix, values, own, parentLabel, catalog, saving, onSave, onReset, draft, onDraft }) {
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
  const isSet = (keys) => keys.some((k) => !!(own && own.has && own.has[k]));
  // "Use inherited" clears exactly the keys this layer set, so a row that
  // overrides one of its three fields hands back only that one.
  const resetKeys = (keys) => keys.filter((k) => !!(own && own.has && own.has[k]));
  const reset = (keys) => <button type="button" className="btn btn-ghost btn-sm" disabled={saving} onClick={() => onReset(resetKeys(keys))}>Use inherited</button>;
  const source = (keys) => <span className="set-src">{isSet(keys) ? "Set here" : parentLabel}</span>;
  const rowClass = (keys) => "set-row" + (isSet(keys) ? " is-set" : "");
  return (
    <div className="set-rows">
      <div className={rowClass(["compactionEnabled"])}>
        <label className="set-label" htmlFor={prefix + "-compact"}>Auto-compact{source(["compactionEnabled"])}</label>
        <div className="set-ctl">
          <Switch.Root
            id={prefix + "-compact"}
            className="rx-switch"
            checked={!!values.compactionEnabled}
            onCheckedChange={(v) => onSave({ compactionEnabled: v })}
          >
            <Switch.Thumb className="rx-switch-thumb" />
          </Switch.Root>
          {isSet(["compactionEnabled"]) ? reset(["compactionEnabled"]) : null}
        </div>
      </div>
      <div className={rowClass(["steeringMode"])}>
        <label className="set-label" htmlFor={prefix + "-steer"}>Steering{source(["steeringMode"])}</label>
        <div className="set-ctl">
          <select id={prefix + "-steer"} value={values.steeringMode || "one-at-a-time"} onChange={(e) => onSave({ steeringMode: e.target.value })}>
            {MODES.map((m) => <option key={m} value={m}>{m}</option>)}
          </select>
          {isSet(["steeringMode"]) ? reset(["steeringMode"]) : null}
        </div>
      </div>
      <div className={rowClass(["followUpMode"])}>
        <label className="set-label" htmlFor={prefix + "-follow"}>Follow-up{source(["followUpMode"])}</label>
        <div className="set-ctl">
          <select id={prefix + "-follow"} value={values.followUpMode || "one-at-a-time"} onChange={(e) => onSave({ followUpMode: e.target.value })}>
            {MODES.map((m) => <option key={m} value={m}>{m}</option>)}
          </select>
          {isSet(["followUpMode"]) ? reset(["followUpMode"]) : null}
        </div>
      </div>
      <div className={rowClass(["defaultProvider", "defaultModel", "defaultThinkingLevel"]) + " set-row-stack"}>
        <span className="set-label">Defaults{source(["defaultProvider", "defaultModel", "defaultThinkingLevel"])}</span>
        <div className="set-ctl">
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
          {isSet(["defaultProvider", "defaultModel", "defaultThinkingLevel"]) ? reset(["defaultProvider", "defaultModel", "defaultThinkingLevel"]) : null}
        </div>
      </div>
      <div className={rowClass(["enabledModels"]) + " set-row-stack"} id={prefix === "g" ? "scoped-models" : undefined}>
        <span className="set-label">Scoped models{source(["enabledModels"])}</span>
        <div className="set-ctl">
          <PatternField list={values.enabledModels || []} draft={draft} onDraft={onDraft} onSave={(enabledModels) => onSave({ enabledModels })} />
          {isSet(["enabledModels"]) ? reset(["enabledModels"]) : null}
        </div>
      </div>
      <div className={rowClass(["defaultTools"]) + " set-row-stack"}>
        <span className="set-label">Tools{source(["defaultTools"])}</span>
        <div className="set-ctl set-ctl-stack">
          <div className="set-tools" data-align-row data-align-wrap>
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
          {isSet(["defaultTools"]) ? reset(["defaultTools"]) : null}
        </div>
      </div>
    </div>
  );
}

function PatternField({ list, draft, onDraft, onSave }) {
  const setDraft = onDraft;
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
