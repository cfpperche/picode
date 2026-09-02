import { useEffect, useRef, useState } from "react";
import QRCode from "qrcode";
import { api } from "../lib/api.js";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";
import { relTime, absTime } from "../lib/relTime.js";
import { subscribeFeed } from "../lib/feed.js";
import { touches } from "../lib/feedReducers.js";

const MODES = [
  { id: "remote", label: "This machine is trusted", hint: "Browsers on this machine pair on their own; every other device needs a pairing link." },
  { id: "all", label: "Every device pairs", hint: "For a shared or public server: even a browser on this machine must pair." },
  { id: "off", label: "Off", hint: "No pairing at all. Only behind a proxy you trust, or while developing." },
];

// Preferences → Server → Devices (ADR-0049): who is paired, pairing a new
// device, the mode, and the install token for scripts.
export default function DevicesSection({ hidden }) {
  const [me, setMe] = useState(null);
  const [rows, setRows] = useState(null);
  const [tokenPath, setTokenPath] = useState("");
  const [pairing, setPairing] = useState(null); // {url, expiresAt}
  const canvasRef = useRef(null);

  async function load() {
    try {
      const [s, list] = await Promise.all([api("/api/auth/session"), api("/api/auth/sessions")]);
      setMe(s);
      setRows(list.items || []);
      setTokenPath(list.tokenPath || "");
    } catch (e) { toastError(e); }
  }
  useEffect(() => { if (!hidden) load(); }, [hidden]);
  useEffect(() => subscribeFeed((ev) => { if (touches(ev, ["session", "pairing", "setting"])) load(); }), []);
  useEffect(() => {
    if (!pairing || !canvasRef.current) return;
    QRCode.toCanvas(canvasRef.current, pairing.url, { width: 180, margin: 1, color: { dark: "#16181d", light: "#ffffff" } });
  }, [pairing]);

  async function pair() {
    try { setPairing(await api("/api/auth/pairings", { method: "POST" })); } catch (e) { toastError(e); }
  }
  async function revoke(row) {
    const ok = await askConfirm({ title: "Forget " + (row.label || row.id) + "?", message: row.current ? "This is the device you are using — you will need to pair it again." : "That device will need a new pairing link.", confirmLabel: "Forget", danger: true });
    if (!ok) return;
    try { await api("/api/auth/sessions/" + encodeURIComponent(row.id), { method: "DELETE" }); if (row.current) location.reload(); else load(); } catch (e) { toastError(e); }
  }
  async function setMode(mode) {
    try { await api("/api/auth/mode", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ mode }) }); toast.ok("Saved."); load(); } catch (e) { toastError(e); }
  }
  async function rotate() {
    const ok = await askConfirm({ title: "Rotate the install token?", message: "Scripts and the Chrome extension read it from the file, so they keep working; anything that copied the old value stops.", confirmLabel: "Rotate" });
    if (!ok) return;
    try { await api("/api/auth/token/rotate", { method: "POST" }); toast.ok("Token rotated."); } catch (e) { toastError(e); }
  }

  const mode = me ? me.mode : "remote";
  return (
    <div className="devs">
      <h4 className="devs-h">Devices</h4>
      {rows === null ? <div className="mcp-skel"><span className="skel-line w-70" /><span className="skel-line w-40" /></div> : rows.length === 0 ? (
        <div className="mcp-empty"><p>No paired devices yet.</p><button type="button" className="btn btn-primary" onClick={pair}>Pair a device</button></div>
      ) : (
        <>
          <ul className="devs-list">
            {rows.map((r) => (
              <li key={r.id} className="devs-row" data-align-row>
                <span className="devs-label">{r.label || r.id}{r.current ? <span className="devs-tag">this device</span> : null}{r.kind === "token" ? <span className="devs-tag">token</span> : null}</span>
                <span className="devs-when" title={absTime(r.lastSeenAt)}>{r.ip ? r.ip + " · " : ""}seen {relTime(r.lastSeenAt)}</span>
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => revoke(r)}>Forget</button>
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
            <p>Scan with the phone, or open this link on the other device. It works once and expires in ten minutes.</p>
            <code className="auto-code">{pairing.url}</code>
            <div className="devs-actions" data-align-row>
              <button type="button" className="btn btn-ghost" onClick={async () => { try { await navigator.clipboard.writeText(pairing.url); toast.ok("Copied."); } catch { /* ignore */ } }}>Copy link</button>
              <button type="button" className="btn btn-ghost" onClick={() => setPairing(null)}>Done</button>
            </div>
          </div>
        </div>
      ) : null}

      <h4 className="devs-h">Who must pair</h4>
      <div className="devs-modes">
        {MODES.map((m) => (
          <label key={m.id} className="devs-mode">
            <input type="radio" name="auth-mode" value={m.id} checked={mode === m.id} onChange={() => setMode(m.id)} />
            <span><strong>{m.label}</strong><br /><span className="devs-hint">{m.hint}</span></span>
          </label>
        ))}
      </div>

      <h4 className="devs-h">Scripts and the Chrome extension</h4>
      <div className="devs-actions" data-align-row>
        <code className="auto-code">{tokenPath || "…"}</code>
        <button type="button" className="btn btn-ghost" onClick={rotate}>Rotate</button>
      </div>
    </div>
  );
}
