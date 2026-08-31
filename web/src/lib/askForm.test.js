import assert from "node:assert/strict";
import { test } from "node:test";
import { putAsk, answerAsk, timeoutAsk, cancelOpenAsks, stitchIndex, askJustAnswered, fieldLabel, summaryLine, summaryParts, backAsk, walkReply, noteAsk, unanswerAsk, BACK } from "./askForm.js";

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
  // row 3: full four-step edit → one definition line
  assert.equal(
    summaryLine([
      { status: "answered", title: "Edit which role?", answer: "vision" },
      { status: "answered", title: "Model for vision — provider", answer: "xai" },
      { status: "answered", title: "Model for vision — model", answer: "grok-4.5" },
      { status: "answered", title: "Thinking level", answer: "medium" },
    ]),
    "vision — xai/grok-4.5 · medium",
  );
  // thinking "none" is not part of the definition
  assert.equal(
    summaryLine([
      { status: "answered", title: "Model for fast — provider", answer: "xai" },
      { status: "answered", title: "Model for fast — model", answer: "grok-4.5" },
      { status: "answered", title: "Thinking level", answer: "none" },
    ]),
    "xai/grok-4.5",
  );
});

test("row 2: role-only pick is labeled, and real when the notify said the model", () => {
  const steps = [{ status: "answered", title: "Roles (current: auto)", answer: "default" }];
  assert.equal(summaryLine(steps), "Role — default");
  assert.equal(
    summaryLine(steps, "xai/grok-4.6 · high · lock /default"),
    "default — xai/grok-4.6 · high",
  );
});

test("skipped provider select: the note fills the provider in", () => {
  const steps = [
    { status: "answered", title: "Edit which role?", answer: "vision" },
    { status: "answered", title: "Model for vision — model", answer: "grok-4.5" },
    { status: "answered", title: "Thinking level", answer: "medium" },
  ];
  assert.equal(summaryLine(steps, "Saved vision → xai/grok-4.5"), "vision — xai/grok-4.5 · medium");
  // no note: honest fallback still names what was picked
  assert.equal(summaryLine(steps), "vision — grok-4.5 · medium");
});

test("Save to step: labeled Save, workspace suffix on the definition line", () => {
  assert.equal(fieldLabel("Save to"), "Save");
  const steps = [
    { status: "answered", title: "Edit which role?", answer: "vision" },
    { status: "answered", title: "Model for vision — provider", answer: "xai" },
    { status: "answered", title: "Model for vision — model", answer: "grok-4.5" },
    { status: "answered", title: "Thinking level", answer: "medium" },
  ];
  assert.equal(
    summaryLine([...steps, { status: "answered", title: "Save to", answer: "workspace" }]),
    "vision — xai/grok-4.5 · medium (workspace)",
  );
  assert.equal(
    summaryLine([...steps, { status: "answered", title: "Save to", answer: "this agent" }]),
    "vision — xai/grok-4.5 · medium",
  );
});

test("clear flow: labeled Clear, line is the extension's result", () => {
  assert.equal(fieldLabel("Clear which config?"), "Clear");
  const steps = [
    { status: "answered", title: "Clear which config?", answer: "workspace" },
    { status: "answered", title: "Delete this roles file?", answer: "Yes" },
  ];
  assert.equal(summaryLine(steps, "Cleared .pi/roles.json"), "Cleared .pi/roles.json");
  assert.equal(summaryLine(steps), "workspace · Yes");
});

test("regression: 'Delete this roles file?' must not read as the Role field", () => {
  // fieldLabel's role-picker match ("roles?") once also matched any title
  // that merely mentions "roles" in passing — the delete-confirm title —
  // mislabeling its "Yes" answer as a role name and corrupting the line.
  assert.equal(fieldLabel("Delete this roles file?"), "Clear");
  assert.notEqual(fieldLabel("Delete this roles file?"), "Role");
});

test("/roles clear <scope> (arg form, no select step) still folds the note", () => {
  // /roles clear agent skips the "Clear which config?" select and goes
  // straight to confirm — one step only, and it alone must mark the flow
  // as a clear so the note (not a note/model false-positive) wins.
  const steps = [{ status: "answered", title: "Delete this roles file?", answer: "Yes" }];
  assert.equal(
    summaryLine(steps, "Cleared .pi/roles/qa-213680.json"),
    "Cleared .pi/roles/qa-213680.json",
  );
});

test("role picker titles are unaffected by the narrower regex", () => {
  assert.equal(fieldLabel("Roles (current: auto)"), "Role");
  assert.equal(fieldLabel("Edit which role?"), "Role");
});

test("noteAsk folds the completion notify into the answered card once", () => {
  let items = putAsk([user], sel("a", "Roles (current: auto)", ["auto", "default"]), "open");
  items = answerAsk(items, "a", "default", false);
  items = noteAsk(items, "xai/grok-4.6 · high · lock /default");
  assert.equal(items[1].note, "xai/grok-4.6 · high · lock /default");
  const again = noteAsk(items, "second notice");
  assert.equal(again[1].note, "xai/grok-4.6 · high · lock /default");
  // an assistant block after the card means the notify is not its result
  const later = noteAsk([...items, { kind: "block", cls: "", text: "hi" }], "x");
  assert.equal(later[1].note, items[1].note);
});

test("unanswerAsk reopens a step the server rejected", () => {
  let items = putAsk([user], sel("a", "Model for vision — provider", ["xai", "zai"]), "open");
  items = answerAsk(items, "a", "xai", false);
  assert.equal(items[1].status, "answered");
  items = unanswerAsk(items, "a");
  assert.equal(items[1].status, "open");
  assert.equal(items[1].steps[0].status, "open");
  assert.equal(items[1].steps[0].answer, "");
});

