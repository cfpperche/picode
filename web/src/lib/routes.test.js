import assert from "node:assert/strict";
import { test } from "node:test";
import { parseRoute, ROUTES, providersNew, providersLlama, pinRoute, prefSection, agentRoute, workspaceHash } from "./routes.js";

test("preferences and settings are distinct", () => {
  assert.equal(parseRoute("#/preferences"), "preferences");
  assert.equal(parseRoute("#/preferences/backup"), "preferences");
  assert.equal(prefSection("#/preferences"), "appearance");
  assert.equal(prefSection("#/preferences/backup"), "backup");
  assert.equal(parseRoute("#/settings"), "settings");
  assert.equal(ROUTES.preferences, "/preferences");
  assert.equal(ROUTES.settings, "/settings");
  assert.equal(parseRoute("#/providers/new"), "providers");
  assert.equal(providersNew("#/providers/new"), true);
  assert.equal(providersLlama("#/providers/llama"), true);
  assert.equal(parseRoute("#/pins/new"), "pins");
  assert.deepEqual(pinRoute("#/pins/new"), { mode: "new", id: "" });
  assert.deepEqual(pinRoute("#/pins/hello-abc"), { mode: "edit", id: "hello-abc" });
});

test("agent hash is still the workspace shell", () => {
  assert.equal(parseRoute("#/agent/grok-c87aca"), "workspace");
  assert.equal(parseRoute("#/"), "workspace");
  assert.equal(agentRoute("#/agent/grok-c87aca"), "grok-c87aca");
  assert.equal(agentRoute("#/agent/a%2Fb"), "a/b");
  assert.equal(agentRoute("#/"), null);
  assert.equal(agentRoute("#/mcps"), null);
  assert.equal(workspaceHash("grok-c87aca"), "#/agent/grok-c87aca");
  assert.equal(workspaceHash(null), "#/");
});
