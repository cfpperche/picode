import assert from "node:assert/strict";
import { test } from "node:test";
import { pendingFollowUps, dropQueued, startEditQueued, saveEditQueued, cancelEditQueued } from "./queue.js";

const fu = { kind: "block", cls: "user", chip: "follow_up", pending: true, qid: "q1", text: "later" };

test("pendingFollowUps skips dropped and editing", () => {
  assert.equal(pendingFollowUps([fu]).length, 1);
  assert.equal(pendingFollowUps(dropQueued([fu], "q1")).length, 0);
  assert.equal(pendingFollowUps(startEditQueued([fu], "q1")).length, 0);
});

test("dropQueued marks dropped", () => {
  const [it] = dropQueued([fu], "q1");
  assert.equal(it.dropped, true);
  assert.equal(it.pending, false);
});

test("saveEditQueued updates text; empty removes", () => {
  const [ed] = saveEditQueued([fu], "q1", "  new  ");
  assert.equal(ed.text, "new");
  assert.equal(ed.editing, false);
  const [gone] = saveEditQueued([fu], "q1", "   ");
  assert.equal(gone.dropped, true);
});

test("cancelEditQueued leaves text", () => {
  const started = startEditQueued([fu], "q1");
  const [it] = cancelEditQueued(started, "q1");
  assert.equal(it.text, "later");
  assert.equal(it.editing, false);
});
