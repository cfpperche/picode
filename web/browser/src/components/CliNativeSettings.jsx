import { useCallback, useEffect, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { cliSettingsHash } from "@picode/shared/domain/cliSettings.js";
import {
  coerce, dangerNote, defaultLayer, groupFields, layerToScope, rowState, scopeToLayer,
  cyclePosition, isListField, isRoleField, joinSelector, listValue, roleAliases, selectorsInUse, splitSelector,
} from "@picode/shared/domain/cliNative.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import SearchCombo from "./SearchCombo.jsx";
import CliDoctor from "./CliDoctor.jsx";
import { supportsCliDoctor } from "@picode/shared/domain/cliDoctor.js";

// The guest CLIs' own settings, edited in their own files (ADR-0163). One
// layer at a time, the file it writes named under the switcher, provenance on
// every row — the same language the Pi pane has used since 2026-09-12, drawn
// from the schema the server declares instead of a hand-written form.

export default function CliNativeSettings({ route, workspaceId = "" }) {
  const cli = route.id;
  // The model list is a subprocess in the CLI's own words, and the answer
  // depends on the workspace (measured 2026-09-22: 55 models in one folder,
  // 0 in the next). It is fetched the first time a picker opens, never on
  // mount, and the pane works without it.
  const [models, setModels] = useState({ state: "idle", rows: [], error: "" });
  const [report, setReport] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState("");
  const [saveError, setSaveError] = useState("");
  const [reload, setReload] = useState(0);

  const load = useCallback(() => {
    const query = new URLSearchParams({ cli });
    if (workspaceId) query.set("workspace", workspaceId);
    return api("/api/cli-settings?" + query);
  }, [cli, workspaceId]);

  useEffect(() => {
    let active = true;
    setLoading(true);
    load().then((rep) => {
      if (!active) return;
      setReport(rep);
      setError(null);
    }).catch((err) => {
      if (active) setError(err);
    }).finally(() => {
      if (active) setLoading(false);
    });
    return () => { active = false; };
  }, [load, reload]);

  // The CLI writes these files, and so may another PiCode view.
  useEffect(() => {
    let timer;
    const stop = subscribeFeed((event) => {
      if (event.type === "cli.settings" && event.data?.cli === cli) {
        clearTimeout(timer);
        timer = setTimeout(() => setReload((n) => n + 1), 80);
      }
    });
    return () => { clearTimeout(timer); stop(); };
  }, [cli]);

  const loadModels = useCallback(() => {
    setModels((m) => {
      if (m.state !== "idle") return m;
      const query = new URLSearchParams({ cli });
      if (workspaceId) query.set("workspace", workspaceId);
      api("/api/cli-models?" + query)
        .then((rep) => setModels({ state: "ready", rows: rep?.models || [], error: "" }))
        .catch((err) => setModels({ state: "error", rows: [], error: err.message || "" }));
      return { ...m, state: "loading" };
    });
  }, [cli, workspaceId]);

  // A workspace change makes the previous answer wrong, not stale.
  useEffect(() => { setModels({ state: "idle", rows: [], error: "" }); }, [cli, workspaceId]);

  const layers = report?.layers || [];
  const layer = defaultLayer(layers, route.layer);
  const scope = layerToScope(layer);
  const current = layers.find((l) => l.scope === scope) || layers[0];

  const save = async (patch) => {
    if (!current) return;
    setSaving(patch.key);
    setSaveError("");
    const body = { cli, workspaceId, scope: current.scope, revision: current.revision };
    if (patch.reset) body.reset = [patch.key];
    else body.set = { [patch.key]: patch.value };
    try {
      // Encoded here, the way every other pane sends a body: `api` hands
      // its options to fetch untouched, and a raw object reached the server
      // as "[object Object]" — the pane reported "invalid request body" on
      // every save, in every browser, since it shipped (found by driving a
      // real write on a scratch instance, 2026-09-22).
      const next = await api("/api/cli-settings", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (next?.layers) setReport(next);
      else setReload((n) => n + 1);
    } catch (err) {
      setSaveError(err.message || "The save did not go through.");
      setReload((n) => n + 1);
    } finally {
      setSaving("");
    }
  };

  if (loading && !report) {
    return <div className="cli-loading" aria-label={"Loading " + terminalCliLabel(cli) + " settings"}><div /><div /><div /></div>;
  }
  if (error) {
    return (
      <div className="cli-notice is-error" role="alert">
        <span>{error.message}</span>
        <button type="button" className="btn btn-ghost btn-sm" onClick={() => setReload((n) => n + 1)}>Try again</button>
      </div>
    );
  }
  if (!current) {
    return <div className="cli-notice" role="status"><span>{terminalCliLabel(cli)} keeps no settings file PiCode can edit.</span></div>;
  }

  const groups = groupFields(report.fields || []);
  const roles = report.roles || null;
  // The quick-switch cycle is a row like any other, and it is also what the
  // badge on every role row reads — resolved through the same layer rules.
  const cycleField = roles?.cycleKey ? (report.fields || []).find((f) => f.key === roles.cycleKey) : null;
  const cycle = cycleField ? listValue(rowState(cycleField, layers, current.scope).value) : [];
  const picks = roles
    ? { selectors: selectorsInUse(report.fields || [], layers), aliases: roleAliases(report.fields || []) }
    : null;
  return (
    <>
      {/* The checks read the folder, not a layer, so they sit above the
          layer switcher (slice 4). */}
      {supportsCliDoctor(cli) ? <CliDoctor cli={cli} workspaceId={workspaceId} /> : null}
      {layers.length > 1 ? (
        <div className="settings-layer">
          <span className="settings-layer-label">Edit</span>
          <div className="pkg-scope" role="radiogroup" aria-label="Which file to edit">
            {layers.map((l) => (
              <a
                key={l.scope}
                className="pkg-scope-btn"
                role="radio"
                aria-checked={l.scope === current.scope}
                href={cliSettingsHash(cli, { layer: scopeToLayer(l.scope) })}
              >{l.label}</a>
            ))}
          </div>
        </div>
      ) : null}
      {current.path ? <p className="settings-file">Writes {current.path}</p> : null}

      {current.note ? (
        <div className="cli-notice" role="status"><span>{current.note}</span></div>
      ) : null}
      {current.error ? (
        <div className="cli-notice is-error" role="alert">
          <span>PiCode cannot read this file, so it will not write it.</span>
          <a className="btn btn-ghost btn-sm" href={"#/file/" + encodeURIComponent(current.path)}>Open the file</a>
        </div>
      ) : null}
      {saveError ? <div className="cli-notice is-error" role="alert"><span>{saveError}</span></div> : null}

      {current.path ? groups.map((group) => (
        <section className="settings-section" key={group.name || "general"}>
          {group.name ? <h3>{group.name}</h3> : null}
          {group.name === roles?.chainGroup && roles?.chainHelp ? <p className="settings-desc">{roles.chainHelp}</p> : null}
          {group.fields.map((field) => (
            <Row
              key={field.key}
              field={field}
              state={rowState(field, layers, current.scope)}
              cliLabel={terminalCliLabel(cli)}
              busy={saving === field.key}
              disabled={!current.writable}
              unreadable={(current.unreadable || []).includes(field.key)}
              filePath={current.path}
              cycle={cycle}
              levels={roles?.levels || []}
              models={models}
              onLoadModels={loadModels}
              picks={picks}
              onSet={(value) => save({ key: field.key, value })}
              onReset={() => save({ key: field.key, reset: true })}
            />
          ))}
          {roles && group.name === roles.group ? (
            <AddRow
              label="New role"
              hint="Name, e.g. review"
              disabled={!current.writable}
              onAdd={(name) => save({ key: (roles.tagPrefix || "modelTags.") + name + ".name", value: name.toUpperCase() })}
            />
          ) : null}
          {roles && roles.chainPrefix && group.name === roles.chainGroup ? (
            <>
              {group.fields.every((f) => !isListField(f)) ? (
                <p className="set-src">No fallback chains yet.</p>
              ) : null}
              <AddRow
                label="New fallback"
                hint="Role, provider/model, or provider/*"
                disabled={!current.writable}
                onAdd={(name) => save({ key: roles.chainPrefix + name, value: [] })}
              />
            </>
          ) : null}
        </section>
      )) : null}
    </>
  );
}

// A new role or a new chain is a key that does not exist yet, so the control
// that creates one is a name, not a value: the row appears as soon as the file
// has it, and is edited like every other row. Hidden behind one deliberate
// reveal, because the common case is assigning a role the CLI already has.
export function AddRow({ label, hint, disabled, onAdd }) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  if (disabled) return null;
  if (!open) {
    return (
      <button type="button" className="btn btn-ghost btn-sm role-add-btn" onClick={() => setOpen(true)}>+ {label}…</button>
    );
  }
  const commit = () => {
    const trimmed = name.trim();
    if (!trimmed) return;
    onAdd(trimmed);
    setName("");
    setOpen(false);
  };
  return (
    <div className="role-add">
      <span className="role-add-label">{label}</span>
      <input
        className="set-text"
        autoFocus
        value={name}
        placeholder={hint}
        aria-label={label}
        onChange={(e) => setName(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") { e.preventDefault(); commit(); }
          if (e.key === "Escape") { setOpen(false); setName(""); }
        }}
      />
      <button type="button" className="btn btn-sm" disabled={!name.trim()} onClick={commit}>Add</button>
      <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setOpen(false); setName(""); }}>Cancel</button>
    </div>
  );
}

