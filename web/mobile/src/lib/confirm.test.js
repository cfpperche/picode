import assert from "node:assert/strict";
import { test } from "node:test";
import { fmtBytes } from "./confirm.js";

test("fmtBytes", () => {
  assert.equal(fmtBytes(0), "0 B");
  assert.equal(fmtBytes(512), "512 B");
  assert.equal(fmtBytes(2048), "2.0 KB");
  assert.equal(fmtBytes(12 * 1024), "12 KB");
  assert.equal(fmtBytes(Math.round(3.4 * 1024 * 1024)), "3.4 MB");
});
