import assert from "node:assert/strict";
import { test } from "node:test";
import { workEmpty } from "./workEmpty.js";

const rows = [
  ["workspaces", false, { line: "No workspaces yet.", action: "Add workspace", page: true }],
  ["agents", false, { line: "No agents yet.", action: "New agent", page: true }],
  ["terminals", false, { line: "No terminals yet.", action: "New terminal", page: true }],
  ["workspaces", true, { line: "No matching workspaces.", action: "Clear search", page: false }],
  ["agents", true, { line: "No matching agents.", action: "Clear search", page: false }],
  ["terminals", true, { line: "No matching terminals.", action: "Clear search", page: false }],
  ["nope", false, { line: "No workspaces yet.", action: "Add workspace", page: true }],
];

test("work empty copy is one line and one action per view", () => {
  for (const [section, searching, want] of rows) {
    assert.deepEqual(workEmpty(section, searching), want, section + " searching=" + searching);
  }
});

test("chrome never says free", () => {
  for (const [section, searching] of rows) {
    const { line, action } = workEmpty(section, searching);
    assert.equal(/free/i.test(line + " " + action), false);
  }
});
