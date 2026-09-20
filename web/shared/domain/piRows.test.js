import assert from "node:assert/strict";
import test from "node:test";
import { PI_ROWS, piRowsFor, piRowState, rowKeys } from "./piRows.js";

test("every row declares what it reads and where it belongs", () => {
  for (const row of PI_ROWS) {
    assert.ok(row.label, "a row needs a label");
    assert.ok(row.group, `${row.label} needs a group`);
    assert.ok(row.kind, `${row.label} needs a kind`);
    assert.ok(rowKeys(row).length > 0, `${row.label} reads nothing`);
    if (row.kind === "select") assert.ok(row.options?.length, `${row.label} is a select with no options`);
    if (row.kind === "bool") assert.equal(typeof row.defaultOn, "boolean", `${row.label} must say what pi does when nobody sets it`);
  }
});

test("the machine layer sees every row; a workspace layer sees only what pi reads there", () => {
  const machine = piRowsFor("global").flatMap((g) => g.rows);
  const project = piRowsFor("project").flatMap((g) => g.rows);
  assert.equal(machine.length, PI_ROWS.length);
  assert.ok(project.length < machine.length);
  // Nothing pi keeps for the machine is offered on another layer: writing it
  // there would write a key pi never reads.
  assert.equal(project.filter((r) => r.machine).length, 0);
  for (const key of ["theme", "hideThinkingBlock", "quietStartup", "defaultProjectTrust", "shellPath"]) {
    assert.ok(!project.some((r) => r.key === key), `${key} must not be offered on a workspace layer`);
    assert.ok(machine.some((r) => r.key === key), `${key} is missing from the machine layer`);
  }
});

test("rows keep their declared order inside their group", () => {
  const groups = piRowsFor("global");
  assert.deepEqual(groups.map((g) => g.name), ["Session", "Model", "Approvals", "Interface"]);
  assert.deepEqual(groups[0].rows.map((r) => r.key), ["compactionEnabled", "steeringMode", "followUpMode"]);
});

test("a row says whether this layer set it, and hands back only what it set", () => {
  const model = PI_ROWS.find((r) => r.kind === "model");
  const own = { has: { defaultModel: true } };
  const state = piRowState(model, { defaultModel: "opus" }, own, "From This machine");
  assert.equal(state.setHere, true);
  // One of three fields is overridden, so only that one is handed back.
  assert.deepEqual(state.resetKeys, ["defaultModel"]);
  assert.equal(state.source, "Set here");

  const untouched = piRowState(model, {}, { has: {} }, "Pi default");
  assert.equal(untouched.setHere, false);
  assert.equal(untouched.source, "Pi default");
  assert.deepEqual(untouched.resetKeys, []);
});
