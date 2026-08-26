import assert from "node:assert/strict";
import { test } from "node:test";
import { safeImgSrc } from "./mdSafe.js";

test("safeImgSrc", () => {
  assert.ok(safeImgSrc("https://example.com/a.png"));
  assert.ok(safeImgSrc("data:image/png;base64,xxx"));
  assert.equal(safeImgSrc("javascript:alert(1)"), "");
  assert.equal(safeImgSrc("file:///etc/passwd"), "");
});
