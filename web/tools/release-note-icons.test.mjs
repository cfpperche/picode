import { it } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

// web/shared/data/whats-new.json is rendered by two apps and neither of them
// validates an `icon` name at runtime: an unknown name silently draws the
// sparkle. v0.2.0 shipped exactly that — "matrix", on the headline highlight
// of the release. The maps are read from source on purpose: the point is to
// fail when a note is published before the glyph exists in the app that
// draws it.
const notes = JSON.parse(readFileSync(fileURLToPath(new URL("../shared/data/whats-new.json", import.meta.url)), "utf8"));

function mappedIcons(app) {
  const source = readFileSync(fileURLToPath(new URL(`../${app}/src/components/WhatsNew.jsx`, import.meta.url)), "utf8");
  const map = source.match(/const ICONS = \{([\s\S]*?)\n\};/);
  assert.ok(map, `${app} WhatsNew.jsx no longer has an ICONS map`);
  return new Set([...map[1].matchAll(/^\s*([A-Za-z0-9_]+):\s*[A-Za-z0-9_]+,/gm)].map((m) => m[1]));
}

const published = [...new Set(notes.flatMap((release) => release.highlights.map((item) => item.icon)))];

it("whats-new.json publishes at least one icon name", () => {
  assert.ok(published.length > 0);
});

for (const app of ["desktop", "mobile"]) it(`${app} releases notes draw every published icon name`, () => {
  const known = mappedIcons(app);
  assert.deepEqual(published.filter((name) => !known.has(name)), []);
});
