import assert from "node:assert/strict";
import { test } from "node:test";
import { parseRoute, ROUTES, providersNew, providersLlama } from "./routes.js";

test("preferences and settings are distinct", () => {
  assert.equal(parseRoute("#/preferences"), "preferences");
  assert.equal(parseRoute("#/settings"), "settings");
  assert.equal(ROUTES.preferences, "/preferences");
  assert.equal(ROUTES.settings, "/settings");
  assert.equal(parseRoute("#/providers/new"), "providers");
  assert.equal(providersNew("#/providers/new"), true);
  assert.equal(providersLlama("#/providers/llama"), true);
});
