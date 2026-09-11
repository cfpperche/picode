import { useEffect, useMemo, useRef, useState } from "react";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { askConfirm } from "../lib/confirm.js";
import { paneContext } from "@picode/shared/domain/tree.js";
import { fileHash } from "../lib/routes.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { rolesConfigSchema, ROLES_RESERVED } from "@picode/shared/contracts/schemas.js";
import PageFrame from "./PageFrame.jsx";
import PiSpinner from "./PiSpinner.jsx";

// pi-roles configuration (ADR-0028/0033; GUI added by ADR-0033 amendment
// #3): the page edits the same workspace file and per-agent overlay the
// extension reads — never a copy. There is deliberately no machine layer
// (pi-roles has none), "Save" writes the file, and the page says when the
// change takes effect instead of claiming a running agent picked it up.

const BUILTIN = ["default", "vision", "plan"];
const THINKING_NONE = "none"; // placeholder: leave the thinking level alone

// One line of state per builtin role (what triggers it), so a row reads
// without the README. Custom presets carry no descriptor — the name is it.
const ROLE_DESC = {
  default: "Text-only messages in auto mode",
  vision: "Messages with images or image paths",
  plan: "/plan — model plus plan-mode instructions",
};

const EMPTY = { builtin: { default: null, vision: null, plan: null }, custom: [] };

function layerToDraft(layer) {
  if (!layer || layer.invalid || !layer.exists) return structuredClone(EMPTY);
  const builtin = {};
  for (const name of BUILTIN) {
    const a = layer.config.builtin && layer.config.builtin[name];
    builtin[name] = a ? { model: a.model, thinking: a.thinking || "" } : null;
  }
  return {
    builtin,
    custom: (layer.config.custom || []).map((c) => ({ name: c.name, model: c.model, thinking: c.thinking || "" })),
  };
}

function draftEquals(a, b) {
  return JSON.stringify(a) === JSON.stringify(b);
}

function draftToConfig(draft) {
  const builtin = {};
  for (const name of BUILTIN) {
    const a = draft.builtin[name];
    if (a && a.model) builtin[name] = a.thinking ? { model: a.model, thinking: a.thinking } : { model: a.model };
  }
  const custom = draft.custom
    .filter((c) => c.model)
    .map((c) => (c.thinking ? { name: c.name, model: c.model, thinking: c.thinking } : { name: c.name, model: c.model }));
  return { builtin, custom };
}

