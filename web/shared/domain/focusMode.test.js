import assert from "node:assert/strict";
import { test } from "node:test";
import {
  DWELL_IN_MS,
  DWELL_OUT_MS,
  EDGE_PX,
  FOCUS_KEY,
  FOCUS_SEEN_KEY,
  activeZones,
  chromeState,
  focusAvailable,
  focusReduce,
  focusRowLabel,
  hotZone,
  initialFocusState,
  readFocusPrefs,
  shellClasses,
  writeFocusPrefs,
} from "./focusMode.js";

// Drives a list of events through the reducer, returning the last state.
function run(state, events) {
  return events.reduce((s, ev) => focusReduce(s, ev), state);
}

function onState(extra = {}) {
  return { ...focusReduce(initialFocusState(), { type: "enter", railOpen: true }), ...extra };
}

function fakeStorage(seed = {}) {
  const map = new Map(Object.entries(seed));
  return {
    getItem: (k) => (map.has(k) ? map.get(k) : null),
    setItem: (k, v) => map.set(k, String(v)),
    dump: () => Object.fromEntries(map),
  };
}

// --- row: off + menu row / shortcut -> on ---------------------------------

test("off: the menu row turns the mode on and hides every piece of chrome", () => {
  const next = focusReduce(initialFocusState(), { type: "toggle", railOpen: true });
  assert.equal(next.on, true);
  assert.deepEqual(chromeState(next), { sidebar: "hidden", tabs: "hidden", rail: "hidden" });
  assert.equal(shellClasses(next), "focus-on");
});

test("off: entering leaves the browser window alone, and the first run is unseen", () => {
  const before = initialFocusState();
  const after = focusReduce(before, { type: "enter", railOpen: false });
  assert.equal(after.on, true);
  assert.equal(after.seen, false);
  assert.equal(focusReduce(after, { type: "seen" }).seen, true);
  // The state carries PiCode's chrome and nothing about the window: no
  // half of this mode lives in the browser, so none of it can fall out of
  // sync with the browser. A new key here is a new row owed a test.
  assert.deepEqual(Object.keys(after).sort(),
    ["armed", "closing", "on", "pinned", "railWasOpen", "reveal", "seen"]);
});

test("off: chrome is shown and no strip exists", () => {
  const off = initialFocusState();
  assert.deepEqual(chromeState(off), { sidebar: "shown", tabs: "shown", rail: "shown" });
  assert.deepEqual(activeZones(off), []);
  assert.equal(shellClasses(off), "");
});

// --- row: on + pointer dwells 120 ms on the left strip -> sidebar overlay --

test("on: 120 ms of dwell on the left strip reveals the sidebar as an overlay", () => {
  const next = run(onState(), [
    { type: "point", zone: "left", inside: null, at: 1000 },
    { type: "tick", at: 1000 + DWELL_IN_MS },
  ]);
  assert.equal(next.reveal, "left");
  assert.deepEqual(chromeState(next), { sidebar: "revealed", tabs: "hidden", rail: "hidden" });
  assert.equal(shellClasses(next), "focus-on focus-reveal-left");
});

test("on: the top strip reveals the tab strip, the right strip the rail", () => {
  const top = run(onState(), [{ type: "point", zone: "top", inside: null, at: 0 }, { type: "tick", at: DWELL_IN_MS }]);
  assert.deepEqual(chromeState(top), { sidebar: "hidden", tabs: "revealed", rail: "hidden" });
  const right = run(onState(), [{ type: "point", zone: "right", inside: null, at: 0 }, { type: "tick", at: DWELL_IN_MS }]);
  assert.deepEqual(chromeState(right), { sidebar: "hidden", tabs: "hidden", rail: "revealed" });
});

// --- row: on + pointer leaves for 250 ms -> overlay out -------------------

