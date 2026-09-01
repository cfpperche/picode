import { useEffect, useMemo, useRef, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import PageFrame from "./PageFrame.jsx";
import ConfigFields from "./ConfigFields.jsx";
import { api } from "../lib/api.js";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";
import { automationSchema, parseForm } from "../lib/schemas.js";
import { relTime, absTime } from "../lib/relTime.js";
import { sparklinePath } from "../lib/sparkline.js";
import { PRESETS, DOW, presetToCron, cronToPreset, describeCron, cronError } from "../lib/cron.js";
import { automationRoute, automationsHash, workspaceHash } from "../lib/routes.js";
import { mentionAgents } from "../lib/tree.js";
import { IconPlay, IconPlus, IconCopy, IconTrash, IconPencil, IconChevronLeft } from "./Icons.jsx";

const REFRESH_MS = 15_000;
const GUIDE = "https://cfpperche.github.io/picode/guide/automations";

// Automations (ADR-0044): trigger + prompt + bounds; every run is an
// ordinary session on the automation's own agent. Adapted from Devin's
// Automations list/editor/activity log (docs/benchmarks/2026-09-01-devin-
// automations.md): schedule + webhook only, bounds instead of babysitting,
// results in the Inbox. Polls like the dashboard (ADR-0042): 15 s, paused
// while hidden, last good results kept.
export default function Automations({ hidden, catalog, workspaces, freeAgents, system }) {
  const [sub, setSub] = useState(() => automationRoute());
  const [items, setItems] = useState(null);
  const [loadErr, setLoadErr] = useState("");
  const [reveal, setReveal] = useState(null); // {id, secret} shown once after create/rotate

  useEffect(() => {
    const on = () => setSub(automationRoute());
    window.addEventListener("hashchange", on);
    return () => window.removeEventListener("hashchange", on);
  }, []);

  async function load() {
    try {
      const d = await api("/api/automations");
      setItems(d.items || []);
      setLoadErr("");
    } catch (ex) {
      setLoadErr(ex.message || "Could not load automations.");
    }
  }

  useEffect(() => {
    if (hidden) return;
    load();
    let t = null;
    const start = () => { if (!t) t = setInterval(load, REFRESH_MS); };
    const stop = () => { if (t) clearInterval(t); t = null; };
    const vis = () => { if (document.hidden) stop(); else { load(); start(); } };
    if (!document.hidden) start();
    document.addEventListener("visibilitychange", vis);
    return () => { stop(); document.removeEventListener("visibilitychange", vis); };
  }, [hidden, sub]);

  const piMissing = !!(system && Array.isArray(system.warnings) && system.warnings.some((w) => /^pi is not installed/i.test(w)));
  const agents = useMemo(() => mentionAgents(workspaces, freeAgents, null), [workspaces, freeAgents]);
  const current = sub && sub !== "new" && items ? items.find((a) => a.id === sub) : null;

  async function toggle(a, enabled) {
    setItems((list) => (list || []).map((x) => (x.id === a.id ? { ...x, enabled } : x)));
    try {
      await api("/api/automations/" + encodeURIComponent(a.id), { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ enabled }) });
    } catch (ex) {
      toastError(ex);
    }
    load();
  }

  async function runNow(a) {
    try {
      await api("/api/automations/" + encodeURIComponent(a.id) + "/run", { method: "POST" });
      toast.ok(a.name + " started.");
    } catch (ex) {
      toast.info(/busy/i.test(ex.message || "") ? "The previous run is still in progress." : ex.message);
    }
    load();
  }

  async function remove(a) {
    const ok = await askConfirm({
      title: "Delete " + a.name,
      message: "The schedule and its run history go away. The agent and its sessions stay.",
      confirmLabel: "Delete",
      danger: true,
    });
    if (!ok) return;
    try {
      await api("/api/automations/" + encodeURIComponent(a.id), { method: "DELETE" });
      location.hash = automationsHash("");
      load();
    } catch (ex) {
      toastError(ex);
    }
  }

  async function rotate(a) {
    try {
      const d = await api("/api/automations/" + encodeURIComponent(a.id) + "/secret", { method: "POST" });
      setReveal({ id: a.id, secret: d.webhookSecret });
      load();
    } catch (ex) {
      toastError(ex);
    }
  }

  function onSaved(d) {
    const a = d.automation;
    if (d.webhookSecret) setReveal({ id: a.id, secret: d.webhookSecret });
    location.hash = automationsHash(a.id);
    load();
  }

  let body;
  if (sub === "new" || (sub && current && sub === current.id && location.hash.endsWith("?edit"))) {
    body = null;
  }
  if (sub === "new") {
    body = <Editor catalog={catalog} workspaces={workspaces} agents={agents} onSaved={onSaved} onCancel={() => { location.hash = automationsHash(""); }} />;
  } else if (sub) {
    body = current ? (
      <Detail
        a={current}
        catalog={catalog}
        workspaces={workspaces}
        agents={agents}
        reveal={reveal && reveal.id === current.id ? reveal.secret : ""}
        onDismissSecret={() => setReveal(null)}
        onRun={() => runNow(current)}
        onDelete={() => remove(current)}
        onRotate={() => rotate(current)}
        onToggle={(v) => toggle(current, v)}
        onSaved={onSaved}
      />
    ) : items ? (
      <div className="mcp-empty"><p>That automation is gone.</p><a className="btn btn-ghost" href={automationsHash("")}>All automations</a></div>
    ) : <Skeleton />;
  } else {
    body = (
      <List items={items} loadErr={loadErr} piMissing={piMissing} onToggle={toggle} onRun={runNow} />
    );
  }

  return (
    <PageFrame id="automations-view" title="Automations" hidden={hidden} wide>
      {body}
    </PageFrame>
  );
}

