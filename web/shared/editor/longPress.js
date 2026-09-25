// Press-and-hold to open a link in the Live editor on a touch screen, where
// Ctrl/⌘+click does not exist. A plain tap still edits the link (it reveals
// its source), exactly as a click does with a mouse.

export const HOLD_MS = 500;
export const SLOP_PX = 10;

// pressTracker is the gesture as a state machine, with the clock passed in so
// it is tested without a DOM: `start` a press on a link, `move` cancels it once
// the finger travels past the slop (the reader is scrolling), and the timer
// fires `open` after the hold. `end` answers what the gesture was — "opened"
// (the hold fired: swallow the tap that follows), "tap" (a short press on a
// link) or "none".
export function pressTracker({ open, setTimer = setTimeout, clearTimer = clearTimeout, holdMs = HOLD_MS, slop = SLOP_PX }) {
  let press = null;
  let timer = null;
  let fired = false;
  const cancel = () => {
    if (timer != null) clearTimer(timer);
    timer = null;
    press = null;
  };
  return {
    start(x, y, href) {
      cancel();
      fired = false;
      if (!href) return;
      press = { x, y, href };
      timer = setTimer(() => {
        timer = null;
        fired = true;
        open(press.href);
      }, holdMs);
    },
    move(x, y) {
      if (press && Math.hypot(x - press.x, y - press.y) > slop) cancel();
    },
    end() {
      const was = fired ? "opened" : press ? "tap" : "none";
      cancel();
      fired = false;
      return was;
    },
    cancel() { cancel(); fired = false; },
    // A long press also asks for the system's context menu; while one of ours
    // is under way (or just fired) that menu is ours to refuse.
    holding() { return !!press || fired; },
  };
}

// A browser may still synthesize mouse events for the finger lifted after a
// hold; they must not land wherever the page now is.
const SWALLOW_MS = 800;
function swallowNextTap() {
  const kinds = ["mousedown", "mouseup", "click"];
  const stop = (e) => { e.preventDefault(); e.stopPropagation(); if (e.type === "click") done(); };
  const done = () => { for (const k of kinds) document.removeEventListener(k, stop, true); clearTimeout(timer); };
  const timer = setTimeout(done, SWALLOW_MS);
  for (const k of kinds) document.addEventListener(k, stop, true);
}

// attachLongPress wires a tracker to an element's touch events. `hrefAt(x, y,
// target)` names the link under the finger ("" for none); `onTap` hears a
// short press on a link (for the hint). Returns the detach function.
//
// Opening a link scrolls or replaces the page, so the node the finger went
// down on is usually gone when it lifts, and its touchend reaches no one
// (not `el`, not the document). What matters is caught anyway: the mouse
// events the browser synthesizes next hit what is under the finger now, and
// swallowNextTap takes them at the document; the press is reset on a timer.
export function attachLongPress(el, { hrefAt, open, onTap }) {
  const tracker = pressTracker({
    open: (href) => {
      swallowNextTap();
      // If the finger's touchend is lost with its node, the press still ends.
      setTimeout(() => { unwatch(); tracker.cancel(); }, SWALLOW_MS);
      // Chrome refuses (and logs) a vibration before any user activation.
      if (navigator.userActivation?.hasBeenActive) { try { navigator.vibrate?.(10); } catch { /* not every device vibrates */ } }
      open(href);
    },
  });
  let last = null;
  const onEnd = (e) => {
    unwatch();
    const was = tracker.end();
    if (was === "opened") { e.preventDefault(); e.stopPropagation(); }
    else if (was === "tap" && last) onTap?.(last.x, last.y);
  };
  const onCancel = () => { unwatch(); tracker.cancel(); };
  const onMove = (e) => { const t = e.touches[0]; if (t) tracker.move(t.clientX, t.clientY); };
  const watch = () => {
    document.addEventListener("touchmove", onMove, { capture: true, passive: true });
    document.addEventListener("touchend", onEnd, { capture: true, passive: false });
    document.addEventListener("touchcancel", onCancel, true);
  };
  const unwatch = () => {
    document.removeEventListener("touchmove", onMove, { capture: true });
    document.removeEventListener("touchend", onEnd, { capture: true });
    document.removeEventListener("touchcancel", onCancel, true);
  };
  const onStart = (e) => {
    unwatch();
    if (e.touches.length !== 1) { tracker.cancel(); return; }
    const t = e.touches[0];
    last = { x: t.clientX, y: t.clientY };
    tracker.start(t.clientX, t.clientY, hrefAt(t.clientX, t.clientY, e.target));
    if (tracker.holding()) watch();
  };
  const onMenu = (e) => { if (tracker.holding()) e.preventDefault(); };
  el.addEventListener("touchstart", onStart, { passive: true });
  el.addEventListener("contextmenu", onMenu);
  return () => {
    unwatch();
    tracker.cancel();
    el.removeEventListener("touchstart", onStart);
    el.removeEventListener("contextmenu", onMenu);
  };
}
