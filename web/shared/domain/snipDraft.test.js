import test from "node:test";
import assert from "node:assert/strict";
import { parseSnip, expandSnip, snipSlug, insertLiteralBraces, draftToRestore, formFromSnip, tagsFromInput, replaceSnipToken, setDefaultInBody, encodeDefault, titleFromText, detectConversions, applyConversions, convName, readDraft, writeDraft, clearDraft } from "./snipDraft.js";

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
  assert.equal(formFromSnip({ kind: "shell" }).kind, "shell");
  assert.equal(formFromSnip(null).kind, "prompt");
  assert.deepEqual(tagsFromInput(" a, b , "), ["a", "b"]);
});

test("replaceSnipToken keeps surrounding draft (D4)", () => {
  assert.equal(replaceSnipToken("see /snip:review please", "review", "look at PR 1"), "see look at PR 1 please");
  assert.equal(replaceSnipToken("/snip:review-pr ", "review-pr", "done"), "done ");
  assert.equal(replaceSnipToken("", "x", "hi"), "hi");
});

test("setDefaultInBody writes the first occurrence and drops optionality", () => {
  assert.equal(setDefaultInBody("{{a}} {{a}}", "a", "1", true), "{{a=1}} {{a}}");
  assert.equal(setDefaultInBody("{{a=x}}", "a", "y", true), "{{a=y}}");
  assert.equal(setDefaultInBody("{{a=x}}", "a", "", true), "{{a=}}");
  assert.equal(setDefaultInBody("{{a=x}}", "a", "", false), "{{a}}");
  assert.equal(setDefaultInBody("nope", "a", "1", true), "nope");
  assert.equal(setDefaultInBody("{{{{a}}}} then {{b}}", "b", "2", true), "{{{{a}}}} then {{b=2}}");
});

test("encodeDefault escapes brace pairs", () => {
  assert.equal(encodeDefault("{{ x }}"), "{{{{ x }}}}");
  assert.equal(encodeDefault("one } two"), "one } two");
});

test("parse errors carry an excerpt", () => {
  const r = parseSnip("ok {{1bad}} tail");
  assert.equal(r.ok, false);
  assert.ok(r.excerpt.includes("1bad"));
});

test("titleFromText takes the first words of a selection", () => {
  assert.equal(titleFromText("Please review {{file}} and report"), "Please review {{file}} and report");
  assert.equal(titleFromText(""), "");
  assert.equal(titleFromText("   \n  "), "");
  assert.equal(titleFromText("- Review this diff before merging it into main and shipping"), "Review this diff before merging it into main and shipping");
  assert.equal(titleFromText("- 1. Review  the   diff"), "Review the diff");
  assert.equal(titleFromText("# Heading\n\nBody text"), "Heading Body text");
  assert.equal(titleFromText("One very long opening sentence that will not fit in sixty characters at all"), "One very long opening sentence that will not fit in sixty");
  assert.equal(titleFromText("Explain this error."), "Explain this error");
  assert.equal(titleFromText("a"), "a");
});

test("detectConversions finds brackets and UPPER_CASE, skipping placeholders", () => {
  const d = detectConversions("Check [FILE_NAME] for MAX_RETRIES and [FILE_NAME] again");
  assert.deepEqual(d.kinds.map((k) => [k.kind, k.count]), [["bracket", 2], ["upper", 1]]);
  assert.deepEqual(d.kinds[0].names, ["file_name"]);
  assert.deepEqual(d.kinds[0].sample, ["[FILE_NAME]", "[FILE_NAME]"]);
  assert.deepEqual(detectConversions("nothing here").kinds, []);
  // A token inside an existing placeholder is not a conversion.
  assert.deepEqual(detectConversions("{{keep_me}} and [TAKE_ME]").kinds.map((k) => k.kind), ["bracket"]);
  assert.deepEqual(detectConversions("{{a=[X]}}").kinds, []);
  // Bare lowercase words and single-capitalized words are left alone.
  assert.deepEqual(detectConversions("lower and Upper are fine").kinds, []);
});

test("applyConversions rewrites only the accepted kinds", () => {
  const body = "Check [FILE] with MAX_RETRIES now";
  assert.equal(applyConversions(body, ["bracket"]), "Check {{file}} with MAX_RETRIES now");
  assert.equal(applyConversions(body, ["upper"]), "Check [FILE] with {{max_retries}} now");
  assert.equal(applyConversions(body, ["bracket", "upper"]), "Check {{file}} with {{max_retries}} now");
  assert.equal(applyConversions(body, []), body);
  assert.equal(applyConversions("keep {{ok}} and [TAKE]", ["bracket"]), "keep {{ok}} and {{take}}");
  assert.equal(applyConversions("", ["bracket"]), "");
});

test("convName always produces a valid placeholder name", () => {
  assert.equal(convName("FILE_NAME"), "file_name");
  assert.equal(convName("File Name"), "file_name");
  assert.equal(convName("file-name"), "file_name");
  assert.equal(convName("  ..  "), "");
  assert.equal(convName("2FA_CODE"), "v_2fa_code");
  assert.ok(parseSnip("{{" + convName("Weird!!Name") + "}}").ok);
});

test("converted bodies parse", () => {
  const out = applyConversions("Fix [THE BUG] in [FILE] on UPPER_ENV", ["bracket", "upper"]);
  const parsed = parseSnip(out);
  assert.equal(parsed.ok, true);
  assert.deepEqual(parsed.placeholders.map((p) => p.name), ["the_bug", "file", "upper_env"]);
});

// A draft carries why it exists: a capture or an import is not a crash, and
// only an origin-less draft may be offered as "unsaved changes restored".
test("drafts record their origin", () => {
  const store = new Map();
  const fake = { getItem: (k) => (store.has(k) ? store.get(k) : null), setItem: (k, v) => store.set(k, v), removeItem: (k) => store.delete(k) };
  const form = { ...formFromSnip(null), body: "hello", title: "Hello" };
  writeDraft(fake, "", form, "", "capture");
  assert.equal(readDraft(fake, "").origin, "capture");
  assert.equal(readDraft(fake, "").body, "hello");
  // The editors fold the read draft into their form, so an edit after the
  // handoff keeps the origin; a draft that never carried one stays plain.
  writeDraft(fake, "", { ...readDraft(fake, ""), body: "hello there" }, "");
  assert.equal(readDraft(fake, "").origin, "capture");
  assert.equal(readDraft(fake, "").body, "hello there");
  // A plain draft (typed in the studio) has no origin.
  writeDraft(fake, "new", form, "");
  assert.equal(readDraft(fake, "new").origin, "");
  clearDraft(fake, "");
  assert.equal(readDraft(fake, ""), null);
  assert.equal(draftToRestore({ ...form, origin: "import" }, null).origin, "import");
});
