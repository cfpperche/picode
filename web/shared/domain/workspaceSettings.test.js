import { test } from "node:test";
import assert from "node:assert/strict";
import { blocksLanding, cleanChecks, draftFrom, followsRows, inheritedLine, integrationSave, integrationSource, machinePage, rulesSummary } from "./workspaceSettings.js";
import { workspaceSettingsSchema } from "../contracts/schemas.js";

const DEFAULT = { declared: false, effective: { fromScope: "default", ffOnly: true } };
// The server's omitempty drops the machine scope "" — the fixture is the wire.
const MACHINE = { declared: false, effective: { scope: "", ffOnly: false, checks: ["go test ./..."], version: 1 } };
const OWN = { declared: true, settings: { ffOnly: true, checks: ["make ci"], version: 3 }, effective: { fromScope: "w1", ffOnly: true, checks: ["make ci"] } };

test("the source names the layer in force", () => {
  assert.equal(integrationSource(DEFAULT), "default");
  assert.equal(integrationSource(MACHINE), "machine");
  assert.equal(integrationSource(OWN), "own");
  // The runner blocks both of these, so the line says so (ADR-0182).
  assert.equal(inheritedLine(MACHINE), "This machine's rules: fast-forward off, so authorized branches stay blocked");
  assert.match(inheritedLine(DEFAULT), /^No rules yet/);
  assert.equal(inheritedLine({ declared: false, effective: { ffOnly: true, checks: ["go test ./..."] } }), "This machine's rules: fast-forward only, after go test ./...");
  assert.equal(rulesSummary({ ffOnly: true }), "fast-forward only, no checks");
  assert.equal(blocksLanding({ ffOnly: false }), true);
  assert.equal(blocksLanding({ ffOnly: true }), false);
});

// One row per line of the decision table in workspaceSettings.js.
test("save follows the decision table", () => {
  assert.deepEqual(integrationSave(DEFAULT, "machine", {}), { action: "none" });
  assert.deepEqual(integrationSave(OWN, "machine", {}), { action: "delete" });
  assert.deepEqual(integrationSave(MACHINE, "own", { ffOnly: true, checks: ["make ci", "  "] }), { action: "put", body: { ffOnly: true, checks: ["make ci"] } });
  assert.deepEqual(integrationSave(OWN, "own", { ffOnly: true, checks: [" make ci ", ""] }), { action: "none" });
  assert.deepEqual(integrationSave(OWN, "own", { ffOnly: false, checks: ["make ci"] }),
    { action: "put", body: { ffOnly: false, checks: ["make ci"], expectedVersion: 3 } });
});

test("switching to own starts from the rules in force", () => {
  assert.deepEqual(draftFrom(MACHINE), { ffOnly: false, checks: ["go test ./..."] });
  assert.deepEqual(draftFrom(DEFAULT), { ffOnly: true, checks: [] });
  assert.deepEqual(draftFrom(OWN), { ffOnly: true, checks: ["make ci"] });
  assert.deepEqual(cleanChecks(["", " a ", null]), ["a"]);
});

test("the form schema mirrors the store's limits", () => {
  const ok = (v) => workspaceSettingsSchema.safeParse(v).success;
  assert.equal(ok({ name: "App", ffOnly: true, checks: ["make ci"] }), true);
  assert.equal(ok({ name: "  ", ffOnly: true, checks: [] }), false);
  assert.equal(ok({ name: "App", ffOnly: true, checks: ["a\nb"] }), false);
  assert.equal(ok({ name: "App", ffOnly: true, checks: Array(9).fill("x") }), false);
  assert.equal(ok({ name: "App", ffOnly: true, checks: ["é".repeat(151)] }), false);
});

test("the follows list reads each workspace's layer, then the machine's", () => {
  const ws = [{ id: "a", name: "A" }, { id: "b", name: "B" }, { id: "c", name: "C" }];
  const own = { a: { ffOnly: true, checks: ["make ci"] }, c: { ffOnly: false } };
  const pick = (rows) => rows.map((r) => [r.id, r.state, r.text]);
  assert.deepEqual(pick(followsRows(ws, { machine: { ffOnly: true }, workspaces: own })), [
    ["a", "own", "Own rules · fast-forward only, after make ci"],
    ["b", "machine", "These rules"],
    ["c", "blocked", "Own rules · fast-forward off, so authorized branches stay blocked"],
  ]);
  assert.deepEqual(followsRows(ws, { machine: { ffOnly: false }, workspaces: {} })[1].text, "These rules — blocked");
  assert.deepEqual(followsRows(ws, { machine: null, workspaces: {} })[0], { id: "a", name: "A", state: "blocked", text: "Blocked — no rules" });
  assert.deepEqual(followsRows([], null), []);
});

test("the machine page feeds the same save table", () => {
  const none = machinePage({ machine: null });
  assert.equal(none.declared, false);
  assert.deepEqual(integrationSave(none, "own", { ffOnly: true, checks: ["git diff --check"] }), { action: "put", body: { ffOnly: true, checks: ["git diff --check"] } });
  const m = machinePage({ machine: { ffOnly: true, checks: [], version: 2 } });
  assert.deepEqual(integrationSave(m, "own", { ffOnly: true, checks: [] }), { action: "none" });
  assert.deepEqual(integrationSave(m, "own", { ffOnly: true, checks: ["x"] }).body.expectedVersion, 2);
});
