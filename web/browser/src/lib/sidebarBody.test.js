import { test } from "node:test";
import assert from "node:assert/strict";
import { sidebarBody } from "./sidebarBody.js";

// | fleet read | rows | shows    |
// | ---------- | ---- | -------- |
// | no         | 0    | skeleton |
// | yes        | 0    | empty    |
// | either     | > 0  | list     |
test("a list is empty only once the fleet has been read", () => {
  assert.equal(sidebarBody(false, 0), "skeleton");
  assert.equal(sidebarBody(true, 0), "empty");
  assert.equal(sidebarBody(false, 2), "list");
  assert.equal(sidebarBody(true, 2), "list");
});
