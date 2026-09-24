import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { BENDS, GUIDE, bendQuoteFailures, parseBends } from "./docs-quotes.mjs";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const real = (p) => readFileSync(join(root, p), "utf8");

// The real tree with one file rewritten: the drift a refactor would cause.
function failuresWith(path, edit) {
  return bendQuoteFailures(root, (p) => (p === path ? edit(real(p)) : real(p)));
}

function lineOf(path, needle) {
  return real(path).slice(0, real(path).indexOf(needle)).split("\n").length;
}

test("the guide and its sources agree today", () => {
  assert.deepEqual(bendQuoteFailures(root), []);
});

test("every card on the page has exactly one row", () => {
  const cards = parseBends(real(GUIDE));
  assert.equal(cards.length, 13);
  assert.deepEqual(cards.map((c) => c.when).sort(), BENDS.map((b) => b.when).sort());
});

// Each row: the file a refactor touches, the edit → the failure it must report.
const drift = [
  {
    name: "a script rewords its refusal",
    path: "scripts/land.mjs",
    edit: (s) => s.replace("either it is merged already", "either it was merged already"),
    want: [`${GUIDE}:${lineOf(GUIDE, "main cannot fast-forward to")}:`, "make land cannot fast-forward", "no longer matches scripts/land.mjs"],
  },
  {
    name: "the card rewords the quote",
    path: GUIDE,
    edit: (s) => s.replace("fix it (the branch is already merged;", "fix it (the branch was merged;"),
    want: [`${GUIDE}:${lineOf(GUIDE, "land: make ci FAILED")}:`, "make ci fails after the fast-forward", "no longer in the card"],
  },
  {
    name: "a word is added to a quote",
    path: GUIDE,
    edit: (s) => s.replace("Refusing to commit a new session note directly on main", "Refusing to commit a brand new session note directly on main"),
    want: ["A commit on main is refused", "no longer in the card"],
  },
  {
    name: "unchecked text inside a quote",
    path: GUIDE,
    edit: (s) => s.replace("<code>land: make ci FAILED", "<code>oops land: make ci FAILED"),
    want: ["make ci fails after the fast-forward", 'unchecked text "oops"'],
  },
  {
    name: "a hook splits its message differently",
    path: ".githooks/pre-commit",
    edit: (s) => s.replace("likely wrote into this worktree", "probably wrote into this worktree"),
    want: ["A commit is refused over a living doc", "no longer matches .githooks/pre-commit"],
  },
  {
    name: "a constant the prose cites changes",
    path: "scripts/handoff-board.mjs",
    edit: (s) => s.replace("const NOTE_FRESH_DAYS = 7;", "const NOTE_FRESH_DAYS = 10;"),
    want: ["A debt outlives its note", "NOTE_FRESH_DAYS = 7"],
  },
  {
    name: "a new card without a row",
    path: GUIDE,
    edit: (s) =>
      s.replace(
        '<div class="bend-item">',
        '<div class="bend-item">\n<p class="bend-when">Something new</p>\n<p class="bend-says"><code>x</code></p>\n</div>\n\n<div class="bend-item">',
      ),
    want: ['card "Something new" has no source row'],
  },
  {
    name: "a card is removed",
    path: GUIDE,
    edit: (s) => s.replace('<p class="bend-when">A debt outlives its note</p>', '<p class="bend-when">Renamed</p>'),
    want: ['card "A debt outlives its note" is gone'],
  },
];

for (const c of drift) {
  test(`drift: ${c.name}`, () => {
    const fails = failuresWith(c.path, c.edit);
    assert.ok(fails.length > 0, "expected a failure");
    const joined = fails.join("\n");
    for (const w of c.want) assert.ok(joined.includes(w), `missing ${JSON.stringify(w)} in:\n${joined}`);
  });
}
