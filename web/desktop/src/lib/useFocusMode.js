// The DOM half of fullscreen (focus) mode. Every decision lives in
// @picode/shared/domain/focusMode.js; this file only wires it to the DOM:
// the pointer, Escape, localStorage and the xterm refit that a chrome
// change owes every live pane.
//
// The browser window is not wired here at all. The mode hides PiCode's own
// chrome and leaves the window alone, so there is no Fullscreen API call to
// make, nothing to undo when the browser moves on its own, and no
// fullscreenchange listener: Escape reaches this file's handler on the
// first press, which is what makes the two steps (close the reveal, then
// leave) real. F11 is the reader's gesture — we neither send it nor watch
// for it, and a mode entered before or after it behaves the same.
import { useCallback, useEffect, useRef, useState } from "react";
import {
  DWELL_IN_MS,
  DWELL_OUT_MS,
  EDGE_PX,
  FOCUS_FIRST_RUN,
  activeZones,
  focusReduce,
  hotZone,
  initialFocusState,
  readFocusPrefs,
  shellClasses,
  writeFocusPrefs,
} from "@picode/shared/domain/focusMode.js";
import { scheduleTermFit } from "@picode/shared/domain/termFit.js";
import { terms } from "./terms.js";
import { paneAt } from "./termActions.js";
import { toast } from "./toast.js";

const now = () => (typeof performance !== "undefined" ? performance.now() : Date.now());

// Which revealed panel the pointer is over. Anything portaled to the body
// — a menu opened from the sidebar, a dialog, a toast — belongs to
// whatever opened it, so it keeps the reveal that is already open instead
// of counting as "the pointer left".
function panelUnder(target, current) {
  if (!target || typeof target.closest !== "function") return null;
  if (target.closest("#sidebar")) return "left";
  if (target.closest("#main-tabs")) return "top";
  if (target.closest("#inspector")) return "right";
  if (target.closest("[data-radix-popper-content-wrapper], [data-sonner-toaster], [role='dialog'], [role='alertdialog']")) return current;
  return null;
}

// An overlay owns Escape while it is open: Radix closes the menu or the
// dialog, and the mode must not read the same press as "leave".
function overlayOpen() {
  if (typeof document === "undefined") return false;
  return !!document.querySelector("[data-radix-popper-content-wrapper], [role='dialog'], [role='alertdialog']");
}

// "The rail is on screen" is what the viewer actually saw: inspectorLayout
// can report shown while the Home dashboard is suppressing the rail, and a
// right strip that reveals nothing is a dead hover. The shell itself is the
// honest answer — read at the click that needs it, and watched (below) for
// the changes a running mode has to hear about.
function railOnScreen(fallback) {
  if (typeof document === "undefined") return fallback;
  const el = document.getElementById("inspector");
  return el ? !el.hidden : fallback;
}

// Hiding or showing the chrome resizes every pane; a reveal does not (it
// floats above the surface), so only the mode itself refits.
function refitPanes() {
  requestAnimationFrame(() => {
    for (const entry of terms.values()) scheduleTermFit(entry, true);
  });
}

