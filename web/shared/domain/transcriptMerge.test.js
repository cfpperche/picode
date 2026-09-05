import { test } from "node:test";
import assert from "node:assert/strict";
import { reconcileTranscript, liveSince, transcriptGate } from "./transcriptMerge.js";
const tool = (id, status = "···", preview = null) => ({ kind: "tool", id, status, preview });
const block = (text, ts) => ({ kind: "block", cls: "", actor: "agent", text, ts });

test("history arriving after a frame retains active tool identity and preview", () => {
  const active = tool("a", "···", { image: "new" });
  const merged = reconcileTranscript([block("Before", 1), tool("a")], [active, tool("b")]);
  assert.deepEqual(merged, [block("Before", 1), active, tool("b")]);
});

test("completion ordering matrix: final beats pending on either side", () => {
  for (const [stored, live, expected] of [
    [tool("a"), tool("a", "ok", { image: "final" }), "final"],
    [tool("a", "ok", { image: "persisted" }), tool("a", "···", { image: "old" }), "persisted"],
    [tool("a", "ok", { image: "persisted" }), tool("a", "ok", { image: "new" }), "new"],
  ]) {
    const result = reconcileTranscript([stored], [live]);
    assert.equal(result.length, 1);
    assert.equal(result[0].preview.image, expected);
    assert.equal(result[0].status, "ok");
  }
});

test("missing final frame stays absent; tool names do not establish identity", () => {
  const result = reconcileTranscript([tool("a", "ok")], [tool("a", "···", { image: "temporary" }), tool("b")]);
  assert.equal(result[0].preview, null);
  assert.equal(result.length, 2);
});

test("repeated equal messages match in order and different timestamps stay distinct", () => {
  const history = [block("OK", 1), tool("a", "ok"), block("OK", 2)];
  assert.deepEqual(reconcileTranscript(history, history), history);
  assert.equal(reconcileTranscript(history, [block("OK", 3)]).length, 4);
});

test("snapshot asks and live deltas survive history and do not duplicate tools", () => {
  const ask = { kind: "ask", id: "q", status: "open" };
  const active = tool("a");
  const result = reconcileTranscript([active], [ask, active, block("Typing", undefined)]);
  assert.equal(result.filter((it) => it.kind === "tool").length, 1);
  assert.equal(result.filter((it) => it.kind === "ask").length, 1);
  assert.equal(result.filter((it) => it.kind === "block").length, 1);
});

test("only changed/running suffix survives a read; old compaction prefix is not restored", () => {
  const baseline = [block("Old", 1), tool("a", "ok")];
  assert.deepEqual(liveSince(baseline, baseline), []);
  const next = [...baseline, tool("b")];
  assert.deepEqual(liveSince(next, baseline), [tool("b")]);
  assert.deepEqual(liveSince(next, next), [tool("b")]);
  assert.deepEqual(liveSince(next, next, false), []);
  const completed = [...baseline, tool("b", "ok")];
  assert.deepEqual(liveSince(completed, next), [tool("b", "ok")]);
});

test("newer request, agent switch, and switch-away-and-back invalidate old reads", () => {
  const gate = transcriptGate();
  const first = gate.begin();
  assert.ok(gate.current(first));
  const second = gate.begin();
  assert.ok(!gate.current(first));
  assert.ok(gate.current(second));
  gate.invalidate(); gate.invalidate();
  assert.ok(!gate.current(second));
  const third = gate.begin();
  assert.ok(gate.current(third));
});
