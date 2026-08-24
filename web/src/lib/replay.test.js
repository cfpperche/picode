import { eventsToItems } from "./replay.js";
import assert from "node:assert/strict";
import { test } from "node:test";

test("maps user assistant tool", () => {
  const items = eventsToItems([
    { kind: "user", text: "hi" },
    { kind: "tool", id: "t1", name: "bash", args: "ls", status: "ok", detail: "a" },
    { kind: "assistant", text: "a" },
  ]);
  assert.equal(items[0].kind, "block");
  assert.equal(items[0].actor, "You");
  assert.equal(items[1].kind, "tool");
  assert.equal(items[1].name, "bash");
  assert.equal(items[2].actor, "agent");
});
