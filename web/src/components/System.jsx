import { useEffect, useState } from "react";
import { api } from "../lib/api.js";
import PageFrame from "./PageFrame.jsx";

export default function System({ hidden, version, system: systemProp }) {
  const [fetched, setFetched] = useState(null);
  const [ver, setVer] = useState(version || "");
  useEffect(() => {
    if (hidden) return;
    if (!systemProp) api("/api/system").then(setFetched).catch(() => {});
    if (!version) api("/api/version").then((v) => setVer(v.version || v || "")).catch(() => {});
  }, [hidden, systemProp, version]);
  const system = systemProp || fetched;
  return (
    <PageFrame id="system-view" title="System" hidden={hidden}>
      <section className="settings-section">
        <h3>Host</h3>
        <dl className="sys-rows" id="system-host">
          {hostRows(system).map(([k, v]) => (
            <div className="sys-row" key={"h" + k}><dt>{k}</dt><dd>{v}</dd></div>
          ))}
        </dl>
      </section>

      <section className="settings-section">
        <h3>Network</h3>
        <dl className="sys-rows" id="system-net">
          {netRows(system).map(([k, v]) => (
            <div className="sys-row" key={"n" + k}><dt>{k}</dt><dd>{v}</dd></div>
          ))}
        </dl>
      </section>

      <section className="settings-section">
        <h3>Dependencies</h3>
        <dl className="sys-rows" id="system-deps">
          {depRows(system).map(([k, v]) => (
            <div className="sys-row" key={"d" + k + v}><dt>{k}</dt><dd>{v}</dd></div>
          ))}
        </dl>
      </section>

      <section className="settings-section">
        <h3>About</h3>
        <dl className="sys-rows">
          <div className="sys-row"><dt>Version</dt><dd id="about-ver">{ver ? "v" + ver : "—"}</dd></div>
          <div className="sys-row"><dt>Repository</dt><dd><a className="settings-link" href="https://github.com/cfpperche/picode" target="_blank" rel="noopener noreferrer">cfpperche/picode ↗</a></dd></div>
          <div className="sys-row"><dt>License</dt><dd>MIT</dd></div>
        </dl>
      </section>
    </PageFrame>
  );
}

function hostRows(system) {
  if (!system || !system.host) return [["Status", "unavailable"]];
  const h = system.host;
  let os = h.os || "—";
  if (h.arch) os += " · " + h.arch;
  if (h.wsl) os += " (WSL)";
  return [
    ["Name", h.name || "—"],
    ["OS", os],
  ];
}

function netRows(system) {
  if (!system || !system.network) return [["Status", "unavailable"]];
  const n = system.network;
  const bind = n.port ? n.bind + ":" + n.port : (n.bind || "—");
  return [
    ["Bind", bind],
    ["HTTPS", n.https ? "on" : "off"],
    ["LAN", (n.lan && n.lan.length) ? n.lan.join(", ") : "—"],
    ["Tailscale", n.tailscale || "—"],
  ];
}

function depRows(system) {
  if (!system) return [["Status", "unavailable"]];
  const rows = [
    ["tmux", system.tmux && system.tmux.installed ? (system.tmux.version || "installed") : "not installed"],
    ["pi", system.pi && system.pi.installed ? (system.pi.version || "installed") : "not installed"],
    ["mkcert", system.mkcert && system.mkcert.installed ? "installed" : "not installed · optional"],
    ["tailscale", tailscaleValue(system.tailscale)],
  ];
  if (system.warnings) {
    for (const w of system.warnings) rows.push(["note", w]);
  }
  return rows;
}

function tailscaleValue(ts) {
  if (!ts || !ts.installed) return "not installed · optional";
  return ts.ip || "installed";
}
