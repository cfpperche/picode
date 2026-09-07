import { test } from "node:test";
import assert from "node:assert/strict";
import { createFleetReload } from "./fleetReload.js";

function fixture() {
  const reads = [];
  const loader = createFleetReload(() => new Promise((resolve, reject) => reads.push({ resolve, reject })));
  return { loader, reads };
}
const tick = () => new Promise(resolve => setImmediate(resolve));

// Decision table: idle => start; active poll => share; active forced =>
// queue and await fresh; forced calls before that read => share follow-up;
// forced during follow-up => queue again; rejected old read => still refresh.
for (const force of [false, true]) {
  test(`idle reload starts one read, force=${force}`, async () => {
    const { loader, reads } = fixture();
    const result = loader.reload({ force });
    assert.equal(loader.pending, true);
    await tick();
    assert.equal(reads.length, 1);
    reads[0].resolve("first");
    assert.equal(await result, "first");
    assert.equal(loader.pending, false);
  });
}

test("ordinary refreshes share the active read", async () => {
  const { loader, reads } = fixture();
  const first = loader.reload();
  assert.equal(loader.reload(), first);
  await tick();
  assert.equal(reads.length, 1);
  reads[0].resolve("shared");
  assert.equal(await first, "shared");
});

for (const rejectOld of [false, true]) {
  test(`forced refresh waits for a newer result after old read ${rejectOld ? "fails" : "succeeds"}`, async () => {
    const { loader, reads } = fixture();
    const old = loader.reload();
    old.catch(() => {});
    await tick();
    const fresh = loader.reload({ force: true });
    assert.equal(loader.reload({ force: true }), fresh, "concurrent mutations share one follow-up");
    let settled = false;
    fresh.then(() => { settled = true; });
    if (rejectOld) reads[0].reject(new Error("old request failed"));
    else reads[0].resolve("before mutation");
    await tick();
    assert.equal(reads.length, 2);
    assert.equal(settled, false, "caller cannot navigate with the old result");
    reads[1].resolve("after mutation");
    assert.equal(await fresh, "after mutation");
    assert.equal(loader.pending, false);
  });
}

test("a mutation during the follow-up waits for another read", async () => {
  const { loader, reads } = fixture();
  loader.reload();
  await tick();
  const firstMutation = loader.reload({ force: true });
  reads[0].resolve("before mutation");
  await tick();
  const secondMutation = loader.reload({ force: true });
  assert.notEqual(secondMutation, firstMutation);
  reads[1].resolve("before second mutation");
  assert.equal(await firstMutation, "before second mutation");
  await tick();
  assert.equal(reads.length, 3);
  reads[2].resolve("after second mutation");
  assert.equal(await secondMutation, "after second mutation");
});

test("a failed forced read rejects its caller and allows retry", async () => {
  const { loader, reads } = fixture();
  loader.reload();
  await tick();
  const fresh = loader.reload({ force: true });
  const rejected = assert.rejects(fresh, /refresh failed/);
  reads[0].resolve("old");
  await tick();
  reads[1].reject(new Error("refresh failed"));
  await rejected;
  assert.equal(loader.pending, false);
  const retry = loader.reload({ force: true });
  await tick();
  reads[2].resolve("recovered");
  assert.equal(await retry, "recovered");
});
