import test from "node:test";
import assert from "node:assert/strict";
import { nativeApps, appTile, nativeSurfaceFor } from "./nativeApps.js";

const Canvas = () => null;
const desktop = nativeApps({ canvas: Canvas });
const primitives = { id: "inbox", name: "Inbox", apiVersion: 1, surface: "" };
const native = { id: "canvas", name: "Canvas", apiVersion: 1, surface: "native" };

test("nativeApps: explicit assembly, frozen, ids is the gate's Set", () => {
  assert.equal(desktop.get("canvas"), Canvas);
  assert.equal(desktop.get("nope"), null);
  assert.equal(desktop.has("canvas"), true);
  assert.deepEqual([...desktop.ids], ["canvas"]);
  assert.ok(Object.isFrozen(desktop));
  assert.deepEqual([...nativeApps().ids], []);
});

// ADR-0109's decision table, the desktop rows (tile column).
test("appTile: primitives app is enabled with or without a registry", () => {
  assert.deepEqual(appTile(primitives, desktop), { ok: true, title: "Inbox" });
  assert.deepEqual(appTile(primitives, null), { ok: true, title: "Inbox" });
});

test("appTile: native app registered in this shell is enabled", () => {
  assert.deepEqual(appTile(native, desktop), { ok: true, title: "Canvas" });
});

test("appTile: native app this build did not compile in needs a newer PiCode", () => {
  const tile = appTile(native, nativeApps({}));
  assert.equal(tile.ok, false);
  assert.match(tile.title, /needs a newer PiCode/);
  assert.match(tile.title, /no Canvas surface/);
  assert.equal(appTile(native, null).ok, false);
});

test("appTile: unknown surface value is unsupported even when the id is registered", () => {
  const tile = appTile({ ...native, surface: "hologram" }, desktop);
  assert.equal(tile.ok, false);
  assert.match(tile.title, /needs a newer PiCode/);
  assert.match(tile.title, /hologram/);
});

test("appTile: apiVersion other than 1 is unsupported as today", () => {
  const tile = appTile({ ...primitives, apiVersion: 9 }, desktop);
  assert.equal(tile.ok, false);
  assert.match(tile.title, /speaks v9/);
  assert.equal(appTile({ ...native, apiVersion: 2 }, desktop).ok, false);
});

// The open column: which component the tab mounts.
test("nativeSurfaceFor: the registered component for a native manifest, null otherwise", () => {
  assert.equal(nativeSurfaceFor(native, desktop), Canvas);
  assert.equal(nativeSurfaceFor(primitives, desktop), null, "primitives → AppSurface");
  assert.equal(nativeSurfaceFor({ ...native, id: "other" }, desktop), null, "not registered → AppSurface's honest line");
  assert.equal(nativeSurfaceFor({ ...native, surface: "hologram" }, desktop), null);
  assert.equal(nativeSurfaceFor(null, desktop), null, "manifest not loaded yet");
  assert.equal(nativeSurfaceFor(native, null), null);
});
