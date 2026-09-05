import test from "node:test";
import assert from "node:assert/strict";
import { parseRoute } from "../desktop/src/lib/routes.js";
import { mobileRoute } from "../mobile/src/lib/mobileRoutes.js";

test("CLI routes and old preferences reach both application managers", () => {
  for (const hash of ["#/clis", "#/clis/codex", "#/clis/terminals", "#/preferences/status"]) {
    assert.equal(parseRoute(hash), "clis");
    assert.equal(mobileRoute(hash).section, "clis");
  }
});