function Skeleton() {
  return (
    <div className="mcp-skel" aria-hidden="true">
      <span className="skel-line w-70" />
      <span className="skel-line w-50" />
      <span className="skel-line w-40" />
    </div>
  );
}

function List({ items, loadErr, piMissing, onToggle, onRun }) {
  if (items === null && !loadErr) return <Skeleton />;
  if (loadErr && items === null) {
    return <div className="mcp-empty"><p>{loadErr}</p></div>;
  }
  return (
    <>
      {piMissing ? (
        <div className="auto-blocked" role="status">
          <p>pi is not installed, so automations cannot start agents.</p>
          <a className="btn btn-ghost" href={GUIDE} target="_blank" rel="noreferrer">Set up pi</a>
        </div>
      ) : null}
      {items.length === 0 ? (
        <div className="mcp-empty">
          <p>No automations yet.</p>
          <a className="btn btn-primary" href={automationsHash("new")}><IconPlus /> Create automation</a>
        </div>
      ) : (
        <>
          <div className="auto-toolbar" data-align-row>
            <span className="auto-count">{items.length} automation{items.length === 1 ? "" : "s"}</span>
            <a className="btn btn-primary" href={automationsHash("new")}><IconPlus /> Create automation</a>
          </div>
          <ul className="auto-list">
            {items.map((a) => (
              <li key={a.id} className={"auto-row" + (a.enabled ? "" : " off")}>
                <Switch.Root className="rx-switch" checked={a.enabled} onCheckedChange={(v) => onToggle(a, v)} aria-label={(a.enabled ? "Disable " : "Enable ") + a.name}>
                  <Switch.Thumb className="rx-switch-thumb" />
                </Switch.Root>
                <a className="auto-row-main" href={automationsHash(a.id)}>
                  <span className="auto-name">{a.name}</span>
                  <span className="auto-when">{whenLine(a)}</span>
                </a>
                <Spark points={a.sparkline} />
                <LastRun run={a.lastRun} />
                <div className="auto-row-actions" data-align-row>
                  <button type="button" className="btn btn-ghost" onClick={() => onRun(a)} disabled={a.running}>
                    <IconPlay /> Run now
                  </button>
                </div>
              </li>
            ))}
          </ul>
        </>
      )}
    </>
  );
}

function whenLine(a) {
  const parts = [];
  if (a.cron) parts.push(describeCron(a.cron));
  if (a.webhook) parts.push("Webhook");
  if (a.action === "message") parts.push("Messages an agent");
  if (a.enabled && a.nextFireAt) parts.push("next " + untilText(a.nextFireAt));
  return parts.join(" · ");
}

