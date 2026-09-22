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
import { piRowsFor, piRowState, rowKey, rowKeys } from "@picode/shared/domain/piRows.js";
import { IconX } from "./Icons.jsx";
import PiKeys from "./PiKeys.jsx";


export default function PiSettings({ hidden, agent: originalAgent, workspace, catalog, onAgentConfig, agentOnly = false, embedded = false, focus = "", disabled = false, layer = "", keys = false, onLayerChange = () => {} }) {
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
    { id: "global", label: "Global", file: (rep && rep.global && rep.global.path) || "" },
    ...(workspace ? [{ id: "project", label: workspace.name || "This folder", file: (rep && rep.project && rep.project.path) || "" }] : []),
    ...(agent ? [{ id: "agent", label: displayAgentName(agent, workspace), file: "" }] : []),
  ];
  const wanted = focus === "scoped-models" ? "global" : layer;
  // The quick sheet (agentOnly) is one agent, one layer: no switcher, and the
  // keyboard map is the pane next door, not a sub-tab (docs/plans/cli-settings-ux.md).
  const active = agentOnly ? layers[layers.length - 1] : (layers.find((l) => l.id === wanted) || layers[layers.length - 1]);
  const chrome = !agentOnly && !keys;
  const own = active.id === "global" ? rep && rep.global : active.id === "project" ? rep && rep.project : null;

  return (
    <PageFrame id="pi-settings-view" title="Pi settings" embedded={embedded} wide hidden={hidden}>
      {loadError ? <div className="pi-settings-notice" role="alert"><span title={loadError}>Could not load Pi defaults.</span><button type="button" className="btn btn-ghost btn-sm" onClick={() => setRetry(value => value + 1)}>Try again</button></div> : null}
      {saveError ? <div className="pi-settings-notice" role="alert"><span>{saveError}</span><button type="button" className="btn btn-ghost btn-sm" onClick={() => setSaveError("")}>Dismiss</button></div> : null}
      {saving ? <p role="status">Saving…</p> : null}
      {keys ? (
        <PiKeys disabled={disabled || saving || !!pending} />
      ) : (
        <>
          {chrome ? <div className="settings-layer">
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
          </div> : null}
          {chrome && active.file ? <p className="settings-file" title={active.file}>{active.file}</p> : null}
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
          <div className="settings-layer-body" data-layer={active.id}>
            {!canProject && active.id === "project" ? (
              <div className="cli-notice" role="status"><span>This folder is not trusted.</span><a className="btn btn-ghost btn-sm" href={"#/agent/" + encodeURIComponent(agent.id)}>Open agent to trust</a></div>
            ) : (
              <LayerKnobs
                key={active.id}
                // pi keeps these five in its machine file and nowhere else, so
                // the rows appear on that layer only rather than writing a key
                // pi would never read there.
                machine={active.id === "global"}
                prefix={active.id === "global" ? "g" : "w"}
                draft={drafts[active.id] || ""}
                onDraft={(value) => setDrafts((current) => ({ ...current, [active.id]: value }))}
                values={active.id === "global" ? g : p}
                own={own}
                parentLabel={active.id === "project" ? "From Global" : "Pi default"}
                catalog={catalog}
                saving={saving}
                onSave={(patch) => save(active.id, patch, active.id === "global" ? "Saved for every pi on this machine." : "Saved for this folder.")}
                onReset={(keys) => save(active.id, { reset: keys }, "Back to inherited.")}
              />
            )}
          </div>
          </fieldset>
          )}
          </fieldset>
        </>
      )}
    </PageFrame>
  );
}

function inheritedLabel(workspace) {
  return workspace ? "this folder" : "Global";
}