test("on, sidebar revealed: 250 ms away from the sidebar and the strip closes it", () => {
  const revealed = run(onState(), [
    { type: "point", zone: "left", inside: null, at: 0 },
    { type: "tick", at: DWELL_IN_MS },
  ]);
  assert.equal(revealed.reveal, "left");
  const leaving = focusReduce(revealed, { type: "point", zone: null, inside: null, at: 500 });
  assert.equal(leaving.reveal, "left", "the panel stays through the grace window");
  assert.equal(focusReduce(leaving, { type: "tick", at: 500 + DWELL_OUT_MS - 1 }).reveal, "left");
  const closed = focusReduce(leaving, { type: "tick", at: 500 + DWELL_OUT_MS });
  assert.equal(closed.reveal, null);
  assert.equal(closed.on, true, "the mode outlives the reveal");
});

// --- row: pointer moves from the strip into the sidebar -> stays ----------

test("on, sidebar revealed: moving from the strip into the sidebar keeps it", () => {
  let s = run(onState(), [{ type: "point", zone: "left", inside: null, at: 0 }, { type: "tick", at: DWELL_IN_MS }]);
  // Off the 6px strip, but inside the panel it opened.
  s = focusReduce(s, { type: "point", zone: null, inside: "left", at: 300 });
  assert.equal(s.closing, null);
  s = focusReduce(s, { type: "tick", at: 300 + DWELL_OUT_MS * 4 });
  assert.equal(s.reveal, "left");
});

test("on, sidebar revealed: coming back before the grace expires cancels the close", () => {
  let s = run(onState(), [{ type: "point", zone: "left", inside: null, at: 0 }, { type: "tick", at: DWELL_IN_MS }]);
  s = focusReduce(s, { type: "point", zone: null, inside: null, at: 200 });
  assert.ok(s.closing);
  s = focusReduce(s, { type: "point", zone: null, inside: "left", at: 300 });
  assert.equal(s.closing, null);
  s = focusReduce(s, { type: "tick", at: 900 });
  assert.equal(s.reveal, "left");
});

// --- row: pointer crosses the strip without dwelling -> nothing -----------

test("on: a pointer crossing the strip without dwelling reveals nothing", () => {
  const next = run(onState(), [
    { type: "point", zone: "left", inside: null, at: 0 },
    { type: "point", zone: null, inside: null, at: DWELL_IN_MS - 40 },
    { type: "tick", at: DWELL_IN_MS + 200 },
  ]);
  assert.equal(next.reveal, null);
  assert.equal(next.armed, null);
});

test("on: sliding along the edge from one strip to another restarts the dwell", () => {
  let s = focusReduce(onState(), { type: "point", zone: "top", inside: null, at: 0 });
  s = focusReduce(s, { type: "point", zone: "left", inside: null, at: 100 });
  assert.equal(s.armed.zone, "left");
  assert.equal(s.armed.at, 100);
  assert.equal(focusReduce(s, { type: "tick", at: 100 + DWELL_IN_MS - 1 }).reveal, null);
  assert.equal(focusReduce(s, { type: "tick", at: 100 + DWELL_IN_MS }).reveal, "left");
});

// --- row: Escape with a reveal open -> reveal closes, mode stays ----------

test("on, a reveal open: Escape closes the reveal and keeps the mode", () => {
  const revealed = run(onState(), [{ type: "point", zone: "top", inside: null, at: 0 }, { type: "tick", at: DWELL_IN_MS }]);
  const next = focusReduce(revealed, { type: "escape" });
  assert.equal(next.reveal, null);
  assert.equal(next.on, true);
});

// --- row: Escape with no reveal -> the mode leaves ------------------------

test("on, no reveal: Escape leaves the mode", () => {
  const next = focusReduce(onState(), { type: "escape" });
  assert.equal(next.on, false);
  assert.deepEqual(chromeState(next), { sidebar: "shown", tabs: "shown", rail: "shown" });
});

// --- row: Escape is two steps, and both of them are ours ------------------

test("on, a reveal open: two Escapes close the strip and then leave", () => {
  // Both presses reach the app: nothing in front of it (there is no browser
  // fullscreen any more) eats the first one, so the step that closes the
  // strip is never skipped.
  const revealed = run(onState(), [{ type: "point", zone: "left", inside: null, at: 0 }, { type: "tick", at: DWELL_IN_MS }]);
  assert.equal(chromeState(revealed).sidebar, "revealed");
  const first = focusReduce(revealed, { type: "escape" });
  assert.equal(first.on, true, "the first Escape only closes the strip");
  assert.equal(first.reveal, null);
  const second = focusReduce(first, { type: "escape" });
  assert.equal(second.on, false, "the second Escape leaves the mode");
});

