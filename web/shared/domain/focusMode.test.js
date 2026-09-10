import assert from "node:assert/strict";
import { test } from "node:test";
import {
  DWELL_IN_MS,
  DWELL_OUT_MS,
  EDGE_PX,
  FOCUS_KEY,
  FOCUS_SEEN_KEY,
  activeZones,
  browserIntent,
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

test("off: entering asks the browser for fullscreen, and the first run is unseen", () => {
  const before = initialFocusState();
  const after = focusReduce(before, { type: "enter", railOpen: false });
  assert.equal(browserIntent(before, after), "request");
  assert.equal(after.seen, false);
  assert.equal(focusReduce(after, { type: "seen" }).seen, true);
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
  assert.equal(browserIntent(revealed, next), "none", "the browser stays fullscreen");
});

// --- row: Escape with no reveal -> mode leaves, browser fullscreen exits --

test("on, no reveal: Escape leaves the mode and exits browser fullscreen", () => {
  const inFs = focusReduce(onState(), { type: "browser", active: true });
  assert.equal(inFs.fs, true);
  const next = focusReduce(inFs, { type: "escape" });
  assert.equal(next.on, false);
  assert.equal(browserIntent(inFs, next), "exit");
  assert.deepEqual(chromeState(next), { sidebar: "shown", tabs: "shown", rail: "shown" });
});

// --- row: browser leaves fullscreen by itself -> mode leaves --------------

test("on: the browser leaving fullscreen on its own ends the mode", () => {
  const inFs = focusReduce(onState(), { type: "browser", active: true });
  const ev = { type: "browser", active: false };
  const next = focusReduce(inFs, ev);
  assert.equal(next.on, false);
  assert.deepEqual(chromeState(next), { sidebar: "shown", tabs: "shown", rail: "shown" });
  assert.equal(browserIntent(inFs, next, ev), "none", "the browser is already out; nothing to exit");
});

// --- row: resume after a reload -> completes the browser part ------------

test("resume with the mode on and no browser fullscreen asks for it", () => {
  const restored = initialFocusState({ on: true });
  assert.equal(restored.on, true);
  assert.equal(restored.fs, false);
  assert.equal(browserIntent(restored, restored, { type: "resume" }), "request");
});

test("resume is silent once the browser part is already there", () => {
  const inFs = focusReduce(onState(), { type: "browser", active: true });
  assert.equal(browserIntent(inFs, inFs, { type: "resume" }), "none");
});

test("resume says nothing when the mode is off", () => {
  const off = focusReduce(initialFocusState(), { type: "leave" });
  assert.equal(browserIntent(off, off, { type: "resume" }), "none");
});

test("the reducer leaves every state untouched on resume — the gesture is the change", () => {
  const restored = initialFocusState({ on: true });
  assert.equal(focusReduce(restored, { type: "resume" }), restored);
});

// --- row: requestFullscreen rejected -> mode stays on ---------------------

test("on: a browser that never granted fullscreen keeps the in-app mode", () => {
  // No `browser {active:true}` ever arrives — the request was rejected.
  const s = onState();
  assert.equal(s.fs, false);
  const next = focusReduce(s, { type: "browser", active: false });
  assert.equal(next.on, true, "the mode does not fail because the browser refused");
  assert.equal(next.reveal, null);
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

test("a reload restores the mode from localStorage without browser fullscreen", () => {
  const prefs = readFocusPrefs(fakeStorage({ [FOCUS_KEY]: "1", [FOCUS_SEEN_KEY]: "1" }));
  assert.deepEqual(prefs, { on: true, seen: true });
  const restored = initialFocusState({ ...prefs, railOpen: true });
  assert.equal(restored.on, true);
  assert.equal(restored.fs, false, "a reload has no user gesture, so no fullscreen was requested");
  // The top strip is still reachable: the mode is on and the zone exists.
  assert.ok(activeZones(restored).includes("top"));
  const next = run(restored, [{ type: "point", zone: "top", inside: null, at: 0 }, { type: "tick", at: DWELL_IN_MS }]);
  assert.equal(chromeState(next).tabs, "revealed");
});

test("a restored mode is not ended by a stray fullscreenchange", () => {
  const restored = initialFocusState({ on: true, seen: true });
  assert.equal(focusReduce(restored, { type: "browser", active: false }).on, true);
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
    { type: "escape" },
    { type: "browser", active: true },
    { type: "leave" },
  ]) assert.deepEqual(focusReduce(off, ev), off, ev.type);
  assert.equal(focusReduce(off, null), off);
  assert.equal(focusReduce(off, { type: "nonsense" }), off);
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
