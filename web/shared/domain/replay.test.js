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

test("replayed tool result details render the persisted preview (ADR-0057)", () => {
  const items = eventsToItems([
    { kind: "tool", id: "t1", name: "agent_browser", status: "ok", detail: "shot",
      result: { preview: { image: "data:image/jpeg;base64,AAA", url: "https://example.com", title: "Example" } } },
    { kind: "tool", id: "t2", name: "bash", status: "ok", detail: "a" },
  ]);
  assert.deepEqual(items[0].preview, { image: "data:image/jpeg;base64,AAA", url: "https://example.com", title: "Example" });
  assert.equal(items[1].preview, null);
});