// --- row: tab switched from the revealed strip -> mode stays --------------

test("on: switching tabs from the revealed strip keeps the mode", () => {
  // A tab click is a pointer inside the revealed strip; nothing about the
  // mode changes, and the strip closes on its own once the pointer leaves.
  let s = run(onState(), [{ type: "point", zone: "top", inside: null, at: 0 }, { type: "tick", at: DWELL_IN_MS }]);
  s = focusReduce(s, { type: "point", zone: null, inside: "top", at: 200 });
  assert.equal(s.on, true);
  assert.equal(s.reveal, "top");
  s = run(s, [{ type: "point", zone: null, inside: null, at: 400 }, { type: "tick", at: 400 + DWELL_OUT_MS }]);
  assert.equal(s.on, true);
  assert.equal(s.reveal, null, "no stuck overlay over the new surface");
});

test("leaving with a strip revealed leaves no stuck overlay", () => {
  const revealed = run(onState(), [{ type: "point", zone: "top", inside: null, at: 0 }, { type: "tick", at: DWELL_IN_MS }]);
  const next = focusReduce(revealed, { type: "toggle" });
  assert.equal(next.on, false);
  assert.equal(next.reveal, null);
  assert.equal(shellClasses(next), "");
});

// --- row: page reloaded -> restored, no browser fullscreen ----------------

test("a reload restores the whole mode from localStorage", () => {
  const prefs = readFocusPrefs(fakeStorage({ [FOCUS_KEY]: "1", [FOCUS_SEEN_KEY]: "1" }));
  assert.deepEqual(prefs, { on: true, seen: true });
  const restored = initialFocusState({ ...prefs, railOpen: true });
  assert.equal(restored.on, true);
  // Nothing is left over for a gesture to complete: the mode is only the
  // chrome it hides, and a restored one is the same state as a fresh one.
  assert.deepEqual(restored, { ...onState(), seen: true });
  // The top strip is still reachable: the mode is on and the zone exists.
  assert.ok(activeZones(restored).includes("top"));
  const next = run(restored, [{ type: "point", zone: "top", inside: null, at: 0 }, { type: "tick", at: DWELL_IN_MS }]);
  assert.equal(chromeState(next).tabs, "revealed");
});

test("preferences round-trip through storage", () => {
  const store = fakeStorage();
  writeFocusPrefs({ on: true, seen: true }, store);
  assert.deepEqual(store.dump(), { [FOCUS_KEY]: "1", [FOCUS_SEEN_KEY]: "1" });
  writeFocusPrefs({ on: false }, store);
  assert.deepEqual(readFocusPrefs(store), { on: false, seen: true });
  // A storage that throws (private mode) never breaks a read or a write.
  const hostile = { getItem() { throw new Error("denied"); }, setItem() { throw new Error("denied"); } };
  assert.deepEqual(readFocusPrefs(hostile), { on: false, seen: false });
  assert.doesNotThrow(() => writeFocusPrefs({ on: true }, hostile));
  assert.deepEqual(readFocusPrefs(null), { on: false, seen: false });
});

// --- row: rail was closed before -> the right strip reveals nothing -------

test("on, rail closed before the mode started: the right strip does not exist", () => {
  const s = focusReduce(initialFocusState(), { type: "enter", railOpen: false });
  assert.deepEqual(activeZones(s), ["left", "top"]);
  assert.equal(hotZone({ x: 1198, y: 400 }, { width: 1200, height: 800 }, { rail: false }), null);
  const next = run(s, [{ type: "point", zone: null, inside: null, at: 0 }, { type: "tick", at: DWELL_IN_MS }]);
  assert.equal(next.reveal, null);
  assert.equal(chromeState(next).rail, "hidden");
});

