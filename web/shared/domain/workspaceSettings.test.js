import { test } from "node:test";
import assert from "node:assert/strict";
import { cleanChecks, draftFrom, inheritedLine, integrationSave, integrationSource } from "./workspaceSettings.js";
import { workspaceSettingsSchema } from "../contracts/schemas.js";

const DEFAULT = { declared: false, effective: { fromScope: "default", ffOnly: true } };
// The server's omitempty drops the machine scope "" — the fixture is the wire.
const MACHINE = { declared: false, effective: { scope: "", ffOnly: false, checks: ["go test ./..."], version: 1 } };
const OWN = { declared: true, settings: { ffOnly: true, checks: ["make ci"], version: 3 }, effective: { fromScope: "w1", ffOnly: true, checks: ["make ci"] } };

test("the source names the layer in force", () => {
  assert.equal(integrationSource(DEFAULT), "default");
  assert.equal(integrationSource(MACHINE), "machine");
  assert.equal(integrationSource(OWN), "own");
  assert.equal(inheritedLine(MACHINE), "This machine's rules: any merge, after go test ./...");
  assert.equal(inheritedLine(DEFAULT), "PiCode's default: fast-forward only, no checks");
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
