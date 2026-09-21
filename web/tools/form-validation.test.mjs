import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { sourceFiles } from "./boundaries.mjs";

const root = fileURLToPath(new URL("../", import.meta.url));

// AGENTS.md: "Forms: Zod, never native browser validation. Schemas live in
// web/shared/contracts/schemas.js. Forms set noValidate. Same messages in
// every browser."
//
// The rule was 59 of 62 forms. The three that had drifted were all in the
// Pins and Snippets screens, and none carried a `required` or a `pattern`
// yet — so nothing misbehaved, and nothing would have, until the first
// field that did. A rule worth stating is worth checking.
function openingTags(source) {
  const tags = [];
  for (const match of source.matchAll(/<form\b/g)) {
    let depth = 0;
    let i = match.index + match[0].length;
    for (; i < source.length; i++) {
      const c = source[i];
      if (c === "{") depth++;
      else if (c === "}") depth--;
      else if (c === ">" && depth === 0) break;
    }
    tags.push({ tag: source.slice(match.index, i + 1), line: source.slice(0, match.index).split("\n").length });
  }
  return tags;
}

test("every form opts out of native browser validation", () => {
  const offenders = [];
  let seen = 0;
  for (const app of ["browser", "mobile", "shared", "desktop", "launcher"]) {
    let files;
    try {
      files = sourceFiles(resolve(root, app));
    } catch {
      continue; // an application that does not exist here
    }
    for (const path of files) {
      if (!/\.[jt]sx$/.test(path)) continue;
      const source = readFileSync(path, "utf8");
      for (const { tag, line } of openingTags(source)) {
        seen++;
        if (!/\bnoValidate\b/.test(tag)) {
          offenders.push(`${relative(root, path).replaceAll("\\", "/")}:${line}`);
        }
      }
    }
  }
  assert.ok(seen > 20, `found only ${seen} forms — the scan is not reaching the applications`);
  assert.deepEqual(
    offenders,
    [],
    `these <form> tags do not set noValidate, so the browser will validate them its own way:\n  ${offenders.join("\n  ")}`,
  );
});