test("on, rail open before the mode started: all three strips exist", () => {
  assert.deepEqual(activeZones(onState()), ["left", "top", "right"]);
});

// --- row: the Inspector toggle -> the rail, without waiting for the edge --

test("on, rail closed before the mode started: the toggle shows the rail now", () => {
  const s = focusReduce(initialFocusState(), { type: "enter", railOpen: false });
  const next = focusReduce(s, { type: "reveal", zone: "right", pinned: true });
  assert.equal(next.railWasOpen, true, "showing it is the assertion that it exists");
  assert.equal(next.reveal, "right");
  assert.equal(next.pinned, true);
  assert.deepEqual(activeZones(next), ["left", "top", "right"], "the edge works from then on");
  assert.deepEqual(chromeState(next), { sidebar: "hidden", tabs: "hidden", rail: "revealed" });
});

test("on: a panel a control opened does not close behind the pointer", () => {
  const opened = focusReduce(onState(), { type: "reveal", zone: "right", pinned: true });
  // The pointer is elsewhere from the first move and never enters the rail:
  // the toggle lives in the tab strip, so it never had to.
  const away = focusReduce(opened, { type: "point", zone: null, inside: null, at: 10 });
  assert.equal(away.closing, null);
  assert.equal(focusReduce(away, { type: "tick", at: 10 + DWELL_OUT_MS }).reveal, "right");
});

test("on, pinned: a dwell on another strip replaces it, transient again", () => {
  const opened = focusReduce(onState(), { type: "reveal", zone: "right", pinned: true });
  const armed = focusReduce(opened, { type: "point", zone: "top", inside: null, at: 0 });
  assert.equal(armed.armed.zone, "top", "the pinned rail does not block the tabs");
  const tabs = focusReduce(armed, { type: "tick", at: DWELL_IN_MS });
  assert.equal(tabs.reveal, "top");
  assert.equal(tabs.pinned, false);
});

test("on: the control takes the panel down, and Escape is the other way out", () => {
  const opened = focusReduce(onState(), { type: "reveal", zone: "right", pinned: true });
  const down = focusReduce(opened, { type: "reveal", zone: null });
  assert.equal(down.reveal, null);
  assert.equal(down.pinned, false);
  assert.equal(down.on, true, "taking the panel down is not leaving the mode");
  const escaped = focusReduce(opened, { type: "escape" });
  assert.equal(escaped.reveal, null);
  assert.equal(escaped.pinned, false);
  assert.equal(escaped.on, true, "the first Escape closes the reveal");
});

test("on: the shell can put the rail on screen in the middle of the mode", () => {
  const closed = focusReduce(initialFocusState(), { type: "enter", railOpen: false });
  assert.deepEqual(activeZones(closed), ["left", "top"]);
  const opened = focusReduce(closed, { type: "rail", open: true });
  assert.deepEqual(activeZones(opened), ["left", "top", "right"]);
  assert.equal(focusReduce(opened, { type: "rail", open: false }).railWasOpen, false);
});

test("on: a rail that leaves the screen takes its own reveal, not the others", () => {
  const sidebar = run(onState(), [{ type: "point", zone: "left", inside: null, at: 0 }, { type: "tick", at: DWELL_IN_MS }]);
  assert.equal(sidebar.reveal, "left");
  assert.equal(focusReduce(sidebar, { type: "rail", open: false }).reveal, "left");
  const rail = focusReduce(onState(), { type: "reveal", zone: "right", pinned: true });
  const gone = focusReduce(rail, { type: "rail", open: false });
  assert.equal(gone.reveal, null);
  assert.equal(gone.railWasOpen, false);
  assert.deepEqual(activeZones(gone), ["left", "top"]);
});

// --- the hot-zone hit test ------------------------------------------------

