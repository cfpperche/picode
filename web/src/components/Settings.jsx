import { useEffect, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import PushPrefs from "./PushPrefs.jsx";
import { api } from "../lib/api.js";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { IconSun, IconMonitor, IconMoon, IconMore } from "./Icons.jsx";
import PageFrame from "./PageFrame.jsx";
import ThemeCard from "./ThemeCard.jsx";
import { toast, toastError } from "../lib/toast.js";
import { readToastPrefs, persistToastPrefs, TOAST_POSITIONS, TOAST_CLOSE_PLACES } from "../lib/toastPrefs.js";
import { readContextMenuPrefs, persistContextMenuPrefs, CTX_MODIFIERS } from "../lib/contextMenuPrefs.js";
import AppKeys from "./AppKeys.jsx";
import FolderField from "./FolderField.jsx";
import AccessSection from "./AccessSection.jsx";
import TermWiring from "./TermWiring.jsx";
import PiSpinner from "./PiSpinner.jsx";
import { askConfirm, fmtBytes } from "../lib/confirm.js";
import { prefSection } from "../lib/routes.js";
import { z } from "zod";

const publicUrlSchema = z.string().trim().regex(/^https?:\/\/[^\s/]+\/?$/, "An origin like https://box.tailxxxx.ts.net:8445").or(z.literal(""));

export default function Settings({ hidden, themeMode, onTheme }) {
  const [port, setPort] = useState("");
  const [note, setNote] = useState("");
  const [moving, setMoving] = useState(false);
  const [err, setErr] = useState("");
  const [info, setInfo] = useState(null); // GET /api/server
  const [bind, setBind] = useState("");
  const [pubUrl, setPubUrl] = useState("");
  const [reachErr, setReachErr] = useState("");
  const [toastPrefs, setToastPrefs] = useState(readToastPrefs);
  const [ctxPrefs, setCtxPrefs] = useState(readContextMenuPrefs);

  useEffect(() => {
    if (hidden) return;
    api("/api/server").then((info) => {
      setPort(String(info.current));
      setNote(`Serving on port ${info.current} (configured: ${info.configured}).`);
      setMoving(false);
      setInfo(info);
      setBind(info.host || "0.0.0.0");
      setPubUrl(info.publicUrl || "");
    }).catch(() => setNote("Server state unavailable."));
  }, [hidden]);

  // Reconnect on the address the server moves to (ADR-0050): the reply
  // leaves on the old listener; the new one is up 1.5 s later.
  function reconnect(host, portNo, delay = 1500) {
    const scheme = location.protocol === "http:" ? "http" : "https";
    setMoving(true);
    setTimeout(() => { location.replace(`${scheme}://${host}:${portNo}/#/preferences/server`); }, delay);
  }

  async function applyBind() {
    setReachErr("");
    try {
      const res = await api("/api/server/host", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ host: bind }),
      });
      if (res.moving) {
        // Loopback always stays (ADR-0050), and an unspecified bind
        // covers every address: this tab keeps its origin unless it
        // came in on an outside address the new bind leaves behind.
        const here = location.hostname;
        const local = here === "localhost" || here === "127.0.0.1" || here === "::1" || here === "[::1]";
        const keep = local || bind === "0.0.0.0" || bind === "::" || bind === here;
        const host = keep ? here : bind;
        setNote(keep ? "Moving the listener — reconnecting…" : `This address stops answering; reconnecting on ${host}…`);
        reconnect(host, info ? info.current : port, 2500);
      } else toast.ok("Bind unchanged.");
    } catch (e) { setReachErr(e.message); }
  }

  async function applyPublicUrl(value = pubUrl) {
    setReachErr("");
    const parsed = publicUrlSchema.safeParse(value);
    if (!parsed.success) { setReachErr(parsed.error.issues[0].message); return; }
    try {
      const res = await api("/api/server/public-url", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url: parsed.data }),
      });
      setPubUrl(res.publicUrl || "");
      setInfo((i) => (i ? { ...i, publicUrl: res.publicUrl || "" } : i));
      toast.ok(res.publicUrl ? "Public URL saved." : "Public URL cleared.");
    } catch (e) { setReachErr(e.message); }
  }

  function saveToast(patch) {
    setToastPrefs(persistToastPrefs({ ...toastPrefs, ...patch }));
  }

  function saveCtx(patch) {
    setCtxPrefs(persistContextMenuPrefs({ ...ctxPrefs, ...patch }));
  }

  async function applyPort() {
    setErr("");
    if (!port.trim()) return;
    try {
      const res = await api("/api/server/port", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ port: port.trim() }),
      });
      if (res.moving) {
        setNote(`Moving to port ${res.port} — reconnecting…`);
        setMoving(true);
        const host = location.hostname;
        const scheme = location.protocol === "http:" ? "http" : "https";
        setTimeout(() => { location.replace(`${scheme}://${host}:${res.port}/#/preferences`); }, 1500);
      }
    } catch (e) {
      setErr(e.message);
    }
  }

  const [sec, setSec] = useState(() => prefSection());
  useEffect(() => {
    function onHash() { setSec(prefSection()); }
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);
  return (
    <PageFrame id="preferences-view" title="Preferences" hidden={hidden}>
      <nav className="pref-tabs" role="tablist" aria-label="Preferences">
        {[["appearance", "Appearance"], ["shortcuts", "Shortcuts"], ["notifications", "Notifications"], ["status", "Terminal status"], ["server", "Server"], ["backup", "Backup"]].map(([id, label]) => (
          <a
            key={id}
            href={"#/preferences" + (id === "appearance" ? "" : "/" + id)}
            className="pref-tab"
            role="tab"
            aria-selected={sec === id}
          >{label}</a>
        ))}
      </nav>

      <section className="settings-section" hidden={sec !== "appearance"}>
        <h3 className="sr-only">Appearance</h3>
        <div className="theme-cards" role="radiogroup" aria-label="App theme">
          <ThemeCard option="light" label="Light" desc="Bright surfaces" active={themeMode === "light"} onPick={onTheme} icon={<IconSun size={15} />} />
          <ThemeCard option="system" label="System" desc="Match your OS" active={themeMode === "system"} onPick={onTheme} icon={<IconMonitor size={15} />} />
          <ThemeCard option="dark" label="Dark" desc="Low light" active={themeMode === "dark"} onPick={onTheme} icon={<IconMoon size={15} />} />
        </div>
      </section>

      <section className="settings-section" hidden={sec !== "shortcuts"}>
        <h3>Mouse</h3>
        <div className="set-rows">
          <div className="set-row">
            <label htmlFor="ctx-modifier">Browser menu on right-click</label>
            <select id="ctx-modifier" value={ctxPrefs.bypassModifier} onChange={(e) => saveCtx({ bypassModifier: e.target.value })}>
              {CTX_MODIFIERS.map((m) => <option key={m} value={m}>{m[0].toUpperCase() + m.slice(1)} + right-click</option>)}
            </select>
          </div>
        </div>
      </section>
      <AppKeys hidden={sec !== "shortcuts"} />

      <section className="settings-section" hidden={sec !== "status"}>
        <h3>Terminal status</h3>
        <TermWiring hidden={sec !== "status"} />
      </section>

      <section className="settings-section" hidden={sec !== "notifications"}>
        <h3 className="sr-only">Notifications</h3>
        <PushPrefs />
        <h4 className="push-prefs-sub">Toasts</h4>
        <div className="set-rows">
          <div className="set-row">
            <label htmlFor="toast-pos">Position</label>
            <select id="toast-pos" value={toastPrefs.position} onChange={(e) => saveToast({ position: e.target.value })}>
              {TOAST_POSITIONS.map((p) => <option key={p} value={p}>{p.replaceAll("-", " ")}</option>)}
            </select>
          </div>
          <div className="set-row">
            <label htmlFor="toast-dur">Duration (ms)</label>
            <input id="toast-dur" type="number" min={1500} max={15000} step={500} value={toastPrefs.duration} onChange={(e) => saveToast({ duration: Number(e.target.value) })} />
          </div>
          <div className="set-row">
            <label htmlFor="toast-n">Visible at once</label>
            <input id="toast-n" type="number" min={1} max={5} value={toastPrefs.visibleToasts} onChange={(e) => saveToast({ visibleToasts: Number(e.target.value) })} />
          </div>
          <div className="set-row">
            <label htmlFor="toast-expand">Expand stack</label>
            <Switch.Root id="toast-expand" className="rx-switch" checked={toastPrefs.expand} onCheckedChange={(v) => saveToast({ expand: v })}>
              <Switch.Thumb className="rx-switch-thumb" />
            </Switch.Root>
          </div>
          <div className="set-row">
            <label htmlFor="toast-close">Close button</label>
            <Switch.Root id="toast-close" className="rx-switch" checked={toastPrefs.closeButton} onCheckedChange={(v) => saveToast({ closeButton: v })}>
              <Switch.Thumb className="rx-switch-thumb" />
            </Switch.Root>
          </div>
          <div className="set-row">
            <label htmlFor="toast-close-place">Close position</label>
            <select id="toast-close-place" value={toastPrefs.closePlace} disabled={!toastPrefs.closeButton} onChange={(e) => saveToast({ closePlace: e.target.value })}>
              {TOAST_CLOSE_PLACES.map((p) => <option key={p} value={p}>{p.replaceAll("-", " ")}</option>)}
            </select>
          </div>
          <div className="set-row">
            <label htmlFor="toast-rich">Rich colors</label>
            <Switch.Root id="toast-rich" className="rx-switch" checked={toastPrefs.richColors} onCheckedChange={(v) => saveToast({ richColors: v })}>
              <Switch.Thumb className="rx-switch-thumb" />
            </Switch.Root>
          </div>
        </div>
        <button type="button" className="btn btn-ghost btn-sm" style={{ marginTop: 10 }} onClick={() => toast.ok("Sample notification")}>Preview</button>
      </section>

      <section className="settings-section" hidden={sec !== "server"}>
        <h3 className="sr-only">Server</h3>
        <div className="port-row">
          <input id="port-input" type="text" inputMode="numeric" placeholder="e.g. 8446" autoComplete="off" value={port} onChange={(e) => setPort(e.target.value)} />
          <button id="port-save" className="btn btn-primary btn-sm" onClick={applyPort}>Apply</button>
        </div>
        <p id="port-error" className="form-error" hidden={!err}>{err}</p>
        <p id="port-note" className={"port-note" + (moving ? " moving" : "")}>{note}</p>

        <h4 className="devs-h">Reach this server</h4>
        <p className="settings-desc">Where PiCode listens, and the address other machines should use. The public URL goes into pairing links, <code>server.json</code> and the phone drawer; it does not move the listener.</p>
        <div className="set-rows">
          <div className="set-row reach-row">
            <label htmlFor="bind-select">Bind</label>
            <span className="reach-ctl" data-align-row>
              <select id="bind-select" value={bind} onChange={(e) => setBind(e.target.value)} disabled={!info}>
                <option value="0.0.0.0">All interfaces (0.0.0.0)</option>
                <option value="127.0.0.1">This machine only (127.0.0.1)</option>
                {(info ? info.interfaces : []).map((i) => (
                  <option key={i.ip} value={i.ip}>{i.kind === "tailnet" ? "Tailnet and this machine" : "LAN and this machine"} ({i.ip})</option>
                ))}
                {info && bind && !["0.0.0.0", "127.0.0.1"].includes(bind) && !(info.interfaces || []).some((i) => i.ip === bind) ? <option value={bind}>{bind}</option> : null}
              </select>
              <button type="button" className="btn btn-ghost btn-sm" onClick={applyBind} disabled={!info || bind === (info && info.host)}>Apply</button>
            </span>
          </div>
          <div className="set-row reach-row">
            <label htmlFor="public-url">Public URL</label>
            <span className="reach-ctl" data-align-row>
              <input id="public-url" type="url" placeholder="https://box.tailxxxx.ts.net:8445" autoComplete="off" spellCheck={false} value={pubUrl} onChange={(e) => setPubUrl(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter") applyPublicUrl(); }} />
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => applyPublicUrl()} disabled={!info || pubUrl === (info && info.publicUrl)}>Apply</button>
              {info && info.publicUrl ? <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setPubUrl(""); applyPublicUrl(""); }} title="Links and clients go back to an address of this machine">Clear</button> : null}
            </span>
          </div>
        </div>
        <p className="settings-desc">The public URL is an address, not an access rule. To cut a device off, use <a href="#/devices">Devices → Forget</a>; to hide the server from other machines, bind to this machine only.</p>
        {info && (info.suggestions.magicDns || info.suggestions.tailscaleIp) ? (
          <p className="settings-desc reach-suggest">On the tailnet this box is{" "}
            {info.suggestions.magicDns ? <button type="button" className="chip" onClick={() => { const v = `https://${info.suggestions.magicDns}:${info.current}`; setPubUrl(v); applyPublicUrl(v); }}>{info.suggestions.magicDns}</button> : null}
            {info.suggestions.magicDns && info.suggestions.tailscaleIp ? " or " : ""}
            {info.suggestions.tailscaleIp ? <button type="button" className="chip" onClick={() => { const v = `https://${info.suggestions.tailscaleIp}:${info.current}`; setPubUrl(v); applyPublicUrl(v); }}>{info.suggestions.tailscaleIp}</button> : null}
            . Click one to use it.</p>
        ) : null}
        <p className="form-error" hidden={!reachErr}>{reachErr}</p>
        <AccessSection hidden={hidden || sec !== "server"} />
      </section>

      <BackupSection hidden={hidden || sec !== "backup"} />
    </PageFrame>
  );
}

