import { useEffect, useRef, useState } from "react";
import QRCode from "qrcode";
import { api } from "../lib/api.js";
import { feedConnected, subscribeFeed } from "../lib/feed.js";
import { touches } from "../lib/feedReducers.js";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";
import { relTime, absTime } from "../lib/relTime.js";
import PageFrame from "./PageFrame.jsx";

// Devices (ADR-0043 presence + ADR-0049 identity, one surface): who may
// enter, and whether they are here right now. Identity is the paired
// session; liveness is the presence ping that session sends. Access
// rules and the install token live in Preferences → Server.
export default function Devices({ hidden }) {
  const [rows, setRows] = useState(null);
  const [live, setLive] = useState([]);
  const [pairing, setPairing] = useState(null); // {url, qrUrl}
  const canvasRef = useRef(null);

  async function load() {
    try {
      const [s, d] = await Promise.all([api("/api/auth/sessions"), api("/api/devices").catch(() => [])]);
      setRows(s.items || []);
      setLive(Array.isArray(d) ? d : []);
    } catch (e) { toastError(e); }
  }
  useEffect(() => {
    if (hidden) return;
    load();
    const t = setInterval(() => { if (!feedConnected()) load(); }, 4000);
    const unsub = subscribeFeed((ev) => { if (ev.type === "feed.open" || touches(ev, ["device", "session", "pairing"])) load(); });
    return () => { clearInterval(t); unsub(); };
  }, [hidden]);
  useEffect(() => {
    if (!pairing || !canvasRef.current) return;
    QRCode.toCanvas(canvasRef.current, pairing.qrUrl || pairing.url, { width: 180, margin: 1, color: { dark: "#16181d", light: "#ffffff" } });
  }, [pairing]);

  async function pair() {
    try { setPairing(await api("/api/auth/pairings", { method: "POST" })); } catch (e) { toastError(e); }
  }
  async function forget(row) {
    const ok = await askConfirm({
      title: "Forget " + (row.label || row.id) + "?",
      message: row.current ? "This is the device you are using — you will need to pair it again." : "That device will need a new pairing link to get back in.",
      confirmLabel: "Forget", danger: true,
    });
    if (!ok) return;
    try {
      await api("/api/auth/sessions/" + encodeURIComponent(row.id), { method: "DELETE" });
      if (row.current) location.reload(); else load();
    } catch (e) { toastError(e); }
  }

  const unpaired = live.filter((d) => !d.session && d.online);

  return (
    <PageFrame id="devices-view" title="Devices" hidden={hidden}>
      {rows === null ? (
        <div className="mcp-skel" aria-hidden="true"><span className="skel-line w-70" /><span className="skel-line w-40" /></div>
      ) : rows.length === 0 ? (
        <div className="mcp-empty">
          <p>No paired devices yet.</p>
          <button type="button" className="btn btn-primary" onClick={pair}>Pair a device</button>
        </div>
      ) : (
        <>
          <ul className="dev-list">
            {rows.map((r) => (
              <li key={r.id} className={"dev-row" + (r.online ? "" : " off")} data-align-row>
                <span className="dev-dot-cell"><span className={"share-dot" + (r.online ? "" : " off")} /></span>
                <span className="dev-name">
                  {r.label || r.id}
                  {r.current ? <span className="devs-tag">this device</span> : null}
                  {r.kind === "token" ? <span className="devs-tag">token</span> : null}
                  {r.pingKind === "extension" ? <span className="devs-tag">extension</span> : null}
                </span>
                <span className="dev-ip">{r.ip}</span>
                <span className="dev-seen" title={absTime(r.pingSeen || r.lastSeenAt)}>{r.online ? "online" : "seen " + relTime(r.pingSeen || r.lastSeenAt)}</span>
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => forget(r)}>Forget</button>
              </li>
            ))}
          </ul>
          <div className="devs-actions" data-align-row>
            <button type="button" className="btn btn-primary" onClick={pair}>Pair a device</button>
          </div>
        </>
      )}
      {pairing ? (
        <div className="devs-pairing" role="status">
          <canvas ref={canvasRef} />
          <div className="devs-pairing-text">
            <p>Scan with the phone's camera{pairing.qrUrl && pairing.qrUrl !== pairing.url ? " (it installs the certificate first, then pairs)" : ""}, or open this link on the other device. It works once and expires in ten minutes.</p>
            <code className="auto-code">{pairing.url}</code>
            <div className="devs-actions" data-align-row>
              <button type="button" className="btn btn-ghost" onClick={async () => { try { await navigator.clipboard.writeText(pairing.url); toast.ok("Copied."); } catch { /* ignore */ } }}>Copy link</button>
              <button type="button" className="btn btn-ghost" onClick={() => setPairing(null)}>Done</button>
            </div>
          </div>
        </div>
      ) : null}
      {unpaired.length ? (
        <section className="settings-section">
          <h3>Online without pairing</h3>
          <p className="settings-desc">Only possible while pairing is off. Turn it on in Preferences → Server.</p>
          <ul className="dev-list">
            {unpaired.map((d) => (
              <li key={d.id} className="dev-row">
                <span className="share-dot" />
                <span className="dev-name">{d.name}</span>
                <span className="dev-ip">{d.ip}</span>
                <span className="dev-seen">online</span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </PageFrame>
  );
}