test("hotZone: the edges, the corner and the middle", () => {
  const view = { width: 1200, height: 800 };
  const opts = { rail: true };
  assert.equal(hotZone({ x: 0, y: 400 }, view, opts), "left");
  assert.equal(hotZone({ x: EDGE_PX, y: 400 }, view, opts), "left");
  assert.equal(hotZone({ x: EDGE_PX + 1, y: 400 }, view, opts), null);
  assert.equal(hotZone({ x: 600, y: 0 }, view, opts), "top");
  assert.equal(hotZone({ x: 600, y: EDGE_PX + 1 }, view, opts), null);
  assert.equal(hotZone({ x: 1200, y: 400 }, view, opts), "right");
  assert.equal(hotZone({ x: 1200 - EDGE_PX - 1, y: 400 }, view, opts), null);
  // The top-left corner belongs to the top strip: it carries Leave.
  assert.equal(hotZone({ x: 0, y: 0 }, view, opts), "top");
  assert.equal(hotZone({ x: 600, y: 400 }, view, opts), null);
  // Outside the viewport (a pointer leaving the window) is no zone.
  assert.equal(hotZone({ x: -4, y: 400 }, view, opts), null);
  assert.equal(hotZone({ x: 600, y: 900 }, view, opts), null);
  assert.equal(hotZone(null, view, opts), null);
  assert.equal(hotZone({ x: NaN, y: 2 }, view, opts), null);
});

// --- availability, labels, guards ----------------------------------------

test("the mode is not offered in the narrow shell or on a page route", () => {
  assert.equal(focusAvailable(false, false), true);
  assert.equal(focusAvailable(true, false), false, "narrow shell has no chrome to hide");
  assert.equal(focusAvailable(false, true), false, "a page route has no tab strip to reveal");
  const next = focusReduce(onState(), { type: "unavailable" });
  assert.equal(next.on, false);
  assert.equal(next.reveal, null);
});

test("the menu row says what the click will do", () => {
  assert.equal(focusRowLabel(false), "Fullscreen");
  assert.equal(focusRowLabel(true), "Leave fullscreen");
});

test("events are ignored while the mode is off", () => {
  const off = initialFocusState();
  for (const ev of [
    { type: "point", zone: "left", inside: null, at: 0 },
    { type: "tick", at: 9999 },
    { type: "reveal", zone: "right", pinned: true },
    { type: "rail", open: true },
    { type: "escape" },
    { type: "leave" },
  ]) assert.deepEqual(focusReduce(off, ev), off, ev.type);
  assert.equal(focusReduce(off, null), off);
  assert.equal(focusReduce(off, { type: "nonsense" }), off);
  // The browser half is gone, not merely unused: what it used to send is
  // now an event the table has never heard of, on or off.
  for (const ev of [{ type: "browser", active: true }, { type: "browser", active: false }, { type: "resume" }]) {
    assert.equal(focusReduce(off, ev), off, ev.type + " off");
    const on = onState();
    assert.equal(focusReduce(on, ev), on, ev.type + " on");
  }
});

test("entering twice does not restart the mode", () => {
  const s = run(onState(), [{ type: "point", zone: "left", inside: null, at: 0 }, { type: "tick", at: DWELL_IN_MS }]);
  assert.equal(focusReduce(s, { type: "enter", railOpen: true }), s);
});

test("a restored mode reads the right strip from the rail as it stands now", () => {
  assert.deepEqual(activeZones(initialFocusState({ on: true, railOpen: true })), ["left", "top", "right"]);
  assert.deepEqual(activeZones(initialFocusState({ on: true, railOpen: false })), ["left", "top"]);
  // An open rail on a viewer who is not in the mode changes nothing.
  assert.equal(initialFocusState({ on: false, railOpen: true }).railWasOpen, false);
});

test("a tick carrying exactly the deadline resolves the dwell", () => {
  // The hook passes the deadline, not the clock: setTimeout(120) can come
  // back at 118.9 ms of performance.now(), and a tick that resolved
  // nothing would leave the dwell armed with no second timer behind it.
  const armed = focusReduce(onState(), { type: "point", zone: "left", inside: null, at: 1000 });
  assert.equal(focusReduce(armed, { type: "tick", at: 1000 + DWELL_IN_MS - 0.1 }).reveal, null);
  assert.equal(focusReduce(armed, { type: "tick", at: 1000 + DWELL_IN_MS }).reveal, "left");
});
