import { test } from "node:test";
import assert from "node:assert/strict";
import {
  CANVAS_PATTERNS,
  CANVAS_PATTERN_EVENT,
  DEFAULT_CANVAS_PATTERN,
  persistCanvasPattern,
  readCanvasPattern,
} from "./canvasPattern.js";

function memoryStorage(seed = {}, { throws = false } = {}) {
  const mem = { ...seed };
  const fired = [];
  globalThis.localStorage = {
    getItem: (k) => { if (throws) throw new Error("blocked"); return k in mem ? mem[k] : null; },
    setItem: (k, v) => { if (throws) throw new Error("blocked"); mem[k] = String(v); },
  };
  globalThis.window = { dispatchEvent: (e) => fired.push(e.type) };
  globalThis.Event = class { constructor(type) { this.type = type; } };
  return { mem, fired };
}

test("an empty store reads the shipped pattern", () => {
  memoryStorage();
  assert.equal(readCanvasPattern(), DEFAULT_CANVAS_PATTERN);
  assert.equal(DEFAULT_CANVAS_PATTERN, "dots");
});

test("every offered pattern round-trips", () => {
  const { mem } = memoryStorage();
  for (const p of CANVAS_PATTERNS) {
    assert.equal(persistCanvasPattern(p), p);
    assert.equal(readCanvasPattern(), p);
    assert.equal(mem["picode-canvas-pattern"], p);
  }
});

test("junk on disk reads as the default instead of drawing nothing", () => {
  memoryStorage({ "picode-canvas-pattern": "hexagons" });
  assert.equal(readCanvasPattern(), DEFAULT_CANVAS_PATTERN);
  memoryStorage({ "picode-canvas-pattern": "" });
  assert.equal(readCanvasPattern(), DEFAULT_CANVAS_PATTERN);
});

test("a value we do not draw is never written", () => {
  const { mem } = memoryStorage();
  assert.equal(persistCanvasPattern("hexagons"), DEFAULT_CANVAS_PATTERN);
  assert.equal(mem["picode-canvas-pattern"], DEFAULT_CANVAS_PATTERN);
});

test("persisting tells the open planes", () => {
  const { fired } = memoryStorage();
  persistCanvasPattern("lines");
  assert.deepEqual(fired, [CANVAS_PATTERN_EVENT]);
});

test("a store that throws still answers, and still announces", () => {
  const { fired } = memoryStorage({}, { throws: true });
  assert.equal(readCanvasPattern(), DEFAULT_CANVAS_PATTERN);
  assert.equal(persistCanvasPattern("cross"), "cross");
  assert.deepEqual(fired, [CANVAS_PATTERN_EVENT]);
});
