import { useEffect, useRef, useState } from "react";
import ScreenHeader from "../components/ScreenHeader.jsx";
import KeyBar from "../components/KeyBar.jsx";
import TermSurface from "../../components/TermSurface.jsx";
import { terms } from "../../lib/terms.js";
import { scheduleTermFit } from "../../lib/termFit.js";
import { api, humanizeError } from "../../lib/api.js";
import { termLine } from "../../lib/repoLine.js";
import { IconKeyboard, IconGit } from "../../components/Icons.jsx";

// The pushed terminal screen (#/term/<id>, the desktop's route): the same
// xterm the desktop attaches to the tmux session. The keys a soft keyboard
// lacks live behind a floating button so the pane keeps the whole screen
// until they are wanted. Opening (re)creates the tmux session if closed.
export default function TerminalScreen({ term, onBack, onRemove, busy, onOpenChanges }) {
  const [page, setPage] = useState(null);
  const [error, setError] = useState("");
  const [keys, setKeys] = useState(false);
  const [armed, setArmed] = useState({ ctrl: false, alt: false });
  const [hardKeyboard, setHardKeyboard] = useState(false);
  const hostRef = useRef(null);
  const screenRef = useRef(null);
  const id = term && term.id;
  const entryOf = () => terms.get("sh:" + id);

  // The sticky state lives on the attach (ShellTerm filters the phone's
  // keystrokes through it); this screen mirrors it for the bar.
  useEffect(() => {
    if (!page || error) return undefined;
    const tick = setInterval(() => {
      const e = entryOf();
      if (!e || !e.sticky) return;
      const st = e.sticky.state();
      setArmed((cur) => (cur.ctrl === st.ctrl && cur.alt === st.alt ? cur : st));
    }, 250);
    let off = null;
    const e = entryOf();
    if (e && e.sticky) off = e.sticky.subscribe((st) => setArmed(st));
    return () => { clearInterval(tick); if (off) off(); };
  }, [page, error, id]);

  // iOS: the soft keyboard shrinks the visual viewport, not the layout
  // one, so a bar at the bottom of a 100dvh app ends up under the
  // keyboard. Size the screen to the visual viewport and refit xterm,
  // the way terminal-web lifts its bar. No keyboard, no change.
  useEffect(() => {
    const vv = window.visualViewport;
    const el = screenRef.current;
    if (!vv || !el) return undefined;
    let focusedAt = 0;
    const apply = () => {
      const lift = Math.max(0, Math.round(window.innerHeight - vv.height - vv.offsetTop));
      const covered = window.innerHeight - vv.height > 120; // a keyboard, not a toolbar
      el.style.height = covered ? vv.height + "px" : "";
      el.style.transform = covered ? "translateY(" + Math.round(vv.offsetTop) + "px)" : "";
      el.style.setProperty("--m-kb-lift", lift + "px");
      if (covered) setHardKeyboard(false);
      const e = entryOf();
      if (e) scheduleTermFit(e, true);
    };
    // A hardware keyboard: the person tapped the terminal, it took
    // focus, and nothing shrank. Programmatic focus (the attach focuses
    // xterm) opens no soft keyboard anywhere, so only a focus that
    // follows a touch counts.
    let touchedAt = 0;
    const onPointer = () => { touchedAt = Date.now(); };
    const onFocusIn = (ev) => {
      if (!hostRef.current || !hostRef.current.contains(ev.target)) return;
      if (Date.now() - touchedAt > 500) return;
      focusedAt = Date.now();
      setTimeout(() => {
        if (Date.now() - focusedAt < 280) return;
        if (window.innerHeight - vv.height <= 120) setHardKeyboard(true);
      }, 300);
    };
    vv.addEventListener("resize", apply);
    vv.addEventListener("scroll", apply);
    document.addEventListener("focusin", onFocusIn);
    document.addEventListener("pointerdown", onPointer, true);
    apply();
    return () => {
      vv.removeEventListener("resize", apply);
      vv.removeEventListener("scroll", apply);
      document.removeEventListener("focusin", onFocusIn);
      document.removeEventListener("pointerdown", onPointer, true);
      el.style.height = "";
      el.style.transform = "";
    };
  }, [id]);

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

  // A key from the bar never changes whether the phone keyboard is up:
  // refocus xterm only if it already had the focus (the keyboard was
  // open and must stay open); otherwise leave the focus alone.
  function sendKey(seq) {
    const entry = entryOf();
    if (!entry) return;
    const out = entry.sticky ? entry.sticky.applyKey(seq) : seq;
    if (entry.sock && entry.sock.readyState === WebSocket.OPEN) entry.sock.send(new TextEncoder().encode(out));
    const host = hostRef.current;
    const hadFocus = !!(host && document.activeElement && host.contains(document.activeElement));
    if (hadFocus && entry.term) entry.term.focus();
  }

  function armKey(mod) {
    const entry = entryOf();
    if (entry && entry.sticky) setArmed(entry.sticky.arm(mod));
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
    <div className="m-screen m-term-screen" ref={screenRef}>
      <ScreenHeader
        title={live.name || "Terminal"}
        sub={line.text}
        onBack={onBack}
        right={(
          <>
            {live.git && live.git.dirty ? (
              <button type="button" className="btn btn-sm m-changes-btn" title="Uncommitted changes" onClick={() => onOpenChanges("term", term.id, live.name || "Terminal")}><IconGit size={13} /> {live.git.dirty}</button>
            ) : null}
            {page && !error ? (
              <button type="button" className={"btn btn-sm m-keys-btn" + (keys ? " on" : "")} title="Terminal keys" aria-label={keys ? "Hide terminal keys" : "Show terminal keys"} aria-pressed={keys} onPointerDown={(e) => e.preventDefault()} onClick={() => { setHardKeyboard(false); setKeys((k) => !k); }}>
                <IconKeyboard size={16} />
              </button>
            ) : null}
            <button type="button" className="btn btn-sm" disabled={busy} onClick={() => onRemove(term)}>Remove</button>
          </>
        )}
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
      {page && !error && keys && !hardKeyboard ? <KeyBar armed={armed} onArm={armKey} onKey={sendKey} /> : null}
    </div>
  );
}
