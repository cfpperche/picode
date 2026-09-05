import assert from "node:assert/strict";
import { test } from "node:test";
import { locate, firstAgentId, displayAgentName, agentsOf, paneContext, mentionAgents } from "./tree.js";

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
  assert.equal(agentsOf({ id: "w2", name: "Empty", agents: [] }).length, 0);
  assert.equal(agentsOf({ id: "w3", name: "Ghost", agents: [], agent: {} }).length, 0);
});

test("mentionAgents skips the open agent", () => {
  const got = mentionAgents([ws], [{ id: "f1", name: "Grok" }], "a2");
  assert.deepEqual(got.map((a) => a.id), ["a1", "f1"]);
  assert.equal(got[0].name, "Repo");
});

test("paneContext names agent and folder once", () => {
  assert.equal(paneContext("web_search", ""), "web_search");
  assert.equal(paneContext("review", "Repo"), "review · Repo");
  assert.equal(paneContext("Repo", "Repo"), "Repo");
  assert.equal(paneContext("", ""), "");
});
