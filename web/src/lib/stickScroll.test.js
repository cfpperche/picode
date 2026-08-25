import assert from "node:assert/strict";
import { test } from "node:test";
import { stuckToBottom } from "./stickScroll.js";

test("inside composer pad is stuck", () => {
  assert.equal(stuckToBottom({ scrollHeight: 1000, scrollTop: 750, clientHeight: 200 }), true);
});

test("scrolled up is not stuck", () => {
  assert.equal(stuckToBottom({ scrollHeight: 2000, scrollTop: 100, clientHeight: 400 }), false);
});
