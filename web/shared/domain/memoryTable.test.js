import assert from "node:assert/strict";
import { test } from "node:test";
import { agoLabel, blastRadius, bytesLabel, columnsFor, facets, filterItems, health, indexBudget, sortItems } from "./memoryTable.js";

const idx = { id: "MEMORY.md", title: "Index", index: true, bytes: 100, modified: "2026-09-01T00:00:00Z" };
const rows = [
  idx,
  { id: "a.md", title: "Alpha", kind: "feedback", bytes: 2000, modified: "2026-09-10T00:00:00Z", citedBy: 3, indexed: true },
  { id: "b.md", title: "Beta", kind: "project", bytes: 500, modified: "2026-09-15T00:00:00Z", citedBy: 0, indexed: true },
  { id: "c.md", title: "Gamma", kind: "feedback", bytes: 900, modified: "2026-09-05T00:00:00Z", citedBy: 1, indexed: false, broken: ["gone"] },
];

test("the index stays pinned to the top whatever the sort", () => {
  for (const key of ["title", "citedBy", "bytes", "modified"]) {
    for (const dir of ["asc", "desc"]) {
      assert.equal(sortItems(rows, { key, dir })[0].id, "MEMORY.md", `${key} ${dir}`);
    }
  }
});

test("sorting orders the rest by the column asked for", () => {
  assert.deepEqual(sortItems(rows, { key: "citedBy", dir: "desc" }).slice(1).map((r) => r.id), ["a.md", "c.md", "b.md"]);
  assert.deepEqual(sortItems(rows, { key: "bytes", dir: "asc" }).slice(1).map((r) => r.id), ["b.md", "c.md", "a.md"]);
  assert.deepEqual(sortItems(rows, { key: "modified", dir: "desc" }).slice(1).map((r) => r.id), ["b.md", "a.md", "c.md"]);
  assert.deepEqual(sortItems(rows, { key: "title", dir: "asc" }).slice(1).map((r) => r.id), ["a.md", "b.md", "c.md"]);
});

test("health names only things to act on", () => {
  assert.deepEqual(health(rows[1]).map((h) => h.id), []);
  // Cited by nothing is the zero in its own column, not a second chip.
  assert.deepEqual(health(rows[2]).map((h) => h.id), []);
  assert.deepEqual(health(rows[3]).map((h) => h.id), ["unindexed", "broken"]);
  // The index itself wears nothing: it is the map, not a memory.
  assert.deepEqual(health(idx), []);
  // A store with no citation graph reports nothing rather than "cited by 0".
  assert.deepEqual(health({ id: "x.md", indexed: undefined, citedBy: undefined }), []);
});

test("a column a CLI cannot answer is not offered", () => {
  assert.ok(columnsFor(rows).some((c) => c.id === "citedBy"));
  const plain = [{ id: "MEMORY.md", index: true }, { id: "USER.md" }];
  assert.ok(!columnsFor(plain).some((c) => c.id === "citedBy"));
});

test("facets count what is there, so no filter is ever empty", () => {
  assert.deepEqual(facets(rows), [{ kind: "feedback", count: 2 }, { kind: "project", count: 1 }]);
  assert.deepEqual(facets([idx]), []);
});

test("filtering by kind keeps the index, because the CLI still loads it", () => {
  const got = filterItems(rows, { kinds: ["project"] });
  assert.deepEqual(got.map((r) => r.id), ["MEMORY.md", "b.md"]);
});

test("the text filter reaches the summary, the id and a broken link's name", () => {
  assert.deepEqual(filterItems(rows, { text: "gamma" }).map((r) => r.id), ["c.md"]);
  assert.deepEqual(filterItems(rows, { text: "gone" }).map((r) => r.id), ["c.md"]);
  assert.equal(filterItems(rows, { text: "nothing here" }).length, 0);
});

test("the index budget says which limit bites first", () => {
  assert.equal(indexBudget(null), null);
  assert.equal(indexBudget({ exists: false }), null);
  // The owner's real numbers the day this shipped.
  const real = indexBudget({ exists: true, lines: 54, bytes: 10630, lineLimit: 200, byteLimit: 25600 });
  assert.equal(real.percent, 42);
  assert.equal(real.limit, "bytes");
  assert.equal(real.over, false);
  assert.equal(real.near, false);
  // Lines can bite first even when the bytes are fine.
  const wordy = indexBudget({ exists: true, lines: 190, bytes: 1000, lineLimit: 200, byteLimit: 25600 });
  assert.equal(wordy.limit, "lines");
  assert.equal(wordy.near, true);
  const over = indexBudget({ exists: true, lines: 260, bytes: 1000, lineLimit: 200, byteLimit: 25600 });
  assert.equal(over.over, true);
  assert.equal(over.percent, 100);
});

test("sizes and ages read as lengths, not raw numbers", () => {
  assert.equal(bytesLabel(900), "900 B");
  assert.equal(bytesLabel(2048), "2 KB");
  assert.equal(bytesLabel(undefined), "");
  const now = Date.parse("2026-09-20T00:00:00Z");
  assert.equal(agoLabel("2026-09-20T00:00:00Z", now), "today");
  assert.equal(agoLabel("2026-09-19T00:00:00Z", now), "yesterday");
  assert.equal(agoLabel("2026-09-10T00:00:00Z", now), "10d");
  assert.equal(agoLabel("2026-06-20T00:00:00Z", now), "3mo");
  assert.equal(agoLabel("", now), "");
  assert.equal(agoLabel("not a date", now), "");
});

test("the delete confirm names what it breaks, as a sentence", () => {
  assert.equal(blastRadius(2, 0, 0), "Nothing else points at them.");
  assert.equal(blastRadius(1, 0, 0), "Nothing else points at it.");
  assert.equal(blastRadius(1, 1, 0), "1 link in another memory will point at nothing.");
  assert.equal(blastRadius(3, 0, 2), "The index still names 2 of them.");
  assert.equal(blastRadius(3, 4, 1), "4 links in other memories will point at nothing; the index still names 1 of them.");
});
