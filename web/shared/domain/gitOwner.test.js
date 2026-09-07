import assert from "node:assert/strict";
import { test } from "node:test";
import { ownerBase, ownerNamespace } from "./gitOwner.js";

test("each owner kind reads under its own namespace", () => {
  assert.equal(ownerNamespace("agent"), "agents");
  assert.equal(ownerNamespace("term"), "terminals");
  assert.equal(ownerNamespace("workspace"), "workspaces");
});

test("an unknown or absent owner falls back to agents", () => {
  assert.equal(ownerNamespace(""), "agents");
  assert.equal(ownerNamespace("nonsense"), "agents");
  assert.equal(ownerBase(null), "/api/agents/");
  assert.equal(ownerBase(undefined), "/api/agents/");
});

test("ownerBase is the prefix a git URL concatenates onto", () => {
  assert.equal(ownerBase({ kind: "workspace", id: "w1" }) + "w1/git", "/api/workspaces/w1/git");
  assert.equal(ownerBase({ kind: "term", id: "t1" }), "/api/terminals/");
  assert.equal(ownerBase({ kind: "agent", id: "a1" }), "/api/agents/");
});
