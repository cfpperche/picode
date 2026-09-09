import test from "node:test";
import assert from "node:assert/strict";
import { assemble, parseFragment } from "./changelog-assemble.mjs";

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

test("fragments reject unknown sections, stray text and emptiness", () => {
  assert.throws(() => parseFragment("### Bogus\n- x\n"), /unknown section/);
  assert.throws(() => parseFragment("- no heading\n"), /before the first/);
  assert.throws(() => parseFragment("### Added\n\n"), /no entries/);
});
