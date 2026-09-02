import { useEffect, useState } from "react";
import { api } from "../lib/api.js";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";
import { subscribeFeed } from "../lib/feed.js";
import { touches } from "../lib/feedReducers.js";

const MODES = [
  { id: "remote", label: "This machine is trusted", hint: "Browsers on this machine pair on their own; every other device needs a pairing link." },
  { id: "all", label: "Every device pairs", hint: "For a shared or public server: even a browser on this machine must pair." },
  { id: "off", label: "Off", hint: "No pairing at all. Only behind a proxy you trust, or while developing." },
];

// Preferences → Server → access rules (ADR-0049): who must pair, and the
// install token scripts read. The devices themselves live on #/devices.
export default function AccessSection({ hidden }) {
  const [me, setMe] = useState(null);
  const [tokenPath, setTokenPath] = useState("");
  async function load() {
    try {
      const [s, list] = await Promise.all([api("/api/auth/session"), api("/api/auth/sessions")]);
      setMe(s);
      setTokenPath(list.tokenPath || "");
    } catch (e) { toastError(e); }
  }
  useEffect(() => { if (!hidden) load(); }, [hidden]);
  useEffect(() => subscribeFeed((ev) => { if (touches(ev, ["setting"])) load(); }), []);
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
      <h4 className="devs-h">Who must pair</h4>
      <div className="devs-modes">
        {MODES.map((m) => (
          <label key={m.id} className="devs-mode">
            <input type="radio" name="auth-mode" value={m.id} checked={mode === m.id} onChange={() => setMode(m.id)} />
            <span><strong>{m.label}</strong><br /><span className="devs-hint">{m.hint}</span></span>
          </label>
        ))}
      </div>
      <p className="settings-desc" style={{ marginTop: 10 }}>Paired devices are listed under <a href="#/devices">Devices</a>.</p>
      <h4 className="devs-h">Scripts and the Chrome extension</h4>
      <div className="devs-actions" data-align-row>
        <code className="auto-code">{tokenPath || "…"}</code>
        <button type="button" className="btn btn-ghost" onClick={rotate}>Rotate</button>
      </div>
    </div>
  );
}
