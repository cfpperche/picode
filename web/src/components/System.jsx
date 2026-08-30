import { useEffect, useState } from "react";
import { api } from "../lib/api.js";
import { toast } from "../lib/toast.js";
import PiSpinner from "./PiSpinner.jsx";
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

      {system && system.pi && system.pi.updateAvailable ? <PiUpdateCard pi={system.pi} /> : null}

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

function PiUpdateCard({ pi }) {
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState("");
  const [err, setErr] = useState("");
  async function run() {
    setBusy(true);
    setErr("");
    try {
      const res = await api("/api/system/pi-update", { method: "POST" });
      setDone((res && res.version) || "updated");
      toast.ok("pi updated" + ((res && res.version) ? " to " + res.version : "") + " — restart agents to pick it up.");
    } catch (e) {
      setErr((e && e.message) || "Update failed.");
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="settings-section pi-update-card">
      <h3>Pi update</h3>
      <p className="pi-update-line" data-align-row>
        <span>
          <strong>{pi.version || "installed"}</strong> → <strong>{pi.latest}</strong> available
        </span>
        <span className="pi-update-actions" data-align-row>
          <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={() => { navigator.clipboard.writeText("pi update --self"); toast.ok("Command copied."); }} title="Copy the terminal command">Copy command</button>
          <button type="button" className="btn btn-primary btn-sm" disabled={busy} onClick={run} title="Runs pi update --self">
            {busy ? <><PiSpinner title="Updating pi" /> Updating…</> : "Update now"}
          </button>
        </span>
      </p>
      {busy ? <p className="pi-update-note">This runs <code>pi update --self</code> and can take a minute.</p> : null}
      {done ? <p className="pi-update-note ok">Now at <strong>{done}</strong>. Running agents keep the old version until you restart them.</p> : null}
      {err ? <p className="pi-update-note err" title={err}>{err.slice(0, 300)}</p> : null}
      {!done && !busy && !err ? <p className="pi-update-note">Running agents keep the old version until you restart them.</p> : null}
    </section>
  );
}
function tailscaleValue(ts) {
  if (!ts || !ts.installed) return "not installed · optional";
  return ts.ip || "installed";
}