const INTERVALS = [
  { v: 15, label: "Every 15 minutes" },
  { v: 60, label: "Every hour" },
  { v: 360, label: "Every 6 hours" },
  { v: 1440, label: "Every day" },
];
const KEEP = [3, 7, 10, 30, 90];

function BackupSection({ hidden }) {
  const [cfg, setCfg] = useState({
    dir: "", scheduled: false, intervalMin: 60, keepDays: 10, sessions: true, secrets: true,
    enabled: false, lastOk: "", lastError: "", lastBytes: 0, sameFs: false, destOk: false,
  });
  const [snaps, setSnaps] = useState([]);
  const [busy, setBusy] = useState(false);
  const [job, setJob] = useState(null);

  async function load() {
    try {
      const s = await api("/api/backup");
      setCfg(s);
      if (s.dir) {
        const d = await api("/api/backup/snapshots");
        setSnaps(d.snapshots || []);
      } else setSnaps([]);
    } catch (e) { toastError(e); }
  }

  useEffect(() => { if (!hidden) load(); }, [hidden]);

  async function save(patch) {
    const next = { ...cfg, ...patch };
    try {
      const s = await api("/api/backup", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          dir: next.dir, scheduled: next.scheduled, intervalMin: next.intervalMin, keepDays: next.keepDays,
          sessions: next.sessions, secrets: next.secrets,
        }),
      });
      setCfg(s);
      if (s.dir) {
        const d = await api("/api/backup/snapshots");
        setSnaps(d.snapshots || []);
      }
    } catch (e) { toastError(e); }
  }

  async function runNow() {
    if (!cfg.dir || job) return;
    const steps = backupSteps(cfg);
    setJob({ kind: "backup", steps, dest: cfg.dir, step: 0, error: "", done: false });
    setBusy(true);
    const tick = startJobTick(setJob);
    try {
      await api("/api/backup/now", { method: "POST" });
      setJob((j) => j && { ...j, step: steps.length, done: true });
      await load();
      setTimeout(() => setJob(null), 520);
    } catch (e) {
      setJob((j) => j && { ...j, error: e.message || String(e) });
    } finally {
      clearInterval(tick);
      setBusy(false);
    }
  }

  async function reveal(id) {
    try {
      await api("/api/backup/reveal", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(id ? { id } : { root: true }),
      });
    } catch (e) { toastError(e); }
  }

  async function removeSnap(id) {
    const ok = await askConfirm({
      title: "Remove snapshot",
      message: "Delete this backup from disk? This cannot be undone.",
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    try {
      await api("/api/backup/snapshots/" + encodeURIComponent(id), { method: "DELETE" });
      await load();
    } catch (e) { toastError(e); }
  }

  async function restore(id) {
    const ok = await askConfirm({
      title: "Restore backup",
      message: "Replace this machine's PiCode data with snapshot " + id + "? Agents stop first. Project folders are not touched.",
      confirmLabel: "Restore",
      danger: true,
    });
    if (!ok || job) return;
    const steps = restoreSteps();
    setJob({ kind: "restore", steps, dest: id, step: 0, error: "", done: false });
    setBusy(true);
    const tick = startJobTick(setJob);
    try {
      await api("/api/backup/restore", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id }),
      });
      setJob((j) => j && { ...j, step: steps.length, done: true });
    } catch (e) {
      setJob((j) => j && { ...j, error: e.message || String(e) });
    } finally {
      clearInterval(tick);
      setBusy(false);
    }
  }

  const status = cfg.lastError
    ? cfg.lastError
    : cfg.lastOk
      ? "Last backup " + cfg.lastOk.replace("T", " ").replace("Z", " UTC") + (cfg.lastBytes ? " · " + fmtBytes(cfg.lastBytes) : "")
      : "No backup yet.";

  return (
    <section className="settings-section" hidden={hidden}>
      <div className="set-rows">
        <div className="set-row set-row-stack">
          <label>Folder</label>
          <div className="folder-field-wrap">
            <FolderField placeholder="External drive or other folder" value={cfg.dir || ""} onChange={(dir) => save({ dir })} />
            <button type="button" className="btn btn-ghost btn-sm" disabled={!cfg.dir} onClick={() => reveal("")} title="Show in file manager">Reveal</button>
          </div>
        </div>
        <div className="set-row">
          <label htmlFor="bak-on">Schedule</label>
          <Switch.Root id="bak-on" className="rx-switch" checked={!!cfg.scheduled} disabled={!cfg.dir} onCheckedChange={(v) => save({ scheduled: v })}>
            <Switch.Thumb className="rx-switch-thumb" />
          </Switch.Root>
        </div>
        <div className="set-row">
          <label htmlFor="bak-int">Interval</label>
          <select id="bak-int" value={cfg.intervalMin} onChange={(e) => save({ intervalMin: Number(e.target.value) })} disabled={!cfg.dir}>
            {INTERVALS.map((o) => <option key={o.v} value={o.v}>{o.label}</option>)}
          </select>
        </div>
        <div className="set-row">
          <label htmlFor="bak-keep">Keep</label>
          <select id="bak-keep" value={cfg.keepDays} onChange={(e) => save({ keepDays: Number(e.target.value) })} disabled={!cfg.dir}>
            {KEEP.map((d) => <option key={d} value={d}>{d} days</option>)}
          </select>
        </div>
        <div className="set-row">
          <label htmlFor="bak-sess">Include sessions</label>
          <Switch.Root id="bak-sess" className="rx-switch" checked={!!cfg.sessions} disabled={!cfg.dir} onCheckedChange={(v) => save({ sessions: v })}>
            <Switch.Thumb className="rx-switch-thumb" />
          </Switch.Root>
        </div>
        <div className="set-row">
          <label htmlFor="bak-sec">Include secrets</label>
          <Switch.Root id="bak-sec" className="rx-switch" checked={!!cfg.secrets} disabled={!cfg.dir} onCheckedChange={(v) => save({ secrets: v })}>
            <Switch.Thumb className="rx-switch-thumb" />
          </Switch.Root>
        </div>
      </div>
      {cfg.sameFs ? <p className="port-note">This folder is on the same disk as PiCode. It will not survive a dead drive.</p> : null}
      {!snaps.length ? <p className="port-note">{status}</p> : null}
      <div className="port-row" style={{ marginTop: 10 }}>
        <button type="button" className="btn btn-primary btn-sm" disabled={!cfg.dir || busy} onClick={runNow}>Backup now</button>
      </div>
      {snaps.length ? (
        <ul className="bak-snaps">
          {snaps.map((s) => (
            <li key={s.id} className="bak-snap">
              <span>{s.created.replace("T", " ").replace("Z", " UTC")} · {fmtBytes(s.bytes)}</span>
              <div className="bak-snap-actions">
                <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={() => restore(s.id)}>Restore</button>
                <DropdownMenu.Root>
                  <DropdownMenu.Trigger asChild>
                    <button type="button" className="ws-icon-btn" aria-label="Snapshot actions"><IconMore size={14} /></button>
                  </DropdownMenu.Trigger>
                  <DropdownMenu.Portal>
                    <DropdownMenu.Content className="um-popover" side="bottom" align="end" sideOffset={4}>
                      <DropdownMenu.Item className="um-item" onSelect={() => reveal(s.id)}>Reveal</DropdownMenu.Item>
                      <DropdownMenu.Item className="um-item um-danger" onSelect={() => removeSnap(s.id)}>Remove</DropdownMenu.Item>
                    </DropdownMenu.Content>
                  </DropdownMenu.Portal>
                </DropdownMenu.Root>
              </div>
            </li>
          ))}
        </ul>
      ) : null}
      {job ? <BackupJob job={job} onClose={() => setJob(null)} /> : null}
    </section>
  );
}

