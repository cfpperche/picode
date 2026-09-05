import test from "node:test";
import assert from "node:assert/strict";
import { createRefreshQueue } from "./appRefreshQueue.js";

test("a feed burst completes the current read and coalesces a trailing read", async () => {
  let calls = 0;
  let release;
  const gate = new Promise((resolve) => { release = resolve; });
  const queue = createRefreshQueue(async () => { calls++; if (calls === 1) await gate; });
  const done = queue.request();
  for (let i = 0; i < 20; i++) queue.request();
  assert.equal(calls, 1);
  release();
  await done;
  assert.equal(calls, 2);
  await queue.request();
  assert.equal(calls, 3);
});

test("closing the surface cancels queued refreshes", async () => {
  let calls = 0;
  let release;
  const gate = new Promise((resolve) => { release = resolve; });
  const queue = createRefreshQueue(async () => { calls++; await gate; });
  const done = queue.request();
  queue.request();
  queue.stop();
  release();
  await done;
  await queue.request();
  assert.equal(calls, 1);
});
