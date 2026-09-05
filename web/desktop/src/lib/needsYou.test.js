import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { needsYou, verbLabel, methodLabel } from "./needsYou.js";

const dialog = { id: "ui-1", method: "confirm", title: "Allow this?", message: "m" };

describe("needsYou", () => {
  it("lists waiting agents first, then blocking inbox items newest first", () => {
    const workspaces = [{ id: "w1", name: "Proj", agents: [
      { id: "a1", name: "default", mode: "managed", waiting: true, dialog },
      { id: "a2", name: "idle", mode: "managed", waiting: false },
    ] }];
    const freeAgents = [{ id: "a3", name: "zed", mode: "managed", waiting: true, dialog: { id: "ui-2", method: "select", options: ["x", "‹ back"] } }];
    const inbox = [
      { id: "i1", kind: "approval", blocking: true, state: "unread", title: "Old", createdAt: "2026-09-01T10:00:00Z", allowedResponses: ["accept", "ignore"], sourceKind: "agent", sourceId: "a2" },
      { id: "i2", kind: "question", blocking: true, state: "unread", title: "New", createdAt: "2026-09-01T12:00:00Z", allowedResponses: ["respond"] },
      { id: "i3", kind: "fyi", blocking: false, state: "unread", title: "Nope", createdAt: "2026-09-01T13:00:00Z" },
      { id: "i4", kind: "approval", blocking: true, state: "done", title: "Done", createdAt: "2026-09-01T14:00:00Z" },
      { id: "i5", kind: "approval", blocking: true, state: "unread", title: "Later", createdAt: "2026-09-01T15:00:00Z", snoozedUntil: "2999-01-01T00:00:00Z" },
    ];
    const out = needsYou({ workspaces, freeAgents, inbox });
    assert.deepEqual(out.map((e) => e.key), ["ask:a1", "ask:a3", "inbox:i2", "inbox:i1"]);
    assert.equal(out[0].agentName, "Proj");
    assert.equal(out[0].where, "Proj");
    assert.equal(out[0].title, "Allow this?");
    assert.deepEqual(out[1].options, ["x"]);
    assert.equal(out[1].title, "Needs a choice");
    assert.equal(out[3].agentId, "a2");
    assert.deepEqual(out[3].verbs, ["accept", "ignore"]);
  });
  it("is empty and safe on missing input", () => {
    assert.deepEqual(needsYou({}), []);
    assert.deepEqual(needsYou({ workspaces: null, freeAgents: null, inbox: null }), []);
  });
  it("labels", () => {
    assert.equal(verbLabel("respond"), "Reply");
    assert.equal(verbLabel("weird"), "weird");
    assert.equal(methodLabel("input"), "Needs a line of text");
  });
});
