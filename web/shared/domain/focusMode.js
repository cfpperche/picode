// Fullscreen (focus) mode — the pure part.
//
// One shell-wide mode, per viewer: the sidebar, the tab strip and the
// Inspector rail hide, and the selected surface takes the whole window.
// Nothing unmounts; the chrome is only made invisible, exactly like a tab
// that is not on screen. Three thin edge strips bring one piece of chrome
// back as an overlay above the surface, so switching tabs or picking an
// agent never leaves the mode; a control that names a panel — the Inspector
// toggle at the end of the tab strip — shows it the same way, without
// waiting for the pointer to find a strip (a hover is a peek, a click is a
// command).
//
// Study: VS Code / Cursor "Zen Mode" and Zed's zen mode. What we adapt:
// their single command that hides every panel and asks the OS window for
// real fullscreen, and their double-Escape exit. What we changed: Zen Mode
// centres the editor and leaves the chrome unreachable until you leave the
// mode; the owner asked for the sidebar to come back when the pointer
// touches the left edge, so the chrome is reachable *inside* the mode and
// the surface is never re-centred (a terminal must keep every column).
//
// Everything here is data in / data out so node:test can drive the whole
// decision table without a browser. The React side (desktop/src/lib/
// useFocusMode.js) owns the DOM: pointer events, the Fullscreen API, the
// keyboard lock that hands the browser's reserved chords (Ctrl+T, Ctrl+W,
// …) to the page while the mode is on — see browserChord.js — localStorage
// and the refit.

export const FOCUS_KEY = "picode-focus";
export const FOCUS_SEEN_KEY = "picode-focus-seen";

// Pointer dwell before a strip reveals its chrome. Short enough to feel
// like a hover, long enough that a pointer crossing the edge on its way
// somewhere else never flashes the sidebar.
export const DWELL_IN_MS = 120;
// Grace after the pointer leaves the revealed panel: a diagonal move from
// the strip to a row two levels down must not lose it (macOS menu logic).
export const DWELL_OUT_MS = 250;
// The strips themselves. 6px is the same hit width as the sidebar sizer.
export const EDGE_PX = 6;

export const ZONES = ["left", "top", "right"];

export function initialFocusState(prefs = {}) {
  return {
    on: !!(prefs && prefs.on),
    seen: !!(prefs && prefs.seen),
    // The zone currently revealed as an overlay, or null.
    reveal: null,
    // True when that reveal came from a control naming the panel instead of
    // from a pointer dwelling on a strip. A command is not undone by the
    // pointer moving away — the pointer never had to enter the panel — so a
    // pinned reveal leaves on the control, Escape or the mode, never on the
    // dwell-out.
    pinned: false,
    // A pointer dwelling on a strip that has not reached DWELL_IN_MS yet.
    armed: null,
    // A revealed panel the pointer has left, counting down DWELL_OUT_MS.
    closing: null,
    // The browser is in fullscreen *because this mode asked for it*. A
    // viewer who reloaded, or a browser that refused, leaves this false —
    // and then leaving browser fullscreen must not end the mode.
    fs: false,
    // Whether the rail was open when the mode started. The right strip
    // exists only then: a viewer who closed the rail did not ask for it.
    // A restored mode has no "before" to remember, so the caller passes
    // what the rail looks like now.
    railWasOpen: !!(prefs && prefs.on && prefs.railOpen),
  };
}

function started(state, ev) {
  return {
    ...state,
    on: true,
    reveal: null,
    armed: null,
    closing: null,
    pinned: false,
    fs: false,
    railWasOpen: !!(ev && ev.railOpen),
  };
}

function stopped(state) {
  return { ...state, on: false, reveal: null, armed: null, closing: null, pinned: false, fs: false, railWasOpen: false };
}

// Taking a revealed panel back down: every path that closes one shares this
// shape, so none of them can leave the dwell or the pin behind.
function closed(state) {
  return { ...state, reveal: null, armed: null, closing: null, pinned: false };
}

function at(ev) {
  const n = Number(ev && ev.at);
  return Number.isFinite(n) ? n : 0;
}