export default function PackagesConfig({ hidden, embedded = false, pkg, workspaceId, workspaceName, agentId, agentName, catalog, backHash = "#/clis/pi/packages", initialScope = "workspace", onScopeChange = () => {}, beforeMutation = async () => {} }) {
  const [view, setView] = useState(null);
  const [loadErr, setLoadErr] = useState("");
  const scope = initialScope;
  const setScope = onScopeChange;
  const [drafts, setDrafts] = useState({ workspace: null, agent: null });
  const [saving, setSaving] = useState(false);
  const [conflict, setConflict] = useState("");
  const [note, setNote] = useState("");
  const [noteError, setNoteError] = useState(false);
  const [adding, setAdding] = useState(false);
  const current = useRef(null);
  current.current = { view, drafts, saving };
  const request = useRef(0);
  useEffect(() => () => { request.current++; }, []);
  useEffect(() => { setConflict(""); setNote(""); }, [scope]);

  const hasAgentLayer = !!agentId;
  const layer = scope === "agent" && hasAgentLayer ? view?.agent : view?.workspace;
  const other = scope === "agent" && hasAgentLayer ? view?.workspace : null;
  const draft = drafts[scope] || structuredClone(EMPTY);
  const broken = !!(layer && layer.invalid);
  const readOnly = !view || broken || !!loadErr;
  const dirty = !!view && !broken && !draftEquals(draft, layerToDraft(layer));

  const providers = useMemo(
    () => ((catalog && catalog.providers) || []).filter((p) => (p.models || []).length > 0),
    [catalog],
  );
  const thinkingLevels = useMemo(
    () => (catalog && Array.isArray(catalog.thinking) && catalog.thinking.length ? catalog.thinking : ["off", "minimal", "low", "medium", "high", "xhigh", "max"]),
    [catalog],
  );

  function configURL() {
    const q = ["package=pi-roles"];
    if (workspaceId) q.push("workspace=" + encodeURIComponent(workspaceId));
    if (agentId) q.push("agent=" + encodeURIComponent(agentId));
    return "/api/packages/config?" + q.join("&");
  }

  async function load() {
    if (!workspaceId) { setView(null); return; }
    if (current.current.saving) return;
    const seq = ++request.current;
    try {
      const next = await api(configURL());
      if (seq !== request.current) return;
      const previous = current.current;
      setView(next);
      setLoadErr("");
      setDrafts({
        workspace: previous.view && previous.drafts.workspace && !draftEquals(previous.drafts.workspace, layerToDraft(previous.view.workspace)) ? previous.drafts.workspace : layerToDraft(next.workspace),
        agent: previous.view?.agent && previous.drafts.agent && !draftEquals(previous.drafts.agent, layerToDraft(previous.view.agent)) ? previous.drafts.agent : next.agent ? layerToDraft(next.agent) : null,
      });
    } catch (err) {
      if (seq === request.current) setLoadErr(humanizeError(err.message || String(err)));
    }
  }

  useEffect(() => { if (!hidden) load(); }, [hidden, workspaceId, agentId]); // eslint-disable-line
  useEffect(() => {
    if (hidden) return;
    return subscribeFeed((ev) => {
      if (ev.type === "feed.open" || ev.type === "feed.reset") { load(); return; }
      if (ev.type === "packages.config") load();
    });
  }, [hidden, workspaceId, agentId]); // eslint-disable-line

  function setDraft(next) {
    setDrafts((d) => ({ ...d, [scope]: next }));
    setNote("");
  }

  function setSlot(name, patch) {
    if (patch.__reset) {
      setDraft({ ...draft, builtin: { ...draft.builtin, [name]: null } });
      return;
    }
    if (patch.__override) {
      setDraft({ ...draft, builtin: { ...draft.builtin, [name]: { model: patch.model, thinking: patch.thinking || "" } } });
      return;
    }
    const cur = draft.builtin[name] || {};
    setDraft({ ...draft, builtin: { ...draft.builtin, [name]: { ...cur, ...patch } } });
  }

  function overrideCustom(inherited) {
    if (draft.custom.some((c) => c.name === inherited.name)) return;
    setDraft({ ...draft, custom: [...draft.custom, { name: inherited.name, model: inherited.model, thinking: inherited.thinking || "" }] });
  }

  function setCustom(idx, patch) {
    if (patch.__reset) {
      setDraft({ ...draft, custom: draft.custom.filter((_, i) => i !== idx) });
      return;
    }
    setDraft({ ...draft, custom: draft.custom.map((c, i) => (i === idx ? { ...c, ...patch } : c)) });
  }

  function removeCustom(name) {
    setDraft({ ...draft, custom: draft.custom.filter((c) => c.name !== name) });
  }

  function addCustom(row) {
    setDraft({ ...draft, custom: [...draft.custom, row] });
    setAdding(false);
  }

  async function save(force) {
    if (saving || !view || loadErr || (broken && !force)) return;
    const parsed = rolesConfigSchema.safeParse(draftToConfig(draft));
    if (!parsed.success) {
      setNoteError(true);
      setNote(parsed.error.issues[0]?.message || "Check the highlighted fields.");
      return;
    }
    request.current++;
    setSaving(true);
    setConflict("");
    setNote("");
    setNoteError(false);
    try {
      await beforeMutation();
      const next = await api("/api/packages/config", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          package: "pi-roles", scope, workspaceId,
          agentId: agentId || undefined,
          config: parsed.data, force: !!force,
        }),
      });
      setView(next);
      setDrafts(prev => ({ workspace: scope === "workspace" ? layerToDraft(next.workspace) : prev.workspace, agent: scope === "agent" ? (next.agent ? layerToDraft(next.agent) : null) : prev.agent }));
      const rel = scope === "agent" ? next.agent?.rel : next.workspace.rel;
      setNote("Saved to " + rel + ". Roles apply on the agent's next message.");
    } catch (err) {
      if (err.status === 409) setConflict(err.message);
      else { setNoteError(true); setNote(humanizeError(err.message || String(err))); }
    } finally {
      setSaving(false);
    }
  }

  async function clearFile() {
    if (saving || readOnly || !layer?.exists) return;
    const target = scope === "agent" ? view.agent : view.workspace;
    const ok = await askConfirm({
      title: "Clear roles file",
      message: "Delete " + target.rel + "? " + (scope === "agent"
        ? "This agent falls back to the shared workspace roles."
        : "Agents that keep their own overlay are unaffected."),
      confirmLabel: "Delete file",
      danger: true,
    });
    if (!ok) return;
    request.current++;
    setSaving(true);
    setNote("");
    setNoteError(false);
    try {
      await beforeMutation();
      const next = await api(
        "/api/packages/config?package=pi-roles&scope=" + scope + "&workspace=" + encodeURIComponent(workspaceId)
        + (agentId ? "&agent=" + encodeURIComponent(agentId) : ""),
        { method: "DELETE" },
      );
      setView(next);
      setDrafts(prev => ({ workspace: scope === "workspace" ? layerToDraft(next.workspace) : prev.workspace, agent: scope === "agent" ? (next.agent ? layerToDraft(next.agent) : null) : prev.agent }));
      setNote("Cleared " + target.rel + ".");
    } catch (err) {
      setNoteError(true);
      setNote(humanizeError(err.message || String(err)));
    } finally {
      setSaving(false);
    }
  }

  if (hidden) return null;

  if (!workspaceId) {
    return (
      <PageFrame embedded={embedded} id="packages-config" title={pkg + " settings"} context={paneContext(agentName, workspaceName)} hidden={hidden}>
        <div className="pkg-empty">
          <p className="pkg-empty-title">Roles files live in a workspace folder.</p>
          <a className="btn btn-sm" href="#/">Open a workspace</a>
        </div>
      </PageFrame>
    );
  }

  const inheritedCustoms = scope === "agent"
    ? (other?.config?.custom || []).filter((o) => !draft.custom.some((c) => c.name === o.name))
    : [];

  return (
    <PageFrame embedded={embedded} id="packages-config" title={pkg + " settings"} context={paneContext(agentName, workspaceName)} hidden={hidden}>
      <div className="pkc-top">
        <a className="pkg-back" href={backHash}>← All packages</a>
        <span className="pkg-foot-spacer" />
        <div className="pkg-scope" role="radiogroup" aria-label="Which roles file to edit">
          <button type="button" role="radio" className="pkg-scope-btn" disabled={saving} aria-checked={scope === "workspace"} onClick={() => setScope("workspace")}>Workspace — shared</button>
          {hasAgentLayer ? (
            <button type="button" role="radio" className="pkg-scope-btn" disabled={saving} aria-checked={scope === "agent"} onClick={() => setScope("agent")}>{agentName || "This agent"} — overrides</button>
          ) : null}
        </div>
      </div>

      {loadErr ? (
        <p className="pkg-notice warn" role="alert">{loadErr} <button type="button" className="btn btn-ghost btn-sm" onClick={load}>Retry</button></p>
      ) : null}
      {!view && !loadErr ? (
        <p className="pkg-loading" role="status"><PiSpinner title="Loading roles" /> Loading roles…</p>
      ) : null}

      {view ? (
        <>
          {broken ? (
            <div className="pkg-notice err" role="alert">
              <p>{layer.invalid}</p>
              <p className="pkg-fine">The file was left untouched. Fix it in the editor, or replace it with this form.</p>
              <div className="pkg-notice-actions">
                {scope === "workspace" && workspaceId ? <a className="btn btn-sm" href={fileHash("w", workspaceId, layer.rel)}>Open file</a> : null}
                <button type="button" className="btn btn-danger btn-sm" onClick={() => save(true)}>Replace file…</button>
              </div>
            </div>
          ) : null}
          {conflict ? (
            <div className="pkg-notice err" role="alert">
              <p>{conflict}</p>
              <div className="pkg-notice-actions">
                <button type="button" className="btn btn-sm" onClick={() => setConflict("")}>Go back</button>
                <button type="button" className="btn btn-danger btn-sm" onClick={() => save(true)}>Replace anyway</button>
              </div>
            </div>
          ) : null}
          {layer && !layer.exists ? (
            <p className="pkg-notice">No roles file in this layer yet — saving creates <code>{layer.rel}</code>. Until then this layer adds nothing.</p>
          ) : null}

          <div className="pkc-section">
            <h3>Built-in roles</h3>
            {BUILTIN.map((name) => {
              const inheritedBuiltin = scope === "agent" ? (other?.config?.builtin?.[name] || null) : null;
              return (
                <RoleRow
                  key={name}
                  name={name}
                  desc={ROLE_DESC[name]}
                  own={draft.builtin[name]}
                  inherited={inheritedBuiltin}
                  agentScope={scope === "agent"}
                  providers={providers}
                  thinkingLevels={thinkingLevels}
                  disabled={readOnly || saving}
                  onChange={(patch) => setSlot(name, patch)}
                  onOverride={inheritedBuiltin ? () => setSlot(name, { __override: true, model: inheritedBuiltin.model, thinking: inheritedBuiltin.thinking || "" }) : undefined}
                />
              );
            })}
          </div>

          <div className="pkc-section">
            <h3>Custom presets</h3>
            {draft.custom.map((c, i) => (
              <RoleRow
                key={c.name}
                name={c.name}
                custom
                own={c}
                inherited={scope === "agent" ? ((other?.config?.custom || []).find((o) => o.name === c.name) || null) : null}
                agentScope={scope === "agent"}
                providers={providers}
                thinkingLevels={thinkingLevels}
                disabled={readOnly || saving}
                onChange={(patch) => setCustom(i, patch)}
                onRemove={() => removeCustom(c.name)}
              />
            ))}
            {scope === "agent" ? inheritedCustoms.map((o) => (
              <RoleRow key={"inh:" + o.name} name={o.name} own={null} inherited={o} agentScope readOnly={false} providers={providers} thinkingLevels={thinkingLevels} disabled={readOnly || saving} onChange={() => {}} onOverride={() => overrideCustom(o)} />
            )) : null}
            {draft.custom.length === 0 && inheritedCustoms.length === 0 && !adding ? (
              <p className="pkg-fine">No presets in this layer.</p>
            ) : null}
            {adding ? (
              <AddPreset
                providers={providers}
                thinkingLevels={thinkingLevels}
                existing={draft.custom.map((c) => c.name)}
                disabled={readOnly || saving}
                onAdd={addCustom}
                onCancel={() => setAdding(false)}
              />
            ) : (
              <button type="button" className="btn btn-sm" disabled={readOnly || saving} onClick={() => setAdding(true)}>Add preset</button>
            )}
          </div>

          <div className={(dirty ? "pkc-foot dirty" : "pkc-foot")}>
            <span className={dirty ? "pkg-fine pkc-file dirty" : "pkg-fine pkc-file"} title={layer?.path || ""}>
              {layer && layer.exists ? layer.rel : "No file yet"}
              {scope === "agent" ? " · unset slots inherit the workspace file" : ""}
            </span>
            <div className="pkc-actions" data-align-row data-align-wrap>
            <button type="button" className="btn btn-ghost btn-sm" disabled={!dirty || saving} onClick={() => setDraft(layerToDraft(layer))}>Discard</button>
            <button type="button" className="btn btn-ghost btn-sm" disabled={readOnly || saving || !layer?.exists} onClick={clearFile}>Clear file…</button>
            <button type="button" className="btn btn-primary btn-sm" disabled={readOnly || !dirty || saving} onClick={() => save(false)}>{saving ? "Saving…" : "Save"}</button>
            </div>
          </div>
          {dirty ? <p className="pkg-fine pkc-dirty">Unsaved changes in this layer.</p> : null}
          {note ? <p className={"pkg-notice " + (noteError ? "err" : "ok")} role={noteError ? "alert" : "status"}>{note}</p> : null}
        </>
      ) : null}
    </PageFrame>
  );
}

