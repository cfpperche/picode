import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { agentOwnerWs, termOwnerWs } from "./workBack.js";

// Decision table: Back from a pushed agent/terminal lands on the Work
// view the resource belongs to. wsId contract: string = owning
// workspace, null = free, undefined = fleet not answered yet.
describe("workBack ownership", () => {
  it("a workspace agent backs into its workspace group", () => {
    assert.equal(agentOwnerWs({ agent: { id: "a1" }, workspace: { id: "ws1" } }), "ws1");
  });
  it("a free agent backs into the Agents list", () => {
    assert.equal(agentOwnerWs({ agent: { id: "a1" }, workspace: null }), null);
  });
  it("an agent the fleet has not found yet keeps the legacy parent", () => {
    assert.equal(agentOwnerWs(null), undefined);
  });
  it("a workspace terminal backs into its workspace group", () => {
    assert.equal(termOwnerWs({ id: "t1", workspaceId: "ws1" }), "ws1");
  });
  it("a free terminal backs into the Terminals list", () => {
    assert.equal(termOwnerWs({ id: "t1", workspaceId: "ws_free" }), null);
    assert.equal(termOwnerWs({ id: "t1" }), null);
  });
  it("a terminal the fleet has not found yet keeps the legacy parent", () => {
    assert.equal(termOwnerWs(null), undefined);
  });
});
