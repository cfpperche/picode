import { test } from "node:test";
import assert from "node:assert/strict";
import { describeDomainEntry, describeDomainField, parseDomainField } from "./browserDomains.js";

// The field says what it means. Every row here is a shape the schema accepts
// and the matchers read some way; if one of these lies, the person typing a
// grant is told the wrong thing about the agent's reach.

test("the any-site entry says so in words", () => {
  const d = describeDomainEntry("*");
  assert.equal(d.kind, "any");
  assert.match(d.label, /Any site/);
  assert.match(d.covers, /http or https/);
  // The scheme rule survives the wildcard, and the hint has to say it.
  assert.match(d.miss, /never/);
});

test("a host is exact, and the hint teaches the wildcard", () => {
  const d = describeDomainEntry("example.com");
  assert.equal(d.kind, "host");
  assert.match(d.label, /example\.com only/);
  assert.match(d.miss, /\*\.example\.com/);
});

test("both wildcard spellings describe the same promise", () => {
  for (const entry of ["*.example.com", ".example.com"]) {
    const d = describeDomainEntry(entry);
    assert.equal(d.kind, "suffix");
    assert.match(d.covers, /docs\.example\.com/);
    assert.match(d.miss, /notexample\.com/);
  }
});

test("a bare TLD entry reads as every site under it", () => {
  const d = describeDomainEntry("*.com");
  assert.equal(d.kind, "suffix");
  assert.match(d.label, /Every \.com site/);
  assert.match(d.covers, /ending in \.com/);
});

test("entries that open nothing are called out, not silently accepted", () => {
  // The `*` bug this hint exists for: the field accepted it, no host matched
  // it, and nothing said a word.
  for (const entry of ["*.", "*example.com", "*.", ".", ".."]) {
    const d = describeDomainEntry(entry);
    assert.equal(d.kind, "dead", entry);
    assert.match(d.label, /matches nothing/);
    assert.match(d.miss, /\* /, entry);
  }
});

test("parseDomainField strips what the schema strips", () => {
  assert.deepEqual(parseDomainField("https://Example.com/path, localhost:5173, [::1]:80"), [
    "example.com",
    "localhost",
    "::1",
  ]);
  assert.deepEqual(parseDomainField("  ,  "), []);
  assert.deepEqual(parseDomainField(""), []);
});

test("the field's hint lists one line per entry, in order", () => {
  const rows = describeDomainField("* , example.com, *.example.com");
  assert.deepEqual(rows.map((r) => r.entry), ["*", "example.com", "*.example.com"]);
  assert.deepEqual(rows.map((r) => r.kind), ["any", "host", "suffix"]);
});

test("an empty field describes nothing (the empty state is the page's)", () => {
  assert.deepEqual(describeDomainField(""), []);
});
