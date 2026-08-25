import { test } from "node:test";
import assert from "node:assert/strict";
import { commandDocUrl, DOCS_BASE } from "./commandDocs.js";

test("commandDocUrl", () => {
  assert.equal(commandDocUrl("tree"), DOCS_BASE + "/commands/tree/");
  assert.equal(commandDocUrl(""), DOCS_BASE + "/");
  assert.equal(commandDocUrl("../x"), DOCS_BASE + "/");
  assert.equal(commandDocUrl("TREE"), DOCS_BASE + "/commands/tree/");
});
