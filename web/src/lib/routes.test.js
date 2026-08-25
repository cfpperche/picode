import assert from "node:assert/strict";
import { test } from "node:test";
import { parseRoute, ROUTES, docsSlug } from "./routes.js";

test("preferences and settings are distinct", () => {
  assert.equal(parseRoute("#/preferences"), "preferences");
  assert.equal(parseRoute("#/settings"), "settings");
  assert.equal(ROUTES.preferences, "/preferences");
  assert.equal(ROUTES.settings, "/settings");
});

test("docs slug", () => {
  assert.equal(parseRoute("#/docs/tree"), "docs");
  assert.equal(docsSlug("#/docs/tree"), "tree");
  assert.equal(docsSlug("#/docs"), "");
});
