import test from "node:test";
import assert from "node:assert/strict";
import { parseRoute } from "../browser/src/lib/routes.js";
import { mobileRoute } from "../mobile/src/lib/mobileRoutes.js";

test("CLI routes and old preferences reach both application managers", () => {
  for (const hash of ["#/clis", "#/clis/codex", "#/clis/codex/terminals", "#/clis/pi/sessions", "#/clis/pi/providers", "#/clis/pi/settings", "#/clis/pi/packages", "#/clis/pi/connectors", "#/clis/terminals", "#/preferences/status"]) {
    assert.equal(parseRoute(hash), "clis");
    assert.equal(mobileRoute(hash).section, "clis");
  }
});

test("native settings and legacy URLs reach the CLI manager on both applications", () => {
  for (const hash of ["#/settings", "#/more/settings", "#/settings?agentId=A", "#/clis/settings", "#/clis/settings/pi?agentId=A", "#/clis/settings/codex"]) {
    assert.equal(parseRoute(hash), "clis", hash);
    assert.equal(mobileRoute(hash).section, "clis", hash);
  }
});

test("desktop snippets hashes are a host page", () => {
  for (const hash of ["#/snippets", "#/snippets/new", "#/snippets/review-pr-abc"]) {
    assert.equal(parseRoute(hash), "snippets", hash);
  }
  assert.equal(mobileRoute("#/snippets").section, "snippets");
  assert.equal(mobileRoute("#/snippets/abc").screen, "snip");
  assert.equal(mobileRoute("#/snippets/abc/edit").screen, "snipEdit");
});
