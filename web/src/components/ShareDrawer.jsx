import { useEffect, useRef, useState } from "react";
import { api } from "../lib/api.js";
import QRCode from "qrcode";

export default function ShareDrawer({ open, onClose }) {
  const [report, setReport] = useState(null);
  const [picked, setPicked] = useState("");
  const [err, setErr] = useState("");
  const canvasRef = useRef(null);

  useEffect(() => {
    if (!open) return;
    setErr("");
    setPicked("");
    api("/api/share").then(setReport).catch((e) => setErr(e.message));
  }, [open]);

  const targets = (report && report.targets) || [];
  const url = picked || (report && report.url) || "";
  const chosen = targets.find((t) => t.url === url) || (url ? { url, onCert: true, kind: "" } : null);
  const appURL = withHash(url);
  const qrURL = trustPage(report && report.trustUrl, appURL);
  const canQR = !!qrURL;

  useEffect(() => {
    if (!open || !canQR || !canvasRef.current) return;
    QRCode.toCanvas(canvasRef.current, qrURL, { width: 200, margin: 1, color: { dark: "#16181d", light: "#ffffff" } });
  }, [open, canQR, qrURL]);

  if (!open) return null;

  const misses = report ? report.checks.filter((c) => !c.ok) : [];

  return (
    <div className="share-root" onMouseDown={(e) => { if (e.target.classList.contains("share-root")) onClose(); }}>
      <div className="share-panel" role="dialog" aria-label="Open on phone">
        <header className="share-head">
          <h2>Open on phone</h2>
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
                {report.trustUrl ? <p className="share-muted">Scan installs trust, then open PiCode. iPhone: enable the profile under Certificate Trust Settings.</p> : null}
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
                      {!t.onCert ? <span className="share-why">{t.reason}</span> : null}
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

function trustPage(trust, app) {
  if (!trust) return app;
  try {
    const u = new URL(trust);
    if (app) u.searchParams.set("next", app);
    return u.toString();
  } catch {
    return trust;
  }
}
