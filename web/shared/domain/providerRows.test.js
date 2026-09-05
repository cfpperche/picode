import { describe, it } from "node:test";
import assert from "node:assert/strict";
import {
  STATE_LIVE, STATE_STALE, STATE_NONE,
  usageKey, indexUsage, quotaState, quotaNote, formatAge, barWindows,
  sourceLabel, identityLine, blastRadius, matchesQuery, spendByProvider, formatSpend, moneyWindows,
} from "./providerRows.js";

const win = (id, used) => ({ id, label: id, usedPercent: used });

describe("quotaState", () => {
  it("never trusts a row it has no fetch for", () => {
    assert.equal(quotaState(null), STATE_NONE);
    assert.equal(quotaState({ status: "unknown", windows: [] }), STATE_NONE);
    assert.equal(quotaState({ status: "ok", windows: [] }), STATE_NONE);
    assert.equal(quotaState({ status: "auth_required", windows: [win("5h", 10)] }), STATE_NONE);
  });
  it("separates a fresh number from an old one", () => {
    assert.equal(quotaState({ status: "ok", windows: [win("5h", 10)], ageSec: 30 }), STATE_LIVE);
    assert.equal(quotaState({ status: "ok", windows: [win("5h", 10)], ageSec: 300 }), STATE_LIVE);
    assert.equal(quotaState({ status: "ok", windows: [win("5h", 10)], ageSec: 301 }), STATE_STALE);
  });
});

describe("quotaNote", () => {
  it("says which kind of nothing it is", () => {
    assert.equal(quotaNote(null), "not checked");
    assert.equal(quotaNote({ status: "unknown" }), "not checked");
    assert.equal(quotaNote({ status: "auth_required" }), "sign in again");
    assert.equal(quotaNote({ status: "unsupported" }), "no plan windows");
    assert.equal(quotaNote({ status: "ok", windows: [] }), "no plan windows");
    assert.equal(quotaNote({ status: "error", error: "429 from the vendor" }), "429 from the vendor");
    assert.equal(quotaNote({ status: "error" }), "couldn't load");
  });
  it("is empty when there is a bar to show instead", () => {
    assert.equal(quotaNote({ status: "ok", windows: [win("5h", 42)] }), "");
  });
});

describe("formatAge", () => {
  it("reads as a duration, not a timestamp", () => {
    assert.equal(formatAge(0), "now");
    assert.equal(formatAge(59), "now");
    assert.equal(formatAge(60), "1m");
    assert.equal(formatAge(4000), "1h");
    assert.equal(formatAge(200000), "2d");
    assert.equal(formatAge(undefined), "now");
  });
});

describe("barWindows", () => {
  it("keeps percentage windows and drops money ones", () => {
    const e = { windows: [win("5h", 10), { id: "extra", remaining: 4.1, unit: "usd" }, win("7d", 20), win("month", 30)] };
    const got = barWindows(e);
    assert.deepEqual(got.map((w) => w.id), ["5h", "7d"]);
  });
  it("survives an empty entry", () => {
    assert.deepEqual(barWindows(null), []);
  });
});

describe("indexUsage / usageKey", () => {
  it("addresses the active slot and a vault row the same way the API does", () => {
    const idx = indexUsage([
      { provider: "anthropic", accountId: "a1", status: "ok" },
      { provider: "zai", accountId: "", status: "unsupported" },
    ]);
    assert.equal(idx.get(usageKey("anthropic", "a1")).status, "ok");
    assert.equal(idx.get(usageKey("zai", "")).status, "unsupported");
    assert.equal(idx.get(usageKey("zai", "live")).status, "unsupported");
  });
});

describe("sourceLabel", () => {
  it("names only the source the page cannot change", () => {
    assert.equal(sourceLabel({ source: "vault" }), "");
    assert.equal(sourceLabel({ source: "environment", envVar: "GROQ_API_KEY" }), "GROQ_API_KEY");
    assert.equal(sourceLabel({ source: "environment" }), "environment");
    assert.equal(sourceLabel(null), "");
  });
});

describe("identityLine", () => {
  it("prefers what the vendor said and never repeats the alias", () => {
    assert.equal(identityLine({ label: "Work", email: "me@co.com", plan: "Max" }, null), "me@co.com · Max");
    assert.equal(identityLine({ label: "Work" }, { plan: "Pro" }), "Pro");
    assert.equal(identityLine({ label: "Work" }, null), "");
  });
});

describe("blastRadius", () => {
  it("counts what breaks, in words", () => {
    assert.equal(blastRadius({ agents: 3, automations: 1 }), "3 agents and 1 automation use this provider.");
    assert.equal(blastRadius({ agents: 1 }), "1 agent uses this provider.");
    assert.equal(blastRadius({ automations: 2 }), "2 automations use this provider.");
    assert.equal(blastRadius({}), "");
    assert.equal(blastRadius(null), "");
  });
});

describe("matchesQuery", () => {
  const p = { id: "anthropic", accounts: [{ label: "Work", email: "me@company.com", plan: "Max" }] };
  it("finds a provider by id, alias, email or plan", () => {
    assert.equal(matchesQuery(p, ""), true);
    assert.equal(matchesQuery(p, "anth"), true);
    assert.equal(matchesQuery(p, "work"), true);
    assert.equal(matchesQuery(p, "@company"), true);
    assert.equal(matchesQuery(p, "max"), true);
    assert.equal(matchesQuery(p, "grok"), false);
  });
  it("survives a provider with no accounts", () => {
    assert.equal(matchesQuery({ id: "groq" }, "gro"), true);
    assert.equal(matchesQuery({ id: "groq" }, "zzz"), false);
  });
});

describe("spend", () => {
  it("maps the dashboard aggregate and hides sub-cent noise", () => {
    const m = spendByProvider({ byProvider: [{ provider: "anthropic", cost: 4.126 }, { provider: "xai", cost: 0.004 }] });
    assert.equal(formatSpend(m.get("anthropic")), "$4.13");
    assert.equal(formatSpend(m.get("xai")), "");
    assert.equal(formatSpend(m.get("nope")), "");
  });
});

describe("money windows", () => {
  it("a balance is a reading, not a bar", () => {
    const e = { status: "ok", ageSec: 5, windows: [{ id: "credits", label: "Credits", remaining: 12.5, unit: "usd" }] };
    assert.deepEqual(barWindows(e), []);
    assert.equal(moneyWindows(e).length, 1);
    assert.equal(quotaState(e), STATE_LIVE);
    assert.equal(quotaNote(e), "");
  });
  it("a window with neither a percentage nor an amount is nothing to show", () => {
    const e = { status: "ok", ageSec: 5, windows: [{ id: "mystery", label: "Mystery" }] };
    assert.equal(quotaState(e), STATE_NONE);
    assert.equal(quotaNote(e), "no plan windows");
  });
});
