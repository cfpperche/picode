import assert from "node:assert/strict";
import { test } from "node:test";
import { locate, firstAgentId, displayAgentName, agentsOf } from "./tree.js";

const ws = { id: "w1", name: "Repo", agents: [{ id: "a1", name: "default" }, { id: "a2", name: "review" }] };

test("locate agent and legacy workspace id", () => {
  const hit = locate([ws], [], "a2");
  assert.equal(hit.agent.id, "a2");
  assert.equal(locate([ws], [], "w1").agent.id, "a1");
});

test("firstAgentId and display name", () => {
  assert.equal(firstAgentId([ws], []), "a1");
  assert.equal(displayAgentName({ name: "default" }, ws), "Repo");
  assert.equal(displayAgentName({ name: "review" }, ws), "review");
  assert.equal(agentsOf(ws).length, 2);
});
