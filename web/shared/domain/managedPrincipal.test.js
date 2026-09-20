import test from "node:test";
import assert from "node:assert/strict";
import { catalogForPrincipal, catalogForAgent, agentIsPi } from "./managedPrincipal.js";
import { managedPrincipalSchema, parseForm } from "../contracts/schemas.js";

test("catalogForPrincipal keeps installed launchable CLIs", () => {
  const rows = catalogForPrincipal([
    { id: "claude-code", name: "Claude Code", installed: true, launchable: true },
    { id: "grok", name: "Grok", installed: false, launchable: true },
    { id: "detect", name: "X", installed: true, launchable: false },
    { id: "pi", name: "Pi", installed: true },
    null,
  ]);
  assert.deepEqual(rows.map((c) => c.id), ["claude-code", "pi"]);
});

test("catalogForPrincipal is empty when nothing is installed", () => {
  assert.deepEqual(catalogForPrincipal([]), []);
  assert.deepEqual(catalogForPrincipal([{ id: "grok", installed: false, launchable: true }]), []);
});

test("catalogForAgent always leads with Pi", () => {
  const rows = catalogForAgent([
    { id: "claude-code", name: "Claude Code", installed: true, launchable: true },
    { id: "grok", name: "Grok", installed: false, launchable: true },
  ]);
  assert.deepEqual(rows.map((c) => c.id), ["pi", "claude-code"]);
});

test("agentIsPi treats missing cli as Pi", () => {
  assert.equal(agentIsPi({}), true);
  assert.equal(agentIsPi({ cli: "pi" }), true);
  assert.equal(agentIsPi({ cli: "claude-code" }), false);
});

test("managedPrincipalSchema accepts a catalog id and an optional name", () => {
  const ok = parseForm(managedPrincipalSchema, { cli: "claude-code", name: "Claude" });
  assert.equal(ok.ok, true);
  assert.deepEqual(ok.value, { cli: "claude-code", name: "Claude" });
  const anon = parseForm(managedPrincipalSchema, { cli: "codex", name: "  " });
  assert.equal(anon.ok, true);
  assert.equal(anon.value.name, "");
  const missing = parseForm(managedPrincipalSchema, { cli: "", name: "" });
  assert.equal(missing.ok, false);
  assert.match(missing.error, /CLI/);
});