// untilText("2026-09-01T17:53:00-03:00") -> "in 16m" / "in 3h" / "in 2d" / "any minute".
function untilText(iso, now = Date.now()) {
  const d = Date.parse(iso || "") - now;
  if (!Number.isFinite(d) || d < 60_000) return "any minute";
  if (d < 3_600_000) return "in " + Math.round(d / 60_000) + "m";
  if (d < 86_400_000) return "in " + Math.round(d / 3_600_000) + "h";
  return "in " + Math.round(d / 86_400_000) + "d";
}

function Spark({ points }) {
  const spark = points && points.length >= 2 && points.some((p) => p > 0) ? sparklinePath(points, { width: 72, height: 22, pad: 2 }) : null;
  if (!spark) return <span className="auto-spark auto-spark-empty" aria-hidden="true" />;
  return (
    <svg className="auto-spark stat-tile-spark" viewBox={"0 0 " + spark.width + " " + spark.height} aria-label="Runs over the last 30 days">
      {spark.mainPath ? <path d={spark.mainPath} className="stat-tile-spark-hist" /> : null}
      <path d={spark.headPath} className="stat-tile-spark-cur" />
      <circle cx={spark.dot.x} cy={spark.dot.y} r="1.6" className="stat-tile-spark-dot" />
    </svg>
  );
}

function StatusPill({ status, reason }) {
  const label = status === "running" ? "Running" : status === "done" ? "Done" : status === "failed" ? "Failed" : "Skipped";
  return (
    <span className={"auto-pill auto-pill-" + status} title={reason || ""}>
      {label}
      {reason && status !== "done" ? <span className="auto-pill-reason"> · {reason}</span> : null}
    </span>
  );
}

function LastRun({ run }) {
  if (!run) return <span className="auto-last auto-last-none">Never ran</span>;
  return (
    <span className="auto-last" title={absTime(run.firedAt)}>
      <StatusPill status={run.status} reason={run.reason} />
      <span className="auto-last-when">{relTime(run.firedAt)}</span>
    </span>
  );
}

function money(n) {
  if (!n) return "$0.00";
  return "$" + (n < 0.01 ? n.toFixed(3) : n.toFixed(2));
}

function Detail({ a, catalog, workspaces, agents, reveal, onDismissSecret, onRun, onDelete, onRotate, onToggle, onSaved }) {
  const [editing, setEditing] = useState(false);
  const [runs, setRuns] = useState(null);
  const fireURL = location.origin + "/api/automations/" + encodeURIComponent(a.id) + "/fire";

  useEffect(() => {
    setEditing(false);
  }, [a.id]);

  useEffect(() => {
    let on = true;
    const load = () => api("/api/automations/" + encodeURIComponent(a.id) + "/runs?limit=50").then((d) => { if (on) setRuns(d.items || []); }).catch(() => {});
    load();
    const t = setInterval(() => { if (!document.hidden) load(); }, REFRESH_MS);
    return () => { on = false; clearInterval(t); };
  }, [a.id, a.lastRun && a.lastRun.id, a.running]);

  if (editing) {
    return <Editor initial={a} catalog={catalog} workspaces={workspaces} agents={agents} onSaved={(d) => { setEditing(false); onSaved(d); }} onCancel={() => setEditing(false)} />;
  }

  return (
    <div className="auto-detail">
      <div className="auto-detail-head" data-align-row>
        <a className="btn btn-ghost btn-sm" href={automationsHash("")}><IconChevronLeft /> All automations</a>
        <div className="auto-detail-actions" data-align-row>
          <span className="auto-switch-cell">
            <Switch.Root className="rx-switch" checked={a.enabled} onCheckedChange={onToggle} aria-label={(a.enabled ? "Disable " : "Enable ") + a.name}>
              <Switch.Thumb className="rx-switch-thumb" />
            </Switch.Root>
          </span>
          <button type="button" className="btn btn-ghost" onClick={onRun} disabled={a.running}><IconPlay /> Run now</button>
          <button type="button" className="btn btn-ghost" onClick={() => setEditing(true)}><IconPencil /> Edit</button>
          <button type="button" className="btn btn-ghost btn-danger" onClick={onDelete}><IconTrash /> Delete</button>
        </div>
      </div>
      <h3 className="auto-detail-title">{a.name}</h3>
      <p className="auto-when">{whenLine(a)}</p>
      <pre className="auto-prompt">{a.prompt}</pre>
      <dl className="auto-facts">
        {a.agentId ? <><dt>Agent</dt><dd><a href={workspaceHash(a.agentId)}>{a.agentName || a.name}</a></dd></> : null}
        {a.model ? <><dt>Model</dt><dd>{a.model}{a.thinking ? " · " + a.thinking : ""}</dd></> : null}
        {a.maxCostUsd ? <><dt>Max cost per run</dt><dd>{money(a.maxCostUsd)}</dd></> : null}
        {a.maxRuns ? <><dt>Max runs</dt><dd>{a.maxRuns} per {windowLabel(a.maxRunsWindowMin)}</dd></> : null}
      </dl>

      {a.webhook ? (
        <section className="settings-section auto-webhook">
          <h3>Webhook</h3>
          <div className="auto-copy-row" data-align-row>
            <code className="auto-code">{fireURL}</code>
            <CopyButton text={fireURL} label="Copy URL" />
          </div>
          {reveal ? (
            <div className="auto-secret" role="status">
              <div className="auto-copy-row" data-align-row>
                <code className="auto-code">{reveal}</code>
                <CopyButton text={reveal} label="Copy secret" />
              </div>
              <p className="auto-secret-note">Send it as <code>Authorization: Bearer</code>. This is the only time it is shown.</p>
              <button type="button" className="btn btn-ghost btn-sm" onClick={onDismissSecret}>Done</button>
            </div>
          ) : (
            <button type="button" className="btn btn-ghost btn-sm" onClick={onRotate}>Regenerate secret</button>
          )}
        </section>
      ) : null}

      <section className="settings-section">
        <h3>Runs</h3>
        <RunsTable runs={runs} agentId={a.agentId} />
      </section>
    </div>
  );
}

