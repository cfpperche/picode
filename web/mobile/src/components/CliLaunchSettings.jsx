import { useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { askConfirm } from "../lib/confirm.js";
import { registerHashGuard } from "../lib/hashGuard.js";
import { cliLaunchSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { launchDraft, launchConfig, defaultLaunchConfig, launchChanged, launchArgs, launchLines, PICODE_TOOL_FAMILIES } from "@picode/shared/domain/cliLaunch.js";
import { quickSettingsFor, readQuickValue, readQuickList, applyQuickSetting, applyQuickList, argLine } from "@picode/shared/domain/cliLaunchPresets.js";

export const cliJSON = (method, body) => ({ method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
export const confirmDiscard = () => askConfirm({ title: "Discard launch changes?", message: "Your unsaved changes will be lost.", confirmLabel: "Discard changes", danger: true });

export function useLaunchGuard(dirty, confirm = confirmDiscard) {
  const pending = useRef(false);
  const bypass = useRef(false);
  useEffect(() => {
    if (!dirty) { bypass.current = false; return; }
    const before = (e) => { if (!bypass.current) { e.preventDefault(); e.returnValue = ""; } };
    const click = async (e) => {
      const link = e.target.closest?.("a[href]");
      if (bypass.current || !link || link.target === "_blank" || link.hash === location.hash || e.ctrlKey || e.metaKey) return;
      e.preventDefault(); e.stopPropagation();
      if (pending.current) return;
      pending.current = true;
      try { if (await confirm()) { bypass.current = true; location.href = link.href; } } finally { pending.current = false; }
    };
    const hash = async (e) => {
      if (bypass.current) return;
      e.stopImmediatePropagation();
      history.replaceState(history.state, "", e.oldURL);
      if (pending.current) return;
      pending.current = true;
      try { if (await confirm()) { bypass.current = true; location.href = e.newURL; } } finally { pending.current = false; }
    };
    window.addEventListener("beforeunload", before);
    document.addEventListener("click", click, true);
    const releaseHash = registerHashGuard(hash);
    return () => { window.removeEventListener("beforeunload", before); document.removeEventListener("click", click, true); releaseHash(); };
  }, [dirty, confirm]);
  return () => { bypass.current = true; };
}

// QuickListField edits a repeatable flag (one value per line). The text is
// local so trailing newlines survive typing; it re-seeds when the parsed
// argv changes underneath (CLI switch, profile load, an Advanced edit).
function QuickListField({ spec, values, onApply }) {
  const joined = values.join("\n");
  const [text, setText] = useState(joined);
  const [seen, setSeen] = useState(joined);
  if (seen !== joined) { setSeen(joined); setText(joined); }
  return <label>{spec.label}
    <textarea aria-label={spec.label} rows={2} spellCheck={false} placeholder={spec.placeholder} value={text}
      onChange={(e) => { setText(e.target.value); onApply(launchLines(e.target.value)); }} />
    {spec.hint ? <span>{spec.hint}</span> : null}
  </label>;
}

// LaunchFields is the launch editor used by the CLI defaults, a terminal's
// overrides and launch profiles. When the CLI has verified quick controls
// (cliLaunchPresets.js) they render first, patching the same argument
// array the Advanced reveal exposes; without them the form is unchanged.
export function LaunchFields({ draft, setDraft, includeIntegration = false, cli, agentBound = false }) {
  const field = (key) => ({ value: draft[key], onChange: (e) => setDraft({ ...draft, [key]: e.target.value }) });
  const specs = cli ? quickSettingsFor(cli.id, { agentBound }) : null;
  const args = launchArgs(draft.argsText);
  const setQuick = (spec, value) => setDraft({ ...draft, argsText: applyQuickSetting(args, specs, spec, value).map(argLine).join("\n") });
  const quick = specs?.map((spec) => {
    if (spec.type === "list") {
      return <QuickListField key={spec.key} spec={spec} values={readQuickList(args, spec)}
        onApply={(vals) => setDraft({ ...draft, argsText: applyQuickList(args, spec, vals).map(argLine).join("\n") })} />;
    }
    const current = readQuickValue(args, spec);
    if (spec.type === "boolean") {
      return <label key={spec.key} className="cli-checkbox"><input type="checkbox" checked={current === "on"} onChange={(e) => setQuick(spec, e.target.checked ? "on" : "")} />{spec.label}</label>;
    }
    const warn = spec.options?.find((o) => o.value === current)?.warn;
    const listed = spec.options?.some((o) => o.value === current);
    return <label key={spec.key}>{spec.label}
      {spec.type === "select" ? <select value={current} onChange={(e) => setQuick(spec, e.target.value)}>
        <option value="">CLI default</option>
        {spec.options.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        {current && !listed ? <option value={current}>{current}</option> : null}
      </select>
        : <><input type="text" list={`cli-quick-${cli.id}-${spec.key}`} placeholder={spec.placeholder} autoComplete="off" spellCheck={false} value={current} onChange={(e) => setQuick(spec, e.target.value)} />
          {spec.suggestions ? <datalist id={`cli-quick-${cli.id}-${spec.key}`}>{spec.suggestions.map((s) => <option key={s} value={s} />)}</datalist> : null}</>}
      {warn ? <span className="is-warn">{warn}</span> : spec.hint ? <span>{spec.hint}</span> : null}
    </label>;
  });
  const advanced = <>
    <label>Executable<input type="text" placeholder="Automatic detection" autoComplete="off" {...field("executable")} /><span>Empty uses the detected CLI; a path pins this executable.</span></label>
    <details open={!!draft.argsText || undefined}><summary>Additional arguments</summary><label><span>One argument per line</span><textarea aria-label="Additional arguments" rows={3} spellCheck={false} {...field("argsText")} /></label></details>
    <details open={!!draft.pathText || undefined}><summary>Extra PATH entries</summary><label><span>One absolute directory per line</span><textarea aria-label="Extra PATH entries" rows={2} placeholder="/opt/tools/bin" spellCheck={false} {...field("pathText")} /></label></details>
    <details><summary>Environment {draft.envText ? "· customized" : "· no additions"}</summary><label><span>One NAME=value per line · values visible while editing</span><textarea aria-label="Environment" rows={3} autoComplete="off" spellCheck={false} {...field("envText")} /></label></details>
  </>;
  // PiCode tools (ADR-0154): one switch per family, only for a CLI that
  // takes MCP servers at launch; the others get the connector at workspace
  // scope from their Connectors pane, and the form says nothing here.
  const tools = draft.tools || [];
  const toggleTool = (id, on) => setDraft({ ...draft, tools: on ? [...tools.filter((t) => t !== id), id] : tools.filter((t) => t !== id) });
  const toolGroup = cli?.toolsCapable ? <fieldset className="cli-tools" aria-label="PiCode tools">
    <legend>PiCode tools</legend>
    {PICODE_TOOL_FAMILIES.map((f) => <label key={f.id} className="cli-checkbox"><input type="checkbox" checked={tools.includes(f.id)} onChange={(e) => toggleTool(f.id, e.target.checked)} />{f.label}<span>{f.hint}</span></label>)}
  </fieldset> : null;
  return <div className="cli-fields cli-launch-fields">
    {quick}
    {toolGroup}
    {specs ? <details className="cli-advanced" open={!!(draft.executable || draft.argsText || draft.pathText || draft.envText) || undefined}><summary>Advanced</summary>{advanced}</details> : advanced}
    {includeIntegration ? <label className="cli-checkbox"><input type="checkbox" checked={draft.integration} onChange={(e) => setDraft({ ...draft, integration: e.target.checked })} />Report activity on new launches</label> : null}
  </div>;
}

const list = (values, fallback) => values?.length ? values.map((v, i) => <code key={i}>{v === "" ? '""' : v}</code>) : fallback;

export function LaunchSummary({ plan, title = "Launch settings", compact = false }) {
  if (!plan) return null;
  const injection = plan.injection;
  return <section className="cli-launch-summary" aria-label={title}>
    <dl className="cli-summary-grid">
      <dt>Executable</dt><dd><code>{plan.executable || "Not found"}</code><small>{plan.origins?.executable || "Last applied"}</small></dd>
      <dt>Additional arguments</dt><dd>{list(plan.args, "None")}<small>{plan.origins?.args}</small></dd>
      <dt>Extra PATH</dt><dd>{list(plan.path, "None · service PATH")}<small>{plan.path?.length ? plan.origins?.path : null}</small></dd>
      <dt>Environment</dt><dd>{list(plan.envKeys?.map((k) => k + "=••••"), "No additions")}<small>{plan.envKeys?.length ? plan.origins?.env : null}</small></dd>
      <dt>PiCode additions</dt><dd>{plan.integration ? injection?.summary || "Activity reporting on" : "None · activity reporting off"}<small>{plan.origins?.integration}</small></dd>
      {plan.tools?.length ? <><dt>PiCode tools</dt><dd>{plan.toolInjection?.summary || plan.tools.join(", ")}<small>{plan.origins?.tools}</small></dd></> : null}
      {plan.agentInjection ? <><dt>Agent's own skills</dt><dd>{plan.agentInjection.summary}</dd></> : null}
    </dl>
    {!compact ? <details className="cli-effective"><summary>View launch details</summary>
      <div className="cli-injection">
        <h4>Added by PiCode</h4>
        {!plan.integration ? <p>No activity-reporting injection.</p> : <>
          {(injection?.branches || []).map((b) => <div key={b.when}><p>{b.when}</p><pre>{b.args.map((a) => JSON.stringify(a)).join("\n")}</pre></div>)}
          {Object.entries(injection?.environment || {}).map(([k, v]) => <p key={k}><code>{k}={v}</code></p>)}
          <h4>PiCode files</h4><div>{list(injection?.files, "None")}</div>
        </>}
        {plan.agentInjection?.branches?.length ? <>
          <h4>Agent's own skills</h4>
          {plan.agentInjection.branches.map((b) => <div key={b.when}><p>{b.when}</p><pre>{b.args.map((a) => JSON.stringify(a)).join("\n")}</pre></div>)}
          <div>{list(plan.agentInjection.files, "None")}</div>
        </> : null}
        {plan.managedEnv ? <><h4>Launch correlation</h4><p>{plan.managedEnv.join(" · ")}</p></> : null}
        {plan.inheritedPath ? <><h4>Inherited service PATH</h4><div>{list(plan.inheritedPath, "Empty")}</div></> : null}
        <p className="cli-muted">Native CLI settings stay with the CLI. <code>run-{'{next}'}</code> is allocated at launch.</p>
      </div>
    </details> : null}
  </section>;
}

export function LaunchPreview({ cli, config, overrides = {}, terminalId = "", applied, onPreview }) {
  const [result, setResult] = useState(null);
  const [error, setError] = useState("");
  const [retry, setRetry] = useState(0);
  const [pending, setPending] = useState(true);
  const callback = useRef(onPreview); callback.current = onPreview;
  const body = JSON.stringify({ ...(config ? { config } : {}), overrides, terminalId });
  useEffect(() => {
    let cancelled = false; setPending(true);
    const timer = setTimeout(() => api(`/api/clis/${cli}/preview`, cliJSON("POST", JSON.parse(body))).then((v) => {
      if (!cancelled) { setResult(v); setError(""); callback.current?.(v); }
    }).catch((e) => { if (!cancelled) setError(e.message); }).finally(() => { if (!cancelled) setPending(false); }), 180);
    return () => { cancelled = true; clearTimeout(timer); };
  }, [cli, body, retry]);
  return <section className={"cli-preview" + (pending ? " is-updating" : "")} aria-busy={pending}>
    <h4>{error && result ? "Previous preview" : applied ? "Next launch" : "Launch preview"}{pending ? <small>Updating…</small> : null}</h4>
    {error ? <div className="cli-notice is-error" role="alert"><span>{error}</span><button type="button" className="btn btn-ghost btn-sm" onClick={() => setRetry(retry + 1)}>Try again</button></div> : null}
    {!result ? (pending ? <div className="cli-loading" aria-label="Loading launch preview"><div /><div /></div> : null) : <>
      {result.plan.problem ? <p className="cli-field-error">{result.plan.problem}</p> : null}
      <LaunchSummary plan={result.plan} />
      {applied ? <div className="cli-comparison"><p>{launchChanged(applied, result.plan) ? "Different from last launch · restart explicitly to apply" : "Matches the last applied launch settings"}</p><details className="cli-effective"><summary>Compare with last launch · {new Date(applied.startedAt).toLocaleString()}</summary><LaunchSummary plan={applied} /></details></div> : null}
      {result.affected?.length ? <p className="cli-muted">Applies on the next start: {result.affected.map((t) => t.name).join(", ")}.</p> : null}
    </>}
  </section>;
}

export function CLIDefaults({ cli, busy, onSave, editRequested, hideTitle = false, editing, onEditingChange }) {
  const [localEditing, setLocalEditing] = useState(false);
  const isEditing = editing ?? localEditing;
  const setEditing = onEditingChange ?? setLocalEditing;
  const [draft, setDraft] = useState(() => launchDraft(cli.config));
  const [error, setError] = useState("");
  const dirty = isEditing && JSON.stringify({ ...draft, integration: cli.config.integration }) !== JSON.stringify(launchDraft(cli.config));
  useLaunchGuard(dirty);
  useEffect(() => { if (!isEditing) setDraft(launchDraft(cli.config)); }, [cli.config, isEditing]);
  useEffect(() => { if (editRequested) setEditing(true); }, [editRequested]);
  const parsed = parseForm(cliLaunchSchema, { ...draft, integration: cli.config.integration });
  const reset = async () => {
    if (!(await askConfirm({ title: "Restore launch defaults?", message: "Clears the executable, additional arguments, PATH entries and environment. Activity reporting stays unchanged; running terminals are not restarted.", confirmLabel: "Restore defaults" }))) return;
    setDraft(launchDraft(defaultLaunchConfig(cli.config.integration))); setEditing(true);
  };
  return <section className="cli-defaults">
    {hideTitle ? null : <div className="cli-section-heading"><h3>Launch settings</h3>{!isEditing ? <button type="button" className="btn btn-ghost btn-sm" onClick={() => setEditing(true)}>Customize</button> : <span className="cli-muted">Editing defaults</span>}</div>}
    {!isEditing ? <LaunchSummary plan={cli.plan} /> : <form noValidate onSubmit={async (e) => {
      e.preventDefault(); if (!parsed.ok) { setError(parsed.error); return; }
      try { await onSave(launchConfig(parsed.value)); setError(""); setEditing(false); } catch (e) { setError(e.message); }
    }}>
      <LaunchFields draft={draft} setDraft={setDraft} cli={cli} />
      {error ? <p className="cli-field-error" role="alert">{error}</p> : null}
      <div className="cli-actions" data-align-row><button className="btn btn-primary btn-sm" disabled={busy || !dirty}>{busy ? "Saving…" : "Save changes"}</button><button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={async () => { if (!dirty || await confirmDiscard()) { setDraft(launchDraft(cli.config)); setEditing(false); setError(""); } }}>Discard</button></div>
      {parsed.ok ? <LaunchPreview cli={cli.id} config={launchConfig(parsed.value)} /> : null}
    </form>}
    <button type="button" className="btn btn-ghost btn-sm cli-reset" disabled={busy} onClick={reset}>Restore defaults</button>
  </section>;
}
