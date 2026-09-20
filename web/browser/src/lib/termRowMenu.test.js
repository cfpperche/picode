import { test } from "node:test";
import assert from "node:assert/strict";
import { termRowMenu } from "./termRowMenu.js";

// The module lives in shared; this file only proves the browser import
// still resolves so WorkspaceRows and AgentClis do not drift to a fork.
test("the browser import is the shared menu", () => {
  const rows = termRowMenu({ running: true });
  assert.equal(rows.find((r) => r.id === "restart").label, "Restart terminal");
  assert.equal(termRowMenu({ running: true }, { surface: "phone" }).some((r) => r.id === "launch"), false);
});