function windowLabel(min) {
  if (min === 60) return "hour";
  if (min === 1440) return "day";
  if (min === 10080) return "week";
  return min + " min";
}

function CopyButton({ text, label }) {
  const [done, setDone] = useState(false);
  return (
    <button type="button" className="btn btn-ghost" aria-label={label} onClick={async () => {
      try { await navigator.clipboard.writeText(text); setDone(true); setTimeout(() => setDone(false), 1200); } catch { toast.info("Copy failed — select the text instead."); }
    }}>
      <IconCopy /> {done ? "Copied" : "Copy"}
    </button>
  );
}

function RunsTable({ runs, agentId }) {
  if (runs === null) return <Skeleton />;
  if (!runs.length) return <p className="settings-desc">No runs yet. Use Run now to try it.</p>;
  return (
    <table className="auto-runs">
      <thead>
        <tr><th>When</th><th>Trigger</th><th>Result</th><th className="num">Cost</th><th /></tr>
      </thead>
      <tbody>
        {runs.map((r) => (
          <tr key={r.id} className={"auto-run" + (r.status === "running" ? " running" : "")}>
            <td title={absTime(r.firedAt)}>{relTime(r.firedAt)}</td>
            <td>{r.trigger}</td>
            <td><StatusPill status={r.status} reason={r.reason} /></td>
            <td className="num">{r.status === "skipped" ? "" : money(r.costUsd)}</td>
            <td>{r.sessionPath && agentId ? <a href={workspaceHash(agentId)}>Open agent</a> : null}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function emptyForm(initial) {
  const p = cronToPreset(initial && initial.cron ? initial.cron : "0 9 * * 1-5");
  return {
    name: initial ? initial.name : "",
    workspaceId: initial ? initial.workspaceId : "ws_free",
    action: initial ? initial.action : "start",
    targetAgentId: (initial && initial.targetAgentId) || "",
    prompt: initial ? initial.prompt : "",
    provider: (initial && initial.provider) || "",
    model: (initial && initial.model) || "",
    thinking: (initial && initial.thinking) || "",
    scheduleOn: initial ? !!initial.cron : true,
    preset: p.kind,
    time: p.time,
    dow: p.dow,
    cron: p.cron,
    advanced: p.kind === "custom",
    webhook: initial ? !!initial.webhook : false,
    maxCostUsd: initial && initial.maxCostUsd ? String(initial.maxCostUsd) : "",
    maxRuns: initial && initial.maxRuns ? String(initial.maxRuns) : "",
    maxRunsWindowMin: initial && initial.maxRunsWindowMin ? String(initial.maxRunsWindowMin) : "60",
    limits: !!(initial && (initial.maxCostUsd || initial.maxRuns)),
  };
}

function Editor({ initial, catalog, workspaces, agents, onSaved, onCancel }) {
  const [f, setF] = useState(() => emptyForm(initial));
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const nameRef = useRef(null);
  const set = (patch) => setF((x) => ({ ...x, ...patch }));

  useEffect(() => { if (nameRef.current) nameRef.current.focus(); }, []);

  const cron = f.preset === "custom" ? f.cron : presetToCron({ kind: f.preset, time: f.time, dow: f.dow });
  const cronErr = f.scheduleOn ? cronError(cron) : "";

  async function save(e) {
    e.preventDefault();
    const parsed = parseForm(automationSchema, { ...f, cron });
    if (!parsed.ok) { setErr(parsed.error); return; }
    setErr("");
    setBusy(true);
    const body = {
      name: f.name.trim(),
      workspaceId: f.workspaceId,
      action: f.action,
      targetAgentId: f.action === "message" ? f.targetAgentId : "",
      prompt: f.prompt,
      provider: f.provider, model: f.model, thinking: f.thinking,
      cron: f.scheduleOn ? cron : "",
      webhook: f.webhook,
      maxCostUsd: f.maxCostUsd === "" ? 0 : Number(f.maxCostUsd),
      maxRuns: f.maxRuns === "" ? 0 : Number(f.maxRuns),
      maxRunsWindowMin: f.maxRuns === "" ? 0 : Number(f.maxRunsWindowMin),
    };
    try {
      const d = initial
        ? await api("/api/automations/" + encodeURIComponent(initial.id), { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) })
        : await api("/api/automations", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
      toast.ok(initial ? "Saved." : "Automation created.");
      onSaved(d);
    } catch (ex) {
      setErr(ex.message || "Could not save.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="auto-form" onSubmit={save} noValidate>
      <div className="auto-detail-head" data-align-row>
        <button type="button" className="btn btn-ghost btn-sm" onClick={onCancel}><IconChevronLeft /> {initial ? initial.name : "All automations"}</button>
      </div>
      <h3 className="auto-detail-title">{initial ? "Edit automation" : "New automation"}</h3>

      <label className="auto-field">
        <span>Name</span>
        <input ref={nameRef} className="dlg-input" value={f.name} onChange={(e) => set({ name: e.target.value })} placeholder="Nightly test run" maxLength={60} />
      </label>

      <fieldset className="auto-field">
        <legend>What it does</legend>
        <Segmented
          name="auto-action"
          value={f.action}
          onChange={(v) => set({ action: v })}
          options={[{ id: "start", label: "Start a new run" }, { id: "message", label: "Message an agent" }]}
        />
        {f.action === "message" ? (
          <select className="auto-select" value={f.targetAgentId} onChange={(e) => set({ targetAgentId: e.target.value })} aria-label="Agent to message">
            <option value="">Pick an agent</option>
            {agents.map((ag) => <option key={ag.id} value={ag.id}>{ag.name}</option>)}
          </select>
        ) : (
          <div className="auto-inline" data-align-row>
            <select className="auto-select" value={f.workspaceId} onChange={(e) => set({ workspaceId: e.target.value })} aria-label="Workspace">
              <option value="ws_free">No workspace</option>
              {(workspaces || []).map((w) => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
            <ConfigFields catalog={catalog} provider={f.provider} model={f.model} thinking={f.thinking} onChange={(c) => set(c)} idPrefix="auto" row />
            {f.provider || f.model || f.thinking ? (
              <button type="button" className="btn btn-ghost" onClick={() => set({ provider: "", model: "", thinking: "" })}>Use defaults</button>
            ) : <span className="auto-hint">Default model</span>}
          </div>
        )}
      </fieldset>

      <label className="auto-field">
        <span>Prompt</span>
        <textarea className="auto-textarea" value={f.prompt} onChange={(e) => set({ prompt: e.target.value })} rows={5} placeholder="What should the agent do each time?" />
      </label>

      <fieldset className="auto-field">
        <legend className="auto-legend-row">
          <Switch.Root className="rx-switch" checked={f.scheduleOn} onCheckedChange={(v) => set({ scheduleOn: v })} id="auto-schedule-on">
            <Switch.Thumb className="rx-switch-thumb" />
          </Switch.Root>
          <label htmlFor="auto-schedule-on">Schedule</label>
        </legend>
        {f.scheduleOn ? (
          <>
            <Segmented name="auto-preset" value={f.preset} onChange={(v) => set({ preset: v, advanced: v === "custom", cron: v === "custom" ? (cron || f.cron) : f.cron })} options={PRESETS} />
            {f.preset !== "custom" ? (
              <div className="auto-inline" data-align-row>
                {f.preset === "weekly" ? (
                  <select className="auto-select" value={f.dow} onChange={(e) => set({ dow: Number(e.target.value) })} aria-label="Day of week">
                    {DOW.map((d, i) => <option key={d} value={i}>{d}</option>)}
                  </select>
                ) : null}
                <label className="auto-inline-label">{f.preset === "hourly" ? "at minute" : "at"}
                  <input className="dlg-input auto-time" type="time" value={f.time} onChange={(e) => set({ time: e.target.value })} step={60} />
                </label>
                <span className="auto-hint">{describeCron(cron) || "Pick a time"}</span>
              </div>
            ) : (
              <div className="auto-inline" data-align-row>
                <input className="dlg-input auto-cron" value={f.cron} onChange={(e) => set({ cron: e.target.value })} placeholder="*/30 9-18 * * 1-5" aria-label="Cron expression" spellCheck={false} />
                <span className={"auto-hint" + (cronErr ? " bad" : "")}>{cronErr || "minute · hour · day · month · weekday"}</span>
              </div>
            )}
          </>
        ) : null}
      </fieldset>

      <fieldset className="auto-field">
        <legend className="auto-legend-row">
          <Switch.Root className="rx-switch" checked={f.webhook} onCheckedChange={(v) => set({ webhook: v })} id="auto-webhook-on">
            <Switch.Thumb className="rx-switch-thumb" />
          </Switch.Root>
          <label htmlFor="auto-webhook-on">Webhook</label>
          <span className="auto-hint">Runs when another tool POSTs to its URL. The secret is shown after saving.</span>
        </legend>
      </fieldset>

      <details className="auto-details" open={f.limits} onToggle={(e) => set({ limits: e.currentTarget.open })}>
        <summary>Limits</summary>
        <div className="auto-inline" data-align-row>
          <label className="auto-inline-label">Max cost per run $
            <input className="dlg-input auto-num" inputMode="decimal" value={f.maxCostUsd} onChange={(e) => set({ maxCostUsd: e.target.value })} placeholder="none" />
          </label>
          <label className="auto-inline-label">Max runs
            <input className="dlg-input auto-num" inputMode="numeric" value={f.maxRuns} onChange={(e) => set({ maxRuns: e.target.value })} placeholder="none" />
          </label>
          <label className="auto-inline-label">per
            <select className="auto-select" value={f.maxRunsWindowMin} onChange={(e) => set({ maxRunsWindowMin: e.target.value })} disabled={f.maxRuns === ""}>
              <option value="60">hour</option>
              <option value="1440">day</option>
              <option value="10080">week</option>
            </select>
          </label>
        </div>
      </details>

      {err ? <p className="form-error" role="alert">{err}</p> : null}
      <div className="dlg-actions">
        <button type="button" className="btn btn-ghost" onClick={onCancel}>Cancel</button>
        <button type="submit" className="btn btn-primary" disabled={busy}>{initial ? "Save" : "Create automation"}</button>
      </div>
    </form>
  );
}

// Native radios under one control, same recipe as .termset-seg.
function Segmented({ name, value, onChange, options }) {
  return (
    <div className="termset-seg auto-seg" role="radiogroup">
      {options.map((o) => (
        <label key={o.id} className="termset-seg-opt">
          <input type="radio" name={name} value={o.id} checked={value === o.id} onChange={() => onChange(o.id)} />
          <span className="termset-seg-face">{o.label}</span>
        </label>
      ))}
    </div>
  );
}
