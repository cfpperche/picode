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
  namedScope,
  namedLayers,
  dangerNote,
  coerce,
  splitSelector,
  joinSelector,
  listValue,
  cyclePosition,
  selectorsInUse,
  roleAliases,
  isRoleField,
  isListField,
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

// ── Model roles (ADR-0181) ──────────────────────────────────────────────────

test("a selector splits for display and joins back byte for byte", () => {
  const levels = ["off", "minimal", "low", "medium", "high", "xhigh", "max", "auto"];
  // The model id may carry slashes, dots and an @upstream routing suffix; only
  // a known level after the last colon is a level.
  for (const value of [
    "openai/gpt-4.1-mini",
    "openrouter/z-ai/glm-4.7@cerebras:high",
    "deepseek/deepseek-flash:max",
    "@smol",
    "*",
    "provider/model:not-a-level",
  ]) {
    const { base, level } = splitSelector(value, levels);
    assert.equal(joinSelector(base, level), value, value);
  }
  assert.deepEqual(splitSelector("deepseek/deepseek-flash:max", levels), { base: "deepseek/deepseek-flash", level: "max" });
  assert.deepEqual(splitSelector("provider/model:not-a-level", levels), { base: "provider/model:not-a-level", level: "" });
  assert.deepEqual(splitSelector(undefined, levels), { base: "", level: "" });
  // An empty base is never joined into a lone suffix.
  assert.equal(joinSelector("", "high"), "");
  assert.equal(joinSelector("  ", "high"), "");
});

test("a list value tolerates the shapes a config file holds", () => {
  assert.deepEqual(listValue(["a", "b"]), ["a", "b"]);
  assert.deepEqual(listValue("a"), ["a"]);
  assert.deepEqual(listValue(""), []);
  assert.deepEqual(listValue(undefined), []);
  assert.deepEqual(listValue({}), []);
});

test("the cycle badge counts from one, and says zero when the role is out", () => {
  assert.equal(cyclePosition(["smol", "default", "slow"], "smol"), 1);
  assert.equal(cyclePosition(["smol", "default", "slow"], "default"), 2);
  assert.equal(cyclePosition(["smol", "default", "slow"], "slow"), 3);
  assert.equal(cyclePosition(["smol", "default", "slow"], "plan"), 0);
  assert.equal(cyclePosition(undefined, "smol"), 0);
});

test("the picker offers what the files already use, and the CLI's own aliases", () => {
  const fields = [
    { key: "modelRoles.default", kind: "role" },
    { key: "modelRoles.smol", kind: "role" },
    { key: "symbolPreset", kind: "text" },
  ];
  const layers = [
    { scope: "user", values: { "modelRoles.default": "openai/gpt-5", symbolPreset: "ascii" } },
    { scope: "project", values: { "modelRoles.smol": "zai/glm-5.3-flash", "modelRoles.default": "@smol" } },
  ];
  // Sorted, de-duplicated, and an alias is not a model.
  assert.deepEqual(selectorsInUse(fields, layers), ["openai/gpt-5", "zai/glm-5.3-flash"]);
  assert.deepEqual(roleAliases(fields), ["*", "@default", "@smol"]);
  assert.equal(isRoleField(fields[0]), true);
  assert.equal(isRoleField(fields[2]), false);
  assert.equal(isListField({ kind: "list" }), true);
});

test("a bound pane names the workspace layer; an unbound one keeps the generic word", () => {
  const layers = [
    { scope: "user", label: "Global", values: {} },
    { scope: "project", label: "This workspace", values: {} },
  ];
  assert.deepEqual(namedLayers(layers, "delivery").map((l) => l.label), ["Global", "delivery"]);
  // The layer itself survives the rename: scope, values and revision ride along.
  assert.equal(namedLayers(layers, "delivery")[1].scope, "project");
  assert.deepEqual(namedLayers(layers, "").map((l) => l.label), ["Global", "This workspace"]);
  // A driver's suffix survives: Claude Code's uncommitted checkout layer.
  assert.equal(namedScope("This workspace (local)", "Atlas"), "Atlas (local)");
  assert.equal(namedScope("Global", "Atlas"), "Global");
  // A workspace name is free text (trimmed, length-capped, no charset rule):
  // `$` sequences must land literally, never as replace patterns.
  assert.equal(namedScope("This workspace", "$&"), "$&");
  assert.equal(namedScope("This workspace (local)", "A$'B"), "A$'B (local)");
  // Provenance reads the name, because the row reads the renamed layer.
  const named = namedLayers([
    { scope: "project", label: "This workspace", values: { key: "set" } },
    { scope: "user", label: "Global", values: {} },
  ], "delivery");
  assert.equal(rowState({ key: "key" }, named, "user").from, "delivery");
});