function startJobTick(setJob) {
  return setInterval(() => {
    setJob((j) => {
      if (!j || j.error || j.done) return j;
      if (j.step < j.steps.length - 1) return { ...j, step: j.step + 1 };
      return j;
    });
  }, 480);
}

function backupSteps(cfg) {
  return [
    { id: "db", label: "VACUUM INTO picode.db" },
    { id: "pins", label: "Copy pins" + (cfg.secrets ? " and secrets" : "") },
    { id: "sess", label: cfg.sessions ? "Copy pi sessions" : "Skip sessions" },
    { id: "manifest", label: "Write manifest" },
  ];
}

function restoreSteps() {
  return [
    { id: "stop", label: "Stop running agents" },
    { id: "db", label: "Replace picode.db" },
    { id: "pins", label: "Restore pins" },
    { id: "rest", label: "Restore secrets and sessions if present" },
  ];
}

function BackupJob({ job, onClose }) {
  const steps = job.steps || [];
  const restore = job.kind === "restore";
  const title = restore ? "Restoring" : "Backing up";
  return (
    <div className="pkg-job" role="alertdialog" aria-modal="true" aria-labelledby="bak-job-title">
      <div className="pkg-job-card">
        <h3 id="bak-job-title">{title}</h3>
        <p className="pkg-job-src">{job.dest}</p>
        <ol className="pkg-job-steps">
          {steps.map((s, i) => {
            let st = "todo";
            if (job.error && i === job.step) st = "err";
            else if (i < job.step) st = "done";
            else if (i === job.step) st = "run";
            return (
              <li key={s.id} className={"pkg-job-step " + st}>
                <span className="pkg-job-mark" aria-hidden="true">
                  {st === "run" ? <PiSpinner title="Working" /> : st === "done" ? "✓" : st === "err" ? "!" : "○"}
                </span>
                <code>{s.label}</code>
              </li>
            );
          })}
        </ol>
        {job.error ? (
          <>
            <p className="pkg-job-err">{job.error}</p>
            <button type="button" className="btn btn-primary btn-sm" onClick={onClose}>Close</button>
          </>
        ) : job.done && restore ? (
          <>
            <p className="pkg-fine">Restored. Reload to use this snapshot.</p>
            <button type="button" className="btn btn-primary btn-sm" onClick={() => location.reload()}>Reload</button>
          </>
        ) : (
          <p className="pkg-fine">{restore ? "Agents stop first. Project folders stay." : "Stays here until the snapshot is on disk."}</p>
        )}
      </div>
    </div>
  );
}


