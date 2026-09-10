import { useCallback, useEffect, useRef, useState } from "react";
import { IconMonitor } from "./Icons.jsx";

// Browser surface (ADR-0114): a watch-only live view of the agent's
// browser. The daemon proxies the engine's stream WebSocket; this view
// renders JPEG frames on a canvas, shows the page URL, and sends pacing
// control only. Watch-only is phase 1 — the view says so instead of
// pretending interactive input works.

const EMPTY = "no-browser";
const UNREACHABLE = "engine-unreachable";

export default function BrowserSurface({ agentId, onBack }) {
  const canvasRef = useRef(null);
  const wrapRef = useRef(null);
  const frameRef = useRef(null); // last decoded frame (ImageBitmap)
  const socketRef = useRef(null);
  const [phase, setPhase] = useState("connecting"); // connecting | live | idle | unreachable | ended
  const [pageUrl, setPageUrl] = useState("");

  const drawFrame = useCallback(() => {
    const canvas = canvasRef.current;
    const frame = frameRef.current;
    if (!canvas || !frame) return;
    const wrap = wrapRef.current;
    const dpr = window.devicePixelRatio || 1;
    const w = Math.max(1, Math.floor((wrap ? wrap.clientWidth : canvas.clientWidth) * dpr));
    const h = Math.max(1, Math.floor((wrap ? wrap.clientHeight : canvas.clientHeight) * dpr));
    if (canvas.width !== w || canvas.height !== h) {
      canvas.width = w;
      canvas.height = h;
    }
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.fillStyle = "#000";
    ctx.fillRect(0, 0, w, h);
    const scale = Math.min(w / frame.width, h / frame.height);
    const dw = Math.max(1, Math.floor(frame.width * scale));
    const dh = Math.max(1, Math.floor(frame.height * scale));
    ctx.imageSmoothingEnabled = scale < 1;
    ctx.drawImage(frame, Math.floor((w - dw) / 2), Math.floor((h - dh) / 2), dw, dh);
  }, []);

  // Refit the last frame when the pane resizes.
  useEffect(() => {
    const wrap = wrapRef.current;
    if (!wrap || typeof ResizeObserver === "undefined") return;
    const ro = new ResizeObserver(() => drawFrame());
    ro.observe(wrap);
    return () => ro.disconnect();
  }, [drawFrame]);

  useEffect(() => {
    if (!agentId) return;
    let disposed = false;
    let bitmapUrl = null;
    setPhase("connecting");
    setPageUrl("");
    frameRef.current = null;

    const proto = location.protocol === "https:" ? "wss" : "ws";
    const ws = new WebSocket(`${proto}://${location.host}/ws/browser?agent=${encodeURIComponent(agentId)}`);
    socketRef.current = ws;

    ws.onopen = () => {
      // Bound the bandwidth: 12 fps is plenty for supervising an agent.
      try { ws.send(JSON.stringify({ type: "config", maxFps: 12 })); } catch { /* closed */ }
    };

    ws.onmessage = (event) => {
      if (disposed) return;
      let msg;
      try { msg = JSON.parse(event.data); } catch { return; }
      if (msg.type === "status") {
        if (msg.connected) {
          setPhase("live");
        } else if (msg.reason === UNREACHABLE) {
          setPhase("unreachable");
        } else if (msg.reason === EMPTY) {
          setPhase("idle");
        } else {
          setPhase("idle");
        }
        return;
      }
      if (msg.type === "url" && msg.url) {
        setPageUrl(msg.url);
        return;
      }
      if (msg.type === "frame" && msg.data) {
        try {
          const blob = base64ToBlob(msg.data);
          if (bitmapUrl) URL.revokeObjectURL(bitmapUrl);
          bitmapUrl = URL.createObjectURL(blob);
          const img = new Image();
          img.onload = () => {
            if (disposed) return;
            frameRef.current = img;
            drawFrame();
          };
          img.src = bitmapUrl;
        } catch { /* malformed frame: skip */ }
      }
    };

    ws.onclose = () => {
      if (disposed) return;
      setPhase((p) => (p === "live" ? "ended" : p === "connecting" ? "idle" : p));
    };
    ws.onerror = () => { /* onclose follows; it owns the state */ };

    return () => {
      disposed = true;
      if (bitmapUrl) URL.revokeObjectURL(bitmapUrl);
      try { ws.close(); } catch { /* already closed */ }
      if (socketRef.current === ws) socketRef.current = null;
    };
  }, [agentId, drawFrame]);

  const retry = () => {
    // Re-run the connect effect by bumping a key upstream is cleaner, but a
    // manual reconnect here keeps the retry action local and instant.
    if (socketRef.current) { try { socketRef.current.close(); } catch { /* noop */ } }
    setPhase("connecting");
  };

  const live = phase === "live";

  return (
    <section className="browser-surface" aria-label="Agent browser">
      <div className="browser-bar">
        <span className={"browser-dot" + (live ? " live" : "")} title={live ? "Live" : "Not streaming"} />
        <span className="browser-url" title={pageUrl || ""}>{live || phase === "ended" ? pageUrl || "about:blank" : ""}</span>
        <span className="browser-chip" title="You are watching the agent's browser. Interactive control arrives in a later phase.">Watch-only</span>
        {phase === "unreachable" || phase === "ended" ? (
          <button type="button" className="btn btn-sm" onClick={retry}>Retry</button>
        ) : null}
        {phase !== "idle" ? (
          <button type="button" className="btn btn-sm" onClick={onBack}>Back to chat</button>
        ) : null}
      </div>
      <div className="browser-stage" ref={wrapRef} data-phase={phase}>
        <canvas ref={canvasRef} className="browser-canvas" hidden={phase !== "live" && phase !== "ended"} />
        {phase === "connecting" ? (
          <div className="browser-empty" role="status">
            <IconMonitor size={20} />
            <p className="browser-empty-title">Attaching to the browser…</p>
          </div>
        ) : null}
        {phase === "idle" ? (
          <div className="browser-empty">
            <IconMonitor size={20} />
            <p className="browser-empty-title">No browser is running in this session.</p>
            <p className="browser-empty-hint">The view attaches when the agent opens one.</p>
            <button type="button" className="btn btn-primary btn-sm" onClick={onBack}>Back to chat</button>
          </div>
        ) : null}
        {phase === "unreachable" ? (
          <div className="browser-empty">
            <IconMonitor size={20} />
            <p className="browser-empty-title">The browser stopped responding.</p>
            <button type="button" className="btn btn-primary btn-sm" onClick={retry}>Retry</button>
          </div>
        ) : null}
      </div>
    </section>
  );
}

function base64ToBlob(b64) {
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return new Blob([bytes], { type: "image/jpeg" });
}
