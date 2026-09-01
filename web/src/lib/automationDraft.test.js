import test from "node:test";
import assert from "node:assert/strict";
import { normalizeDraft, draftFromTemplate } from "./automationDraft.js";
import { isValidCron } from "./cron.js";

test("normalizeDraft whitelists, coerces and validates", () => {
  const d = normalizeDraft({
    name: "  Nightly ".padEnd(80, "x"), prompt: " run tests ", action: "weird", cron: " 0  2 * * * ",
    webhook: "true", maxCostUsd: "1,5", maxRuns: "2.7", maxRunsWindowMin: 99, evil: "x", source: "automate",
  }, isValidCron);
  assert.equal(d.name.length, 60);
  assert.equal(d.prompt, "run tests");
  assert.equal(d.action, "start");
  assert.equal(d.cron, "0 2 * * *");
  assert.equal(d.webhook, true);
  assert.equal(d.maxCostUsd, 1.5);
  assert.equal(d.maxRuns, 2);
  assert.equal(d.maxRunsWindowMin, 1440);
  assert.equal(d.evil, undefined);
  assert.equal(d.source, "automate");
});

test("normalizeDraft drops a bad cron and unpaired windows", () => {
  const d = normalizeDraft({ name: "a", prompt: "p", cron: "every day", maxRunsWindowMin: 60 }, isValidCron);
  assert.equal(d.cron, "");
  assert.equal(d.maxRuns, 0);
  assert.equal(d.maxRunsWindowMin, 0);
  assert.equal(normalizeDraft(null), null);
  assert.equal(normalizeDraft([1]), null);
  assert.equal(normalizeDraft("x"), null);
});

test("draftFromTemplate carries the template fields and its origin", () => {
  const d = draftFromTemplate({ id: "morning-brief", name: "Morning repo brief", prompt: "p", cron: "0 9 * * 1-5", maxCostUsd: 0.5, maxRuns: 2, maxRunsWindowMin: 1440 });
  assert.equal(d.templateId, "morning-brief");
  assert.equal(d.source, "template");
  assert.equal(d.sourceLabel, "Morning repo brief");
  assert.equal(d.cron, "0 9 * * 1-5");
});
