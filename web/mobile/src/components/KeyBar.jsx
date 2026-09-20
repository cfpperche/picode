// Phone terminal accessory (ADR-0044): one row that scrolls sideways,
// Fable-first, sitting on top of the software keyboard. Ctrl/Alt are
// sticky. A key never summons the IME. Hide (pinned, does not scroll
// away) blurs xterm.
//
// The keyboard belongs to the user (owner, iOS, 2026-09-20): a key-bar
// tap must move the TUI and leave the IME open; only Hide or a real tap
// in the pane closes it. WebKit only honors a NON-PASSIVE touchstart
// with preventDefault for keeping focus — but per the touch-events
// contract canceling touchstart also cancels the native pan, which is
// how the first cut killed scrolling (owner, same day). The Stack
// Overflow consensus for exactly this conflict (8692678, 20915251) is
// gesture discrimination: preventDefault stays, the code tracks the
// touch itself — movement past a threshold becomes manual scrolling
// with momentum, a stationary release becomes a tap and dispatches the
// key on touchend. Mouse acts on pointerdown, Enter/Space on click;
// each input fires exactly one path.

import { useEffect, useRef } from "react";
import { IconKeyboard } from "./Icons.jsx";
import { KEYS } from "../lib/termKeyBar.js";

const TAP_SLOP = 8; // px of movement before a touch becomes a drag
const GLIDE_FRICTION = 0.94; // velocity kept per ~16ms frame

export default function KeyBar({ armed, onArm, onKey, onHide }) {
  const barRef = useRef(null);
  // Fresh every render, so the effect (rebound each render too) and the
  // React handlers always call with the current props.
  const dispatch = (act, arg) => {
    if (act === "key") onKey(arg);
    else if (act === "mod") onArm(arg);
    else onHide();
  };

  useEffect(() => {
    const bar = barRef.current;
    if (!bar) return undefined;
    const scroller = bar.querySelector(".m-keybar-scroller");
    let touch = null; // { id, x0, y0, scroll0, act, arg, dragging }
    let lastX = 0;
    let lastT = 0;
    let vel = 0; // finger px/ms, negative = finger moving left
    let raf = 0;

    const stopGlide = () => { cancelAnimationFrame(raf); raf = 0; };

    const glide = (v) => {
      stopGlide();
      let prev = performance.now();
      const step = (now) => {
        const dt = Math.min(48, now - prev) || 16;
        prev = now;
        const before = scroller.scrollLeft;
        scroller.scrollLeft -= v * dt;
        v *= Math.pow(GLIDE_FRICTION, dt / 16);
        const atEdge = scroller.scrollLeft === before;
        if (!atEdge && Math.abs(v) > 0.02) raf = requestAnimationFrame(step);
      };
      raf = requestAnimationFrame(step);
    };

    const onTouchStart = (e) => {
      if (e.touches.length !== 1) { touch = null; return; }
      const t = e.touches[0];
      const btn = e.target.closest("button[data-act]");
      stopGlide();
      touch = {
        id: t.identifier,
        x0: t.clientX, y0: t.clientY,
        scroll0: scroller ? scroller.scrollLeft : 0,
        act: btn && btn.dataset.act, arg: btn && btn.dataset.arg,
        dragging: false,
      };
      lastX = t.clientX;
      lastT = performance.now();
      vel = 0;
      // Keeps focus (and the IME) on the terminal; the native pan is
      // cancelled with it, so touchmove scrolls by hand below.
      e.preventDefault();
    };

    const onTouchMove = (e) => {
      if (!touch) return;
      const t = [...e.changedTouches].find((x) => x.identifier === touch.id);
      if (!t) return;
      const dx = t.clientX - touch.x0;
      const dy = t.clientY - touch.y0;
      if (!touch.dragging && Math.hypot(dx, dy) > TAP_SLOP) touch.dragging = true;
      if (touch.dragging && scroller) {
        const now = performance.now();
        vel = (t.clientX - lastX) / Math.max(1, now - lastT);
        lastX = t.clientX;
        lastT = now;
        scroller.scrollLeft = touch.scroll0 - dx;
      }
      e.preventDefault();
    };

    const onTouchEnd = (e) => {
      if (!touch) return;
      const ended = [...e.changedTouches].find((x) => x.identifier === touch.id);
      const was = touch;
      touch = null;
      if (!ended) return;
      if (!was.dragging) {
        if (was.act) dispatch(was.act, was.arg);
      } else if (Math.abs(vel) > 0.3) {
        glide(-vel); // scroll velocity is the finger velocity, negated
      }
    };

    const onTouchCancel = () => { touch = null; };

    bar.addEventListener("touchstart", onTouchStart, { passive: false });
    bar.addEventListener("touchmove", onTouchMove, { passive: false });
    bar.addEventListener("touchend", onTouchEnd);
    bar.addEventListener("touchcancel", onTouchCancel);
    return () => {
      stopGlide();
      bar.removeEventListener("touchstart", onTouchStart);
      bar.removeEventListener("touchmove", onTouchMove);
      bar.removeEventListener("touchend", onTouchEnd);
      bar.removeEventListener("touchcancel", onTouchCancel);
    };
  });

  const props = (act, arg) => ({
    "data-act": act,
    "data-arg": arg,
    onPointerDown: (e) => {
      if (e.pointerType === "touch") return; // the touch handlers own it
      e.preventDefault();
      dispatch(act, arg);
    },
    onClick: () => dispatch(act, arg),
  });

  return (
    <div className="m-keybar" role="toolbar" aria-label="Terminal keys" ref={barRef}>
      <div className="m-keybar-scroller">
        {KEYS.map((k) => k.mod ? (
          <button
            key={k.id}
            type="button"
            className={"m-key m-key-mod" + (armed && armed[k.mod] ? " on" : "")}
            aria-pressed={!!(armed && armed[k.mod])}
            title={k.label + " — arms the next key"}
            {...props("mod", k.mod)}
          >{k.label}</button>
        ) : (
          <button
            key={k.id}
            type="button"
            className="m-key"
            title={k.title || k.label}
            {...props("key", k.seq)}
          >{k.label}</button>
        ))}
      </div>
      <button
        type="button"
        className="m-key m-key-hide"
        title="Hide keyboard"
        aria-label="Hide keyboard"
        {...props("hide")}
      ><IconKeyboard size={16} /></button>
    </div>
  );
}
