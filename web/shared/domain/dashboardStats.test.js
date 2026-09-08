import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { deltaPercent, fleetStats, rangeLabel, compareLabel, formatTokens, percent, tokenSegments, dayLabel, folderLabel,
  measured, signalState, coversAll, coverageNote, billingBadge, formatDuration, resetsIn, costPerTurn, costPerLine, spendState } from "./dashboardStats.js";

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

// --- cross-CLI (ADR-0097) ---------------------------------------------

const coverage = [
  { cli: "pi", label: "Pi", billing: "api", signals: { cost: "reported", impact: "not-reported", timing: "not-reported" } },
  { cli: "claude-code", label: "Claude Code", billing: "subscription", signals: { cost: "partial", impact: "reported", timing: "reported" } },
  { cli: "codex", label: "Codex", billing: "unknown", signals: { cost: "not-reported", impact: "not-reported", timing: "not-reported" } },
];
const byCli = [
  { cli: "pi", messages: 10 },
  { cli: "claude-code", messages: 20 },
  { cli: "codex", messages: 5 },
];

describe("measured", () => {
  it("counts reported and partial, nothing else", () => {
    assert.equal(measured("reported"), true);
    assert.equal(measured("partial"), true);
    assert.equal(measured("not-reported"), false);
    assert.equal(measured("unavailable"), false);
    assert.equal(measured(undefined), false);
  });
});

describe("signalState", () => {
  it("reads one CLI's answer", () => {
    assert.equal(signalState(coverage, "claude-code", "cost"), "partial");
  });
  it("defaults to unavailable rather than assuming coverage", () => {
    assert.equal(signalState(coverage, "grok", "cost"), "unavailable");
    assert.equal(signalState(coverage, "pi", "limits"), "unavailable");
  });
});

describe("coversAll", () => {
  it("is false when an active CLI does not report the signal", () => {
    assert.equal(coversAll(coverage, byCli, "impact"), false);
  });
  it("ignores CLIs with no activity in the window", () => {
    const quiet = [{ cli: "pi", messages: 10 }, { cli: "codex", messages: 0 }];
    assert.equal(coversAll(coverage, quiet, "cost"), true);
  });
  it("is true when nothing was active", () => {
    assert.equal(coversAll(coverage, [], "impact"), true);
  });
});

describe("coverageNote", () => {
  it("stays empty when everyone covered it, so the footnote means something", () => {
    assert.equal(coverageNote(coverage, [{ cli: "pi", messages: 1 }], "cost"), "");
  });
  it("names who the number covers when it is partial", () => {
    assert.equal(coverageNote(coverage, byCli, "impact"), "Covers Claude Code only.");
  });
  it("says so when no CLI records it at all", () => {
    assert.equal(coverageNote(coverage, byCli, "limits"), "No CLI that ran records this.");
  });
});

describe("billingBadge", () => {
  it("marks api and subscription, and says nothing for unknown", () => {
    assert.equal(billingBadge("api"), "api");
    assert.equal(billingBadge("subscription"), "sub");
    assert.equal(billingBadge("unknown"), "");
    assert.equal(billingBadge(undefined), "");
  });
});

describe("formatDuration", () => {
  it("reads as agent time, which can exceed the clock", () => {
    assert.equal(formatDuration(0), "0m");
    assert.equal(formatDuration(90 * 1000), "2m");
    assert.equal(formatDuration(3600 * 1000), "1h");
    assert.equal(formatDuration(5400 * 1000), "1h 30m");
    assert.equal(formatDuration(226 * 3600 * 1000), "226h");
  });
});

describe("resetsIn", () => {
  const now = Date.parse("2026-09-07T12:00:00Z");
  it("is empty without a reset time", () => {
    assert.equal(resetsIn("", now), "");
    assert.equal(resetsIn("not a date", now), "");
  });
  it("counts down in the unit that reads", () => {
    assert.equal(resetsIn("2026-09-07T12:30:00Z", now), "resets in 30m");
    assert.equal(resetsIn("2026-09-08T12:00:00Z", now), "resets in 24h");
    assert.equal(resetsIn("2026-09-13T12:00:00Z", now), "resets in 6d");
    assert.equal(resetsIn("2026-09-07T11:00:00Z", now), "resets now");
  });
});

describe("costPerTurn / costPerLine", () => {
  it("returns null rather than a zero that would read as free", () => {
    assert.equal(costPerTurn(10, 0), null);
    assert.equal(costPerLine(10, 0, 0), null);
  });
  it("divides when there is something to divide by", () => {
    assert.equal(costPerTurn(10, 4), 2.5);
    assert.equal(costPerLine(10, 8, 2), 1);
  });
});

describe("spendState", () => {
  it("is not-reported when nothing that ran records a price", () => {
    // The real reading that prompted this: 260 messages, all live Claude
    // Code sessions, none with a cost snapshot yet.
    assert.equal(spendState([{ cli: "claude-code", messages: 260, costState: "not-reported" }]), "not-reported");
  });
  it("is partial when only some active CLIs priced their work", () => {
    assert.equal(spendState([
      { cli: "pi", messages: 10, costState: "reported" },
      { cli: "codex", messages: 5, costState: "not-reported" },
    ]), "partial");
  });
  it("is reported when every active CLI priced its work", () => {
    assert.equal(spendState([
      { cli: "pi", messages: 10, costState: "reported" },
      { cli: "codex", messages: 0, costState: "not-reported" },
    ]), "reported");
  });
  it("ignores CLIs that did nothing", () => {
    assert.equal(spendState([{ cli: "grok", messages: 0, costState: "not-reported" }]), "not-reported");
  });
});
