import { it } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { checkBoundaries } from "./boundaries.mjs";

function fixture(t) {
  const root = mkdtempSync(join(tmpdir(), "picode-boundaries-"));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  for (const app of ["desktop", "mobile", "shared"]) {
    mkdirSync(join(root, app, "src"), { recursive: true });
    writeFileSync(join(root, app, "package.json"), JSON.stringify({ dependencies: app === "shared" ? {} : { react: "19", "@picode/shared": "0.0.0" }, exports: app === "shared" ? { "./api.js": "./api.js" } : undefined }));
  }
  writeFileSync(join(root, "desktop/src/view.jsx"), "export default 'desktop';");
  writeFileSync(join(root, "shared/api.js"), "export const fetchAgents = () => []; ");
  return root;
}

for (const [name, source, message] of [
  ["direct desktop import", 'import View from "../../desktop/src/view.jsx";', /cross-application/],
  ["lazy desktop import", 'const view = () => import("../../desktop/src/view.jsx");', /cross-application/],
  ["CSS desktop import", '@import "../../desktop/src/view.css";', /cross-application/],
  ["undeclared desktop editor dependency", 'import { EditorView } from "@codemirror/view";', /undeclared dependency/],
  ["private shared file", 'import { x } from "@picode/shared/private.js";', /private shared/],
]) it(`rejects ${name}`, t => {
  const root = fixture(t);
  writeFileSync(join(root, "mobile/src/entry" + (name.startsWith("CSS") ? ".css" : ".js")), source);
  assert.throws(() => checkBoundaries(root), message);
});

it("allows an app's own components and the public headless client", t => {
  const root = fixture(t);
  writeFileSync(join(root, "mobile/src/view.jsx"), 'import { fetchAgents } from "@picode/shared/api.js"; export default fetchAgents;');
  writeFileSync(join(root, "mobile/src/entry.js"), 'import View from "./view.jsx";');
  assert.doesNotThrow(() => checkBoundaries(root));
});

it("rejects indirect UI coupling through shared", t => {
  const root = fixture(t);
  writeFileSync(join(root, "shared/api.js"), 'export { default } from "../desktop/src/view.jsx";');
  assert.throws(() => checkBoundaries(root), /cross-application/);
});

it("rejects React in the headless package", t => {
  const root = fixture(t);
  writeFileSync(join(root, "shared/api.js"), 'import { useState } from "react";');
  assert.throws(() => checkBoundaries(root), /presentation dependency/);
});

it("a mobile build checks shared but does not depend on desktop source validity", t => {
  const root = fixture(t);
  writeFileSync(join(root, "desktop/src/view.jsx"), 'import Broken from "missing-editor";');
  assert.doesNotThrow(() => checkBoundaries(root, ["mobile", "shared"]));
  assert.throws(() => checkBoundaries(root), /undeclared dependency/);
});

it("a shared export cannot point directly into an application", t => {
  const root = fixture(t);
  writeFileSync(join(root, "shared/package.json"), JSON.stringify({ exports: { "./api.js": "./../desktop/src/view.jsx" } }));
  assert.throws(() => checkBoundaries(root), /invalid public export/);
});
it("a shared export must exist", t => {
  const root = fixture(t);
  writeFileSync(join(root, "shared/package.json"), JSON.stringify({ exports: { "./missing.js": "./missing.js" } }));
  assert.throws(() => checkBoundaries(root), /invalid public export/);
});
