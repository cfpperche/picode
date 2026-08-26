import assert from "node:assert/strict";
import { test } from "node:test";
import { parseRoute, ROUTES, providersNew, providersLlama, pinRoute } from "./routes.js";

test("preferences and settings are distinct", () => {
  assert.equal(parseRoute("#/preferences"), "preferences");
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
