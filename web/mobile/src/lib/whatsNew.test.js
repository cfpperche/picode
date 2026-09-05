import test from "node:test";
import assert from "node:assert/strict";
import {
  MAX_HIGHLIGHTS,
  MAX_RELEASES,
  compareVersions,
  hasUnseenRelease,
  parseVersion,
  readSeenVersion,
  selectReleaseNotes,
  shouldAutoOpen,
  writeSeenVersion,
} from "./whatsNew.js";

const notes = [
  { version: "0.1.0", date: "2026-08-23", highlights: [{ title: "First", summary: "One" }] },
  { version: "0.2.0", date: "2026-09-04", highlights: [{ title: "Second", summary: "Two" }] },
  { version: "0.3.0", date: "2026-09-10", highlights: [{ title: "Third", summary: "Three" }] },
  { version: "0.4.0", date: "2026-09-17", highlights: [{ title: "Fourth", summary: "Four" }] },
];

test("parseVersion accepts release and source-build forms only", () => {
  assert.deepEqual(parseVersion("v1.2.3"), [1, 2, 3]);
  assert.deepEqual(parseVersion("1.2.3+abc1234"), [1, 2, 3]);
  assert.equal(parseVersion("1.2"), null);
  assert.equal(parseVersion("1.2.x"), null);
  assert.equal(parseVersion(""), null);
});

test("compareVersions handles equal, newer and older semver", () => {
  assert.equal(compareVersions("0.1.0", "0.1.0"), 0);
  assert.equal(compareVersions("0.2.0", "0.1.0"), 1);
  assert.equal(compareVersions("0.1.0", "0.2.0"), -1);
  assert.equal(compareVersions("bad", "0.2.0"), null);
});

test("selection excludes future and already seen releases", () => {
  assert.deepEqual(selectReleaseNotes(notes, "0.2.0", "0.1.0").map((x) => x.version), ["0.2.0"]);
  assert.deepEqual(selectReleaseNotes(notes, "0.2.0", "0.2.0"), []);
  assert.deepEqual(selectReleaseNotes(notes, "0.2.0").map((x) => x.version), ["0.2.0", "0.1.0"]);
  assert.deepEqual(selectReleaseNotes(notes, "0.1.5").map((x) => x.version), ["0.1.0"]);
});

test("selection caps skipped history by releases and highlights", () => {
  const many = Array.from({ length: MAX_RELEASES + 2 }, (_, i) => ({
    version: `1.${i}.0`, highlights: Array.from({ length: 4 }, (_, j) => ({ title: `${i}-${j}` })),
  }));
  const got = selectReleaseNotes(many, "1.9.0");
  assert.ok(got.length <= MAX_RELEASES);
  assert.ok(got.reduce((n, r) => n + r.highlights.length, 0) <= MAX_HIGHLIGHTS);
});

test("selection skips a release without highlights without hiding later notes", () => {
  const got = selectReleaseNotes([
    { version: "1.2.0", highlights: [] },
    { version: "1.1.0", highlights: [{ title: "Useful" }] },
  ], "1.2.0");
  assert.deepEqual(got.map((x) => x.version), ["1.1.0"]);
});

test("auto-open requires a stamped release, product state and unseen notes", () => {
  const common = { current: "0.2.0", entries: notes };
  assert.equal(shouldAutoOpen({ ...common, release: true, hasProductState: true }), true);
  assert.equal(shouldAutoOpen({ ...common, release: false, hasProductState: true }), false);
  assert.equal(shouldAutoOpen({ ...common, release: true, hasProductState: false }), false);
  assert.equal(shouldAutoOpen({ ...common, release: true, blocked: true }), false);
  assert.equal(shouldAutoOpen({ ...common, release: true, seen: "0.2.0" }), false);
  assert.equal(hasUnseenRelease({ ...common, release: true, seen: "0.1.0" }), true);
});

test("seen storage is safe when storage is absent or throws", () => {
  const empty = { getItem: () => null, setItem: () => {} };
  assert.equal(readSeenVersion(empty), "");
  assert.equal(writeSeenVersion("0.2.0", empty), true);
  const broken = { getItem: () => { throw new Error("private"); }, setItem: () => { throw new Error("private"); } };
  assert.equal(readSeenVersion(broken), "");
  assert.equal(writeSeenVersion("0.2.0", broken), false);
  assert.equal(writeSeenVersion("bad", empty), false);
});
