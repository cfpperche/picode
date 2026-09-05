import { useEffect, useRef, useState } from "react";
import { api } from "../lib/api.js";
import { askConfirm } from "../lib/confirm.js";
import { registerHashGuard } from "../lib/hashGuard.js";
import { cliLaunchSchema, parseForm } from "../lib/schemas.js";
import { launchDraft, launchConfig, defaultLaunchConfig, launchChanged } from "../lib/cliLaunch.js";

export const cliJSON = (method, body) => ({ method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
export const confirmDiscard = () => askConfirm({ title: "Discard launch changes?", message: "Your unsaved changes will be lost.", confirmLabel: "Discard changes", danger: true });

export function useLaunchGuard(dirty) {
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
      try { if (await confirmDiscard()) { bypass.current = true; location.href = link.href; } } finally { pending.current = false; }
    };
    const hash = async (e) => {
      if (bypass.current) return;
      e.stopImmediatePropagation();
      history.replaceState(history.state, "", e.oldURL);
      if (pending.current) return;
      pending.current = true;
      try { if (await confirmDiscard()) { bypass.current = true; location.href = e.newURL; } } finally { pending.current = false; }
    };
    window.addEventListener("beforeunload", before);
    document.addEventListener("click", click, true);
    const releaseHash = registerHashGuard(hash);
    return () => { window.removeEventListener("beforeunload", before); document.removeEventListener("click", click, true); releaseHash(); };
  }, [dirty]);
  return () => { bypass.current = true; };
}

export function LaunchFields({ draft, setDraft, includeIntegration = false }) {
  const field = (key) => ({ value: draft[key], onChange: (e) => setDraft({ ...draft, [key]: e.target.value }) });
  return <div className="cli-fields cli-launch-fields">
    <label>Executable<input type="text" placeholder="Automatic detection" autoComplete="off" {...field("executable")} /><span>Empty uses the detected CLI; a path pins this executable.</span></label>
    <details open={!!draft.argsText || undefined}><summary>Additional arguments</summary><label><span>One argument per line</span><textarea aria-label="Additional arguments" rows={3} spellCheck={false} {...field("argsText")} /></label></details>
    <details open={!!draft.pathText || undefined}><summary>Extra PATH entries</summary><label><span>One absolute directory per line</span><textarea aria-label="Extra PATH entries" rows={2} placeholder="/opt/tools/bin" spellCheck={false} {...field("pathText")} /></label></details>
    <details><summary>Environment {draft.envText ? "· customized" : "· no additions"}</summary><label><span>One NAME=value per line · values visible while editing</span><textarea aria-label="Environment" rows={3} autoComplete="off" spellCheck={false} {...field("envText")} /></label></details>
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
    </dl>
    {!compact ? <details className="cli-effective"><summary>View launch details</summary>
      <div className="cli-injection">
        <h4>Added by PiCode</h4>
        {!plan.integration ? <p>No activity-reporting injection.</p> : <>
          {(injection?.branches || []).map((b) => <div key={b.when}><p>{b.when}</p><pre>{b.args.map((a) => JSON.stringify(a)).join("\n")}</pre></div>)}
          {Object.entries(injection?.environment || {}).map(([k, v]) => <p key={k}><code>{k}={v}</code></p>)}
          <h4>PiCode files</h4><div>{list(injection?.files, "None")}</div>
        </>}
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

export function CLIDefaults({ cli, busy, onSave, editRequested }) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(() => launchDraft(cli.config));
  const [error, setError] = useState("");
  const dirty = editing && JSON.stringify({ ...draft, integration: cli.config.integration }) !== JSON.stringify(launchDraft(cli.config));
  useLaunchGuard(dirty);
  useEffect(() => { if (!editing) setDraft(launchDraft(cli.config)); }, [cli.config, editing]);
  useEffect(() => { if (editRequested) setEditing(true); }, [editRequested]);
  const parsed = parseForm(cliLaunchSchema, { ...draft, integration: cli.config.integration });
  const reset = async () => {
    if (!(await askConfirm({ title: "Restore launch defaults?", message: "Clears the executable, additional arguments, PATH entries and environment. Activity reporting stays unchanged; running terminals are not restarted.", confirmLabel: "Restore defaults" }))) return;
    setDraft(launchDraft(defaultLaunchConfig(cli.config.integration))); setEditing(true);
  };
  return <section className="cli-defaults">
    <div className="cli-section-heading"><h3>Launch settings</h3>{!editing ? <button className="btn btn-ghost btn-sm" onClick={() => setEditing(true)}>Customize</button> : <span className="cli-muted">Editing defaults</span>}</div>
    {!editing ? <LaunchSummary plan={cli.plan} /> : <form noValidate onSubmit={async (e) => {
      e.preventDefault(); if (!parsed.ok) { setError(parsed.error); return; }
      try { await onSave(launchConfig(parsed.value)); setError(""); setEditing(false); } catch (e) { setError(e.message); }
    }}>
      <LaunchFields draft={draft} setDraft={setDraft} />
      {error ? <p className="cli-field-error" role="alert">{error}</p> : null}
      <div className="cli-actions" data-align-row><button className="btn btn-primary btn-sm" disabled={busy || !dirty}>{busy ? "Saving…" : "Save changes"}</button><button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={async () => { if (!dirty || await confirmDiscard()) { setDraft(launchDraft(cli.config)); setEditing(false); setError(""); } }}>Discard</button></div>
      {parsed.ok ? <LaunchPreview cli={cli.id} config={launchConfig(parsed.value)} /> : null}
    </form>}
    <button type="button" className="btn btn-ghost btn-sm cli-reset" disabled={busy} onClick={reset}>Restore defaults</button>
  </section>;
}
