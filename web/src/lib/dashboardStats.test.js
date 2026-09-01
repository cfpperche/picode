import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { deltaPercent, fleetStats, rangeLabel, compareLabel } from "./dashboardStats.js";

describe("deltaPercent", () => {
  it("is null with no prior or a zero prior", () => {
    assert.equal(deltaPercent(10, null), null);
    assert.equal(deltaPercent(10, 0), null);
  });
  it("computes a signed percent change", () => {
    assert.equal(deltaPercent(12, 10), 20);
    assert.equal(deltaPercent(8, 10), -20);
  });
});

describe("fleetStats", () => {
  it("counts running vs total across workspaces and free agents", () => {
    const workspaces = [
      { id: "w1", agents: [{ id: "a1", mode: "managed" }, { id: "a2", mode: "stopped" }] },
      { id: "w2", agents: [] },
    ];
    const freeAgents = [{ id: "a3", mode: "interactive" }, { id: "a4", mode: "stopped" }];
    assert.deepEqual(fleetStats(workspaces, freeAgents), { running: 2, total: 4 });
  });
  it("handles the legacy single-agent workspace shape via agentsOf", () => {
    const workspaces = [{ id: "w1", agent: { id: "a1", mode: "managed" } }];
    assert.deepEqual(fleetStats(workspaces, []), { running: 1, total: 1 });
  });
  it("is zero-safe on empty input", () => {
    assert.deepEqual(fleetStats([], []), { running: 0, total: 0 });
    assert.deepEqual(fleetStats(null, null), { running: 0, total: 0 });
  });
});

describe("rangeLabel / compareLabel", () => {
  it("labels every supported range", () => {
    assert.equal(rangeLabel("today"), "Today");
    assert.equal(rangeLabel("30d"), "30 days");
    assert.equal(rangeLabel("bogus"), "7 days");
  });
  it("has no compare label for all-time", () => {
    assert.equal(compareLabel("all"), "");
    assert.equal(compareLabel("7d"), "vs. prior 7 days");
  });
});
