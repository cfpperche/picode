import assert from "node:assert/strict";
import { test } from "node:test";
import {
  emptyExitDraft, exitHeadline, exitRequestBody, fmtLifetime, fmtTurns, outcomeRows,
  pickOutcome, reasonRows, takesReasons, toggleReason,
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

test("lifetimes and turns read in whole units, unmeasured turns as a dash", () => {
  assert.equal(fmtLifetime(42), "42 s");
  assert.equal(fmtLifetime(125), "2 min");
  assert.equal(fmtLifetime(7300), "2 h");
  assert.equal(fmtLifetime(3 * 86400 + 5), "3 d");
  assert.equal(fmtLifetime(-4), "0 s");
  assert.equal(fmtTurns(null), "—");
  assert.equal(fmtTurns(0), "0");
});

test("the headline divides by what it counts, never by zero", () => {
  const h = exitHeadline({ total: 5, asked: 4, answered: 3, askedAnswered: 2, outcomes: { resolved: 2 } });
  assert.deepEqual(h, { total: 5, answered: 3, resolved: 2, resolvedShare: "67%", answerRate: "50%" });
  const empty = exitHeadline(null);
  assert.equal(empty.resolvedShare, "—");
  assert.equal(empty.answerRate, "—");
});

test("rows use the taxonomy's words and drop zero outcomes", () => {
  const sum = { outcomes: { resolved: 2, partial: 0, unresolved: 1, trial: 0, unanswered: 3 }, reasons: [{ id: "stuck", count: 2 }] };
  assert.deepEqual(outcomeRows(sum, TAX).map((r) => r.label), ["Resolved", "Didn't resolve", "No answer"]);
  assert.deepEqual(reasonRows(sum, TAX), [{ key: "stuck", label: "Got stuck or looped", value: 2, display: "2" }]);
});
