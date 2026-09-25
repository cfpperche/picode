import test from "node:test";
import assert from "node:assert/strict";
import { parseRoute } from "../browser/src/lib/routes.js";
import { mobileRoute } from "../mobile/src/lib/mobileRoutes.js";

test("CLI routes and old preferences reach both application managers", () => {
  for (const hash of ["#/clis", "#/clis/codex", "#/clis/codex/terminals", "#/clis/pi/sessions", "#/clis/pi/providers", "#/clis/pi/providers/custom", "#/clis/pi/providers/custom/cheap", "#/clis/pi/settings", "#/clis/pi/packages", "#/clis/pi/connectors", "#/clis/terminals", "#/preferences/status"]) {
    assert.equal(parseRoute(hash), "clis");
    assert.equal(mobileRoute(hash).section, "clis");
  }
});

test("native settings and the Settings tab reach the CLI manager on both applications", () => {
  for (const hash of ["#/clis/settings", "#/clis/pi/settings?agentId=A", "#/clis/codex/settings"]) {
    assert.equal(parseRoute(hash), "clis", hash);
    assert.equal(mobileRoute(hash).section, "clis", hash);
  }
  // The Pi-era settings addresses were retired on 2026-09-24.
  for (const hash of ["#/settings", "#/more/settings"]) {
    assert.notEqual(parseRoute(hash), "clis", hash);
    assert.notEqual(mobileRoute(hash).section, "clis", hash);
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
