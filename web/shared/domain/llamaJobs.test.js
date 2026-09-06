import { test } from "node:test";
import assert from "node:assert/strict";
import { mergeLlamaJob, mergeLlamaSnapshot } from "./llamaJobs.js";

const job = { id: "job", createdAt: "2026-09-06T12:00:01Z", revision: 1, state: "queued" };
test("late acceptance cannot replace a completed feed event", () => {
  const done = { ...job, revision: 4, state: "succeeded" };
  assert.deepEqual(mergeLlamaJob([done], job), [done]);
  assert.deepEqual(mergeLlamaSnapshot([done], [job], Date.now()), [done]);
});
test("snapshot started before acceptance preserves the new job", () => {
  assert.deepEqual(mergeLlamaSnapshot([job], [], Date.parse("2026-09-06T12:00:00Z")), [job]);
});
test("authoritative newer snapshots replace observations and retire old history", () => {
  const done = { ...job, revision: 4, state: "succeeded" };
  assert.deepEqual(mergeLlamaSnapshot([job], [done], Date.now()), [done]);
  assert.deepEqual(mergeLlamaSnapshot([done], [], Date.now()), []);
});
