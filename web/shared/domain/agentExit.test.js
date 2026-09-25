import assert from "node:assert/strict";
import { test } from "node:test";
import {
  emptyExitDraft, exitHeadline, fmtExitCost, exitRequestBody, fmtLifetime, fmtTurns, outcomeRows,
  instructionsLine, pickOutcome, reasonRows, takesReasons, toggleReason,
} from "./agentExit.js";

const TAX = {
  version: 1,
  outcomes: [
    { id: "resolved", label: "Resolved" },
    { id: "partial", label: "Partly" },
    { id: "unresolved", label: "Didn't resolve" },
    { id: "trial", label: "Just trying" },
  ],
  reasons: [
    { id: "setup", label: "Wrong setup" },
    { id: "stuck", label: "Got stuck or looped" },
  ],
  reasonsFor: ["partial", "unresolved"],
};

test("reasons belong to partial and unresolved only", () => {
  assert.equal(takesReasons(TAX, "partial"), true);
  assert.equal(takesReasons(TAX, "resolved"), false);
  assert.equal(takesReasons(TAX, ""), false);
  assert.equal(takesReasons(null, "partial"), false);
});

test("picking the chosen outcome again clears it, and reasons go with an outcome that has none", () => {
  let d = pickOutcome(TAX, emptyExitDraft(), "unresolved");
  d = toggleReason(d, "stuck");
  d = toggleReason(d, "setup");
  assert.deepEqual(d.reasons, ["stuck", "setup"]);
  d = toggleReason(d, "stuck");
  assert.deepEqual(d.reasons, ["setup"]);
  assert.deepEqual(pickOutcome(TAX, d, "partial").reasons, ["setup"]);
  assert.deepEqual(pickOutcome(TAX, d, "resolved").reasons, []);
  assert.equal(pickOutcome(TAX, d, "unresolved").outcome, "");
});

test("the request body sends an answer only when the question was shown", () => {
  const d = { outcome: "unresolved", reasons: ["stuck"], note: "  loops  " };
  assert.deepEqual(exitRequestBody({ taxonomy: TAX, draft: d, shown: true, origin: "desktop" }), {
    exit: { origin: "desktop", asked: true, outcome: "unresolved", reasons: ["stuck"], note: "loops" },
  });
  assert.deepEqual(exitRequestBody({ taxonomy: TAX, draft: d, shown: false, origin: "mobile" }), {
    exit: { origin: "mobile", asked: false, outcome: "", reasons: [], note: "" },
  });
  // A reason left over from a partial answer does not ride a resolved one.
  const resolved = exitRequestBody({ taxonomy: TAX, draft: { outcome: "resolved", reasons: ["stuck"], note: "" }, shown: true, origin: "desktop" });
  assert.deepEqual(resolved.exit.reasons, []);
  // A note without an outcome is not an answer.
  const noteOnly = exitRequestBody({ taxonomy: TAX, draft: { outcome: "", reasons: [], note: "hm" }, shown: true, origin: "desktop" });
  assert.equal(noteOnly.exit.note, "");
});

test("lifetimes and turns read in whole short units, unmeasured turns as a dash", () => {
  assert.equal(fmtLifetime(42), "42s");
  assert.equal(fmtLifetime(125), "2m");
  assert.equal(fmtLifetime(7300), "2h");
  assert.equal(fmtLifetime(3 * 86400 + 5), "3d");
  assert.equal(fmtLifetime(-4), "0s");
  assert.equal(fmtTurns(null), "—");
  assert.equal(fmtTurns(0), "0");
});

test("the headline divides by what it counts, never by zero, and leaves trials aside", () => {
  const h = exitHeadline({ total: 6, asked: 4, answered: 4, askedAnswered: 2, outcomes: { resolved: 2, unresolved: 1, trial: 1 } });
  assert.deepEqual(h, { total: 6, answered: 4, attempts: 3, resolved: 2, resolvedShare: "67%", answerRate: "50%" });
  const empty = exitHeadline(null);
  assert.equal(empty.resolvedShare, "—");
  assert.equal(empty.answerRate, "—");
});

test("rows use the taxonomy's words and drop zero outcomes", () => {
  const sum = { outcomes: { resolved: 2, partial: 0, unresolved: 1, trial: 0, unanswered: 3 }, reasons: [{ id: "stuck", count: 2 }] };
  assert.deepEqual(outcomeRows(sum, TAX).map((r) => r.label), ["Resolved", "Didn't resolve", "No answer"]);
  assert.deepEqual(reasonRows(sum, TAX), [{ key: "stuck", label: "Got stuck or looped", value: 2, display: "2" }]);
});

test("exit cost reads like spend: unknown is a dash, unpriced says so, estimates carry ~", () => {
  assert.equal(fmtExitCost(null), "—");
  assert.equal(fmtExitCost({ cost: 0, unpriced: 3 }), "not priced");
  assert.equal(fmtExitCost({ cost: 1.234, estimated: 0 }), "$1.23");
  assert.equal(fmtExitCost({ cost: 1.2, estimated: 0.2 }), "~$1.20");
  assert.equal(fmtExitCost({ cost: 0.004, estimated: 0 }), "<$0.01");
  assert.equal(fmtExitCost({ cost: 0, unpriced: 0 }), "$0.00");
});

test("instructionsLine names each file with its short hash and the source", () => {
  assert.equal(instructionsLine(undefined), null);
  assert.deepEqual(instructionsLine({ source: "observed", files: [{ path: "AGENTS.md", sha: "591f4f5c793f" }, { path: "gone.md", sha: "" }] }),
    { text: "AGENTS.md @591f4f5c, gone.md (gone)", source: "as the CLI recorded them" });
  assert.equal(instructionsLine({ source: "declared", files: [] }).text, "None");
});

import { SKILL_MIN_ATTEMPTS, loadedSkillsLine, skillComparisonRows, skillStatsLine } from "./agentExit.js";

test("skills in Outcomes: with and without, few runs marked, coverage said", () => {
  const rows = skillComparisonRows({ rows: [
    { name: "pdf", versions: 2, with: { total: 4, resolved: 2, partial: 1, unresolved: 0, unanswered: 1 }, without: { total: 10, resolved: 3, partial: 2, unresolved: 5 } },
  ] });
  assert.equal(rows[0].with.share, "67%");
  assert.equal(rows[0].with.of, "2 of 3");
  assert.equal(rows[0].without.share, "30%");
  assert.equal(rows[0].few, true, "3 answered attempts is under " + SKILL_MIN_ATTEMPTS);
  assert.deepEqual(skillComparisonRows(null), []);
  assert.equal(skillStatsLine({ recorded: 0, unrecorded: 4 }), "No removed agent recorded its skills yet: agents started from now on do.");
  assert.equal(skillStatsLine({ recorded: 3, unrecorded: 2 }), "3 of 5 removed agents recorded their skills; the other 2 count on neither side.");
  assert.deepEqual(loadedSkillsLine({ source: "launch", skills: [{ name: "pdf", scope: "machine" }, { name: "tried", scope: "agent" }] }), { text: "pdf, tried (the agent's own)", source: "as it started" });
  assert.equal(loadedSkillsLine(null), null);
});
