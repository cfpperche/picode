import { it } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { assertResolvedBoundaries, checkBoundaries } from "./boundaries.mjs";

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
]) it(`rejects ${name}`, async t => {
  const root = fixture(t);
  writeFileSync(join(root, "mobile/src/entry" + (name.startsWith("CSS") ? ".css" : ".js")), source);
  await assert.rejects(() => checkBoundaries(root), message);
});

it("allows an app's own components and the public headless client", async t => {
  const root = fixture(t);
  writeFileSync(join(root, "mobile/src/view.jsx"), 'import { fetchAgents } from "@picode/shared/api.js"; export default fetchAgents;');
  writeFileSync(join(root, "mobile/src/entry.js"), 'import View from "./view.jsx";');
  await assert.doesNotReject(() => checkBoundaries(root));
});

it("rejects indirect UI coupling through shared", async t => {
  const root = fixture(t);
  writeFileSync(join(root, "shared/api.js"), 'export { default } from "../desktop/src/view.jsx";');
  await assert.rejects(() => checkBoundaries(root), /cross-application/);
});

it("rejects React in the headless package", async t => {
  const root = fixture(t);
  writeFileSync(join(root, "shared/api.js"), 'import { useState } from "react";');
  await assert.rejects(() => checkBoundaries(root), /presentation dependency/);
});

it("a mobile build checks shared but does not depend on desktop source validity", async t => {
  const root = fixture(t);
  writeFileSync(join(root, "desktop/src/view.jsx"), 'import Broken from "missing-editor";');
  await assert.doesNotReject(() => checkBoundaries(root, ["mobile", "shared"]));
  await assert.rejects(() => checkBoundaries(root), /undeclared dependency/);
});

it("a shared export cannot point directly into an application", async t => {
  const root = fixture(t);
  writeFileSync(join(root, "shared/package.json"), JSON.stringify({ exports: { "./api.js": "./../desktop/src/view.jsx" } }));
  await assert.rejects(() => checkBoundaries(root), /invalid public export/);
});
it("a shared export must exist", async t => {
  const root = fixture(t);
  writeFileSync(join(root, "shared/package.json"), JSON.stringify({ exports: { "./missing.js": "./missing.js" } }));
  await assert.rejects(() => checkBoundaries(root), /invalid public export/);
});

for (const extension of ["js", "mjs", "cjs", "ts", "mts", "cts", "jsx", "tsx"]) it(`rejects shared React in ${extension}, including same-line imports`, async t => {
  const root = fixture(t);
  writeFileSync(join(root, "shared", "view." + extension), 'const value = 1; import React from "react"; export const View = () => React.createElement("div", null, value);');
  await assert.rejects(() => checkBoundaries(root), /presentation dependency|must not contain JSX/);
});
it("does not interpret an import example in a string as an actual import", async t => {
  const root = fixture(t);
  writeFileSync(join(root, "shared/api.js"), `export const example = 'import React from "react"';`);
  await assert.doesNotReject(() => checkBoundaries(root));
});
it("resolved shared graph rejects transitive presentation dependencies", () => {
  const root = "/project/web/";
  const shared = root + "shared/ui.mjs", helper = root + "node_modules/helper/index.js", react = root + "node_modules/react/index.js";
  const edges = { [shared]: [helper], [helper]: [react], [react]: [] };
  const graph = { getModuleIds: () => Object.keys(edges), getModuleInfo: id => ({ importedIds: edges[id], dynamicallyImportedIds: [] }) };
  assert.throws(() => assertResolvedBoundaries(root, "mobile", graph), /Presentation dependency/);
  edges[helper] = [];
  assert.doesNotThrow(() => assertResolvedBoundaries(root, "mobile", graph));
});
