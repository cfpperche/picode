import { useEffect, useRef, useState } from "react";
import ScreenHeader from "../components/ScreenHeader.jsx";
import KeyBar from "../components/KeyBar.jsx";
import TermSurface from "../../components/TermSurface.jsx";
import { terms } from "../../lib/terms.js";
import { api, humanizeError } from "../../lib/api.js";
import { termLine } from "../../lib/repoLine.js";
import { IconKeyboard, IconX } from "../../components/Icons.jsx";

// The pushed terminal screen (#/term/<id>, the desktop's route): the same
// xterm the desktop attaches to the tmux session. The keys a soft keyboard
// lacks live behind a floating button so the pane keeps the whole screen
// until they are wanted. Opening (re)creates the tmux session if closed.
export default function TerminalScreen({ term, onBack, onRemove, busy }) {
  const [page, setPage] = useState(null);
  const [error, setError] = useState("");
  const [keys, setKeys] = useState(false);
  const hostRef = useRef(null);
  const id = term && term.id;

  // Touch scroll. xterm only scrolls its own scrollback on touch; a tmux
  // pane (alt screen or mouse tracking) scrolls through wheel events —
  // SGR mouse reports from xterm, or lib/termWheel's fallback. A finger
  // never produces a wheel, so a vertical drag is turned into one wheel
  // event per ~16px, dispatched where the desktop's wheel would land.
  useEffect(() => {
    const host = hostRef.current;
    if (!host || !page || error) return undefined;
    let lastY = null;
    let acc = 0;
    const STEP = 16;
    const target = () => host.querySelector(".xterm-screen") || host.querySelector(".xterm");
    const onStart = (e) => { lastY = e.touches.length === 1 ? e.touches[0].clientY : null; acc = 0; };
    const onMove = (e) => {
      if (lastY == null || e.touches.length !== 1) return;
      const y = e.touches[0].clientY;
      acc += lastY - y; // finger up (y decreases) = scroll down = positive deltaY
      lastY = y;
      const el = target();
      if (!el) return;
      while (Math.abs(acc) >= STEP) {
        const dir = acc > 0 ? 1 : -1;
        acc -= dir * STEP;
        el.dispatchEvent(new WheelEvent("wheel", { deltaY: dir * 40, deltaMode: 0, bubbles: true, cancelable: true }));
      }
      e.preventDefault();
    };
    const onEnd = () => { lastY = null; acc = 0; };
    host.addEventListener("touchstart", onStart, { passive: true });
    host.addEventListener("touchmove", onMove, { passive: false });
    host.addEventListener("touchend", onEnd);
    host.addEventListener("touchcancel", onEnd);
    return () => {
      host.removeEventListener("touchstart", onStart);
      host.removeEventListener("touchmove", onMove);
      host.removeEventListener("touchend", onEnd);
      host.removeEventListener("touchcancel", onEnd);
    };
  }, [page, error, id]);

  useEffect(() => {
    setPage(null);
    setError("");
    if (!id) return undefined;
    let stale = false;
    api("/api/terminals/" + encodeURIComponent(id) + "/open", { method: "POST" })
      .then((p) => { if (!stale) setPage(p); })
      .catch((e) => { if (!stale) setError(humanizeError((e && e.message) || String(e))); });
    return () => { stale = true; };
  }, [id]);

  function sendKey(seq) {
    const entry = terms.get("sh:" + id);
    if (!entry) return;
    if (entry.sock && entry.sock.readyState === WebSocket.OPEN) entry.sock.send(new TextEncoder().encode(seq));
    if (entry.term) entry.term.focus();
  }

  if (!term) {
    return (
      <div className="m-screen">
        <ScreenHeader title="Terminal" onBack={onBack} />
        <p className="m-empty-line m-pad">That terminal is gone.</p>
      </div>
    );
  }
  const live = page ? { ...term, ...page } : term;
  const line = termLine(live);
  return (
    <div className="m-screen m-term-screen">
      <ScreenHeader
        title={live.name || "Terminal"}
        sub={line.text}
        onBack={onBack}
        right={<button type="button" className="btn btn-sm" disabled={busy} onClick={() => onRemove(term)}>Remove</button>}
      />
      <div className="m-term" ref={hostRef}>
        {error ? (
          <p className="m-empty-line m-pad">{error}</p>
        ) : page ? (
          <TermSurface term={live} hidden={false} cwdKind="term" />
        ) : (
          <p className="m-empty-line m-pad">Attaching…</p>
        )}
      </div>
      {page && !error && keys ? <KeyBar onKey={sendKey} onClose={() => setKeys(false)} /> : null}
      {page && !error && !keys ? (
        <button type="button" className="m-fab" title="Keys" aria-label="Show terminal keys" onPointerDown={(e) => e.preventDefault()} onClick={() => setKeys(true)}>
          <IconKeyboard size={20} />
        </button>
      ) : null}
    </div>
  );
}