function ModelSelect({ id, model, inheritedModel, providers, disabled, onChange }) {
  const provider = model ? model.slice(0, model.indexOf("/")) : "";
  return (
    <select
      className="pkc-select model"
      aria-label={id + " model"}
      disabled={disabled}
      value={model}
      onChange={(e) => onChange({ model: e.target.value, ...(e.target.value ? {} : { thinking: "" }) })}
    >
      <option value="">{inheritedModel ? "Inherits " + inheritedModel : "Not set"}</option>
      {providers.map((p) => (
        <optgroup key={p.id} label={p.id}>
          {(p.models || []).map((m) => (
            <option key={p.id + "/" + m.id} value={p.id + "/" + m.id}>{m.id}</option>
          ))}
        </optgroup>
      ))}
      {model && !providers.some((p) => p.id === provider) ? <option value={model}>{model}</option> : null}
    </select>
  );
}

function ThinkingSelect({ id, value, inheritedValue, levels, disabled, onChange }) {
  return (
    <select
      className="pkc-select thinking"
      aria-label={id + " thinking level"}
      disabled={disabled}
      value={value || THINKING_NONE}
      onChange={(e) => onChange({ thinking: e.target.value === THINKING_NONE ? "" : e.target.value })}
    >
      <option value={THINKING_NONE}>{value ? "" : inheritedValue ? "Inherits " + inheritedValue : "Thinking: unchanged"}</option>
      {levels.map((t) => <option key={t} value={t}>{t}</option>)}
    </select>
  );
}

