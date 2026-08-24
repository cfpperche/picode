import { useEffect, useState } from "react";
import { api } from "../lib/api.js";
import { IconSun, IconMonitor, IconMoon } from "./Icons.jsx";
import PageFrame from "./PageFrame.jsx";

export default function Settings({ hidden, version, themeMode, onTheme, system }) {
  const [port, setPort] = useState("");
  const [note, setNote] = useState("");
  const [moving, setMoving] = useState(false);
  const [err, setErr] = useState("");

  useEffect(() => {
    if (hidden) return;
    api("/api/server").then((info) => {
      setPort(String(info.current));
      setNote(`Serving on port ${info.current} (configured: ${info.configured}).`);
      setMoving(false);
    }).catch(() => setNote("Server state unavailable."));
  }, [hidden]);

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
        setTimeout(() => { location.replace(`${scheme}://${host}:${res.port}/#/settings`); }, 1500);
      }
    } catch (e) {
      setErr(e.message);
    }
  }

  const rows = [];
  if (!system) {
    rows.push(["Status", "unavailable"]);
  } else {
    rows.push(["tmux", system.tmux.installed ? (system.tmux.version || "installed") : "not installed"]);
    rows.push(["pi", system.pi.installed ? (system.pi.version || "installed") : "not installed"]);
    rows.push(["mkcert", optDep(system.mkcert && system.mkcert.installed)]);
    rows.push(["tailscale", tailscaleValue(system.tailscale)]);

    if (system.warnings) {
      for (const w of system.warnings) rows.push(["note", w]);
    }
  }

  return (
    <PageFrame id="settings-view" title="Settings" hidden={hidden}>
      <section className="settings-section">
        <h3>Appearance</h3>
        <div className="theme-cards" role="radiogroup" aria-label="Theme">
          <ThemeCard option="light" label="Light" desc="Bright surfaces" active={themeMode === "light"} onPick={onTheme} icon={<IconSun size={15} />} />
          <ThemeCard option="system" label="System" desc="Match your OS" active={themeMode === "system"} onPick={onTheme} icon={<IconMonitor size={15} />} />
          <ThemeCard option="dark" label="Dark" desc="Low light" active={themeMode === "dark"} onPick={onTheme} icon={<IconMoon size={15} />} />
        </div>
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

      <section className="settings-section">
        <h3>System</h3>
        <dl className="sys-rows" id="settings-sys">
          {rows.map(([k, v]) => (
            <div className="sys-row" key={k + v}>
              <dt>{k}</dt>
              <dd>{v}</dd>
            </div>
          ))}
        </dl>
      </section>

      <section className="settings-section">
        <h3>About</h3>
        <dl className="sys-rows">
          <div className="sys-row"><dt>Version</dt><dd id="about-ver">{version ? "v" + version : "—"}</dd></div>
          <div className="sys-row"><dt>Repository</dt><dd><a className="settings-link" href="https://github.com/cfpperche/picode" target="_blank" rel="noopener noreferrer">cfpperche/picode ↗</a></dd></div>
          <div className="sys-row"><dt>License</dt><dd>MIT</dd></div>
        </dl>
      </section>
    </PageFrame>
  );
}

function optDep(ok) {
  return ok ? "installed" : "not installed · optional";
}

function tailscaleValue(ts) {
  if (!ts || !ts.installed) return "not installed · optional";
  return ts.ip || "installed";
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
