import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { reduceAgentEvent, initialAgentState, markSent, markUndelivered, markAborted } from "./agentEvents.js";

function run(events, start = initialAgentState) {
  let state = start;
  const effects = [];
  for (const ev of events) {
    const r = reduceAgentEvent(state, ev, 1000);
    state = r.state;
    effects.push(...r.effects);
  }
  return { state, effects };
}

const ask = { type: "extension_ui_request", id: "ui-1", method: "confirm", title: "Allow this?", message: "yes or no" };

describe("reduceAgentEvent", () => {
  it("snapshot restores streaming/waiting and the open dialog", () => {
    const { state } = run([{ type: "snapshot", streaming: false, waiting: true, dialog: { id: "d1", method: "select", title: "Pick", options: ["a", "b"] } }]);
    assert.equal(state.waiting, true);
    assert.equal(state.status, "waiting");
    const card = state.items.find((it) => it.kind === "ask");
    assert.ok(card && card.status === "open");
  });
  it("snapshot without waiting cancels a restored ghost card", () => {
    const first = run([ask]).state;
    const { state } = run([{ type: "snapshot", streaming: false, waiting: false }], first);
    assert.equal(state.items.find((it) => it.kind === "ask").status, "cancelled");
    assert.equal(state.status, "idle");
  });
  it("appends deltas to the running block and keeps thinking separate", () => {
    const { state, effects } = run([
      { type: "agent_start" },
      { type: "message_update", assistantMessageEvent: { type: "text_delta", delta: "Hel" } },
      { type: "message_update", assistantMessageEvent: { type: "text_delta", delta: "lo" } },
      { type: "message_update", assistantMessageEvent: { type: "thinking_delta", delta: "hm" } },
    ]);
    assert.equal(state.streaming, true);
    assert.equal(state.status, "streaming");
    assert.deepEqual(state.items.map((it) => [it.cls, it.text]), [["", "Hello"], ["thinking", "hm"]]);
    assert.ok(effects.every((e) => e.type === "scroll"));
  });
  it("tool start/end pair up by id", () => {
    const { state } = run([
      { type: "tool_execution_start", toolCallId: "t1", toolName: "bash", args: { command: "ls" } },
      { type: "tool_execution_end", toolCallId: "t1", toolName: "bash", result: { output: "a" }, isError: false },
    ]);
    const tool = state.items.find((it) => it.kind === "tool");
    assert.equal(tool.status, "ok");
    assert.equal(tool.name, "bash");
  });
  it("a dialog request flips to waiting and opens a card; the answer path is the hook's", () => {
    const { state, effects } = run([{ type: "agent_start" }, ask]);
    assert.equal(state.waiting, true);
    assert.equal(state.status, "waiting");
    assert.equal(state.items.filter((it) => it.kind === "ask").length, 1);
    assert.ok(effects.some((e) => e.type === "scroll"));
  });
  it("timeout and exit close open cards", () => {
    const t = run([ask, { type: "extension_ui_timeout", id: "ui-1" }]).state;
    assert.equal(t.waiting, false);
    assert.equal(t.items.find((it) => it.kind === "ask").status, "timeout");
    const x = run([{ type: "agent_start" }, ask, { type: "exit" }]).state;
    assert.equal(x.waiting, false);
    assert.equal(x.streaming, false);
    assert.equal(x.items.find((it) => it.kind === "ask").status, "cancelled");
  });
  it("a notify with no card and no slash command is a toast", () => {
    const { effects } = run([{ type: "extension_ui_request", method: "notify", message: "Heads up", notifyType: "info" }]);
    assert.deepEqual(effects, [{ type: "toast", level: "info", text: "Heads up" }]);
  });
  it("enqueue_accepted materialises the pending payload once", () => {
    const start = { ...initialAgentState, pendingPayload: "do it" };
    const { state } = run([{ type: "enqueue_accepted", kind: "prompt" }, { type: "enqueue_accepted", kind: "prompt" }], start);
    assert.equal(state.items.filter((it) => it.cls === "user").length, 1);
    assert.equal(state.pendingPayload, "");
  });
  it("task_failed and enqueue_rejected surface errors", () => {
    const f = run([{ type: "agent_start" }, { type: "task_failed", error: "boom" }]).state;
    assert.equal(f.streaming, false);
    assert.equal(f.items.at(-1).kind, "alert");
    const { effects } = run([{ type: "enqueue_rejected", error: "no" }]);
    assert.equal(effects[0].type, "toast");
    assert.equal(effects[0].level, "error");
  });
  it("unknown events are ignored", () => {
    const { state, effects } = run([{ type: "whatever" }]);
    assert.deepEqual(state, initialAgentState);
    assert.deepEqual(effects, []);
  });
});

describe("local transitions", () => {
  it("markSent is optimistic only when the agent is idle", () => {
    const idle = markSent(initialAgentState, { kind: "prompt", text: "hi", ts: 5, busy: false });
    assert.equal(idle.streaming, true);
    assert.equal(idle.items.at(-1).chip, "prompt");
    const busy = markSent({ ...initialAgentState, waiting: true, status: "waiting" }, { kind: "follow_up", text: "later", ts: 6, busy: true });
    assert.equal(busy.streaming, false);
    assert.equal(busy.status, "waiting");
  });
  it("markUndelivered annotates the bubble and stops the optimistic turn", () => {
    const s = markUndelivered(markSent(initialAgentState, { kind: "prompt", text: "hi", ts: 5, busy: false }), 5, "offline");
    assert.match(s.items.at(-1).text, /not delivered: offline/);
    assert.equal(s.streaming, false);
  });
  it("markAborted drops queued steers and closes cards", () => {
    const start = run([{ type: "agent_start" }, ask]).state;
    const withSteer = { ...start, items: [...start.items, { kind: "block", cls: "user", chip: "steer", text: "x" }] };
    const s = markAborted(withSteer);
    assert.equal(s.streaming, false);
    assert.equal(s.waiting, false);
    assert.equal(s.items.find((it) => it.chip === "steer").dropped, true);
    assert.equal(s.items.find((it) => it.kind === "ask").status, "cancelled");
  });
});
