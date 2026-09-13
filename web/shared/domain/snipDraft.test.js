import test from "node:test";
import assert from "node:assert/strict";
import { parseSnip, expandSnip, snipSlug, insertLiteralBraces, draftToRestore, formFromSnip, tagsFromInput } from "./snipDraft.js";

test("parse golden", () => {
  assert.deepEqual(parseSnip("{{name}}").placeholders.map((p) => p.name), ["name"]);
  assert.equal(parseSnip("{{{{name}}}}").placeholders.length, 0);
  assert.equal(parseSnip("{{x={{{{}}").placeholders[0].default, "{{");
  assert.equal(parseSnip("{{name=foo}}bar}}").placeholders[0].default, "foo");
  assert.equal(parseSnip("{{name=foo}bar}}").placeholders[0].default, "foo}bar");
  assert.equal(parseSnip("{{n=}}").placeholders[0].optional, true);
  assert.equal(parseSnip("{{x}}{{x=ignored}}").placeholders.length, 1);
  assert.equal(parseSnip("{{").ok, false);
  assert.equal(parseSnip("{{1abc}}").ok, false);
  assert.equal(parseSnip("{{Not Valid}}").ok, false);
});

test("expand golden", () => {
  assert.equal(expandSnip("{{{{name}}}}").text, "{{name}}");
  assert.equal(expandSnip("{{x={{{{}}").text, "{{");
  assert.equal(expandSnip("{{name=foo}}bar}}").text, "foobar}}");
  assert.equal(expandSnip("Create {{name}}", { name: "Button" }).text, "Create Button");
  assert.deepEqual(expandSnip("{{a}} {{b}}", { a: "1" }).missing, ["b"]);
  assert.equal(expandSnip("keep {{x}}", { x: "a{{y}}b" }).text, "keep a{{y}}b");
  assert.equal(expandSnip("x{{cwd}}y", { cwd: "/evil" }, { cwd: "/ok" }).text, "x/oky");
  assert.equal(expandSnip("!`rm` {{x}}", { x: "ok" }).text, "!`rm` ok");
});

test("slug hyphenates", () => {
  assert.equal(snipSlug("Review PR"), "review-pr");
  assert.equal(snipSlug("review_pr"), "review-pr");
  assert.equal(snipSlug("***"), "");
});

test("insert {{ writes four braces", () => {
  assert.equal(insertLiteralBraces("ab", 1, 1), "a{{{{b");
});

test("draft restore", () => {
  const d = { title: "A", slug: "a", description: "", body: "x", tags: "" };
  assert.ok(draftToRestore(d, null));
  assert.equal(draftToRestore(d, { title: "A", slug: "a", description: "", body: "x", tags: "", updatedAt: "1" }), null);
  assert.equal(draftToRestore({ ...d, base: "old", body: "y" }, { ...d, updatedAt: "new" }), null);
  assert.equal(draftToRestore({ ...d, base: "", body: "" }, { ...d, updatedAt: "1" }), null);
  assert.equal(draftToRestore({ ...d, base: "1", body: "y" }, { ...d, updatedAt: "1" }).body, "y");
});

test("formFromSnip and tags", () => {
  assert.equal(formFromSnip({ title: "T", tags: ["a", "b"] }).tags, "a, b");
  assert.deepEqual(tagsFromInput(" a, b , "), ["a", "b"]);
});
