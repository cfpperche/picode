import test from "node:test";
import assert from "node:assert/strict";
import { assemble, parseFragment, normalizeBlock } from "./changelog-assemble.mjs";

const changelog = `# Changelog

## [Unreleased]

### Added

- old added

### Fixed

- old fix

## [0.1.0] - 2026-08-23

### Added

- first
`;

test("fragments land at the top of their section, the newest fragment first", () => {
  const older = parseFragment("### Added\n- from a\n");
  const newer = parseFragment("### Added\n- from b\n\n### Fixed\n- fix b\n");
  const out = assemble(changelog, [older, newer]);
  const unreleased = out.split("## [0.1.0]")[0];
  assert.match(unreleased, /### Added\n\n- from b\n\n- from a\n\n- old added/);
  assert.match(unreleased, /### Fixed\n\n- fix b\n\n- old fix/);
  assert.match(out, /## \[0\.1\.0\] - 2026-08-23\n\n### Added\n\n- first\n$/);
});

test("a missing section is created in canonical order", () => {
  const out = assemble(changelog, [parseFragment("### Changed\n- moved\n")]);
  const at = (s) => out.indexOf(s);
  assert.ok(at("### Added") < at("### Changed") && at("### Changed") < at("### Fixed"));
  assert.match(out, /### Changed\n\n- moved\n\n### Fixed/);
});

test("a new fragment recreates Unreleased after a release cut", () => {
  const cut = changelog.replace(/## \[Unreleased\][\s\S]*?(?=## \[0\.1\.0\])/, "");
  const out = assemble(cut, [parseFragment("### Fixed\n- after cut\n")]);
  assert.match(out, /## \[Unreleased\]\n\n### Fixed\n\n- after cut/);
  assert.match(out, /## \[0\.1\.0\] - 2026-08-23\n\n### Added\n\n- first/);
});

test("fragments reject unknown sections, stray text and emptiness", () => {
  assert.throws(() => parseFragment("### Bogus\n- x\n"), /unknown section/);
  assert.throws(() => parseFragment("- no heading\n"), /before the first/);
  assert.throws(() => parseFragment("### Added\n\n"), /no entries/);
});

// The release cut publishes the [Unreleased] block verbatim as the GitHub
// release body, so a repeated heading is a defect a reader sees. Duplicates
// arrived with the direct edits that predate fragments; folding heals them.
test("a block that inherited repeated headings comes back with one of each", () => {
  const drifted = [
    "",
    "### Added",
    "",
    "- one",
    "",
    "### Fixed",
    "",
    "- two",
    "",
    "### Added",
    "",
    "- three",
    "",
    "### Fixed",
    "",
    "- four",
  ];
  const out = normalizeBlock(drifted);
  assert.deepEqual(
    out.filter((l) => l.startsWith("### ")),
    ["### Added", "### Fixed"],
  );
  assert.deepEqual(
    out.filter((l) => l.startsWith("- ")),
    ["- one", "- three", "- two", "- four"],
  );
});

test("normalizing keeps canonical order and refuses a heading it does not know", () => {
  const out = normalizeBlock(["### Fixed", "", "- f", "", "### Added", "", "- a"]);
  assert.deepEqual(out.filter((l) => l.startsWith("### ")), ["### Added", "### Fixed"]);
  assert.throws(() => normalizeBlock(["### Bogus", "", "- x"]), /unknown section "Bogus"/);
});

test("a fold into a drifted changelog lands once, and leaves one heading per type", () => {
  const drifted = [
    "# Changelog",
    "",
    "## [Unreleased]",
    "",
    "### Added",
    "",
    "- old add",
    "",
    "### Added",
    "",
    "- older add",
    "",
    "## [0.1.0] - 2026-08-23",
    "",
    "### Added",
    "",
    "- shipped",
  ].join("\n");
  const out = assemble(drifted, [{ Added: ["- fresh"] }]);
  const unreleased = out.split("## [0.1.0]")[0];
  assert.equal((unreleased.match(/^### Added$/gm) || []).length, 1);
  assert.deepEqual(unreleased.match(/^- .*$/gm), ["- fresh", "- old add", "- older add"]);
  // The released section is never touched.
  assert.match(out, /## \[0\.1\.0\] - 2026-08-23\n\n### Added\n\n- shipped/);
});
