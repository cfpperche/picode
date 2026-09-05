import { atQuery, insertAtPath, mergeAtHits, skillsFromSlash } from "./atMention.js";
import assert from "node:assert/strict";
import { test } from "node:test";

test("atQuery needs a word-starting @", () => {
  assert.equal(atQuery("hello", 5), null);
  assert.equal(atQuery("email@x.com", 11), null);
  assert.deepEqual(atQuery("@", 1), { start: 0, query: "" });
  assert.deepEqual(atQuery("see @fo", 7), { start: 4, query: "fo" });
  assert.equal(atQuery("see @fo more", 12), null);
});

test("insertAtPath replaces the token and adds a space", () => {
  const got = insertAtPath("see @fo", 7, "src/app.go");
  assert.equal(got.text, "see @src/app.go ");
  assert.equal(got.caret, "see @src/app.go ".length);
});

test("insertAtPath quotes paths with spaces", () => {
  const got = insertAtPath("@a", 2, "my file.txt");
  assert.equal(got.text, '@"my file.txt" ');
});

test("mergeAtHits mixes agents skills files", () => {
  const hits = mergeAtHits("g", {
    agents: [{ id: "grok-1", name: "Grok" }],
    skills: [{ name: "mcp-scripting" }],
    files: [{ path: "src/go.go", name: "go.go" }, { path: "README.md", name: "README.md" }],
  });
  assert.equal(hits[0].kind, "agent");
  assert.equal(hits[0].path, "agent:Grok");
  assert.ok(!hits.some((h) => h.path === "README.md"));
});

test("skillsFromSlash only skill extras", () => {
  const sk = skillsFromSlash([
    { id: "skill:ask", hint: "Ask" },
    { id: "tpl:bug" },
  ]);
  assert.deepEqual(sk, [{ name: "ask", hint: "Ask" }]);
});

test("empty query still lists all kinds", () => {
  const hits = mergeAtHits("", {
    agents: [{ name: "Claude" }],
    skills: [{ name: "foo" }],
    files: [{ path: "a.ts", name: "a.ts" }],
  });
  assert.deepEqual(hits.map((h) => h.kind), ["file", "agent", "skill"]);
});
