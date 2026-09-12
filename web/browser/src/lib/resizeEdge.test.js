import test from "node:test";
import assert from "node:assert/strict";
import { clampWidth, dragWidth, stepWidth } from "./resizeEdge.js";

test("clampWidth bounds and rounds", () => {
  assert.equal(clampWidth(300, 260, 560), 300);
  assert.equal(clampWidth(10, 260, 560), 260);
  assert.equal(clampWidth(900, 260, 560), 560);
  assert.equal(clampWidth(300.6, 260, 560), 301);
  assert.equal(clampWidth("x", 260, 560), 260);
});

test("dragWidth grows toward the pointer from the panel's edge", () => {
  // Handle on the right edge (left sidebar): moving right widens.
  assert.equal(dragWidth(244, 100, 140, "right"), 284);
  assert.equal(dragWidth(244, 100, 60, "right"), 204);
  // Handle on the left edge (right rail): moving LEFT widens.
  assert.equal(dragWidth(320, 900, 860, "left"), 360);
  assert.equal(dragWidth(320, 900, 940, "left"), 280);
});

test("stepWidth moves the handle with the arrow keys", () => {
  assert.equal(stepWidth(320, "ArrowLeft", 20, "left"), 340);
  assert.equal(stepWidth(320, "ArrowRight", 20, "left"), 300);
  assert.equal(stepWidth(244, "ArrowRight", 20, "right"), 264);
  assert.equal(stepWidth(244, "ArrowLeft", 20, "right"), 224);
  assert.equal(stepWidth(244, "Enter"), null);
});
