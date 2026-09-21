// Tests for scripts/declaration-order.mjs — the TDZ (use-before-declaration)
// checker that guards the blank-window bug class (main 4f1a68a7: a `const`
// read by an effect's deps array earlier in the same render body). The rule
// must flag the real class and stay silent on everything that merely looks
// like it; the last row proves the shipped frontend trees are clean.
import assert from "node:assert/strict";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

import { DEFAULT_SCAN_ROOTS, scanPaths, scanSource } from "./declaration-order.mjs";

const here = dirname(fileURLToPath(import.meta.url));

test("bad.component.jsx: both reads of the later const are flagged", () => {
  const { violations } = scanPaths([join(here, "fixtures/declaration-order/bad.component.jsx")]);
  assert.equal(
    violations.length,
    2,
    `expected the two early reads of showFullUrl, got: ${JSON.stringify(violations, null, 2)}`,
  );
  assert.ok(violations.every((v) => v.name === "showFullUrl"));
  assert.ok(violations.every((v) => v.function === "BadComponent"));
  // One read sits in the deps array, one in the render body — distinct lines.
  assert.notEqual(violations[0].line, violations[1].line);
});

test("good.component.jsx: shadowing, hoisting and member names stay clean", () => {
  const { violations, skipped } = scanPaths([join(here, "fixtures/declaration-order/good.component.jsx")]);
  assert.deepEqual(
    violations.map((v) => `${v.line}:${v.column} ${v.name}`),
    [],
  );
  assert.deepEqual(skipped, []);
});

test("deps-array read of a later const is the flagged class", () => {
  const src = `
    import { useEffect } from "react";
    export function Panel({ id }) {
      useEffect(() => {}, [id, mode]);
      const mode = "dark";
      return null;
    }
  `;
  const violations = scanSource(src);
  assert.equal(violations.length, 1, JSON.stringify(violations));
  assert.equal(violations[0].name, "mode");
});

test("a read inside a callback is exempt; the deps array is not", () => {
  const src = `
    export function Panel() {
      const once = () => console.log(mode);
      const [mode] = ["dark"];
      return once;
    }
  `;
  assert.deepEqual(scanSource(src), []);
});

test("shadowing hides the outer binding in the inner subtree only", () => {
  // Sibling read before the shadow block still sees the outer const: flagged.
  const flagged = scanSource(`
    export function Panel() {
      console.log(mode);
      if (Math.random() > 2) {
        let mode = "dark";
        console.log(mode);
      }
      const mode = "light";
    }
  `);
  assert.equal(flagged.length, 1, JSON.stringify(flagged));
  assert.equal(flagged[0].name, "mode");

  // Reordering so the read follows the declaration: clean.
  const clean = scanSource(`
    export function Panel() {
      const mode = "light";
      console.log(mode);
      if (Math.random() > 2) {
        let mode = "dark";
        console.log(mode);
      }
    }
  `);
  assert.deepEqual(clean, []);
});

test("hoisted names are never flagged", () => {
  const src = `
    export function Panel() {
      const a = helper();
      const b = legacy;
      var legacy = 1;
      function helper() { return 42; }
      return a + b;
    }
  `;
  assert.deepEqual(scanSource(src), []);
});

test("object keys, member properties and for-head lets are not reads", () => {
  const src = `
    export function Panel({ config, items }) {
      const table = { mode: "dark" };
      const fromConfig = config.mode;
      let sum = 0;
      for (const item of items) sum += item.mode?.length || 0;
      for (let mode = 0; mode < 3; mode++) sum += mode;
      const [mode] = ["dark"];
      return table.mode + fromConfig + sum + mode;
    }
  `;
  assert.deepEqual(scanSource(src), []);
});

test("a later const read in an earlier initializer is flagged", () => {
  const src = `
    export function Panel() {
      const first = second + 1;
      const second = 2;
      return first;
    }
  `;
  const violations = scanSource(src);
  assert.equal(violations.length, 1, JSON.stringify(violations));
  assert.equal(violations[0].name, "second");
});

test("each function body is its own unit: a sibling function's later const is invisible", () => {
  const src = `
    export function Panel() {
      const helper = () => own;
      const own = 1;
      return helper;
    }
    export function Other() {
      return own;
    }
  `;
  assert.deepEqual(scanSource(src), []);
});

test("an unparseable file is skipped, never invented into a violation", () => {
  const dir = mkdtempSync(join(tmpdir(), "declaration-order-"));
  const file = join(dir, "broken.component.jsx");
  writeFileSync(file, "export function Panel( {");
  try {
    const { violations, skipped } = scanPaths([file]);
    assert.deepEqual(violations, []);
    assert.equal(skipped.length, 1);
    assert.match(skipped[0].reason, /./);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test("the shipped frontend trees contain no use-before-declaration", () => {
  const { violations, skipped } = scanPaths(DEFAULT_SCAN_ROOTS);
  assert.deepEqual(
    skipped.map((s) => `${s.file}: ${s.reason}`),
    [],
    "every frontend file must parse; a parse failure hides real checks",
  );
  assert.deepEqual(
    violations.map((v) => `${v.file}:${v.line}:${v.column} ${v.name}`),
    [],
    "if this row fails, the file named is a latent blank-window bug — fix it, do not weaken the checker",
  );
});
