import { useEffect, useRef, useState } from "react";
import { api } from "../lib/api.js";
import QRCode from "qrcode";

export default function ShareDrawer({ open, onClose }) {
  const [report, setReport] = useState(null);
  const [err, setErr] = useState("");
  const canvasRef = useRef(null);

  useEffect(() => {
    if (!open) return;
    setErr("");
    api("/api/share").then((r) => {
      if (r.url && location.hash && location.hash !== "#/") {
        const u = r.url.replace(/\/$/, "") + "/" + location.hash;
        r = { ...r, url: u };
      }
      setReport(r);
    }).catch((e) => setErr(e.message));
  }, [open]);

  useEffect(() => {
    if (!open || !report || !report.ready || !report.url || !canvasRef.current) return;
    QRCode.toCanvas(canvasRef.current, report.url, { width: 200, margin: 1, color: { dark: "#16181d", light: "#ffffff" } });
  }, [open, report]);

  if (!open) return null;

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
            {report.ready ? (
              <div className="share-qr">
                <canvas ref={canvasRef} />
                <p className="share-url">{report.url}</p>
              </div>
            ) : (
              <ul className="share-howto">
                {report.checks.filter((c) => !c.ok).map((c) => (
                  <li key={c.id}>
                    <strong>{c.title}</strong>
                    <span>{c.action}</span>
                  </li>
                ))}
              </ul>
            )}
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
