import assert from "node:assert/strict";
import { test } from "node:test";
import { suppressKeyPasteFor, keyPasteSuppressed, resetPasteClaim } from "./termPasteClaim.js";

test("a claim suppresses only the claimed element", () => {
  resetPasteClaim();
  const a = { id: "a" };
  const b = { id: "b" };
  assert.equal(keyPasteSuppressed(a, 1000), false);
  suppressKeyPasteFor(a, 1000);
  assert.equal(keyPasteSuppressed(a, 1001), true);
  assert.equal(keyPasteSuppressed(b, 1001), false);
  assert.equal(keyPasteSuppressed(null, 1001), false);
});

test("a stale claim suppresses nothing", () => {
  resetPasteClaim();
  const a = { id: "a" };
  suppressKeyPasteFor(a, 1000);
  assert.equal(keyPasteSuppressed(a, 1000 + 1500), false);
  assert.equal(keyPasteSuppressed(a, 1000 + 10_000), false);
});

test("reset clears the claim", () => {
  const a = { id: "a" };
  suppressKeyPasteFor(a, 1000);
  resetPasteClaim();
  assert.equal(keyPasteSuppressed(a, 1001), false);
});
