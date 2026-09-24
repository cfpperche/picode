import { useCallback, useEffect, useMemo, useState } from "react";
import ScopeIcon from "./ScopeIcon.jsx";
import * as Switch from "@radix-ui/react-switch";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { cliModelsHash, cliPaneHash } from "@picode/shared/domain/cliLaunch.js";
import {
  defaultLayer, layerToScope, scopeToLayer, rowState, isRoleField, isListField,
  listValue, namedLayers, roleAliases, selectorsInUse, splitSelector,
} from "@picode/shared/domain/cliNative.js";
import {
  ALLOWED_KEY, HIDDEN_KEY, effectiveList, filterModels, formatContext, formatPrice,
  groupByProvider, isUnreadable, kindCounts, patternsNotListed, toggledList,
  allowedLeavesNothing, splitAllowedExtras, kindLabel,
} from "@picode/shared/domain/cliModels.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import { AddRow, Row } from "./CliNativeSettings.jsx";

// What a CLI says it can reach, and the two lists in its own config that
// decide which of those it may use (ADR-0181; omp only today). The catalog is
// the CLI's own answer, read in the workspace being edited — the same command
// answered 55 models in one folder and 0 in the next (2026-09-22). The lists
// are rows of the settings report, so a save here and a save in Settings share
// one revision and one writer.
//
// Arrays replace across layers: a workspace list is the whole list there. So
// every toggle writes the complete list for the layer being edited
// (`toggledList`), and the pane says which layer a list comes from.

const EMPTY = { state: "loading", rows: [], error: "" };

