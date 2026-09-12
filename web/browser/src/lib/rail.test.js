import assert from "node:assert/strict";
import { test } from "node:test";
import { railAnchors } from "./rail.js";

test("skips sys/tool and empty", () => {
  const a = railAnchors([
    { kind: "sys", text: "hi" },
    { kind: "block", cls: "user", actor: "You", text: "Hello there" },
    { kind: "tool", name: "read", text: "x" },
    { kind: "block", actor: "agent", text: "Answer" },
    { kind: "block", cls: "user", actor: "You", text: "Again" },
  ]);
  assert.equal(a.length, 3);
  assert.equal(a[0].id, "turn-0-user");
  assert.equal(a[0].actor, "You");
  assert.equal(a[1].actor, "Agent");
  assert.equal(a[2].id, "turn-1-user");
});

test("truncates preview", () => {
  const a = railAnchors([{ kind: "block", text: "x".repeat(200) }]);
  assert.equal(a[0].preview.endsWith("…"), true);
  assert.ok(a[0].preview.length <= 140);
});
