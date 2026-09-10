// The browser half of fullscreen (focus) mode. Every decision lives in
// @picode/shared/domain/focusMode.js; this file only wires it to the DOM:
// the pointer, the Fullscreen API, the keyboard lock, Escape, localStorage
// and the xterm refit that a chrome change owes every live pane.
//
// The keyboard lock (Chromium, fullscreen only) is what makes the mode a
// real terminal: in a normal window the browser eats the reserved chords
// (Ctrl+T, Ctrl+W, Ctrl+N) before the page sees any keydown — a guest CLI
// like Codex's Ctrl+T can never get them (browserChord.js). Locked, every
// key reaches the page, xterm encodes the chord and the guest gets it; in
// the composer the keys simply stop firing browser chrome. Escape is part
// of the lock: a pane keeps vim's Esc, Esc outside a pane still leaves the
// mode through this file's own handler, and the browser swaps its one-press
// exit for "press and hold Esc" — the hatch that always works.
import { useCallback, useEffect, useRef, useState } from "react";
import {
  DWELL_IN_MS,
  DWELL_OUT_MS,
  EDGE_PX,
  FOCUS_FIRST_RUN,
  activeZones,
  browserIntent,
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
import { matchAction } from "./appKeys.js";
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

// "The rail was open when the mode started" is what the viewer actually
// saw: inspectorLayout can report shown while the Home dashboard is
// suppressing the rail, and a right strip that reveals nothing is a dead
// hover. The shell itself is the honest answer.
function railOnScreen(fallback) {
  if (typeof document === "undefined") return fallback;
  const el = document.getElementById("inspector");
  return el ? !el.hidden : fallback;
}

// Lock before the request, in the same gesture task: the spec asks for
// this order so the browser shows one combined "hold Esc" message instead
// of two. Without the Fullscreen API the lock would be inert anyway (it
// only processes keys during JS fullscreen), so the early return stands.
function lockKeyboard() {
  try {
    const kb = typeof navigator !== "undefined" ? navigator.keyboard : null;
    if (!kb || typeof kb.lock !== "function") return;
    Promise.resolve(kb.lock()).catch((err) =>
      console.debug("focus mode: keyboard lock refused — browser keys stay active", err));
  } catch (err) {
    console.debug("focus mode: keyboard lock unavailable", err);
  }
}

// Fire-and-forget is enough on exit: the browser drops the lock by itself
// whenever fullscreen ends (its own hold-Esc included) — this only covers
// the path where we leave first.
function unlockKeyboard() {
  try {
    const kb = typeof navigator !== "undefined" ? navigator.keyboard : null;
    if (kb && typeof kb.unlock === "function") kb.unlock();
  } catch { /* nothing to unlock */ }
}

function requestFullscreen(report) {
  if (typeof document === "undefined") return;
  const el = document.documentElement;
  const req = el.requestFullscreen || el.webkitRequestFullscreen;
  if (!req) {
    // A browser without the API still gets the in-app mode.
    console.debug("focus mode: no Fullscreen API; the app chrome is hidden anyway");
    return;
  }
  lockKeyboard();
  try {
    const p = req.call(el, { navigationUI: "hide" });
    if (p && typeof p.then === "function") {
      p.then(() => report({ type: "browser", active: !!document.fullscreenElement }))
        .catch((err) => console.debug("focus mode: the browser refused fullscreen", err));
    }
  } catch (err) {
    console.debug("focus mode: the browser refused fullscreen", err);
  }
}

function exitFullscreen() {
  if (typeof document === "undefined" || !document.fullscreenElement) return;
  unlockKeyboard();
  try {
    const p = document.exitFullscreen ? document.exitFullscreen() : null;
    if (p && typeof p.catch === "function") p.catch((err) => console.debug("focus mode: exit refused", err));
  } catch (err) {
    console.debug("focus mode: exit refused", err);
  }
}

// Hiding or showing the chrome resizes every pane; a reveal does not (it
// floats above the surface), so only the mode itself refits.
function refitPanes() {
  requestAnimationFrame(() => {
    for (const entry of terms.values()) scheduleTermFit(entry, true);
  });
}

export function useFocusMode({ railOpen = false, available = true } = {}) {
  // A reload restores the mode but not the browser fullscreen (there was
  // no gesture) and not what the rail looked like when it started — the
  // rail's state right now is the closest true answer.
  const [state, setState] = useState(() => initialFocusState({ ...readFocusPrefs(), railOpen }));
  const ref = useRef(state);
  ref.current = state;
  const railRef = useRef(railOpen);
  railRef.current = railOpen;
  // True while an enter request we fired is still travelling: a mode that
  // stops in that window must take the browser back out (see send).
  const enterPending = useRef(false);

  const send = useCallback((ev) => {
    const prev = ref.current;
    const next = focusReduce(prev, ev);
    const intent = browserIntent(prev, next, ev);
    // The browser answered a fullscreen change: an in-flight enter request
    // of ours is answered by this very event, so the pending flag is read
    // (wasPending) and cleared before anything else decides with it.
    const wasPending = enterPending.current;
    if (ev.type === "browser") enterPending.current = false;
    if (next === prev && intent === "none" && !wasPending) return prev;
    ref.current = next;
    setState(next);
    if (intent === "request") {
      enterPending.current = true;
      requestFullscreen(send);
    } else if (intent === "exit") exitFullscreen();
    // A mode that stops while our own enter request is still travelling —
    // the same gesture was an Escape, the fullscreen chord, or a click
    // that navigated away — must not land fullscreen with the mode off.
    // exitFullscreen is a no-op until the request lands, so the belt
    // fires again when the browser reports the change (wasPending above).
    if (!next.on && enterPending.current) exitFullscreen();
    if (ev.type === "browser" && wasPending && !next.on && ev.active) exitFullscreen();
    // Going fullscreen resizes the viewport: the restored mode engages it
    // without a mode transition, so the refit must follow fs, not on.
    if (prev.fs !== next.fs) refitPanes();
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

  const toggle = useCallback(() => send({ type: "toggle", railOpen: railOnScreen(railRef.current) }), [send]);
  const leave = useCallback(() => send({ type: "leave" }), [send]);

  // A reload restores the mode without the browser part — there was no
  // gesture to ask with. The first real input is that gesture: the resume
  // hands the request in from its own handler, so the mode the viewer
  // chose keeps its fullscreen and its keyboard lock without anything
  // being done twice. Synthetic events are refused (isTrusted — no real
  // input, no activation), and keys that themselves mean "leave" (Escape,
  // the fullscreen chord) are left to speak first, so the resume never
  // races the mode into a fullscreen nobody asked to keep. One attempt
  // per page load: a browser that refuses fullscreen refuses again.
  const resumeTried = useRef(false);
  useEffect(() => {
    if (!state.on || state.fs) return undefined;
    function onGesture(e) {
      if (!e.isTrusted || resumeTried.current) return;
      if (e.type === "keydown") {
        if (e.key === "Escape") return;
        if (matchAction("app.fullscreen.toggle", e)) return;
      }
      resumeTried.current = true;
      send({ type: "resume" });
    }
    window.addEventListener("keydown", onGesture, true);
    window.addEventListener("click", onGesture, true);
    return () => {
      window.removeEventListener("keydown", onGesture, true);
      window.removeEventListener("click", onGesture, true);
    };
  }, [state.on, state.fs, send]);

  // A shell that cannot host the mode any more — the window narrowed to
  // the single column, or the viewer opened a page route, where the tab
  // strip and its Leave control do not exist — gives it back.
  useEffect(() => { if (!available) send({ type: "unavailable" }); }, [available, send]);

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

  // Escape: the reveal first, then the mode. A terminal pane keeps its
  // own Escape — vim and every agent TUI need it — so the mode is left
  // from there with the chord, the menu row or the Leave control.
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

  // F11 or the browser's own Escape ends the mode it started.
  useEffect(() => {
    function onChange() { send({ type: "browser", active: !!document.fullscreenElement }); }
    document.addEventListener("fullscreenchange", onChange);
    return () => document.removeEventListener("fullscreenchange", onChange);
  }, [send]);

  return {
    on: state.on,
    reveal: state.reveal,
    railWasOpen: state.railWasOpen,
    zones: activeZones(state),
    classes: shellClasses(state),
    toggle,
    leave,
  };
}
