import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { deltaPercent, fleetStats, rangeLabel, compareLabel, formatTokens, percent, tokenSegments, dayLabel, folderLabel } from "./dashboardStats.js";

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
  const workspaces = [
    { id: "w1", name: "One", agents: [{ id: "a1", name: "default", mode: "managed", model: "grok-4.6" }, { id: "a2", mode: "stopped" }] },
    { id: "w2", agents: [] },
  ];
  const freeAgents = [{ id: "a3", name: "free", mode: "interactive" }, { id: "a4", mode: "stopped" }];
  it("counts running vs total across workspaces and free agents", () => {
    const f = fleetStats(workspaces, freeAgents);
    assert.equal(f.running, 2);
    assert.equal(f.total, 4);
    assert.equal(f.idle, 2);
    assert.equal(f.working, 0);
  });
  it("splits running agents into working / waiting / idle from live ids", () => {
    const f = fleetStats(workspaces, freeAgents, { workingIds: ["a1"], waitingId: "a3" });
    assert.deepEqual([f.working, f.waiting, f.idle], [1, 1, 0]);
    assert.deepEqual(f.agents.map((a) => [a.name, a.model, a.state]), [["One", "grok-4.6", "working"], ["free", "", "waiting"]]);
  });
  it("handles the legacy single-agent workspace shape via agentsOf", () => {
    const f = fleetStats([{ id: "w1", agent: { id: "a1", mode: "managed" } }], []);
    assert.deepEqual([f.running, f.total], [1, 1]);
  });
  it("is zero-safe on empty input", () => {
    assert.deepEqual(fleetStats([], []).total, 0);
    assert.deepEqual(fleetStats(null, null).agents, []);
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

describe("formatTokens / percent", () => {
  it("shortens magnitudes", () => {
    assert.equal(formatTokens(0), "0");
    assert.equal(formatTokens(999), "999");
    assert.equal(formatTokens(4800), "4.8K");
    assert.equal(formatTokens(48200), "48K");
    assert.equal(formatTokens(48_200_000), "48.2M");
    assert.equal(formatTokens(1_230_000_000), "1.23B");
  });
  it("rates are null without a denominator", () => {
    assert.equal(percent(1, 0), null);
    assert.equal(percent(70, 9882), "0.7%");
    assert.equal(percent(50, 100), "50%");
  });
});

describe("tokenSegments / dayLabel", () => {
  it("splits four slices in a fixed order summing to 100", () => {
    const s = tokenSegments({ input: 10, output: 10, cacheRead: 70, cacheWrite: 10 });
    assert.equal(s.total, 100);
    assert.deepEqual(s.parts.map((p) => p.key), ["input", "output", "cacheRead", "cacheWrite"]);
    assert.deepEqual(s.parts.map((p) => p.pct), [10, 10, 70, 10]);
  });
  it("is zero-safe", () => {
    assert.equal(tokenSegments(null).total, 0);
    assert.ok(tokenSegments({}).parts.every((p) => p.pct === 0));
  });
  it("labels a series day in local time", () => {
    assert.match(dayLabel("2026-08-25"), /25/);
    assert.equal(dayLabel("nope"), "nope");
  });
});

describe("folderLabel", () => {
  it("takes the last segment, or strips pi's encoded fence", () => {
    assert.equal(folderLabel("/home/goat/picode"), "picode");
    assert.equal(folderLabel("/home/goat/picode/"), "picode");
    assert.equal(folderLabel("--home-goat-picode--"), "home-goat-picode");
    assert.equal(folderLabel(""), "");
  });
});
