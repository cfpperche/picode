import { useEffect, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import { api } from "../lib/api.js";
import { IconSun, IconMonitor, IconMoon } from "./Icons.jsx";
import PageFrame from "./PageFrame.jsx";
import { toast } from "../lib/toast.js";
import { readToastPrefs, persistToastPrefs, TOAST_POSITIONS } from "../lib/toastPrefs.js";

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
    </PageFrame>
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