function RoleRow({ name, desc, custom, own, inherited, agentScope, providers, thinkingLevels, disabled, onChange, onRemove, onOverride }) {
  const isOverride = agentScope && !!own;
  const isInherited = agentScope && !own && !!inherited;
  const effective = own || inherited;
  return (
    <div className={"pkc-row" + (isInherited ? " inherited" : "")}>
      <div className="pkc-row-head">
        <div className="pkc-role-id">
          <span className="pkc-role-name">{name}</span>
          {desc ? <span className="pkc-role-desc">{desc}</span> : null}
        </div>
        {isInherited ? <span className="pkg-type">inherited</span> : null}
        {isOverride ? <span className="pkg-type">this agent</span> : null}
        <span className="pkg-foot-spacer" />
        {isOverride && inherited ? (
          <button type="button" className="btn btn-ghost btn-sm" disabled={disabled} onClick={() => onChange({ __reset: true })}>Use workspace value</button>
        ) : null}
        {isOverride && !inherited && onRemove ? (
          <button type="button" className="btn btn-ghost btn-sm" disabled={disabled} onClick={onRemove}>Remove</button>
        ) : null}
        {!agentScope && !custom && own ? (
          <button type="button" className="btn btn-ghost btn-sm" disabled={disabled} onClick={() => onChange({ __reset: true })}>Clear</button>
        ) : null}
        {!agentScope && custom && onRemove ? (
          <button type="button" className="btn btn-ghost btn-sm" disabled={disabled} onClick={onRemove}>Remove</button>
        ) : null}
      </div>
      {isInherited ? (
        <>
          <p className="pkg-fine">
            {effective.model ? "Workspace routes " + name + " to " + effective.model + (effective.thinking ? " · " + effective.thinking : "") + "." : "Workspace sets no value for " + name + "."}
          </p>
          {onOverride ? (
            <div className="pkg-notice-actions">
              <button type="button" className="btn btn-sm" disabled={disabled} onClick={onOverride}>Override for this agent</button>
            </div>
          ) : null}
        </>
      ) : (
        <div className="pkc-row-fields" data-align-row data-align-wrap>
          <ModelSelect
            id={name}
            model={own?.model || ""}
            inheritedModel={isInherited || (!agentScope && !custom) ? effective?.model || "" : ""}
            providers={providers}
            disabled={disabled}
            onChange={onChange}
          />
          <ThinkingSelect
            id={name}
            value={own?.thinking || ""}
            inheritedValue={effective?.thinking || ""}
            levels={thinkingLevels}
            disabled={disabled || !(own && own.model)}
            onChange={onChange}
          />
        </div>
      )}
    </div>
  );
}

