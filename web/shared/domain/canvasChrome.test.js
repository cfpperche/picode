import { test } from "node:test";
import assert from "node:assert/strict";
import {
  CANVAS_CHROME_EVENT,
  DEFAULT_CANVAS_CHROME,
  persistCanvasChrome,
  readCanvasChrome,
} from "./canvasChrome.js";

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

test("a reader who never chose sees the controls", () => {
  memoryStorage();
  assert.equal(readCanvasChrome(), true);
  assert.equal(DEFAULT_CANVAS_CHROME, true);
});

test("hidden round-trips and announces", () => {
  const { mem, fired } = memoryStorage();
  assert.equal(persistCanvasChrome(false), false);
  assert.equal(mem["picode-canvas-chrome"], "hidden");
  assert.equal(readCanvasChrome(), false);
  assert.deepEqual(fired, [CANVAS_CHROME_EVENT]);
  assert.equal(persistCanvasChrome(true), true);
  assert.equal(readCanvasChrome(), true);
});

// Only this module's own word hides the chrome. A stale value, another
// product's key collision or a hand-edited string must not leave a reader
// with no controls and no idea why.
test("only the literal hidden hides", () => {
  for (const raw of ["", "true", "false", "0", "HIDDEN", "none", "null"]) {
    memoryStorage({ "picode-canvas-chrome": raw });
    assert.equal(readCanvasChrome(), true, `${JSON.stringify(raw)} must not hide the chrome`);
  }
  memoryStorage({ "picode-canvas-chrome": "hidden" });
  assert.equal(readCanvasChrome(), false);
});

test("blocked storage still shows the controls and still applies the choice", () => {
  const { fired } = memoryStorage({}, { throws: true });
  assert.equal(readCanvasChrome(), true);
  // The write throws inside; the caller still gets the value it asked for,
  // and every open plane is still told, so this page obeys.
  assert.equal(persistCanvasChrome(false), false);
  assert.deepEqual(fired, [CANVAS_CHROME_EVENT]);
});

test("anything truthy or falsy is coerced, never stored raw", () => {
  const { mem } = memoryStorage();
  assert.equal(persistCanvasChrome(undefined), false);
  assert.equal(mem["picode-canvas-chrome"], "hidden");
  assert.equal(persistCanvasChrome(1), true);
  assert.equal(mem["picode-canvas-chrome"], "shown");
});
