import { useEffect, useState } from "react";
import { AUTH_REQUIRED_EVENT } from "../lib/api.js";

// Shown once the API answered 401 "pairing required" (ADR-0049): this
// browser is not a paired device. Nothing else renders behind it; the
// next click is either the link the user was sent, or a link pasted here.
export default function PairingScreen() {
  const [needed, setNeeded] = useState(false);
  const [login, setLogin] = useState(""); // the gateway's sign-in page (ADR-0052), when that is what 401 asked for
  const [link, setLink] = useState("");
  useEffect(() => {
    const on = (e) => { setNeeded(true); if (e && e.detail && e.detail.login) setLogin(e.detail.login); };
    window.addEventListener(AUTH_REQUIRED_EVENT, on);
    return () => window.removeEventListener(AUTH_REQUIRED_EVENT, on);
  }, []);
  if (!needed) return null;
  if (login) {
    const next = location.pathname + location.search + location.hash;
    return (
      <div className="pair-root" role="dialog" aria-label="Sign in">
        <div className="pair-card">
          <h1>Sign in</h1>
          <p>This PiCode is behind a gateway. Sign in with the account its admin added; your device pairs after that.</p>
          <a className="btn btn-primary" href={login + "?next=" + encodeURIComponent(next)}>Sign in</a>
        </div>
      </div>
    );
  }
  function go(e) {
    e.preventDefault();
    const raw = link.trim();
    if (!raw) return;
    try {
      const u = new URL(raw, location.origin);
      const code = u.searchParams.get("code") || (u.pathname === "" ? raw : "");
      if (code) { location.assign("/pair?code=" + encodeURIComponent(code)); return; }
    } catch { /* fallthrough */ }
    location.assign("/pair?code=" + encodeURIComponent(raw));
  }
  return (
    <div className="pair-root" role="dialog" aria-label="Pair this device">
      <div className="pair-card">
        <h1>Pair this device</h1>
        <p>This browser is not paired with this PiCode yet. On a device that is, open <strong>Devices</strong> in the user menu and choose <strong>Pair a device</strong>, then open the link or scan the code here.</p>
        <form onSubmit={go} className="pair-form" noValidate>
          <input className="dlg-input" value={link} onChange={(e) => setLink(e.target.value)} placeholder="Paste a pairing link or code" autoFocus />
          <button type="submit" className="btn btn-primary">Pair</button>
        </form>
        <p className="pair-muted">On the machine that runs PiCode, <code>picode pair</code> prints a link too.</p>
      </div>
    </div>
  );
}
