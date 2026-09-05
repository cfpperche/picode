import assert from "node:assert/strict";
import { test } from "node:test";
import { newHist, histPush, histUp, histDown } from "./composerHist.js";

test("up/down like the TUI", () => {
  const h = newHist();
  histPush(h, "one");
  histPush(h, "two");
  assert.equal(histUp(h, "now"), "two");
  assert.equal(histUp(h, "now"), "one");
  assert.equal(histUp(h, "now"), "one");
  assert.equal(histDown(h, "now"), "two");
  assert.equal(histDown(h, "now"), "now");
});
