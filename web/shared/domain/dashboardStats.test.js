import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { deltaPercent, fleetStats, rangeLabel, compareLabel, formatTokens, percent, tokenSegments, bucketLabel, folderLabel,
  measured, signalState, coversAll, coverageNote, billingBadge, formatDuration, resetsIn, costPerTurn, costPerLine, spendState,
  limitReading, observedAge,
  FLEET_WORKING, FLEET_NEEDS_YOU, FLEET_IDLE, FLEET_UNREPORTED, FLEET_ORDER, FLEET_LABELS } from "./dashboardStats.js";

describe("deltaPercent", () => {
  it("counts bound Pi once and keeps unreported activity unknown", () => {
    const term = {id:"t",running:true,cli:"pi"};
    const a = {id:"a",cli:"pi",terminalId:"t",mode:"interactive"};
    const stats = fleetStats([], [a], [term], {});
    assert.equal(stats.live,1);
    assert.equal(stats[FLEET_UNREPORTED],1);
    assert.equal(stats.terminals.total,0);
  });
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
  const term = (over = {}) => ({ id: "t1", name: "canvas", running: true, ...over });

  it("counts agents by state across workspaces and free agents", () => {
    const f = fleetStats(workspaces, freeAgents, [], {});
    assert.deepEqual([f.agents.running, f.agents.total], [2, 4]);
    assert.deepEqual([f.agents.idle, f.agents[FLEET_WORKING], f.agents[FLEET_NEEDS_YOU]], [2, 0, 0]);
  });

  it("splits live agents into the buckets from their live ids", () => {
    const f = fleetStats(workspaces, freeAgents, [], { workingIds: ["a1"], waitingId: "a3" });
    assert.deepEqual([f[FLEET_WORKING], f[FLEET_NEEDS_YOU], f[FLEET_IDLE]], [1, 1, 0]);
    assert.deepEqual(f.units.map((u) => [u.name, u.detail, u.state]),
      [["free", "", FLEET_NEEDS_YOU], ["One", "grok-4.6", FLEET_WORKING]]);
  });

  // The decision table this release exists for: a terminal's CLI state is
  // presence (cli/tui) crossed with activity (state), and the two holes in
  // that grid — no CLI, no reported activity — are counted, never guessed.
  it("counts agent-CLI terminals beside agents, by their own vocabulary", () => {
    const terminals = [
      term({ id: "t1", cli: "claude-code", state: "working" }),
      term({ id: "t2", cli: "grok", state: "needs-you" }),
      term({ id: "t3", cli: "pi", state: "idle" }),
      term({ id: "t4", tui: { cli: "codex" } }),
      term({ id: "t5", cli: "pi", running: false }),
    ];
    const f = fleetStats([], [], terminals, {});
    assert.deepEqual([f.terminals.running, f.terminals.total], [4, 5]);
    assert.deepEqual([f[FLEET_WORKING], f[FLEET_NEEDS_YOU], f[FLEET_IDLE], f[FLEET_UNREPORTED]], [1, 1, 1, 1]);
    assert.equal(f.live, 4);
  });

  it("keeps a CLI with no activity signal out of idle", () => {
    const f = fleetStats([], [], [term({ cli: "codex" })], {});
    assert.deepEqual([f[FLEET_IDLE], f[FLEET_UNREPORTED]], [0, 1]);
    assert.equal(f.units[0].state, FLEET_UNREPORTED);
  });

  it("counts shells apart and never lists them as agents", () => {
    const f = fleetStats([], [], [term({ id: "s1", name: "build" })], {});
    assert.deepEqual([f.shells.running, f.shells.total, f.terminals.running], [1, 1, 0]);
    assert.deepEqual(f.units, []);
    assert.equal(f.live, 0);
  });

  it("ranks the units by who is waiting on the reader", () => {
    const f = fleetStats(workspaces, freeAgents, [term({ cli: "pi", state: "idle" })], { waitingId: "a3" });
    assert.deepEqual(f.units.map((u) => u.state), [FLEET_NEEDS_YOU, FLEET_IDLE, FLEET_IDLE]);
    assert.deepEqual(FLEET_ORDER, [FLEET_NEEDS_YOU, FLEET_WORKING, FLEET_IDLE, FLEET_UNREPORTED]);
    assert.equal(FLEET_LABELS[FLEET_UNREPORTED], "no signal");
  });

  it("carries the tab identity each unit opens by, ties broken by name", () => {
    const f = fleetStats(workspaces, [], [term({ id: "grid-layout-3a1c81", cli: "claude-code", state: "idle" })], {});
    const byKind = Object.fromEntries(f.units.map((u) => [u.kind, u.id]));
    assert.equal(byKind.agent, "a1");
    assert.equal(byKind.terminal, "grid-layout-3a1c81");
    // Two idle units and no winner on state: the alphabet decides, so the
    // list does not reshuffle between renders, not insertion order.
    assert.deepEqual(f.units.map((u) => u.key), ["terminal:grid-layout-3a1c81", "agent:a1"]);
  });

  it("handles the legacy single-agent workspace shape via agentsOf", () => {
    const f = fleetStats([{ id: "w1", agent: { id: "a1", mode: "managed" } }], [], [], {});
    assert.deepEqual([f.agents.running, f.agents.total], [1, 1]);
  });

  it("is zero-safe on empty input", () => {
    assert.equal(fleetStats([], [], [], {}).live, 0);
    assert.equal(fleetStats([], [], [], {}).agents.total, 0);
    assert.deepEqual(fleetStats(null, null, null).units, []);
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

describe("tokenSegments / bucketLabel", () => {
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
    assert.match(bucketLabel("2026-08-25"), /25/);
    assert.equal(bucketLabel("nope"), "nope");
  });
  // range=today is bucketed by hour, and the key says so: no second flag, and
  // an older client that never learned the shape still prints the key.
  it("labels an hour key by its clock hour", () => {
    assert.equal(bucketLabel("2026-09-13T00"), "00:00");
    assert.equal(bucketLabel("2026-09-13T14"), "14:00");
    assert.equal(bucketLabel("2026-09-13T23"), "23:00");
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

describe("limitReading / observedAge", () => {
  const now = Date.parse("2026-09-14T12:00:00Z");
  const at = (iso) => iso;
  it("is current while the window it describes is still open", () => {
    assert.equal(limitReading({ resetsAt: "2026-09-20T00:00:00Z" }, now), "current");
  });
  // The failure this exists for: a reading whose window has already reset next
  // to a countdown would present a past window's usage as the current one.
  it("is expired once the window it describes has reset", () => {
    assert.equal(limitReading({ resetsAt: "2026-09-14T11:59:00Z" }, now), "expired");
    assert.equal(limitReading({ resetsAt: "2026-09-14T12:00:00Z" }, now), "expired");
  });
  it("is unknown without a reset time rather than guessing either way", () => {
    assert.equal(limitReading({}, now), "unknown");
    assert.equal(limitReading({ resetsAt: "not a date" }, now), "unknown");
    assert.equal(limitReading(null, now), "unknown");
  });
  it("measures how old the snapshot is", () => {
    assert.equal(observedAge({ observedAt: "2026-09-14T09:00:00Z" }, now), 3 * 3600 * 1000);
    assert.equal(observedAge({ observedAt: "2026-09-14T13:00:00Z" }, now), 0, "a reading from the future is not negative age");
    assert.equal(observedAge({}, now), null);
    assert.equal(observedAge({ observedAt: at("") }, now), null);
  });
});

// resetsIn is the other half of that contract: it must never claim a window
// resets in the future when its reset instant is already past.
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
