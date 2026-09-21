import test from "node:test";
import assert from "node:assert/strict";
import { sessionHostTab } from "./sessionBrowser.js";

test("sessionHostTab binds to the session the human is watching", () => {
  assert.equal(sessionHostTab({
    agentId: "ag-1",
    termId: "term-9",
    openTabs: ["ag-1", "t:term-9"],
    agentTerminalId: "term-9",
  }), "t:term-9");
  // The agent's own terminal wins over a chat tab when the caller did not
  // name a different terminal and the chat is not the running session.
  assert.equal(sessionHostTab({
    agentId: "ag-1",
    openTabs: ["ag-1", "t:term-9"],
    agentTerminalId: "term-9",
  }), "t:term-9");
  assert.equal(sessionHostTab({
    agentId: "ag-1",
    openTabs: ["ag-1"],
  }), "ag-1");
  // Not open yet: still name the terminal the CLI runs in, so the page can open it.
  assert.equal(sessionHostTab({
    agentId: "ag-1",
    termId: "term-9",
    openTabs: [],
  }), "t:term-9");
  assert.equal(sessionHostTab({}), "");
});
