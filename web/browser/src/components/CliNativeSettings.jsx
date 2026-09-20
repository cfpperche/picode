import { useCallback, useEffect, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { cliSettingsHash } from "@picode/shared/domain/cliSettings.js";
import { coerce, dangerNote, defaultLayer, groupFields, layerToScope, rowState, scopeToLayer } from "@picode/shared/domain/cliNative.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";

// The guest CLIs' own settings, edited in their own files (ADR-0163). One
// layer at a time, the file it writes named under the switcher, provenance on
// every row — the same language the Pi pane has used since 2026-09-12, drawn
// from the schema the server declares instead of a hand-written form.

export default function CliNativeSettings({ route, workspaceId = "" }) {
  const cli = route.id;
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
      const next = await api("/api/cli-settings", { method: "PATCH", body });
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
  return (
    <>
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
          {group.fields.map((field) => (
            <Row
              key={field.key}
              field={field}
              state={rowState(field, layers, current.scope)}
              cliLabel={terminalCliLabel(cli)}
              busy={saving === field.key}
              disabled={!current.writable}
              onSet={(value) => save({ key: field.key, value })}
              onReset={() => save({ key: field.key, reset: true })}
            />
          ))}
        </section>
      )) : null}
    </>
  );
}

function Row({ field, state, cliLabel, busy, disabled, onSet, onReset }) {
  const { value, setHere, from } = state;
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
  return (
    <div className={"set-row" + (setHere ? " is-set" : "")}>
      <span className="set-label">
        {field.label}
        <span className="set-src">{source}</span>
        {warn ? <span className="set-src set-warn">{warn}</span> : field.help ? <span className="set-src">{field.help}</span> : field.kind === "bool" && !setHere && field.fallback && field.fallback !== "On" && field.fallback !== "Off" ? <span className="set-src">{field.fallback}</span> : null}
      </span>
      <span className="set-ctl">
        <Control field={field} value={value} busy={busy} disabled={disabled} onSet={onSet} />
        {setHere && !disabled ? (
          <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={onReset}>Use inherited</button>
        ) : null}
      </span>
    </div>
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

function Control({ field, value, busy, disabled, onSet }) {
  const [draft, setDraft] = useState(value === undefined ? "" : String(value));
  useEffect(() => { setDraft(value === undefined ? "" : String(value)); }, [value]);

  if (field.kind === "bool") return <BoolControl field={field} value={value} busy={busy} disabled={disabled} onSet={onSet} />;
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
