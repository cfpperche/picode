import test from "node:test";
import assert from "node:assert/strict";
import { missionLocation, missionActions, missionDraft, missionDraftPayload, latestMissionEvidence } from "./missions.js";
import { missionDraftSchema } from "../contracts/schemas.js";

test("mission routes preserve workspace filters and reject malformed ids", () => {
  assert.deepEqual(missionLocation("#/missions/new?workspace=w"), { id: "", create: true, workspace: "w" });
  assert.equal(missionLocation("#/mission/%E0"), null);
  assert.equal(missionLocation("#/mission/task").id, "task");
});
test("draft keeps the objective and criterion identity", () => {
  const v = { id: "m", workspaceId: "w", title: "Task", objective: "Result", criteria: [{ id: "c", text: "Works" }] };
  const draft = missionDraft(v);
  assert.equal(missionDraftSchema.safeParse(draft).success, true);
  assert.deepEqual(missionDraftPayload(draft, v).criteria, v.criteria);
  assert.equal(missionDraftSchema.safeParse({ ...draft, criteriaText: "" }).success, false);
});
test("uncertain and terminal reservations never offer another send", () => {
  const v = { state: "ready", assignment: { reserved: true, delivery: "unconfirmed" } };
  assert.equal(missionActions(v).includes("dispatch"), false);
  assert.equal(missionActions(v).includes("acknowledge"), true);
  assert.deepEqual(missionActions({ ...v, state: "cancelled" }), ["release"]);
  assert.deepEqual(missionActions({ state: "completed" }), ["reopen", "archive"]);
});
test("evidence is specific to current scope and revision", () => {
  const v = { scopeVersion: 2, repository: "r", evidence: [{ criterionId: "c", scopeVersion: 1, revision: "a", outcome: "pass" }, { criterionId: "c", scopeVersion: 2, revision: "b", outcome: "pass" }] };
  assert.equal(latestMissionEvidence(v, "c", "a"), null);
  assert.equal(latestMissionEvidence(v, "c", "b").outcome, "pass");
});

test("lost response reuses its exact receipt after a form reload changes the version", async () => {
  const { missionRequestReceipt } = await import("./missions.js");
  let nonce = 0;
  const id = () => "request-" + ++nonce;
  const original = missionRequestReceipt("m", "report", { note: "Checkpoint", expectedVersion: 7 }, 7, null, id);
  const restored = missionRequestReceipt("m", "report", { note: "Checkpoint", expectedVersion: 8 }, 8, original, id);
  assert.equal(restored, original);
  assert.equal(restored.payload.expectedVersion, 7);
  assert.equal(nonce, 1);
  const changed = missionRequestReceipt("m", "report", { note: "Different checkpoint", expectedVersion: 8 }, 8, original, id);
  assert.equal(changed.requestId, "request-2");
});
