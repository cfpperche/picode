import test from "node:test";
import assert from "node:assert/strict";
import { sessionHostTab, sessionReveal } from "./sessionBrowser.js";

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

test("sessionReveal brings the session forward only to open it", () => {
  assert.equal(sessionReveal({ hostOpen: false, splitExists: true, method: "Runtime.evaluate" }), "open");
  assert.equal(sessionReveal({ hostOpen: false, splitExists: false, method: "shell.open" }), "open");
  assert.equal(sessionReveal({ hostOpen: true, splitExists: false, method: "Input.dispatchMouseEvent" }), "select");
  assert.equal(sessionReveal({ hostOpen: true, splitExists: true, method: "shell.open" }), "select");
  // The 2026-09-24 report: every click/evaluate/screenshot re-selected the
  // agent's tab and took the human's focus away.
  for (const method of ["Runtime.evaluate", "Input.dispatchMouseEvent", "Page.captureScreenshot", "Page.navigate"]) {
    assert.equal(sessionReveal({ hostOpen: true, splitExists: true, method }), "none");
  }
});