// The one reducer. Events:
//   enter | leave | toggle   { railOpen }   the menu row, the chord, Escape
//   unavailable                             the shell can no longer host it
//   point   { zone, inside, at }            pointer over a strip / a panel
//   tick    { at }                          resolves both dwells
//   reveal  { zone, pinned }                a control names a panel: show it
//                                           now (a null zone takes it down)
//   rail    { open }                        the shell put the rail on screen
//                                           or took it off, mode still on
//   escape                                  reveal first, then the mode
//   browser { active }                      document.fullscreenElement changed
//   resume                                  a real gesture after a reload:
//                                           the restored mode completes the
//                                           browser fullscreen + keyboard lock
//   seen                                    the first-run toast was shown
export function focusReduce(state, ev) {
  if (!state || !ev) return state;
  switch (ev.type) {
    case "toggle":
      return state.on ? stopped(state) : started(state, ev);
    case "enter":
      return state.on ? state : started(state, ev);
    case "leave":
    case "unavailable":
      return state.on ? stopped(state) : state;
    case "point": {
      if (!state.on) return state;
      if (state.reveal) {
        // The pointer keeps the panel while it is over the panel itself or
        // still over the strip that opened it.
        const keep = ev.inside === state.reveal || ev.zone === state.reveal;
        if (keep) return state.closing ? { ...state, closing: null } : state;
        // A pinned panel does not close behind the viewer's back: the
        // pointer never had to enter it (the Inspector toggle lives in the
        // tab strip), so the dwell-out would shut it on the first move. It
        // still does not hold the other strips hostage — a pointer can only
        // dwell on one, and a dwell replaces the pinned panel (see tick).
        if (state.pinned) {
          if (!ev.zone || (state.armed && state.armed.zone === ev.zone)) return state;
          return { ...state, armed: { zone: ev.zone, at: at(ev) } };
        }
        return state.closing ? state : { ...state, closing: { at: at(ev) } };
      }
      if (!ev.zone) return state.armed ? { ...state, armed: null } : state;
      if (state.armed && state.armed.zone === ev.zone) return state;
      return { ...state, armed: { zone: ev.zone, at: at(ev) } };
    }
    case "tick": {
      if (!state.on) return state;
      const now = at(ev);
      let next = state;
      if (next.armed && now - next.armed.at >= DWELL_IN_MS) {
        // A reveal the pointer asked for is the transient kind: the pointer
        // leaving it still ends it (pinned stays false here).
        next = { ...next, reveal: next.armed.zone, armed: null, closing: null, pinned: false };
      }
      if (next.closing && now - next.closing.at >= DWELL_OUT_MS) {
        next = closed(next);
      }
      return next;
    }
    // A control that names a panel — the Inspector toggle at the end of the
    // tab strip, whose rail the mode had otherwise made reachable only from
    // its own edge. The panel appears now instead of after 120ms of pointer
    // dwell, and stays until the same control takes it down, Escape clears
    // it or the mode ends.
    case "reveal": {
      if (!state.on) return state;
      if (!ev.zone) return state.reveal ? closed(state) : state;
      const next = { ...state, reveal: ev.zone, armed: null, closing: null, pinned: !!ev.pinned };
      // Naming the rail is also the assertion that it exists: the click can
      // land in the same tick that turns the rail on, before the shell has
      // reported it, and the right strip must exist from then on.
      if (ev.zone === "right" && !next.railWasOpen) next.railWasOpen = true;
      return next;
    }
    // The rail came on screen or left it while the mode is on — the toggle,
    // a tab picked from the revealed strip, a window that fits it again.
    // `railWasOpen` is the answer taken when the mode started, and the whole
    // right strip hangs off it, so the shell reports every change.
    case "rail": {
      if (!state.on) return state;
      const open = !!ev.open;
      if (open === state.railWasOpen) return state;
      const next = { ...state, railWasOpen: open };
      if (open) return next;
      // A rail that left the screen takes its own reveal (and its pending
      // dwell) with it, and leaves the other panels' alone.
      const onRail = state.reveal === "right" || (state.armed && state.armed.zone === "right");
      return onRail ? closed(next) : next;
    }
    case "escape": {
      if (!state.on) return state;
      if (state.reveal) return closed(state);
      return stopped(state);
    }
    case "browser": {
      if (!state.on) return state;
      if (ev.active) return state.fs ? state : { ...state, fs: true };
      // The browser left fullscreen. Only a fullscreen this mode asked for
      // ties the two together: a reloaded session (no gesture, so no
      // request) and a browser that refused both stay in the in-app mode.
      return state.fs ? stopped(state) : state;
    }
    case "seen":
      return state.seen ? state : { ...state, seen: true };
    // A reload restores the mode without the browser part (no gesture, so
    // no request); the wiring watches for the first real input and reports
    // it as "resume". The state does not change here — the wiring reads the
    // intent and asks the browser for fullscreen in that same handler.
    case "resume":
      return state;
    default:
      return state;
  }
}

