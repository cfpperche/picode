import { useEffect, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import { api } from "../lib/api.js";
import { IconSun, IconMonitor, IconMoon } from "./Icons.jsx";
import PageFrame from "./PageFrame.jsx";
import { toast, toastError } from "../lib/toast.js";
import { readToastPrefs, persistToastPrefs, TOAST_POSITIONS, TOAST_CLOSE_PLACES } from "../lib/toastPrefs.js";
import FolderField from "./FolderField.jsx";
import { askConfirm, fmtBytes } from "../lib/confirm.js";

export default function Settings({ hidden, themeMode, onTheme }) {
  const [port, setPort] = useState("");
  const [note, setNote] = useState("");
  const [moving, setMoving] = useState(false);
  const [err, setErr] = useState("");
  const [toastPrefs, setToastPrefs] = useState(readToastPrefs);

  useEffect(() => {
    if (hidden) return;
    api("/api/server").then((info) => {
      setPort(String(info.current));
      setNote(`Serving on port ${info.current} (configured: ${info.configured}).`);
      setMoving(false);
    }).catch(() => setNote("Server state unavailable."));
  }, [hidden]);

  function saveToast(patch) {
    setToastPrefs(persistToastPrefs({ ...toastPrefs, ...patch }));
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

  return (
    <PageFrame id="preferences-view" title="Preferences" hidden={hidden}>
      <section className="settings-section">
        <h3>Appearance</h3>
        <div className="theme-cards" role="radiogroup" aria-label="Theme">
          <ThemeCard option="light" label="Light" desc="Bright surfaces" active={themeMode === "light"} onPick={onTheme} icon={<IconSun size={15} />} />
          <ThemeCard option="system" label="System" desc="Match your OS" active={themeMode === "system"} onPick={onTheme} icon={<IconMonitor size={15} />} />
          <ThemeCard option="dark" label="Dark" desc="Low light" active={themeMode === "dark"} onPick={onTheme} icon={<IconMoon size={15} />} />
        </div>
      </section>

      <section className="settings-section">
        <h3>Notifications</h3>
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

      <section className="settings-section">
        <h3>Server</h3>
        <div className="port-row">
          <input id="port-input" type="text" inputMode="numeric" placeholder="e.g. 8446" autoComplete="off" value={port} onChange={(e) => setPort(e.target.value)} />
          <button id="port-save" className="btn btn-primary btn-sm" onClick={applyPort}>Apply</button>
        </div>
        <p id="port-error" className="form-error" hidden={!err}>{err}</p>
        <p id="port-note" className={"port-note" + (moving ? " moving" : "")}>{note}</p>
      </section>

      <BackupSection hidden={hidden} />
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
    dir: "", intervalMin: 60, keepDays: 10, sessions: true, secrets: true,
    enabled: false, lastOk: "", lastError: "", lastBytes: 0, sameFs: false, destOk: false,
  });
  const [snaps, setSnaps] = useState([]);
  const [busy, setBusy] = useState(false);

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
          dir: next.dir, intervalMin: next.intervalMin, keepDays: next.keepDays,
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
    setBusy(true);
    try {
      await api("/api/backup/now", { method: "POST" });
      toast.ok("Backup saved.");
      await load();
    } catch (e) { toastError(e); }
    finally { setBusy(false); }
  }

  async function restore(id) {
    const ok = await askConfirm({
      title: "Restore backup",
      message: "Replace this machine's PiCode data with snapshot " + id + "? Agents stop first. Project folders are not touched.",
      confirmLabel: "Restore",
      danger: true,
    });
    if (!ok) return;
    setBusy(true);
    try {
      await api("/api/backup/restore", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id }),
      });
      toast.ok("Restored. Reload the page.");
    } catch (e) { toastError(e); }
    finally { setBusy(false); }
  }

  const status = cfg.lastError
    ? cfg.lastError
    : cfg.lastOk
      ? "Last backup " + cfg.lastOk.replace("T", " ").replace("Z", " UTC") + (cfg.lastBytes ? " · " + fmtBytes(cfg.lastBytes) : "")
      : "No backup yet.";

  return (
    <section className="settings-section">
      <h3>Backup</h3>
      <p className="set-hint">Local snapshots of PiCode and pi sessions. Project folders stay out.</p>
      <div className="set-rows">
        <div className="set-row set-row-stack">
          <label>Folder</label>
          <FolderField placeholder="External drive or other folder" value={cfg.dir || ""} onChange={(dir) => save({ dir })} />
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
      <p className="port-note">{status}</p>
      <div className="port-row" style={{ marginTop: 10 }}>
        <button type="button" className="btn btn-primary btn-sm" disabled={!cfg.dir || busy} onClick={runNow}>Backup now</button>
      </div>
      {snaps.length ? (
        <ul className="bak-snaps">
          {snaps.map((s) => (
            <li key={s.id} className="bak-snap">
              <span>{s.created.replace("T", " ").replace("Z", " UTC")} · {fmtBytes(s.bytes)}</span>
              <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={() => restore(s.id)}>Restore</button>
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}

function ThemeCard({ option, label, desc, active, onPick, icon }) {
  return (
    <button
      type="button"
      className="theme-card"
      data-theme-option={option}
      role="radio"
      aria-checked={active}
      onClick={() => onPick(option)}
    >
      {icon}
      <span className="theme-card-name">{label}</span>
      <span className="theme-card-desc">{desc}</span>
    </button>
  );
}
