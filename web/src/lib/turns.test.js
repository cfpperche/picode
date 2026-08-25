import assert from "node:assert/strict";
import { test } from "node:test";
import { groupTurns, fmtWorked, stepLabel, turnDurationMs } from "./turns.js";

test("groups user / work / reply", () => {
  const t = groupTurns([
    { kind: "block", cls: "user", text: "hi", ts: 1000 },
    { kind: "block", cls: "thinking", text: "hmm", ts: 1100 },
    { kind: "tool", name: "read", args: "/a/README.md", ts: 1200 },
    { kind: "block", cls: "", text: "done", ts: 5000 },
  ]);
  assert.equal(t.length, 1);
  assert.equal(t[0].work.length, 2);
  assert.equal(t[0].replies.length, 1);
  assert.equal(turnDurationMs(t[0]), 4000);
  assert.equal(fmtWorked(4000), "Worked for 4s");
});

test("step labels stay factual", () => {
  assert.equal(stepLabel({ cls: "thinking" }), "Thought");
  assert.equal(stepLabel({ kind: "tool", name: "read", args: "/x/foo.go" }), "Read foo.go");
  assert.equal(stepLabel({ kind: "tool", name: "read", args: "@" + "/home/goat/very/long/path/to/README.md" }), "Read README.md");
  assert.equal(stepLabel({ kind: "tool", name: "web_search", args: "lula" }), "Searched lula");
});
