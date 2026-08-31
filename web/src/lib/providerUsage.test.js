import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { showUsageButton, barTone, formatMoney, usageCopy, activeAccountLine, resetLine } from "./providerUsage.js";

describe("showUsageButton", () => {
  it("hides when unsigned or wrong method", () => {
    assert.equal(showUsageButton(null), false);
    assert.equal(showUsageButton({ signedIn: false, quotaKind: "oauth", authType: "oauth" }), false);
    assert.equal(showUsageButton({ signedIn: true, quotaKind: "", authType: "api_key" }), false);
    assert.equal(showUsageButton({ signedIn: true, quotaKind: "oauth", authType: "api_key" }), false);
  });
  it("shows for oauth quota providers", () => {
    assert.equal(showUsageButton({ signedIn: true, quotaKind: "oauth", authType: "oauth" }), true);
    assert.equal(showUsageButton({ signedIn: true, quotaKind: "api_key", authType: "api_key" }), true);
  });
});

describe("barTone", () => {
  it("steps at 70 and 90", () => {
    assert.equal(barTone(0), "ok");
    assert.equal(barTone(69), "ok");
    assert.equal(barTone(70), "warn");
    assert.equal(barTone(90), "bad");
  });
});

describe("formatMoney", () => {
  it("usd two decimals", () => {
    assert.equal(formatMoney(12.4, "usd"), "$12.40");
  });
});

describe("usageCopy", () => {
  it("covers empty error and auth", () => {
    assert.equal(usageCopy({ status: "auth_required" }).action, "signin");
    assert.equal(usageCopy({ status: "error", error: "Rate limited." }).line, "Rate limited.");
    assert.equal(usageCopy({ status: "ok", windows: [] }).line, "No usage windows on this plan.");
    assert.equal(usageCopy({ status: "ok", windows: [{ id: "5h" }] }).line, "");
    assert.equal(usageCopy({ status: "ok", windows: [], resets: [{ id: "r1" }] }).line, "");
  });
});

describe("resetLine", () => {
  it("hides when empty", () => {
    assert.equal(resetLine([]), "");
    assert.equal(resetLine(null), "");
  });
  it("counts without inventing a bar", () => {
    assert.match(resetLine([{ id: "a" }]), /1 reset available/);
  });
});

describe("activeAccountLine", () => {
  it("uses report then vault", () => {
    assert.equal(activeAccountLine({ authType: "oauth", accounts: [{ label: "work", active: true }] }, { accountLabel: "work", authType: "oauth" }), "work · account");
  });
});
