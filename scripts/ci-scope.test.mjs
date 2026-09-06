import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

import { classifyPaths, localScope, pathScope } from "./ci-scope.mjs";

const cases = [
  {
    name: "an empty or untrusted diff runs everything",
    paths: [],
    want: { scope: "full", full: true, docs: true },
  },
  {
    name: "Go product code runs everything",
    paths: ["internal/server/server.go"],
    want: { scope: "full", full: true, docs: true },
  },
  {
    name: "frontend code runs everything because it affects docs parity",
    paths: ["web/src/main.jsx"],
    want: { scope: "full", full: true, docs: true },
  },
  {
    name: "workflow changes prove the complete replacement workflow",
    paths: [".github/workflows/ci.yml"],
    want: { scope: "full", full: true, docs: true },
  },
  {
    name: "internal docs need only the always-on scope gate",
    paths: ["docs/handoff.md", "docs/architecture.md"],
    want: { scope: "metadata", full: false, docs: false },
  },
  {
    name: "root Markdown is internal project metadata",
    paths: ["CHANGELOG.md", "README.md"],
    want: { scope: "metadata", full: false, docs: false },
  },
  {
    name: "public docs build and validate without the platform matrix",
    paths: ["www/guide/getting-started.md"],
    want: { scope: "docs", full: false, docs: true },
  },
  {
    name: "docs media and its tooling use the docs gate",
    paths: ["docs-videos/compositions/index.html", "scripts/docs-video-manifest.mjs"],
    want: { scope: "docs", full: false, docs: true },
  },
  {
    name: "Vale configuration uses the docs gate",
    paths: [".vale.ini", "styles/PiCode/Spelling.yml"],
    want: { scope: "docs", full: false, docs: true },
  },
  {
    name: "one product file promotes a mixed change set to full",
    paths: ["docs/handoff.md", "www/index.md", "go.mod"],
    want: { scope: "full", full: true, docs: true },
  },
  {
    name: "Windows separators cannot hide a product file",
    paths: ["internal\\desktop\\desktop.go"],
    want: { scope: "full", full: true, docs: true },
  },
];

test("CI path decision table", async (t) => {
  for (const tc of cases) {
    await t.test(tc.name, () => {
      const got = classifyPaths(tc.paths);
      assert.deepEqual(
        { scope: got.scope, full: got.full, docs: got.docs },
        tc.want,
      );
    });
  }
});

test("path scopes are explicit and fail safe", () => {
  assert.equal(pathScope("scripts/docs-check.mjs"), "docs");
  assert.equal(pathScope("scripts/lib/uitree.mjs"), "docs");
  assert.equal(pathScope("scripts/lib/docs-surfaces.mjs"), "docs");
  assert.equal(pathScope("scripts/ci-scope.mjs"), "full");
  assert.equal(pathScope("LICENSE"), "full");
});

test("hosted workflow preserves the optimized platform decision table", () => {
  const here = dirname(fileURLToPath(import.meta.url));
  const root = join(here, "..");
  const workflow = readFileSync(join(root, ".github", "workflows", "ci.yml"), "utf8");
  const makefile = readFileSync(join(root, "Makefile"), "utf8");
  const goStart = workflow.indexOf("\n  go:\n");
  const embeddedStart = workflow.indexOf("\n  embedded:\n");

  assert.ok(goStart > 0 && embeddedStart > goStart, "go and embedded jobs must remain distinct");
  const goJob = workflow.slice(goStart, embeddedStart);
  assert.doesNotMatch(goJob, /setup-node|make web|make docs|npm ci/);
  assert.match(goJob, /ubuntu-latest, macos-latest, windows-latest/);
  assert.match(goJob, /go test -race -run '\^\$' \.\/\.\.\./);
  assert.match(goJob, /matrix\.os != 'windows-latest'/);
  assert.equal((workflow.match(/run: make web/g) ?? []).length, 1);
  assert.match(workflow, /actions\/upload-artifact@v7/);
  assert.match(workflow, /actions\/download-artifact@v8/);
  assert.match(workflow, /cancel-in-progress: true/);
  assert.match(workflow, /needs: \[changes, frontend, docs, go, embedded\]/);

  assert.match(
    makefile,
    /ci-docs:[\s\S]*\$\(MAKE\) docs-check\n\t\$\(MAKE\) docs/,
    "local CI must check committed parity before docs generation",
  );
});

// Local worktree scopes (ADR-0086): run what the diff can break.
const localCases = [
  { name: "empty diff runs everything", paths: [], want: { full: true } },
  { name: "CSS only is a web scope", paths: ["web/desktop/src/styles/app.css"], want: { full: false, go: false, web: true } },
  { name: "a handler is a go scope with its path", paths: ["internal/server/server.go"], want: { full: false, go: true, web: false, goPaths: ["internal/server/server.go"] } },
  { name: "a pi package is test-js only", paths: ["packages/pi-checklist/index.ts"], want: { full: false, packages: true, go: false } },
  { name: "public docs are the docs scope", paths: ["www/guide/api.md"], want: { full: false, docs: true, packages: false } },
  { name: "docs scripts are docs and test-js", paths: ["scripts/docs-shots.mjs"], want: { docs: true, packages: true, full: false } },
  { name: "handoff is metadata", paths: ["docs/handoff/2026-09-06-x.md"], want: { metadata: true, full: false } },
  { name: "the Makefile fails safe to full", paths: ["Makefile", "web/a.jsx"], want: { full: true } },
  { name: "an unknown root file fails safe to full", paths: ["Dockerfile"], want: { full: true } },
];
for (const c of localCases) {
  test("localScope: " + c.name, () => {
    const got = localScope(c.paths);
    for (const [k, v] of Object.entries(c.want)) assert.deepEqual(got[k], v, k);
  });
}