export function Row({ field, state, cliLabel, busy, disabled, unreadable, filePath, cycle, levels, models, onLoadModels, picks, onSet, onReset, note = "" }) {
  const { value, setHere, from } = state;
  // A value this layer holds in a shape PiCode will not rewrite is named and
  // left alone: one line, one action, never a control that cannot save.
  if (unreadable) {
    return (
      <div className="set-row">
        <span className="set-label">
          {field.tag ? <span className="role-tag">{field.tag}</span> : null}
          {field.label}
          <span className="set-src">This file holds it in a shape PiCode does not rewrite.</span>
        </span>
        <span className="set-ctl">
          <a className="btn btn-ghost btn-sm" href={"#/file/" + encodeURIComponent(filePath)}>Open the file</a>
        </span>
      </div>
    );
  }
  // The sub-label answers "where does this value come from" and nothing else.
  // It used to fall back to the CLI's default, which the control already
  // shows — so an unset row printed the same sentence twice (visual review,
  // 2026-09-20). Provenance here; the default lives on the control, and for a
  // switch that means the control reads the CLI's answer while this line says
  // nobody has overridden it.
  // The same three answers Pi's pane gives: this file set it, a parent layer
  // set it, or nobody did and the CLI's own default is what runs.
  const source = setHere ? "Set here" : from ? "From " + from : cliLabel + " default";
  const warn = dangerNote(field, value);
  const spot = isRoleField(field) ? cyclePosition(cycle, field.key.split(".").pop()) : 0;
  return (
    <div data-key={field.key} className={"set-row" + (setHere ? " is-set" : "") + (isListField(field) ? " set-row-stack" : "") + (isRoleField(field) ? " set-row-role" : "")}>
      <span className="set-label">
        <span className="set-name">
          {field.tag ? <span className="role-tag">{field.tag}</span> : null}
          {field.label}
          {/* The CLI's own badge, in the CLI's own words: `⟳ 2` is the second
              stop of its quick-switch cycle (omp's model hub). */}
          {spot > 0 ? <span className="role-loop" title="Position in the quick-switch cycle">⟳ {spot}</span> : null}
        </span>
        <span className="set-src">{source}</span>
        {note ? <span className="set-src set-warn">{note}</span> : null}
        {warn ? <span className="set-src set-warn">{warn}</span> : field.help ? <span className="set-src">{field.help}</span> : field.kind === "bool" && !setHere && field.fallback && field.fallback !== "On" && field.fallback !== "Off" ? <span className="set-src">{field.fallback}</span> : null}
      </span>
      <span className="set-ctl">
        <Control
          field={field} value={value} busy={busy} disabled={disabled} onSet={onSet}
          levels={levels} models={models} onLoadModels={onLoadModels} picks={picks}
        />
        {setHere && !disabled ? (
          <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={onReset}>Use inherited</button>
        ) : null}
      </span>
    </div>
  );
}

