import { test } from "node:test";
import assert from "node:assert/strict";
import { PRESETS, ZOOM_STEPS, stepZoom, zoomLabel, activePreset, presetTitle } from "./responsive.js";

test("the preset ladder is ascending and matches the shell's", () => {
  assert.ok(PRESETS.length >= 3);
  // The shell normalizes against the same widths (desktop-shell/src/responsive.rs).
  assert.deepEqual(PRESETS.map((p) => p.width), [390, 768, 1024, 1280]);
  for (let i = 1; i < PRESETS.length; i += 1) {
    assert.ok(PRESETS[i].width > PRESETS[i - 1].width);
  }
});

test("every preset names a device class", () => {
  for (const p of PRESETS) {
    assert.equal(typeof p.name, "string");
    assert.ok(p.name.length > 0);
    assert.match(presetTitle(p), new RegExp(`^${p.name} · ${p.width}px$`));
  }
});

test("the zoom ladder is the reference's steps inside WebView2's limits", () => {
  assert.equal(ZOOM_STEPS[0], 0.25);
  assert.equal(ZOOM_STEPS[ZOOM_STEPS.length - 1], 5);
  for (let i = 1; i < ZOOM_STEPS.length; i += 1) {
    assert.ok(ZOOM_STEPS[i] > ZOOM_STEPS[i - 1]);
  }
});

test("stepping walks the ladder and pins at both ends", () => {
  assert.equal(stepZoom(1, 1), 1.1);
  assert.equal(stepZoom(1, -1), 0.9);
  assert.equal(stepZoom(0.9, -1), 0.8);
  assert.equal(stepZoom(5, 1), 5);
  assert.equal(stepZoom(0.25, -1), 0.25);
});

test("a zoom between steps steps from the rung at or above it", () => {
  assert.equal(stepZoom(1.05, 1), 1.25);
  assert.equal(stepZoom(0.31, -1), 0.25);
});

test("the percent label rounds the way the zoom row prints", () => {
  assert.equal(zoomLabel(1), "100%");
  assert.equal(zoomLabel(0.33), "33%");
  assert.equal(zoomLabel(1.5), "150%");
});

test("only a stored width highlights a preset", () => {
  assert.equal(activePreset(0), null);
  assert.equal(activePreset(undefined), null);
  assert.equal(activePreset(Number.NaN), null);
  assert.equal(activePreset(768), PRESETS[1]);
  assert.equal(activePreset(390), PRESETS[0]);
});