test("backAsk reopens the clicked pill", () => {
  let items = putAsk([user], sel("a", "Edit which role?", ["vision"]), "open");
  items = answerAsk(items, "a", "vision", false);
  items = putAsk(items, sel("b", "Model for vision — provider", ["xai", "anthropic"]), "open");
  items = answerAsk(items, "b", "xai", false);
  items = putAsk(items, sel("c", "Model for vision — model", ["grok-4.6"]), "open");
  items = answerAsk(items, "c", "grok-4.6", false);
  items = putAsk(items, sel("d", "Thinking level", ["medium"]), "open");
  items = backAsk(items, "d", 1);
  assert.equal(items[1].backTo, "Provider");
  assert.equal(items[1].steps.length, 2);
  assert.equal(items[1].steps[0].answer, "vision");
  assert.equal(items[1].steps[1].status, "open");
  assert.equal(items[1].steps[1].answer, "");
  // row 4: wrong fields on the way back are answered BACK, never shown
  assert.equal(walkReply(items, sel("e", "Model for vision — model", ["grok-4.6", BACK])), BACK);
  // a dialog that cannot go back is shown rather than eaten
  assert.equal(walkReply(items, sel("e2", "Edit which role?", ["vision"])), "");
  // the target field ends the walk
  assert.equal(walkReply(items, sel("f", "Model for vision — provider", ["xai", BACK])), "");
  items = putAsk(items, sel("f", "Model for vision — provider", ["xai", "anthropic", BACK]), "open");
  assert.equal(items[1].steps.length, 2);
  assert.equal(items[1].id, "f");
  assert.equal(items[1].backTo, "");
  assert.equal(items[1].steps[1].status, "open");
});

test("row 5: back to Model keeps Provider and drops Thinking", () => {
  let items = putAsk([user], sel("a", "Model for vision — provider", ["xai", "zai"]), "open");
  items = answerAsk(items, "a", "xai", false);
  items = putAsk(items, sel("b", "Model for vision — model", ["grok-4.6", "grok-4.5", BACK]), "open");
  items = answerAsk(items, "b", "grok-4.6", false);
  items = putAsk(items, sel("c", "Thinking level", ["none", "medium", BACK]), "open");
  items = backAsk(items, "c", 1); // click the Model pill
  assert.equal(items[1].backTo, "Model");
  assert.deepEqual(items[1].steps.map((s) => [fieldLabel(s.title), s.status]), [
    ["Provider", "answered"],
    ["Model", "open"],
  ]);
  items = putAsk(items, sel("d", "Model for vision — model", ["grok-4.6", "grok-4.5", BACK]), "open");
  assert.equal(items[1].id, "d");
  assert.equal(items[1].steps.length, 2);
});

test("row 6: cancel closes the whole card and later cards do not resurrect it", () => {
  let items = putAsk([user], sel("a", "Model for vision — provider", ["xai", "zai"]), "open");
  items = answerAsk(items, "a", "xai", false);
  items = putAsk(items, sel("b", "Thinking level", ["none", "medium", BACK]), "open");
  items = answerAsk(items, "b", "Cancelled", true);
  assert.equal(items[1].status, "cancelled");
  assert.equal(stitchIndex(items), -1);
  assert.equal(walkReply(items, sel("c", "whatever", ["x"])), "");
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

test("a new card remembers the slash command that opened it", () => {
  const items = putAsk(
    [{ kind: "block", cls: "user", text: "/roles clear agent" }],
    sel("a", "Delete this roles file?", []),
    "open",
  );
  assert.equal(items[1].cmd, "/roles clear agent");
  // a plain-prose turn yields no command identity
  const plain = putAsk(
    [{ kind: "block", cls: "user", text: "please do things" }],
    sel("b", "Pick one", ["x"]),
    "open",
  );
  assert.equal(plain[1].cmd, "");
});

test("summaryParts types the outcome", () => {
  const def = summaryParts([
    { status: "answered", title: "Edit which role?", answer: "vision" },
    { status: "answered", title: "Model for vision — provider", answer: "xai" },
    { status: "answered", title: "Model for vision — model", answer: "grok-4.5" },
    { status: "answered", title: "Thinking level", answer: "medium" },
    { status: "answered", title: "Save to", answer: "workspace" },
  ]);
  assert.deepEqual(def, {
    kind: "definition", role: "vision", model: "xai/grok-4.5",
    provider: "xai", modelId: "grok-4.5", thinking: "medium",
    scope: "workspace", text: "",
  });
  const cleared = summaryParts(
    [{ status: "answered", title: "Delete this roles file?", answer: "Yes" }],
    "Cleared .pi/roles.json",
  );
  assert.deepEqual(cleared, { kind: "cleared", file: ".pi/roles.json", text: "Cleared .pi/roles.json" });
  const kept = summaryParts(
    [{ status: "answered", title: "Delete this roles file?", answer: "No" }],
    "Kept .pi/roles/qa.json",
  );
  assert.deepEqual(kept, { kind: "kept", file: ".pi/roles/qa.json", text: "Kept .pi/roles/qa.json" });
  const empty = summaryParts(
    [{ status: "answered", title: "Clear which config?", answer: "workspace" }],
    "Nothing to clear — .pi/roles.json does not exist.",
  );
  assert.equal(empty.kind, "empty");
  const rolePick = summaryParts(
    [{ status: "answered", title: "Roles (current: auto)", answer: "auto" }],
    "Auto: image → vision, text → default",
  );
  assert.equal(rolePick.kind, "role");
  assert.equal(rolePick.role, "auto");
});