// A model selector: the model itself in a searchable picker fed by the CLI, and
// the thinking suffix beside it. The two are written as one string, joined from
// exactly the pieces they were split from, so a value PiCode did not recognise
// survives a round trip (cliNative.splitSelector).
function RoleControl({ field, value, busy, disabled, onSet, levels, models, onLoadModels, picks }) {
  const { base, level } = splitSelector(value, levels);
  const options = [];
  if (models.state === "ready") {
    for (const m of models.rows) {
      options.push({ id: m.selector, label: m.name || m.id, hint: m.provider + (m.kind && m.kind !== "chat" ? " · " + m.kind : "") });
    }
  } else if (picks) {
    for (const s of picks.selectors) options.push({ id: s, label: s, hint: "" });
  }
  for (const alias of picks?.aliases || []) {
    if (alias === "@" + field.key.split(".").pop()) continue;
    options.push({ id: alias, label: alias, hint: alias === "*" ? "same as the default role" : "use that role's model" });
  }
  if (base && !options.some((o) => o.id === base)) options.unshift({ id: base, label: base, hint: "in this file" });
  const note =
    models.state === "loading" ? "Asking the CLI…"
      : models.state === "error" ? models.error
        : models.state === "ready" && models.rows.length === 0 ? "The CLI reports no models it can reach here."
          : "";
  return (
    <>
      <SearchCombo
        value={base}
        onChange={(next) => onSet(joinSelector(next, level))}
        options={options}
        label={base || "auto"}
        searchPlaceholder="Search models"
        ariaLabel={field.label + " model"}
        disabled={busy || disabled}
        triggerClassName="role-pick"
        side="bottom"
        collisionPadding={{ top: 56, right: 8, bottom: 8, left: 8 }}
        markCurrent
        onOpen={onLoadModels}
        footer={note ? <p className="combo-note">{note}</p> : null}
      />
      <select
        className="role-level"
        value={level}
        disabled={busy || disabled || !base}
        aria-label={field.label + " thinking level"}
        onChange={(e) => onSet(joinSelector(base, e.target.value))}
      >
        <option value="">Default thinking</option>
        {levels.map((l) => <option key={l} value={l}>{"Thinking: " + l}</option>)}
      </select>
    </>
  );
}

