import { test } from "node:test";
import assert from "node:assert/strict";
import { emptyFleet, mergeFleetReads, fleetRouteReady } from "./fleetReads.js";

const ok = value => ({ status: "fulfilled", value });
const failed = { status: "rejected", reason: new Error("offline") };
const initial = [ok([{ id: "workspace" }]), ok([{ id: "free-agent" }]), ok({ terminals: [{ id: "terminal" }] })];

for (let mask = 0; mask < 8; mask++) {
  test(`fleet refresh preserves last successful sources, failure mask ${mask}`, () => {
    const previous = mergeFleetReads(emptyFleet(), initial);
    const reads = [ok([{ id: "new-workspace" }]), ok([]), ok({ terminals: [] })].map((read, i) => mask & (1 << i) ? failed : read);
    const next = mergeFleetReads(previous, reads);
    assert.deepEqual(next.workspaces, mask & 1 ? previous.workspaces : [{ id: "new-workspace" }]);
    assert.deepEqual(next.freeAgents, mask & 2 ? previous.freeAgents : []);
    assert.deepEqual(next.terminals, mask & 4 ? previous.terminals : []);
    assert.equal(next.loaded, true);
    assert.equal(!!next.error, mask !== 0);
  });
}

test("initial partial failure remains recoverable until every source answered", () => {
  const partial = mergeFleetReads(emptyFleet(), [initial[0], failed, initial[2]]);
  assert.equal(partial.loaded, false);
  assert.ok(partial.error);
  assert.equal(partial.workspaces[0].id, "workspace");
  const recovered = mergeFleetReads(partial, [failed, initial[1], initial[2]]);
  assert.equal(recovered.loaded, true);
  assert.ok(recovered.error, "workspace is still stale until it refreshes");
  assert.deepEqual(mergeFleetReads(recovered, initial).known, { workspaces: true, freeAgents: true, terminals: true });
});

test("invalid responses do not erase lists or falsely complete loading", () => {
  const next = mergeFleetReads(emptyFleet(), [ok({}), ok(null), ok({})]);
  assert.equal(next.loaded, false);
  assert.ok(next.error);
  assert.deepEqual(next.known, {});
});

// Decision table: a found resource opens regardless of other sources;
// absence waits only for its possible sources. Unrelated routes never wait.
for (const [route, required] of [
  [{ screen: "agent" }, ["workspaces", "freeAgents"]],
  [{ screen: "term" }, ["terminals"]],
  [{ screen: "inspector", section: "agent" }, ["workspaces", "freeAgents"]],
  [{ screen: "inspector", section: "term" }, ["terminals"]],
  [{ screen: "inspector", section: "workspace" }, ["workspaces"]],
  ...["files", "git"].flatMap(screen => [
    [{ screen, section: "agent" }, ["workspaces", "freeAgents"]],
    [{ screen, section: "term" }, ["terminals"]],
    [{ screen, section: "workspace" }, ["workspaces"]],
  ]),
  [{ screen: "work" }, []],
  [{ screen: "more", section: "providers" }, []],
]) {
  test(`route readiness: ${route.screen}/${route.section || ""}`, () => {
    for (let mask = 0; mask < 8; mask++) {
      const known = Object.fromEntries(["workspaces", "freeAgents", "terminals"].map((key, i) => [key, !!(mask & (1 << i))]));
      assert.equal(fleetRouteReady(route, known, true), true, `found resource, mask ${mask}`);
      assert.equal(fleetRouteReady(route, known), required.every(key => known[key]), `missing resource, mask ${mask}`);
    }
  });
}
