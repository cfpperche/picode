import assert from "node:assert/strict";
import { test } from "node:test";
import { putAsk, answerAsk, timeoutAsk, cancelOpenAsks, stitchIndex, askJustAnswered, fieldLabel, summaryLine, backAsk } from "./askForm.js";

const user = { kind: "block", cls: "user", text: "/roles edit" };
const sel = (id, title, options) => ({ id, method: "select", title, options });

test("putAsk decision table", () => {
  const rows = [
    {
      name: "1 no prior ask → new card",
      items: [user],
      dialog: sel("a", "Edit which role?", ["default"]),
      wantCards: 1,
      wantSteps: 1,
    },
    {
      name: "2 cancelled prior → new card",
      items: [user, { kind: "ask", id: "a", status: "cancelled", steps: [{ id: "a", status: "cancelled" }] }],
      dialog: sel("b", "Edit which role?", ["default"]),
      wantCards: 2,
      wantSteps: 1,
    },
    {
      name: "3 answered prior → append step",
      items: [user, {
        kind: "ask", id: "a", status: "answered",
        steps: [{ id: "a", status: "answered", answer: "default" }],
      }],
      dialog: sel("b", "provider", ["xai"]),
      wantCards: 1,
      wantSteps: 2,
    },
    {
      name: "4 assistant reply between → new card",
      items: [user, {
        kind: "ask", id: "a", status: "answered",
        steps: [{ id: "a", status: "answered", answer: "default" }],
      }, { kind: "block", cls: "", text: "ok" }],
      dialog: sel("b", "next", ["x"]),
      wantCards: 2,
      wantSteps: 1,
    },
  ];
  for (const row of rows) {
    const out = putAsk(row.items, row.dialog, "open");
    const asks = out.filter((it) => it.kind === "ask");
    assert.equal(asks.length, row.wantCards, row.name + " cards");
    const last = asks[asks.length - 1];
    assert.equal((last.steps || []).length, row.wantSteps, row.name + " steps");
    assert.equal(last.id, row.dialog.id, row.name + " id");
    assert.equal(last.status, "open", row.name + " open");
  }
});

test("putAsk same id updates the open step", () => {
  const first = putAsk([user], sel("a", "one", ["x"]), "open");
  const out = putAsk(first, sel("a", "one?", ["x", "y"]), "open");
  assert.equal(out.filter((it) => it.kind === "ask").length, 1);
  assert.deepEqual(out[1].options, ["x", "y"]);
});

test("answerAsk then stitch grows one card", () => {
  let items = putAsk([user], sel("a", "role", ["default", "vision"]), "open");
  items = answerAsk(items, "a", "default", false);
  assert.equal(items[1].status, "answered");
  assert.equal(items[1].steps[0].answer, "default");
  items = putAsk(items, sel("b", "provider", ["xai", "anthropic"]), "open");
  assert.equal(items.filter((it) => it.kind === "ask").length, 1);
  assert.equal(items[1].steps.length, 2);
  assert.equal(items[1].status, "open");
  items = answerAsk(items, "b", "anthropic", false);
  items = putAsk(items, sel("c", "model", ["opus"]), "open");
  items = answerAsk(items, "c", "opus", false);
  assert.equal(items[1].steps.map((s) => s.answer).join(" · "), "default · anthropic · opus");
  assert.equal(askJustAnswered(items), true);
});

test("fieldLabel and summaryLine", () => {
  assert.equal(fieldLabel("Edit which role?"), "Role");
  assert.equal(fieldLabel("Model for vision — provider"), "Provider");
  assert.equal(fieldLabel("Model for vision — model"), "Model");
  assert.equal(fieldLabel("Thinking level"), "Thinking");
  assert.equal(
    summaryLine([
      { status: "answered", answer: "vision" },
      { status: "answered", answer: "xai" },
      { status: "answered", answer: "grok-4.5" },
      { status: "answered", answer: "medium" },
    ]),
    "vision — xai/grok-4.5 · medium",
  );
});

test("backAsk keeps earlier pills", () => {
  let items = putAsk([user], sel("a", "role", ["vision"]), "open");
  items = answerAsk(items, "a", "vision", false);
  items = putAsk(items, sel("b", "provider", ["xai"]), "open");
  items = backAsk(items, "b", 0);
  assert.equal(items[1].steps.length, 0);
  assert.equal(items[1].status, "open");
});

test("timeout and cancel only close the open step", () => {
  let items = putAsk([user], sel("a", "role", ["default"]), "open");
  items = timeoutAsk(items, "a");
  assert.equal(items[1].status, "timeout");
  items = putAsk([user], sel("b", "role", ["default"]), "open");
  items = cancelOpenAsks(items);
  assert.equal(items[1].status, "cancelled");
  assert.equal(stitchIndex(items), -1);
});
