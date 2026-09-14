import test from "node:test";
import assert from "node:assert/strict";
import { parseFragment } from "./changelog-assemble.mjs";
import { renderPublicChangelog } from "./docs-changelog.mjs";

const changelog = `# Changelog

All notable changes to this project are documented in this file.

**Agent contract:** every commit with a user-visible change MUST add an entry
to the \`[Unreleased]\` section.

## [Unreleased]

### Added

- old added

## [0.1.0] - 2026-08-23

### Added

- first
`;

test("public page drops the agent contract and keeps Unreleased plus cuts", () => {
  const page = renderPublicChangelog(changelog, []);
  assert.match(page, /editLink: false/);
  assert.doesNotMatch(page, /Agent contract/);
  assert.match(page, /## \[Unreleased\]/);
  assert.match(page, /## \[0\.1\.0\] - 2026-08-23/);
  assert.match(page, /- old added/);
});

test("fragments appear under Unreleased without being the only copy", () => {
  const page = renderPublicChangelog(changelog, [parseFragment("### Added\n- from a fragment\n")]);
  assert.match(page, /- from a fragment/);
  assert.match(page, /- old added/);
});

test("bare angle brackets are escaped so VitePress does not see HTML tags", () => {
  const page = renderPublicChangelog("# Changelog\n\n## [Unreleased]\n\n### Added\n\n- Open <agent> and `<$0.01>`\n", []);
  assert.match(page, /Open &lt;agent>/);
  assert.match(page, /`&lt;\$0\.01>`/);
  assert.doesNotMatch(page, /Open <agent>/);
});
