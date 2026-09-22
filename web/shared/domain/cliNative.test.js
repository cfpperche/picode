import assert from "node:assert/strict";
import test from "node:test";
import {
  NATIVE_SETTINGS_CLIS,
  supportsNativeSettings,
  layerToScope,
  scopeToLayer,
  defaultLayer,
  groupFields,
  rowState,
  dangerNote,
  coerce,
  memoryEmptyLine,
  memoryKindLabel,
  cliMemoryHash,
  cliMemoryLocation,
} from "./cliNative.js";

test("Pi is not a native-settings CLI: it keeps its own editor", () => {
  assert.equal(supportsNativeSettings("pi"), false);
  assert.equal(NATIVE_SETTINGS_CLIS.length, 8);
  assert.ok(!NATIVE_SETTINGS_CLIS.includes("pi"));
});

test("the route's layer and the driver's scope map one way each", () => {
  assert.equal(layerToScope("project"), "project");
  assert.equal(layerToScope("global"), "user");
  assert.equal(layerToScope(""), "user");
  assert.equal(scopeToLayer("project"), "project");
  assert.equal(scopeToLayer("user"), "global");
});

test("a CLI with one file opens on that layer whatever the route says", () => {
  const single = [{ scope: "user", label: "Global" }];
  assert.equal(defaultLayer(single, "project"), "global");
  const both = [{ scope: "user" }, { scope: "project" }];
  assert.equal(defaultLayer(both, "project"), "project");
  assert.equal(defaultLayer(both, ""), "global");
});

test("fields keep their declared order inside their group", () => {
  const groups = groupFields([
    { key: "a", group: "Model" },
    { key: "b", group: "Approvals" },
    { key: "c", group: "Model" },
  ]);
  assert.deepEqual(groups.map((g) => g.name), ["Model", "Approvals"]);
  assert.deepEqual(groups[0].fields.map((f) => f.key), ["a", "c"]);
});

test("a row says where its value comes from, and never claims an inheritance that did not happen", () => {
  const field = { key: "ui.yolo" };
  const layers = [
    { scope: "user", label: "Global", values: { "ui.yolo": false } },
    { scope: "project", label: "This workspace", values: {} },
  ];
  assert.deepEqual(rowState(field, layers, "user"), { value: false, setHere: true, from: "" });
  assert.deepEqual(rowState(field, layers, "project"), { value: false, setHere: false, from: "Global" });

  const unset = rowState({ key: "models.default" }, layers, "project");
  assert.equal(unset.value, undefined);
  assert.equal(unset.from, "");
});

test("a dangerous value is named with its own cost, and only when it is chosen", () => {
  const sandbox = { key: "sandbox_mode", danger: "danger-full-access", dangerNote: "No sandbox: full file and network access." };
  assert.equal(dangerNote(sandbox, "danger-full-access"), "No sandbox: full file and network access.");
  assert.equal(dangerNote(sandbox, "read-only"), "");
  const yolo = { key: "ui.yolo", danger: "true", dangerNote: "Every tool call is approved without asking." };
  assert.equal(dangerNote(yolo, true), "Every tool call is approved without asking.");
  assert.equal(dangerNote(yolo, false), "");
  assert.equal(dangerNote({ key: "model" }, "anything"), "");
  // Two rows in one group must not print the same sentence twice.
  const approvals = { key: "approval_policy", danger: "never", dangerNote: "The agent runs commands without asking." };
  assert.notEqual(dangerNote(approvals, "never"), dangerNote(sandbox, "danger-full-access"));
});

test("values are coerced to the kind the field declares", () => {
  assert.equal(coerce({ kind: "bool" }, "yes"), true);
  assert.equal(coerce({ kind: "number" }, " 42 "), 42);
  assert.equal(coerce({ kind: "number" }, "abc"), null);
  assert.equal(coerce({ kind: "text" }, 7), "7");
});

test("a CLI without memory gets its one line, not an empty well", () => {
  const none = memoryEmptyLine({ tier: "none", note: "Pi has no memory of its own between sessions." });
  assert.equal(none.line, "Pi has no memory of its own between sessions.");
  assert.equal(none.action, "");

  const off = memoryEmptyLine({ tier: "editable", toggle: "memory.backend" }, { resolved: false, note: "Omp keeps memory off until a backend is chosen in Settings." });
  assert.match(off.line, /Settings/);
  assert.equal(off.action, "settings");

  const empty = memoryEmptyLine({ tier: "editable", toggle: "autoMemoryEnabled" }, { resolved: true, exists: false });
  assert.equal(empty.line, "Nothing remembered yet.");
  assert.equal(empty.action, "settings");
});

test("memory kinds read as words, and an unknown kind passes through", () => {
  assert.equal(memoryKindLabel("feedback"), "Feedback");
  assert.equal(memoryKindLabel("habit"), "habit");
  assert.equal(memoryKindLabel(""), "");
});

test("the memory route round-trips", () => {
  const hash = cliMemoryHash("claude-code", { workspaceId: "w1", scope: "workspace" });
  assert.equal(hash, "#/clis/claude-code/memory?workspaceId=w1&scope=workspace");
  assert.deepEqual(cliMemoryLocation(hash), {
    view: "clis",
    pane: "memory",
    id: "claude-code",
    workspaceId: "w1",
    scope: "workspace",
  });
  assert.equal(cliMemoryLocation("#/clis/pi/settings"), null);
  // An unknown scope is dropped, never adopted.
  assert.equal(cliMemoryLocation("#/clis/pi/memory?scope=elsewhere").scope, "");
});
