import { test } from "node:test";
import assert from "node:assert/strict";
import { splitPathQuery, filterDirs, bestMatch } from "./pathFilter.js";

test("splitPathQuery splits dir and fragment", () => {
  assert.deepEqual(splitPathQuery("/home/goat/pi"), { dir: "/home/goat", q: "pi" });
  assert.deepEqual(splitPathQuery("/home/goat/"), { dir: "/home/goat/", q: "" });
  assert.deepEqual(splitPathQuery("/home"), { dir: "/", q: "home" });
  assert.deepEqual(splitPathQuery("/"), { dir: "/", q: "" });
  assert.deepEqual(splitPathQuery("~"), { dir: "~", q: "" });
  assert.deepEqual(splitPathQuery("~/co"), { dir: "~", q: "co" });
  assert.deepEqual(splitPathQuery("C:\\Users\\x"), { dir: "C:\\Users", q: "x" });
  assert.deepEqual(splitPathQuery(""), { dir: "", q: "" });
  assert.deepEqual(splitPathQuery("  /a/b  "), { dir: "/a", q: "b" });
});

const dirs = [
  { name: "Pictures", path: "/p" },
  { name: "picode", path: "/pc" },
  { name: "api-picker", path: "/a" },
  { name: "zzz", path: "/z" },
];

test("filterDirs ranks prefix before includes, case-insensitive", () => {
  assert.deepEqual(filterDirs(dirs, "pi").map((d) => d.name), ["Pictures", "picode", "api-picker"]);
  assert.deepEqual(filterDirs(dirs, ""), dirs);
  assert.deepEqual(filterDirs(dirs, "none"), []);
});

test("bestMatch: exact > prefix > includes > null", () => {
  assert.equal(bestMatch(dirs, "picode").name, "picode");
  assert.equal(bestMatch(dirs, "pi").name, "Pictures");
  assert.equal(bestMatch(dirs, "picker").name, "api-picker");
  assert.equal(bestMatch(dirs, "none"), null);
  assert.equal(bestMatch(dirs, ""), null);
});
