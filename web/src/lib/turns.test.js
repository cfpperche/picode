import assert from "node:assert/strict";
import { test } from "node:test";
import { groupTurns, fmtWorked, fmtElapsed, stepLabel, turnDurationMs, dayKey, fmtDayMark, workingIndex, pathsFromTurn } from "./turns.js";

test("pathsFromTurn unique edit paths", () => {
  const t = {
    work: [
      { kind: "tool", name: "read" },
      { kind: "tool", name: "write", change: { path: "a.py" } },
      { kind: "tool", name: "edit", change: { path: "a.py" } },
      { kind: "tool", name: "edit", change: { path: "b.py" } },
    ],
  };
  assert.deepEqual(pathsFromTurn(t), ["a.py", "b.py"]);
  assert.deepEqual(pathsFromTurn({ work: [] }), []);
});

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
  assert.equal(fmtElapsed(4000), "4s");
});

test("ask card is a top-level loose item", () => {
  const t = groupTurns([
    { kind: "block", cls: "user", text: "hi", ts: 1000 },
    { kind: "ask", id: "ui-ask", method: "confirm", status: "open", title: "Allow this?" },
    { kind: "block", cls: "", text: "done", ts: 2000 },
  ]);
  assert.equal(t.length, 3);
  assert.equal(t[1].kind, "loose");
  assert.equal(t[1].item.kind, "ask");
});

test("bash is a top-level loose item", () => {
  const t = groupTurns([
    { kind: "block", cls: "user", text: "hi", ts: 1000 },
    { kind: "bash", id: "b1", command: "ls", output: "", status: "run" },
    { kind: "block", cls: "", text: "done", ts: 2000 },
  ]);
  assert.equal(t.length, 3);
  assert.equal(t[1].kind, "loose");
  assert.equal(t[1].item.kind, "bash");
});

test("workingIndex ignores queued follow-up", () => {
  const turns = groupTurns([
    { kind: "block", cls: "user", text: "a" },
    { kind: "tool", name: "read", args: "x" },
    { kind: "block", cls: "user", text: "b" },
  ]);
  assert.equal(workingIndex(turns, false), -1);
  assert.equal(workingIndex(turns, true), 0);
});

test("day marks", () => {
  const now = Date.parse("2026-08-24T18:00:00Z");
  assert.equal(dayKey(now), "2026-8-24");
  assert.match(fmtDayMark(now, now), /^Today at /);
  assert.match(fmtDayMark(now - 86400000, now), /^Yesterday at /);
});

test("step labels stay factual", () => {
  assert.equal(stepLabel({ cls: "thinking" }), "Thought");
  assert.equal(stepLabel({ kind: "tool", name: "read", args: "/x/foo.go" }), "Read foo.go");
  assert.equal(stepLabel({ kind: "tool", name: "read", args: "@" + "/home/goat/very/long/path/to/README.md" }), "Read README.md");
  assert.equal(stepLabel({ kind: "tool", name: "web_search", args: "lula" }), "Searched lula");
});

test("groupTurns: compaction card is its own loose row, never swallowed by a turn", () => {
  const turns = groupTurns([
    { kind: "block", cls: "user", text: "hi", ts: 1 },
    { kind: "compaction", text: "SUMMARY", ts: 2 },
    { kind: "block", cls: "", text: "hello again", ts: 3 },
  ]);
  const loose = turns.filter((t) => t.kind === "loose");
  assert.equal(loose.length, 1);
  assert.equal(loose[0].item.kind, "compaction");
  assert.equal(loose[0].item.text, "SUMMARY");
});