export default function CliModels({ cli, route = {}, workspaceId = "", workspaceName = "" }) {
  const label = terminalCliLabel(cli);
  const [report, setReport] = useState(null);
  const [reportError, setReportError] = useState(null);
  const [catalog, setCatalog] = useState(EMPTY);
  const [kind, setKind] = useState("");
  const [query, setQuery] = useState("");
  const [saving, setSaving] = useState("");
  const [saveError, setSaveError] = useState("");
  const [tick, setTick] = useState(0);
  const [catalogTick, setCatalogTick] = useState(0);
  // A Refresh asks the CLI again even when the server holds a current answer.
  const [fresh, setFresh] = useState(false);

  const q = useMemo(() => {
    const p = new URLSearchParams({ cli });
    if (workspaceId) p.set("workspace", workspaceId);
    return p.toString();
  }, [cli, workspaceId]);

  useEffect(() => {
    let active = true;
    api("/api/cli-settings?" + q)
      .then((rep) => { if (active) { setReport(rep); setReportError(null); } })
      .catch((err) => { if (active) setReportError(err); });
    return () => { active = false; };
  }, [q, tick]);

  // The catalog is the reason the pane exists, so it loads on mount. It keeps
  // the last answer on screen while a new one is asked for.
  useEffect(() => {
    let active = true;
    setCatalog((c) => ({ ...c, state: c.rows.length ? "refreshing" : "loading", error: "" }));
    api("/api/cli-models?" + q + (fresh ? "&fresh=1" : ""))
      .then((rep) => { if (active) { setCatalog({ state: "ready", rows: rep?.models || [], error: "", askedAt: rep?.askedAt || "" }); setFresh(false); } })
      .catch((err) => { if (active) setCatalog({ state: "error", rows: [], error: err.message || "" }); });
    return () => { active = false; };
  }, [q, catalogTick]);

  useEffect(() => {
    let timer;
    const stop = subscribeFeed((event) => {
      if (event.type === "cli.settings" && event.data?.cli === cli) {
        clearTimeout(timer);
        timer = setTimeout(() => setTick((n) => n + 1), 80);
      }
    });
    return () => { clearTimeout(timer); stop(); };
  }, [cli]);

  // The pane names the workspace it is bound to (the Pi pane's rule): one
  // rename here feeds the switcher and the "From …" line on every list.
  const layers = namedLayers(report?.layers || [], workspaceName);
  const layerName = defaultLayer(layers, route.layer);
  const scope = layerToScope(layerName);
  const current = layers.find((l) => l.scope === scope) || layers[0];
  const allowed = effectiveList(layers, scope, ALLOWED_KEY);
  const hidden = effectiveList(layers, scope, HIDDEN_KEY);
  const allowedLocked = isUnreadable(current, ALLOWED_KEY);
  const hiddenLocked = isUnreadable(current, HIDDEN_KEY);
  const writable = !!current?.writable && !!current?.path;

  const save = useCallback(async (key, value, { reset = false, recatalog = false } = {}) => {
    if (!current) return;
    setSaving(key + (reset ? ":reset" : ""));
    setSaveError("");
    const body = { cli, workspaceId, scope: current.scope, revision: current.revision };
    if (reset) body.reset = [key];
    else body.set = { [key]: value };
    try {
      const next = await api("/api/cli-settings", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (next?.layers) setReport(next); else setTick((n) => n + 1);
      // Hiding or showing a provider changes what the CLI reports; allowing a
      // model does not.
      if (recatalog) setCatalogTick((n) => n + 1);
    } catch (err) {
      setSaveError(err.message || "The save did not go through.");
      setTick((n) => n + 1);
    } finally {
      setSaving("");
    }
  }, [cli, workspaceId, current]);

  const visible = filterModels(catalog.rows, { kind, query });
  const groups = groupByProvider(visible);
  const counts = kindCounts(catalog.rows);
  const kinds = Object.keys(counts).sort((a, b) => (a === "chat" ? -1 : b === "chat" ? 1 : a.localeCompare(b)));
  const extras = catalog.state === "ready" ? patternsNotListed(allowed.list, catalog.rows) : [];
  const { patterns, unreachable } = splitAllowedExtras(extras);
  const leavesNothing = catalog.state === "ready" && allowedLeavesNothing(allowed.list, catalog.rows);
  // Arrays replace: say what a change here does, in the tense that is true.
  // Each list says where it comes from on its own line; the note only says
  // what a change to an inherited one does, which is true of either list.
  const inheritNote = scope === "project" && current?.path && ((!allowed.setHere && allowed.list.length) || (!hidden.setHere && hidden.list.length))
    ? "Changing a list marked From Global gives " + (workspaceName || "this workspace") + " its own copy, which then replaces the global one here."
    : "";
  const allHidden = catalog.state === "ready" && catalog.rows.length === 0 && hidden.list.length > 0;

  // The role matrix lives here since 2026-09-22 (owner): omp's own model hub
  // keeps roles and the model list on one screen, and a role picks from the
  // very catalog below. Its rows are the settings report's `pane: "models"`
  // fields that carry a group; the two scope lists carry none.
  const roles = report?.roles || null;
  const matrix = [];
  for (const f of report?.fields || []) {
    if (f.pane !== "models" || !f.group) continue;
    let g = matrix.find((x) => x.name === f.group);
    if (!g) { g = { name: f.group, fields: [] }; matrix.push(g); }
    g.fields.push(f);
  }
  // Inside a group: roles first, then lists (the cycle, the chains), then the
  // scalar knobs — except Fallbacks, where the knobs that decide whether a
  // chain is used come before the chains.
  for (const g of matrix) {
    const rank = (f) => isRoleField(f) ? 0 : isListField(f) ? (g.name === roles?.chainGroup ? 2 : 1) : (g.name === roles?.chainGroup ? 1 : 2);
    g.fields = g.fields.map((f, i) => [f, i]).sort((a, b) => rank(a[0]) - rank(b[0]) || a[1] - b[1]).map(([f]) => f);
  }
  const levels = roles?.levels || [];
  const cycleField = roles?.cycleKey ? (report?.fields || []).find((f) => f.key === roles.cycleKey) : null;
  const cycle = cycleField && current ? listValue(rowState(cycleField, layers, current.scope).value) : [];
  const picks = roles ? { selectors: selectorsInUse(report.fields || [], layers), aliases: roleAliases(report.fields || []) } : null;
  const pickerModels = { state: catalog.state === "ready" || catalog.state === "refreshing" ? "ready" : catalog.state, rows: catalog.rows, error: catalog.error };
  const reachable = new Set(catalog.rows.map((m) => m.selector));
  const globAllowed = allowed.list.some((a) => /[*?[\]{}]/.test(a) || !a.includes("/"));
  // A role pointing at a model this folder cannot use is the failure a
  // separate screen hid: say it on the role's own row.
  const roleNote = (field, value) => {
    if (!isRoleField(field) || catalog.state !== "ready" || typeof value !== "string") return "";
    const { base } = splitSelector(value, levels);
    if (!base.includes("/") || base.startsWith("@")) return "";
    if (!reachable.has(base)) return "Not reachable here.";
    if (allowed.list.length && !globAllowed && !allowed.list.includes(base)) return "Not in the allowed list below.";
    return "";
  };

  if (reportError) {
    return (
      <div className="cli-notice is-error" role="alert">
        <span>{reportError.message}</span>
        <button type="button" className="btn btn-ghost btn-sm" onClick={() => setTick((n) => n + 1)}>Try again</button>
      </div>
    );
  }

  return (
    <section id="cli-models-view" className="cli-models">
      {layers.length > 1 ? (
        <div className="settings-layer">
          <span className="settings-layer-label">Edit</span>
          <div className="pkg-scope" role="radiogroup" aria-label="Which file to edit">
            {layers.map((l) => (
              <a
                key={l.scope}
                className="pkg-scope-btn"
                role="radio"
                aria-checked={l.scope === current?.scope}
                href={cliModelsHash(cli, { workspaceId, layer: scopeToLayer(l.scope) })}
              ><ScopeIcon scope={l.scope} />{l.label}</a>
            ))}
          </div>
        </div>
      ) : null}
      {current?.path ? <p className="settings-file">Writes {current.path}</p> : null}
      {current?.note ? <div className="cli-notice" role="status"><span>{current.note}</span></div> : null}
      {saveError ? <div className="cli-notice is-error" role="alert"><span>{saveError}</span></div> : null}

      {current?.path ? matrix.map((group) => (
        <section className="settings-section" key={group.name}>
          <h3>{group.name}</h3>
          {group.name === roles?.chainGroup && roles?.chainHelp ? <p className="settings-desc">{roles.chainHelp}</p> : null}
          {group.fields.map((field) => {
            const state = rowState(field, layers, current.scope);
            return (
              <Row
                key={field.key}
                field={field}
                state={state}
                cliLabel={label}
                busy={saving === field.key}
                disabled={!writable}
                unreadable={(current.unreadable || []).includes(field.key)}
                filePath={current.path}
                cycle={cycle}
                levels={levels}
                models={pickerModels}
                onLoadModels={() => {}}
                picks={picks}
                note={roleNote(field, state.value)}
                workspaceName={workspaceName}
                onSet={(value) => save(field.key, value)}
                onReset={() => save(field.key, null, { reset: true })}
              />
            );
          })}
          {roles && group.name === roles.group ? (
            <AddRow label="New role" hint="Name, e.g. review" disabled={!writable}
              onAdd={(name) => save((roles.tagPrefix || "modelTags.") + name + ".name", name.toUpperCase())} />
          ) : null}
          {roles && roles.chainPrefix && group.name === roles.chainGroup ? (
            <>
              {group.fields.every((f) => !isListField(f)) ? <p className="set-src">No fallback chains yet.</p> : null}
              <AddRow label="New fallback" hint="Role, provider/model, or provider/*" disabled={!writable}
                onAdd={(name) => save(roles.chainPrefix + name, [])} />
            </>
          ) : null}
        </section>
      )) : null}

      <section className="settings-section models-all">
      <h3>All models</h3>
      <div className="models-scope">
        <ScopeLine
          text={allowedLocked
            ? "This file limits models by folder; PiCode leaves that as written."
            : allowed.list.length === 0
              ? label + " may use every model below."
              : label + " uses only the " + allowed.list.length + " allowed model" + (allowed.list.length === 1 ? "" : "s") + "."}
          source={allowed.setHere ? "Set here" : allowed.from ? "From " + allowed.from : ""}
          canReset={allowed.setHere && writable && !allowedLocked}
          busy={saving === ALLOWED_KEY + ":reset"}
          onReset={() => save(ALLOWED_KEY, null, { reset: true })}
          filePath={allowedLocked ? current?.path : ""}
        />
        {leavesNothing ? (
          <div className="models-scope-line" role="alert">
            <span className="models-warn">None of the allowed models is reachable here, so {label} has no model to use in this folder.</span>
            {writable && !allowedLocked ? (
              <button type="button" className="btn btn-ghost btn-sm" disabled={!!saving} onClick={() => save(ALLOWED_KEY, [])}>Allow every model here</button>
            ) : null}
          </div>
        ) : null}
        {extras.length ? (
          <div className="models-chips" aria-label="Allowed entries not listed below">
            <span className="models-chips-label">{unreachable.length && !patterns.length ? "Allowed, not reachable here:" : patterns.length && !unreachable.length ? "Also allowed, as written:" : "Allowed, not listed below:"}</span>
            {extras.map((p) => (
              <span className="models-chip" key={p}>
                <code>{p}</code>
                {writable && !allowedLocked ? (
                  <button type="button" className="btn btn-ghost btn-sm" disabled={!!saving}
                    onClick={() => save(ALLOWED_KEY, toggledList(layers, scope, ALLOWED_KEY, p, false))}>Remove</button>
                ) : null}
              </span>
            ))}
          </div>
        ) : null}
        {hiddenLocked ? (
          <ScopeLine text="This file hides providers by folder; PiCode leaves that as written." filePath={current?.path} />
        ) : hidden.list.length ? (
          <div className="models-chips" aria-label="Hidden providers">
            <span className="models-chips-label">Hidden · {hidden.setHere ? "Set here" : "From " + (hidden.from || "Global")}:</span>
            {hidden.list.map((p) => (
              <span className="models-chip" key={p}>
                <code>{p}</code>
                {writable ? (
                  <button type="button" className="btn btn-ghost btn-sm" disabled={!!saving}
                    onClick={() => save(HIDDEN_KEY, toggledList(layers, scope, HIDDEN_KEY, p, false), { recatalog: true })}>Show</button>
                ) : null}
              </span>
            ))}
          </div>
        ) : null}
        {inheritNote ? <p className="set-src">{inheritNote}</p> : null}
      </div>

      <div className="models-bar" data-align-row>
        <input
          className="cli-memory-search"
          type="search"
          placeholder="Filter models"
          aria-label="Filter models"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <button type="button" className="btn btn-ghost btn-sm" disabled={catalog.state === "loading" || catalog.state === "refreshing"}
          title={catalog.askedAt ? "Asked " + new Date(catalog.askedAt).toLocaleTimeString() : ""}
          onClick={() => { setFresh(true); setCatalogTick((n) => n + 1); }}>{catalog.state === "refreshing" ? "Asking…" : "Refresh"}</button>
      </div>
      <div className="pkg-scope models-kinds" role="radiogroup" aria-label="Kind">
        <button type="button" className="pkg-scope-btn" role="radio" aria-checked={kind === ""} onClick={() => setKind("")}>All{catalog.rows.length ? " " + catalog.rows.length : ""}</button>
        {kinds.map((k) => (
          <button key={k} type="button" className="pkg-scope-btn" role="radio" aria-checked={kind === k} onClick={() => setKind(kind === k ? "" : k)}>{kindLabel(k)} {counts[k]}</button>
        ))}
      </div>

      {catalog.state === "loading" ? (
        <div className="cli-loading" aria-label={"Asking " + label + " which models it can reach"}><div /><div /><div /></div>
      ) : catalog.state === "error" ? (
        <div className="cli-notice is-error" role="alert">
          <span>{catalog.error || label + " did not answer."}</span>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => setCatalogTick((n) => n + 1)}>Try again</button>
        </div>
      ) : allHidden ? (
        <div className="cli-notice" role="status">
          <span>Every provider {label} reaches is hidden in this layer.</span>
          {writable && !hiddenLocked ? (
            <button type="button" className="btn btn-ghost btn-sm" disabled={!!saving} onClick={() => save(HIDDEN_KEY, [], { recatalog: true })}>Show all</button>
          ) : null}
        </div>
      ) : catalog.rows.length === 0 ? (
        // Zero is ambiguous: a provider the project hid and a provider with no
        // credential look the same from outside, so the line does not guess.
        <div className="cli-notice" role="status">
          <span>{label} reports no models it can reach here.</span>
          <a className="btn btn-ghost btn-sm" href={cliPaneHash(cli, "providers")}>Open Providers</a>
        </div>
      ) : groups.length === 0 ? (
        <div className="cli-notice" role="status">
          <span>No model matches this filter.</span>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setQuery(""); setKind(""); }}>Clear filter</button>
        </div>
      ) : (
        <div className={"models-list" + (catalog.state === "refreshing" ? " is-refreshing" : "")}>
          <div className="models-row models-cols" aria-hidden="true">
            <span>Model</span>
            <span className="models-meta"><span className="models-num">Context</span><span className="models-num">Per 1M in / out</span></span>
            <span className="models-allow">Allowed</span>
          </div>
          {groups.map((g) => (
            <section className="models-group" key={g.provider} aria-label={g.provider}>
              <header className="models-group-head">
                <span className="models-provider">{g.provider}</span>
                <span className="set-src">{g.models.length}</span>
                {writable && !hiddenLocked ? (
                  <button type="button" className="btn btn-ghost btn-sm" disabled={!!saving}
                    onClick={() => save(HIDDEN_KEY, toggledList(layers, scope, HIDDEN_KEY, g.provider, true), { recatalog: true })}>Hide provider</button>
                ) : null}
              </header>
              {g.models.map((m) => {
                const on = allowed.list.includes(m.selector);
                return (
                  <div className="models-row" key={m.selector}>
                    <span className="models-name">
                      <span>{m.name || m.id}</span>
                      <code className="models-sel">{m.selector}</code>
                    </span>
                    <span className="models-meta">
                      {m.kind && m.kind !== "chat" ? <span className="role-tag">{kindLabel(m.kind)}</span> : null}
                      {(m.input || []).includes("image") ? <span className="role-tag" title="Accepts images as input">sees images</span> : null}
                      <span className="models-num" title="Context window">{formatContext(m.contextWindow)}</span>
                      <span className="models-num" title="USD per million tokens, input / output">{formatPrice(m.cost)}</span>
                    </span>
                    <span className="models-allow">
                      <Switch.Root
                        className="rx-switch"
                        checked={on}
                        disabled={!writable || allowedLocked || !!saving}
                        onCheckedChange={(v) => save(ALLOWED_KEY, toggledList(layers, scope, ALLOWED_KEY, m.selector, v))}
                        aria-label={"Allow " + (m.name || m.id)}
                      >
                        <Switch.Thumb className="rx-switch-thumb" />
                      </Switch.Root>
                    </span>
                  </div>
                );
              })}
            </section>
          ))}
        </div>
      )}
      </section>
    </section>
  );
}

function ScopeLine({ text, source = "", canReset = false, busy = false, onReset, filePath = "" }) {
  return (
    <div className="models-scope-line">
      <span>{text}{source ? <span className="set-src"> · {source}</span> : null}</span>
      {filePath ? <a className="btn btn-ghost btn-sm" href={"#/file/" + encodeURIComponent(filePath)}>Open the file</a> : null}
      {canReset ? <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={onReset}>Use inherited</button> : null}
    </div>
  );
}