export function useFocusMode({ railOpen = false, available = true } = {}) {
  // A reload restores the mode, but not what the rail looked like when it
  // started — the rail's state right now is the closest true answer. The
  // caller's value seeds it; the observer below keeps it honest from the
  // first commit on.
  const [state, setState] = useState(() => initialFocusState({ ...readFocusPrefs(), railOpen }));
  const ref = useRef(state);
  ref.current = state;

  const send = useCallback((ev) => {
    const prev = ref.current;
    const next = focusReduce(prev, ev);
    if (next === prev) return prev;
    ref.current = next;
    setState(next);
    if (prev.on !== next.on) {
      writeFocusPrefs({ on: next.on });
      refitPanes();
      // One line, the first time only: what happened and how to get out.
      if (next.on && !next.seen) {
        toast.info(FOCUS_FIRST_RUN);
        const seen = focusReduce(next, { type: "seen" });
        ref.current = seen;
        setState(seen);
        writeFocusPrefs({ seen: true });
        return seen;
      }
    }
    return next;
  }, []);

  const toggle = useCallback(() => send({ type: "toggle", railOpen: railOnScreen(false) }), [send]);
  const leave = useCallback(() => send({ type: "leave" }), [send]);
  // A control that names a panel — the Inspector toggle at the end of the
  // tab strip. The pointer may be nowhere near the rail's own edge, so the
  // click reveals it outright and pins it: what a click opened, a click (or
  // Escape, or leaving the mode) closes.
  const show = useCallback((zone) => send({ type: "reveal", zone, pinned: true }), [send]);

  // A shell that cannot host the mode any more — the window narrowed to
  // the single column, or the viewer opened a page route, where the tab
  // strip and its Leave control do not exist — gives it back.
  useEffect(() => { if (!available) send({ type: "unavailable" }); }, [available, send]);

  // The rail's own `hidden` is what the mode follows: the Inspector toggle
  // can turn the rail on while the mode runs, a tab picked from the revealed
  // strip replaces the Home dashboard that was suppressing it, and a window
  // that stopped fitting it can fit again. React has no value for "the
  // element is on screen" — the caller's `railOpen` only seeds the state —
  // so the wiring watches the attribute the element really wears. While the
  // mode is off the event is a no-op, and the click that starts it reads the
  // same source.
  useEffect(() => {
    if (typeof document === "undefined") return undefined;
    const el = document.getElementById("inspector");
    if (!el) return undefined;
    const report = () => send({ type: "rail", open: !el.hidden });
    report();
    const observer = new MutationObserver(report);
    observer.observe(el, { attributes: true, attributeFilter: ["hidden"] });
    return () => observer.disconnect();
  }, [send]);

  // The pointer. The strips are real elements so the events still fire
  // over an embedded frame (the PDF preview), but the zone itself is
  // decided by the shared hit test, from this one listener.
  useEffect(() => {
    if (!state.on) return undefined;
    const rail = state.railWasOpen;
    function onMove(e) {
      const zone = hotZone(
        { x: e.clientX, y: e.clientY },
        { width: window.innerWidth, height: window.innerHeight },
        { rail, edge: EDGE_PX },
      );
      send({ type: "point", zone, inside: panelUnder(e.target, ref.current.reveal), at: now() });
    }
    function onOut(e) {
      // Only the pointer really leaving the window, not a child boundary.
      if (e.relatedTarget) return;
      send({ type: "point", zone: null, inside: null, at: now() });
    }
    window.addEventListener("pointermove", onMove, { passive: true });
    document.addEventListener("pointerout", onOut, { passive: true });
    return () => {
      window.removeEventListener("pointermove", onMove);
      document.removeEventListener("pointerout", onOut);
    };
  }, [state.on, state.railWasOpen, send]);

  // One timer per pending dwell, armed for exactly what is left of it.
  // The tick carries the *deadline*, not the clock: a setTimeout(120) can
  // come back at 118.9 ms of performance.now(), and a tick that resolves
  // nothing would leave the dwell armed forever — the state did not
  // change, so this effect would never schedule a second one.
  useEffect(() => {
    if (!state.on) return undefined;
    const due = [];
    if (state.armed) due.push(state.armed.at + DWELL_IN_MS);
    if (state.closing) due.push(state.closing.at + DWELL_OUT_MS);
    if (!due.length) return undefined;
    const at = Math.min(...due);
    const id = setTimeout(() => send({ type: "tick", at: Math.max(now(), at) }), Math.max(0, at - now()));
    return () => clearTimeout(id);
  }, [state.on, state.armed, state.closing, send]);

  // Escape: the reveal first, then the mode — two presses, both ours,
  // because no browser fullscreen is in front of them to eat the first. A
  // terminal pane keeps its own Escape — vim and every agent TUI need it —
  // so the mode is left from there with the chord, the menu row or the
  // Leave control in the top strip.
  useEffect(() => {
    if (!state.on) return undefined;
    function onKey(e) {
      if (e.key !== "Escape" || e.defaultPrevented) return;
      if (e.ctrlKey || e.altKey || e.metaKey || e.shiftKey) return;
      if (overlayOpen()) return;
      if (paneAt(e.target)) return;
      e.preventDefault();
      send({ type: "escape" });
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [state.on, send]);

  return {
    on: state.on,
    reveal: state.reveal,
    railWasOpen: state.railWasOpen,
    zones: activeZones(state),
    classes: shellClasses(state),
    toggle,
    leave,
    show,
  };
}
