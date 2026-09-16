import { test } from "node:test";
import assert from "node:assert/strict";
import { OPEN_DIALOG_SELECTOR, subscribeAppOverlays } from "./appOverlays.js";

// A stubbed DOM: one element that can be "open" or not, and an observer that
// records the callback so the test can fire it.
function stubDom() {
  const state = { open: false };
  let notify = null;
  globalThis.MutationObserver = class {
    constructor(cb) {
      notify = cb;
    }
    observe() {}
  };
  globalThis.document = {
    body: {},
    querySelector(sel) {
      assert.equal(sel, OPEN_DIALOG_SELECTOR);
      return state.open ? {} : null;
    },
  };
  return { state, poke: () => notify && notify() };
}

test("subscribeAppOverlays reports now and on change, and unsubscribes", () => {
  const dom = stubDom();
  const seen = [];
  const unsubscribe = subscribeAppOverlays((v) => seen.push(v));
  assert.deepEqual(seen, [false]);

  dom.state.open = true;
  dom.poke();
  assert.deepEqual(seen, [false, true]);

  dom.state.open = false;
  dom.poke();
  assert.deepEqual(seen, [false, true, false]);

  unsubscribe();
  dom.state.open = true;
  dom.poke();
  assert.deepEqual(seen, [false, true, false]);

  // Leave the shared module state closed for any later test.
  dom.state.open = false;
  dom.poke();
  assert.deepEqual(seen, [false, true, false]);
});
