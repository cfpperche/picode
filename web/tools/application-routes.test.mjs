import test from "node:test";
import assert from "node:assert/strict";
import { parseRoute } from "../desktop/src/lib/routes.js";
import { mobileRoute } from "../mobile/src/lib/mobileRoutes.js";

test("CLI routes and old preferences reach both application managers", () => {
  for (const hash of ["#/clis", "#/clis/codex", "#/clis/codex/terminals", "#/clis/pi/sessions", "#/clis/terminals", "#/preferences/status"]) {
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
