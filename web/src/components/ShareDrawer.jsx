import { useEffect, useRef, useState } from "react";
import { api } from "../lib/api.js";
import QRCode from "qrcode";

// OPEN_EVENT lets any surface (Devices, the pairing screen's neighbour)
// open this drawer without threading state through the shells.
export const OPEN_EVENT = "picode-open-pair";
export function openPairDrawer() {
  if (typeof window !== "undefined") window.dispatchEvent(new Event(OPEN_EVENT));
}

export default function ShareDrawer({ open, onClose }) {
  const [report, setReport] = useState(null);
  const [picked, setPicked] = useState("");
  const [err, setErr] = useState("");
  const canvasRef = useRef(null);

  const [pairCode, setPairCode] = useState("");
  useEffect(() => {
    if (!open) return;
    setErr("");
    setPicked("");
    setPairCode("");
    api("/api/share").then(setReport).catch((e) => setErr(e.message));
    // The phone needs to be a paired device (ADR-0049): mint a one-time
    // code and put it in the link the QR carries.
    api("/api/auth/pairings", { method: "POST" }).then((p) => setPairCode(p.code || "")).catch(() => {});
  }, [open]);

  const targets = (report && report.targets) || [];
  const url = picked || (report && report.url) || "";
  const chosen = targets.find((t) => t.url === url) || (url ? { url, onCert: true, kind: "" } : null);
  const appURL = pairCode && url ? url.replace(/\/+$/, "") + "/pair?code=" + encodeURIComponent(pairCode) : withHash(url);
  const host = chosen && chosen.addr;
  // A publicly trusted target (the tailnet name with its Tailscale
  // certificate) skips the trust page: there is nothing to install.
  const qrURL = chosen && chosen.trusted ? appURL : trustPage(report && report.trustPort, host, appURL);
  const canQR = !!qrURL;

  useEffect(() => {
    if (!open || !canQR || !canvasRef.current) return;
    QRCode.toCanvas(canvasRef.current, qrURL, { width: 200, margin: 1, color: { dark: "#16181d", light: "#ffffff" } });
  }, [open, canQR, qrURL]);

  if (!open) return null;

  const misses = report ? report.checks.filter((c) => !c.ok) : [];

  return (
    <div className="share-root" onMouseDown={(e) => { if (e.target.classList.contains("share-root")) onClose(); }}>
      <div className="share-panel" role="dialog" aria-label="Pair a device">
        <header className="share-head">
          <h2>Pair a device</h2>
          <button type="button" className="dock-icon" onClick={onClose} aria-label="Close">×</button>
        </header>
        {err ? <p className="form-error">{err}</p> : null}
        {!report && !err ? <p className="share-muted">Checking…</p> : null}
        {report ? (
          <>
            {canQR ? (
              <div className="share-qr">
                <canvas ref={canvasRef} />
                <p className="share-url">{qrURL}</p>
                <p className="share-muted">Scan with the phone's camera. The link pairs this phone; it works once and expires in ten minutes.{report.trustPort ? " iPhone: the Camera app opens Safari, which can install the certificate; Chrome cannot." : ""}</p>
              </div>
            ) : null}
            {misses.length ? (
              <ul className="share-howto">
                {misses.map((c) => (
                  <li key={c.id}>
                    <strong>{c.title}</strong>
                    <span>{c.action}</span>
                  </li>
                ))}
              </ul>
            ) : null}
            {targets.length ? (
              <ul className="share-targets">
                {targets.map((t) => (
                  <li key={t.url}>
                    <button
                      type="button"
                      className={"share-target" + (t.url === url ? " active" : "") + (t.onCert ? "" : " off")}
                      onClick={() => setPicked(t.url)}
                      disabled={!t.onCert}
                    >
                      <span className="share-kind">{t.kind}</span>
                      <span className="share-addr">{t.addr}</span>
                      {!t.onCert ? <span className="share-why">{t.reason}</span> : t.note ? <span className="share-note">{t.note}</span> : null}
                    </button>
                  </li>
                ))}
              </ul>
            ) : null}
            <ul className="share-checks">
              {report.checks.map((c) => (
                <li key={c.id} className={c.ok ? "ok" : "bad"}>
                  <span className="share-dot" />
                  {c.title}
                </li>
              ))}
            </ul>
          </>
        ) : null}
      </div>
    </div>
  );
}

function withHash(url) {
  if (!url) return "";
  if (!location.hash || location.hash === "#/") return url;
  return url.replace(/\/$/, "") + "/" + location.hash;
}

function trustPage(port, addr, app) {
  if (!addr) return app || "";
  if (!port) return app || "";
  const u = new URL("http://" + addr + ":" + port + "/");
  if (app) u.searchParams.set("next", app);
  return u.toString();
}
