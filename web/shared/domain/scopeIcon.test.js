import test from "node:test";
import assert from "node:assert/strict";
import { scopeKind } from "./scopeIcon.js";

test("every pane's scope words read as global, workspace or agent", () => {
  for (const s of ["machine", "user", "global"]) assert.equal(scopeKind(s), "global");
  for (const s of ["workspace", "project", "local"]) assert.equal(scopeKind(s), "workspace");
  assert.equal(scopeKind("agent"), "agent");
  assert.equal(scopeKind("stdio"), "");
  assert.equal(scopeKind(undefined), "");
});
