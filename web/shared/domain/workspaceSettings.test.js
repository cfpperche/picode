import { test } from "node:test";
import assert from "node:assert/strict";
import { blocksLanding, cleanChecks, draftFrom, followsRows, inheritedLine, integrationIntro, integrationSave, integrationSource, machinePage, rulesSummary } from "./workspaceSettings.js";
import { workspaceSettingsSchema } from "../contracts/schemas.js";

const DEFAULT = { declared: false, effective: { fromScope: "default", ffOnly: true } };
// The server's omitempty drops the machine scope "" — the fixture is the wire.
const MACHINE = { declared: false, effective: { scope: "", mode: "local", ffOnly: false, checks: ["go test ./..."], version: 1 } };
const OWN = { declared: true, settings: { mode: "local", ffOnly: true, checks: ["make ci"], version: 3 }, effective: { fromScope: "w1", mode: "local", ffOnly: true, checks: ["make ci"] } };

test("the source names the layer in force", () => {
  assert.equal(integrationSource(DEFAULT), "default");
  assert.equal(integrationSource(MACHINE), "machine");
  assert.equal(integrationSource(OWN), "own");
  // The runner blocks both of these, so the line says so (ADR-0182).
  assert.equal(inheritedLine(MACHINE), "This machine's rules: fast-forward off, so authorized branches stay blocked");
  assert.match(inheritedLine(DEFAULT), /^No rules yet/);
  assert.equal(inheritedLine({ declared: false, effective: { mode: "local", ffOnly: true, checks: ["go test ./..."] } }), "This machine's rules: fast-forward only, after go test ./...");
  assert.equal(rulesSummary({ mode: "local", ffOnly: true }), "fast-forward only, no checks");
  assert.equal(rulesSummary({ mode: "provider", ffOnly: true }), "the project's own merge queue integrates it");
  assert.equal(rulesSummary({ ffOnly: true }), "no integration mode declared");
  assert.equal(blocksLanding({ ffOnly: false }), true);
  assert.equal(blocksLanding({ mode: "local", ffOnly: false }), true);
  assert.equal(blocksLanding({ mode: "local", ffOnly: true }), false);
  assert.equal(blocksLanding({ mode: "provider", ffOnly: false }), false);
});

// One row per line of the decision table in workspaceSettings.js.
test("save follows the decision table", () => {
  assert.deepEqual(integrationSave(DEFAULT, "machine", {}), { action: "none" });
  assert.deepEqual(integrationSave(OWN, "machine", {}), { action: "delete" });
  assert.deepEqual(integrationSave(MACHINE, "own", { mode: "local", ffOnly: true, checks: ["make ci", "  "] }), { action: "put", body: { mode: "local", ffOnly: true, checks: ["make ci"] } });
  assert.deepEqual(integrationSave(OWN, "own", { mode: "local", ffOnly: true, checks: [" make ci ", ""] }), { action: "none" });
  assert.deepEqual(integrationSave(OWN, "own", { mode: "local", ffOnly: false, checks: ["make ci"] }),
    { action: "put", body: { mode: "local", ffOnly: false, checks: ["make ci"], expectedVersion: 3 } });
  // The provider runs its own checks, so a provider draft carries none.
  assert.deepEqual(integrationSave(OWN, "own", { mode: "provider", ffOnly: true, checks: ["make ci"] }),
    { action: "put", body: { mode: "provider", ffOnly: true, checks: [], expectedVersion: 3 } });
  // Switching the mode alone is a change.
  assert.deepEqual(integrationSave(OWN, "own", { mode: "provider", ffOnly: true, checks: ["make ci", ""] }).action, "put");
});

// The sentence above the editor must be true in every mode: PiCode does not
// run the checks of a project that integrates through its own queue (ADR-0186).
test("the intro says who integrates", () => {
  assert.match(integrationIntro({ mode: "local", ffOnly: true }), /^When you authorize a branch an agent delivered, PiCode runs these checks/);
  assert.match(integrationIntro({ mode: "provider" }), /^Your project's own merge queue integrates it: PiCode enqueues and observes/);
  assert.match(integrationIntro({}), /^Choose who integrates before anything runs/);
  for (const rules of [{ mode: "provider" }, {}]) {
    assert.doesNotMatch(integrationIntro(rules), /PiCode runs these checks/);
  }
});

test("switching to own starts from the rules in force", () => {
  assert.deepEqual(draftFrom(MACHINE), { mode: "local", ffOnly: false, checks: ["go test ./..."] });
  assert.deepEqual(draftFrom(DEFAULT), { mode: "", ffOnly: true, checks: [] });
  assert.deepEqual(draftFrom(OWN), { mode: "local", ffOnly: true, checks: ["make ci"] });
  assert.deepEqual(cleanChecks(["", " a ", null]), ["a"]);
});

test("the form schema mirrors the store's limits", () => {
  const ok = (v) => workspaceSettingsSchema.safeParse(v).success;
  assert.equal(ok({ name: "App", mode: "local", ffOnly: true, checks: ["make ci"] }), true);
  assert.equal(ok({ name: "App", mode: "provider", ffOnly: true, checks: [] }), true);
  assert.equal(ok({ name: "App", mode: "automatic", ffOnly: true, checks: [] }), false);
  assert.equal(ok({ name: "  ", mode: "local", ffOnly: true, checks: [] }), false);
  assert.equal(ok({ name: "App", mode: "local", ffOnly: true, checks: ["a\nb"] }), false);
  assert.equal(ok({ name: "App", mode: "local", ffOnly: true, checks: Array(9).fill("x") }), false);
  assert.equal(ok({ name: "App", mode: "local", ffOnly: true, checks: ["é".repeat(151)] }), false);
});

test("the follows list reads each workspace's layer, then the machine's", () => {
  const ws = [{ id: "a", name: "A" }, { id: "b", name: "B" }, { id: "c", name: "C" }];
  const own = { a: { mode: "local", ffOnly: true, checks: ["make ci"] }, c: { mode: "local", ffOnly: false } };
  const pick = (rows) => rows.map((r) => [r.id, r.state, r.text]);
  assert.deepEqual(pick(followsRows(ws, { machine: { mode: "local", ffOnly: true }, workspaces: own })), [
    ["a", "own", "Own rules · fast-forward only, after make ci"],
    ["b", "machine", "These rules"],
    ["c", "blocked", "Own rules · fast-forward off, so authorized branches stay blocked"],
  ]);
  assert.deepEqual(followsRows(ws, { machine: { mode: "local", ffOnly: false }, workspaces: {} })[1].text, "These rules — blocked");
  assert.deepEqual(followsRows(ws, { machine: null, workspaces: {} })[0], { id: "a", name: "A", state: "blocked", text: "Blocked — no rules" });
  assert.deepEqual(followsRows([], null), []);
});

test("the machine page feeds the same save table", () => {
  const none = machinePage({ machine: null });
  assert.equal(none.declared, false);
  assert.deepEqual(integrationSave(none, "own", { mode: "local", ffOnly: true, checks: ["git diff --check"] }), { action: "put", body: { mode: "local", ffOnly: true, checks: ["git diff --check"] } });
  const m = machinePage({ machine: { mode: "local", ffOnly: true, checks: [], version: 2 } });
  assert.deepEqual(integrationSave(m, "own", { mode: "local", ffOnly: true, checks: [] }), { action: "none" });
  assert.deepEqual(integrationSave(m, "own", { mode: "local", ffOnly: true, checks: ["x"] }).body.expectedVersion, 2);
});
