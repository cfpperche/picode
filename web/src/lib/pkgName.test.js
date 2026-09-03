import { test } from "node:test";
import assert from "node:assert/strict";
import { pkgName } from "./pkgName.js";

test("pkgName: the name a source is known by", () => {
  assert.equal(pkgName("npm:pi-web-search"), "pi-web-search");
  assert.equal(pkgName("npm:pi-web-search@1.2.0"), "pi-web-search");
  assert.equal(pkgName("npm:@companion-ai/feynman"), "@companion-ai/feynman");
  assert.equal(pkgName("../packages/pi-inbox"), "pi-inbox");
  assert.equal(pkgName("/home/goat/picode/packages/pi-checklist/"), "pi-checklist");
  assert.equal(pkgName("git:github.com/user/repo.git"), "repo");
  assert.equal(pkgName(""), "");
});