// What the caller must do to the browser between two states. Kept apart
// from the reducer because it is the one part that needs a user gesture:
// requestFullscreen has to run inside the click or keydown handler, never
// in an effect that observes the new state. A transition the browser
// itself caused asks for nothing — it has already moved.
// "resume" asks for the browser part exactly when the restored mode is
// still missing it (mode on, browser never went fullscreen): the gesture
// carrying the resume event is the activation the request needs.
export function browserIntent(prev, next, ev) {
  if (!prev || !next) return "none";
  if (ev && ev.type === "browser") return "none";
  if (ev && ev.type === "resume") return next.on && !next.fs ? "request" : "none";
  if (!prev.on && next.on) return "request";
  if (prev.on && !next.on && prev.fs) return "exit";
  return "none";
}

// Which chrome each state hides. "shown" = in the layout as usual,
// "hidden" = mounted but invisible, "revealed" = floating above the
// surface (it never pushes the layout, so a terminal keeps its columns).
export function chromeState(state) {
  if (!state || !state.on) return { sidebar: "shown", tabs: "shown", rail: "shown" };
  return {
    sidebar: state.reveal === "left" ? "revealed" : "hidden",
    tabs: state.reveal === "top" ? "revealed" : "hidden",
    rail: state.railWasOpen && state.reveal === "right" ? "revealed" : "hidden",
  };
}

// The classes the shell wears. One string so App stays declarative.
export function shellClasses(state) {
  if (!state || !state.on) return "";
  return state.reveal ? `focus-on focus-reveal-${state.reveal}` : "focus-on";
}

// Which strips exist right now. The right one only when the rail was open
// when the mode started.
export function activeZones(state) {
  if (!state || !state.on) return [];
  return state.railWasOpen ? ZONES.slice() : ZONES.filter((z) => z !== "right");
}

// The hot-zone hit test. `view` is the viewport; `rail` says whether the
// right strip exists at all. The top strip wins in a corner: it carries
// the Leave control, so it must always be reachable.
export function hotZone(point, view, opts = {}) {
  if (!point || !view) return null;
  const edge = Number(opts.edge) > 0 ? Number(opts.edge) : EDGE_PX;
  const w = Number(view.width) || 0;
  const h = Number(view.height) || 0;
  const x = Number(point.x);
  const y = Number(point.y);
  if (!Number.isFinite(x) || !Number.isFinite(y)) return null;
  if (x < 0 || y < 0 || x > w || y > h) return null;
  if (y <= edge) return "top";
  if (x <= edge) return "left";
  if (opts.rail && w > 0 && x >= w - edge) return "right";
  return null;
}

// Where the mode can live. Below 768px the desktop shell is a single
// column with no sidebar and no rail (ADR-0072) — there is no chrome to
// hide. On a page route (Preferences, Agent CLIs, …) the whole tab strip
// is inside the hidden workspace view, so the top reveal would carry no
// tabs and no way out: the mode belongs to the tabs, exactly as the owner
// asked for it, and navigating to a page leaves it.
export function focusAvailable(narrow, onPane) {
  return !narrow && !onPane;
}

export function focusRowLabel(on) {
  return on ? "Leave fullscreen" : "Fullscreen";
}

// The one first-run line: what happened, and how to get out. Esc reaches
// the page (keyboard lock), so it still leaves the mode — unless a terminal
// pane owns it, where the guest gets it and holding Esc is the way out.
export const FOCUS_FIRST_RUN =
  "Fullscreen. Left edge shows the sidebar, top edge the tabs. Esc leaves — hold Esc inside a terminal.";

function safeStorage() {
  try { return typeof localStorage !== "undefined" ? localStorage : null; } catch { return null; }
}

// Per-viewer, not navigable — the same shape as the Inspector's
// preferences (picode-inspector-*): localStorage, no hash route.
export function readFocusPrefs(storage = safeStorage()) {
  let on = false, seen = false;
  try {
    on = !!storage && storage.getItem(FOCUS_KEY) === "1";
    seen = !!storage && storage.getItem(FOCUS_SEEN_KEY) === "1";
  } catch { /* private mode, quota — defaults are fine */ }
  return { on, seen };
}

export function writeFocusPrefs(prefs, storage = safeStorage()) {
  if (!prefs || !storage) return;
  try {
    if (prefs.on === true || prefs.on === false) storage.setItem(FOCUS_KEY, prefs.on ? "1" : "0");
    if (prefs.seen === true) storage.setItem(FOCUS_SEEN_KEY, "1");
  } catch { /* preference is optional */ }
}