function AddPreset({ providers, thinkingLevels, existing, disabled, onAdd, onCancel }) {
  const [name, setName] = useState("");
  const [model, setModel] = useState("");
  const [thinking, setThinking] = useState("");
  const reserved = ROLES_RESERVED.includes(name);
  const badShape = !!name && !/^[a-zA-Z][a-zA-Z0-9_-]*$/.test(name);
  const duplicate = existing.includes(name);
  const nameBad = !!name && (badShape || reserved || duplicate);
  return (
    <div className="pkc-row add">
      <div className="pkc-row-fields" data-align-row data-align-wrap>
        <input
          className="pkc-input"
          value={name}
          placeholder="preset name"
          aria-label="Preset name"
          disabled={disabled}
          onChange={(e) => setName(e.target.value)}
        />
        <ModelSelect id="new preset" model={model} providers={providers} disabled={disabled} onChange={(p) => { setModel(p.model); setThinking(p.thinking || ""); }} />
        <ThinkingSelect id="new preset" value={thinking} levels={thinkingLevels} disabled={disabled || !model} onChange={(p) => setThinking(p.thinking || "")} />
      </div>
      {nameBad ? (
        <p className="pkg-fine">{duplicate ? '"' + name + '" already exists in this layer.' : reserved ? '"' + name + '" is reserved.' : "Use letters, digits, - or _ (must start with a letter)."}</p>
      ) : null}
      <div className="pkg-notice-actions">
        <button type="button" className="btn btn-ghost btn-sm" onClick={onCancel}>Cancel</button>
        <button
          type="button"
          className="btn btn-primary btn-sm"
          disabled={disabled || !name || !model || nameBad}
          onClick={() => onAdd({ name, model, ...(thinking ? { thinking } : {}) })}
        >Add preset</button>
      </div>
    </div>
  );
}