// An ordered list of strings — a fallback chain, or the quick-switch cycle.
// Order carries meaning in both, so the rows move rather than sort, and an
// empty list is a value the CLI reads as "never fall back", not an absence.
function ListControl({ field, value, busy, disabled, onSet }) {
  const items = listValue(value);
  const [draft, setDraft] = useState("");
  const commit = (next) => onSet(next);
  return (
    <span className="role-list">
      {items.length === 0 ? <span className="set-src">Empty — the CLI takes this as "never".</span> : null}
      {items.map((item, i) => (
        <span className="role-item" key={item + i}>
          <span className="role-item-name">{item}</span>
          <button type="button" className="btn btn-ghost btn-sm" disabled={busy || disabled || i === 0} title="Move up"
            onClick={() => { const next = [...items]; [next[i - 1], next[i]] = [next[i], next[i - 1]]; commit(next); }}>↑</button>
          <button type="button" className="btn btn-ghost btn-sm" disabled={busy || disabled || i === items.length - 1} title="Move down"
            onClick={() => { const next = [...items]; [next[i + 1], next[i]] = [next[i], next[i + 1]]; commit(next); }}>↓</button>
          <button type="button" className="btn btn-ghost btn-sm" disabled={busy || disabled} title={"Remove " + item}
            onClick={() => commit(items.filter((_, j) => j !== i))}>Remove</button>
        </span>
      ))}
      {!disabled ? (
        <span className="role-item">
          <input
            className="set-text"
            value={draft}
            placeholder="Add an entry"
            aria-label={"Add to " + field.label}
            disabled={busy}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key !== "Enter") return;
              e.preventDefault();
              const trimmed = draft.trim();
              if (!trimmed) return;
              setDraft("");
              commit([...items, trimmed]);
            }}
          />
          <button type="button" className="btn btn-sm" disabled={busy || !draft.trim()}
            onClick={() => { const trimmed = draft.trim(); setDraft(""); commit([...items, trimmed]); }}>Add</button>
        </span>
      ) : null}
    </span>
  );
}

// One control for a boolean in every pane (owner's call, 2026-09-20): the
// same Radix switch Pi's Auto-compact row has always used.
//
// A switch has no third state, so an unset key cannot be drawn as "off" — that
// would tell the reader the opposite of the truth for a key whose CLI default
// is on. It is drawn at the CLI's own default instead, which every boolean
// field declares, and the row's source line carries whether this file is what
// set it.
function BoolControl({ field, value, busy, disabled, onSet }) {
  const effective = value === true || value === false ? value : !!field.defaultOn;
  return (
    <Switch.Root
      className="rx-switch"
      checked={effective}
      disabled={busy || disabled}
      onCheckedChange={(v) => onSet(v)}
      aria-label={field.label}
    >
      <Switch.Thumb className="rx-switch-thumb" />
    </Switch.Root>
  );
}

function Control({ field, value, busy, disabled, onSet, levels, models, onLoadModels, picks }) {
  const [draft, setDraft] = useState(value === undefined ? "" : String(value));
  useEffect(() => { setDraft(value === undefined ? "" : String(value)); }, [value]);

  if (field.kind === "bool") return <BoolControl field={field} value={value} busy={busy} disabled={disabled} onSet={onSet} />;
  if (isRoleField(field)) {
    return (
      <RoleControl
        field={field} value={value} busy={busy} disabled={disabled} onSet={onSet}
        levels={levels || []} models={models || { state: "idle", rows: [] }} onLoadModels={onLoadModels} picks={picks}
      />
    );
  }
  if (isListField(field)) return <ListControl field={field} value={value} busy={busy} disabled={disabled} onSet={onSet} />;
  if (field.kind === "select") {
    return (
      <select
        className="set-wide"
        value={value === undefined ? "" : String(value)}
        disabled={busy || disabled}
        onChange={(e) => onSet(e.target.value)}
      >
        <option value="" disabled>{field.fallback || "Not set"}</option>
        {(field.options || []).map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))}
      </select>
    );
  }
  const commit = () => {
    const next = coerce(field, draft);
    if (next === null || String(next) === String(value ?? "")) return;
    onSet(next);
  };
  return (
    <input
      type={field.kind === "number" ? "number" : "text"}
      className="set-text"
      value={draft}
      placeholder={field.fallback || ""}
      disabled={busy || disabled}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); e.currentTarget.blur(); } }}
    />
  );
}
