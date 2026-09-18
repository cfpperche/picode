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

  // ADR-0082 decision table: live sidecar frames ride capture_frame events
  describe("sidecar capture frames", () => {
    const png = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aX1kAAAAASUVORK5CYII=";
    const frame = (seq, extra = {}) => ({ toolCallId: "s1", seq, ts: 1000 + seq, image: png, url: "http://127.0.0.1:9/p", title: "P", ...extra });
    const start = [
      { type: "agent_start" },
      { type: "tool_execution_start", toolCallId: "s1", toolName: "agent_browser", args: { args: ["open", "http://127.0.0.1:9/"] } },
    ];
    const item = (state, id = "s1") => state.items.find((it) => it.kind === "tool" && it.id === id);

    it("a running tool shows the newest frame and keeps its sequence", () => {
      const { state } = run([...start, { type: "capture_frame", ...frame(3) }, { type: "capture_frame", ...frame(4) }]);
      assert.equal(item(state).preview.image, png);
      assert.equal(item(state).captureSeq, 4);
      assert.equal(item(state).status, "···");
    });
    it("an out-of-order frame is ignored", () => {
      const { state } = run([...start, { type: "capture_frame", ...frame(5) }, { type: "capture_frame", ...frame(4) }]);
      assert.equal(item(state).captureSeq, 5);
    });
    it("a frame for another tool call never lands", () => {
      const { state } = run([...start, { type: "capture_frame", ...frame(1), toolCallId: "other" }]);
      assert.equal(item(state).preview, null);
    });
    it("a final frame lands even after the tool result settled", () => {
      const { state } = run([...start,
        { type: "capture_frame", ...frame(2) },
        { type: "tool_execution_end", toolCallId: "s1", isError: false, result: { content: [] } },
        { type: "capture_frame", ...frame(3, { final: true }) },
      ]);
      const it2 = item(state);
      assert.equal(it2.status, "ok");
      assert.equal(it2.captureSeq, 3);
      assert.equal(it2.preview.image, png);
    });
    it("an invalid frame never touches the item", () => {
      const { state } = run([...start, { type: "capture_frame", toolCallId: "s1", seq: 1, image: "https://example.com/not-data" }]);
      assert.equal(item(state).preview, null);
    });
  });

  // ADR-0057 decision table: tool_execution_update carries preview frames
  describe("tool live preview", () => {
    const frame = { image: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aX1kAAAAASUVORK5CYII=", url: "https://example.com/", title: "Example" };
    const frame2 = { image: frame.image, title: "Second capture" };
    const start = [
      { type: "agent_start" },
      { type: "tool_execution_start", toolCallId: "b1", toolName: "agent_browser", args: { args: ["open", "https://example.com"] } },
    ];
    const item = (state, id = "b1") => state.items.find((it) => it.kind === "tool" && it.id === id);

    it("a partial result with details.preview sets the live frame", () => {
      const { state } = run([...start, { type: "tool_execution_update", toolCallId: "b1", partialResult: { details: { preview: frame } } }]);
      assert.deepEqual(item(state).preview, { image: frame.image, url: frame.url, title: frame.title });
      assert.equal(item(state).status, "···");
    });
    it("a second frame replaces the first — never accumulates", () => {
      const { state } = run([...start,
        { type: "tool_execution_update", toolCallId: "b1", partialResult: { details: { preview: frame } } },
        { type: "tool_execution_update", toolCallId: "b1", partialResult: { details: { preview: frame2 } } },
      ]);
      assert.equal(item(state).preview.image, frame2.image);
    });
    it("an update without a valid preview leaves the item untouched", () => {
      const { state } = run([...start,
        { type: "tool_execution_update", toolCallId: "b1", partialResult: { details: { preview: frame } } },
        { type: "tool_execution_update", toolCallId: "b1", partialResult: { details: { other: 1 } } },
        { type: "tool_execution_update", toolCallId: "b1", partialResult: null },
      ]);
      assert.equal(item(state).preview.image, frame.image);
    });
    it("an update for an unknown or non-tool id is ignored", () => {
      const { state } = run([...start,
        { type: "tool_execution_update", toolCallId: "ghost", partialResult: { details: { preview: frame } } },
      ]);
      assert.equal(item(state).preview, null);
    });
    it("the computer tool's end result carries its capture (ADR-0148)", () => {
      const { state } = run([
        { type: "agent_start" },
        { type: "tool_execution_start", toolCallId: "k1", toolName: "computer", args: { action: "left_click", coordinate: [412, 88] } },
        { type: "tool_execution_end", toolCallId: "k1", toolName: "computer", result: { details: { preview: { ...frame2, source: "computer" } } }, isError: false },
      ]);
      const it = item(state, "k1");
      assert.equal(it.status, "ok");
      assert.equal(it.args, "left_click [412,88]");
      assert.equal(it.preview.image, frame2.image);
      assert.equal(it.preview.source, "computer");
    });
    it("the final result is authoritative: an end preview replaces the live one", () => {
      const { state } = run([...start,
        { type: "tool_execution_update", toolCallId: "b1", partialResult: { details: { preview: frame } } },
        { type: "tool_execution_end", toolCallId: "b1", toolName: "agent_browser", result: { details: { preview: frame2 } }, isError: false },
      ]);
      assert.equal(item(state).status, "ok");
      assert.equal(item(state).preview.image, frame2.image);
    });
    it("an explicit unavailable marker on end clears the live frame (ADR-0057)", () => {
      const { state } = run([...start,
        { type: "tool_execution_update", toolCallId: "b1", partialResult: { details: { preview: frame } } },
        { type: "tool_execution_end", toolCallId: "b1", toolName: "agent_browser", result: { details: { preview: { unavailable: true } } }, isError: false },
      ]);
      assert.equal(item(state).preview, null);
      assert.equal(item(state).previewError, "Capture unavailable");
    });
    it("a silent end result keeps the live frame — the sidecar owns it (ADR-0082)", () => {
      const { state } = run([...start,
        { type: "tool_execution_update", toolCallId: "b1", partialResult: { details: { preview: frame } } },
        { type: "tool_execution_end", toolCallId: "b1", toolName: "agent_browser", result: {}, isError: false },
      ]);
      assert.equal(item(state).preview.image, frame.image);
    });
    it("duplicate starts, older captures and updates after completion cannot replace the row", () => {
      const state = run([...start, start[1],
        { type: "tool_execution_update", toolCallId: "b1", partialResult: { details: { preview: { ...frame, ts: 200 } } } },
        { type: "tool_execution_update", toolCallId: "b1", partialResult: { details: { preview: { ...frame2, ts: 100 } } } },
      ]).state;
      assert.equal(state.items.filter((it) => it.kind === "tool").length, 1);
      assert.equal(item(state).preview.title, "Example");
      const ended = reduceAgentEvent(state, { type: "tool_execution_end", toolCallId: "b1", result: {}, isError: true }).state;
      const late = reduceAgentEvent(ended, { type: "tool_execution_update", toolCallId: "b1", partialResult: { details: { preview: frame } } }).state;
      assert.equal(item(late).preview.title, "Example");
      assert.equal(item(late).status, "error");
    });
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
