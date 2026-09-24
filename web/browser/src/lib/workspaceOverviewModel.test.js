import test from "node:test";
import assert from "node:assert/strict";
import { activeMissions, attentionItems, activityLabel, recentActivity } from "./workspaceOverviewModel.js";

test("attention favors scoped questions, review and blocked missions without duplicate agent asks", () => {
  const rows = attentionItems({ workspaceId: "w1", inbox: [
    { id: "i1", title: "Choose", workspaceId: "w1", sourceId: "a1", state: "unread" },
    { id: "i2", title: "Elsewhere", workspaceId: "w2", state: "unread" },
  ], missions: [{ id: "m1", title: "Fix", state: "blocked" }, { id: "m2", title: "Review", state: "in-review" }],
  agents: [{ id: "a1" }, { id: "a2", name: "Second" }], statusOf: () => "needs-you",
  pr: { url: "https://example.test/pr", checks: { failed: 1 } } });
  assert.deepEqual(rows.map((r) => r.key), ["inbox:i1", "mission:m1", "mission:m2", "agent:a2", "checks"]);
});

test("mission order and recent activity preserve useful work and visit boundary", () => {
  assert.deepEqual(activeMissions([{ id: "done", state: "completed" }, { id: "ready", state: "ready" }, { id: "blocked", state: "blocked" }]).map((m) => m.id), ["blocked", "ready"]);
  const now = Date.parse("2026-09-24T12:00:00Z");
  const rows = recentActivity([{ id: 1, kind: "mission", action: "block", createdAt: "2026-09-24T11:00:00Z" }],
    [{ hash: "old", subject: "Earlier", at: Date.parse("2026-09-23T00:00:00Z") / 1000 }, { hash: "new", subject: "Commit", at: Date.parse("2026-09-24T11:30:00Z") / 1000 }], Date.parse("2026-09-24T10:00:00Z"), now);
  assert.deepEqual(rows.map((r) => r.id), ["new", 1]);
  assert.equal(activityLabel(rows[1]), "Mission blocked");
});