function LayerKnobs({ prefix, values, own, parentLabel, catalog, saving, onSave, onReset, draft, onDraft, machine }) {
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
  // Every row comes from the table (web/shared/domain/piRows.js). Adding a
  // setting pi gained is a line there, the way it already is for the eight
  // guest CLIs — this pane used to be hand-written JSX, which is why pi
  // persisted forty keys and showed eight.
  const groups = piRowsFor(machine ? "global" : "project");
  return (
    <div className="set-rows">
      {groups.map((group) => (
        <section className="settings-section" key={group.name}>
          <h3>{group.name}</h3>
          {group.rows.map((row) => (
            <PiRow
              key={rowKeys(row).join("+")}
              row={row}
              state={piRowState(row, values, own, parentLabel)}
              values={values}
              prefix={prefix}
              catalog={catalog}
              saving={saving}
              draft={draft}
              onDraft={onDraft}
              onSave={onSave}
              onReset={onReset}
            />
          ))}
        </section>
      ))}
    </div>
  );
}

// One row, whatever its kind. The label, the provenance and "Use inherited"
// are the same for all of them; only the control differs, and the three that
// are not scalars (model defaults, scoped models, tools) keep the controls
// they always had rather than being flattened into something they are not.
function PiRow({ row, state, values, prefix, catalog, saving, draft, onDraft, onSave, onReset }) {
  const { setHere, source, resetKeys, value } = state;
  const key = rowKey(row);
  const id = prefix + "-" + key;
  const reset = setHere ? (
    <button type="button" className="btn btn-ghost btn-sm" disabled={saving} onClick={() => onReset(resetKeys)}>Use inherited</button>
  ) : null;
  let control = null;
  switch (row.kind) {
    case "bool":
      control = (
        <Switch.Root id={id} className="rx-switch" checked={value === true || value === false ? value : !!row.defaultOn} disabled={saving} onCheckedChange={(v) => onSave({ [key]: v })} aria-label={row.label}>
          <Switch.Thumb className="rx-switch-thumb" />
        </Switch.Root>
      );
      break;
    case "select":
      control = (
        <select id={id} value={value === undefined || value === null || value === "" ? (row.fallback || "") : String(value)} disabled={saving} onChange={(e) => onSave({ [key]: e.target.value })}>
          {row.options.map((o) => <option key={o} value={o}>{(row.optionLabels && row.optionLabels[o]) || o}</option>)}
        </select>
      );
      break;
    case "text":
      control = <PiTextField id={id} value={value} placeholder={row.fallback} saving={saving} onSave={(v) => onSave({ [key]: v })} />;
      break;
    case "model":
      control = (
        <ConfigFields
          catalog={catalog}
          provider={values.defaultProvider || ""}
          model={values.defaultModel || ""}
          thinking={values.defaultThinkingLevel || ""}
          onChange={(cfg) => onSave({ defaultProvider: cfg.provider, defaultModel: cfg.model, defaultThinkingLevel: cfg.thinking })}
          idPrefix={prefix + "-def"}
          row
        />
      );
      break;
    case "patterns":
      control = <PatternField list={values.enabledModels || []} draft={draft} onDraft={onDraft} onSave={(enabledModels) => onSave({ enabledModels })} />;
      break;
    case "tools":
      control = (
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
      );
      break;
    default:
      control = null;
  }
  return (
    <div className={"set-row" + (setHere ? " is-set" : "") + (row.stack ? " set-row-stack" : "")} id={row.anchor && prefix === "g" ? row.anchor : undefined}>
      <label className="set-label" htmlFor={row.kind === "bool" || row.kind === "select" || row.kind === "text" ? id : undefined}>
        {row.label}
        <span className="set-src">{source}</span>
        {row.help ? <span className="set-src">{row.help}</span> : null}
      </label>
      <div className={"set-ctl" + (row.kind === "tools" ? " set-ctl-stack" : "")}>
        {control}
        {reset}
      </div>
    </div>
  );
}

// A text row commits on blur, so a half-typed path is never saved.
function PiTextField({ id, value, placeholder, saving, onSave }) {
  const [text, setText] = useState(value === undefined || value === null ? "" : String(value));
  useEffect(() => { setText(value === undefined || value === null ? "" : String(value)); }, [value]);
  return (
    <input
      id={id}
      className="set-text"
      value={text}
      placeholder={placeholder}
      disabled={saving}
      onChange={(e) => setText(e.target.value)}
      onBlur={() => { const next = text.trim(); if (next !== String(value ?? "")) onSave(next); }}
      onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); e.currentTarget.blur(); } }}
    />
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
