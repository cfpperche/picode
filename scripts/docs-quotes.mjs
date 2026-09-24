#!/usr/bin/env node
// docs-quotes — the "When the flow bends" cards of docs-site/guide/dev-flow.md
// quote what scripts print; this pass ties each quote to its source.
//
// A refactor of land.mjs or the pre-commit hook used to leave the guide
// confidently wrong: nothing linked a card to the line that prints it
// (docs/handoff/open/docs-audit.md). BENDS below is that link. For every card
// it checks:
//
//   1. the card exists, and every card on the page has a row here;
//   2. each documented phrase is in the card AND in its source file;
//   3. every quoted <code> in the card's "says" line is covered by a checked
//      phrase, a generic placeholder (<sha>, …) or a declared example — so a
//      word added to a quote cannot slip past unchecked.
//
// Backticks are stripped from sources before matching: a script prints
// `make close` in backticks, the card renders it inside <code>. Where the
// printed text is assembled (shell arguments, Go format verbs, a string split
// over two lines), the row names the source pattern that produces it instead
// of the literal text. Examples — the "codex" terminal, the `x` worktree —
// stand for runtime values and are not checked.
//
//   node scripts/docs-quotes.mjs     # exit 0 ok, 1 failures

import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

export const GUIDE = "docs-site/guide/dev-flow.md";

// One row per card, keyed by its "when" line. A check is
// [phrase in the card, source file, pattern(s) in the source]; the pattern
// defaults to the phrase itself. A RegExp matches the raw source, a string
// matches the source with backticks stripped and whitespace collapsed.
export const BENDS = [
  {
    when: "make ci fails after the fast-forward",
    checks: [
      ["land: make ci FAILED on main at", "scripts/land.mjs"],
      ["— fix it (the branch is already merged; a follow-up branch is the honest fix).", "scripts/land.mjs"],
    ],
  },
  {
    when: "make land cannot fast-forward",
    checks: [
      ["main cannot fast-forward to", "scripts/land.mjs"],
      [": either it is merged already, or it needs make close (merge main into the branch) first.", "scripts/land.mjs"],
    ],
  },
  {
    when: "make land refuses the root",
    checks: [
      ["the root checkout has local changes to", "scripts/land.mjs"],
      ["path(s) this branch also changes:", "scripts/land.mjs"],
    ],
    examples: ["N"],
  },
  {
    when: "make deploy refuses",
    checks: [
      ["a restart would end", "internal/install/readiness.go"],
      ["turn(s):", "internal/install/readiness.go", "a restart would end %d turn(s):"],
      ["Wait, or picode deploy --force (PICODE_DEPLOY_FORCE=1 for make deploy).", "internal/install/readiness.go"],
    ],
    // One busy owner, rendered from `%s %q is %s` with the daemon's values.
    examples: ["1", 'terminal "codex" is working'],
  },
  {
    when: "make handoff names an invisible topic file",
    checks: [
      ["handoff: a topic file is invisible — it lists bullets but none under a ## Next or ## Debts heading", "scripts/handoff-board.mjs"],
      ["Fix: add the heading, or delete the bullets if they are not open items.", "scripts/handoff-board.mjs"],
    ],
  },
  {
    when: "The board is over its target",
    checks: [
      ["A warning, never a gate (ADR-0145)", "scripts/handoff-board.mjs", "ADR-0145: over the target is a warning, never a gate"],
      ["it still renders", "scripts/handoff-board.mjs", "The view still rendered"],
      // close runs the board; the over-target branch reaches `return 0`.
      ["make close still passes", "scripts/close.sh", "make --no-print-directory handoff || exit 1"],
      ["make close still passes", "scripts/handoff-board.mjs", /The view still rendered(?:(?!return 1)[\s\S])*?return 0;/],
    ],
  },
  {
    when: "A session cannot finish",
    checks: [
      ["stalled: …", "scripts/worktree-status.mjs", "stalled: ${stall}"],
      ["no commits yet", "scripts/worktree-status.mjs", 'return "no commits yet"'],
      ["nothing committed — empty branch", "scripts/worktree-status.mjs", 'return "nothing committed — empty branch"'],
      ["idle past a day", "scripts/worktree-status.mjs", "idleHours = 24)"],
    ],
  },
  {
    when: "A commit is refused over a living doc",
    checks: [
      ["CHANGELOG.md no longer starts with # Changelog.", ".githooks/pre-commit", ['refuse "$1 no longer starts with $2."', 'check_shape CHANGELOG.md "# Changelog"']],
      ["a parallel session likely wrote into this worktree", ".githooks/pre-commit", /a parallel session"\s*\\\s*"likely wrote into this worktree/],
    ],
  },
  {
    when: "A commit on main is refused",
    checks: [["Refusing to commit a new session note directly on main (ADR-0149).", ".githooks/pre-commit"]],
  },
  {
    when: "make vale flags repo vocabulary",
    checks: [
      ["Possible typo: '<word>'.", "styles/PiCode/Spelling.yml", `message: "Possible typo: '%s'."`],
      ["error", "styles/PiCode/Spelling.yml", "level: error"],
      // Vale names a rule after its style folder and file.
      ["PiCode.Spelling", "styles/PiCode/Spelling.yml", "extends: spelling"],
      ["styles/config/vocabularies/PiCode/accept.txt", "styles/config/vocabularies/PiCode/accept.txt", /[\s\S]/],
    ],
  },
  {
    when: "A docs link is dead",
    checks: [
      ["ignoreDeadLinks covers only localhost and 127.0.0.1", "docs-site/.vitepress/config.mjs", "ignoreDeadLinks: [/^https?:\\/\\/localhost/, /^https?:\\/\\/127\\.0\\.0\\.1/],"],
    ],
  },
  {
    when: "make worktree-gc keeps a tree",
    checks: [
      ["keep", "scripts/worktree-gc.sh", 'echo "keep $path ($branch: written'],
      [": written in the last hour; FORCE=1 to remove)", "scripts/worktree-gc.sh"],
      ["not merged", "scripts/worktree-gc.sh", "($branch: not merged)"],
      ["dirty", "scripts/worktree-gc.sh", "($branch: dirty)"],
    ],
    examples: [".worktrees/x (feat/x"],
  },
  {
    when: "A debt outlives its note",
    checks: [["after 7 days", "scripts/handoff-board.mjs", "const NOTE_FRESH_DAYS = 7;"]],
  },
];

const ENTITIES = { lt: "<", gt: ">", amp: "&", quot: '"', "#39": "'" };
const decode = (s) => s.replace(/&(lt|gt|amp|quot|#39);/g, (_, e) => ENTITIES[e]);
const squash = (s) => s.replace(/\s+/g, " ").trim();
const text = (html) => squash(decode(html.replace(/<[^>]*>/g, "")));

// The cards as the page holds them: when-line, the text a reader sees, the
// <code> quotes of the "says" line, and the line number for failures.
export function parseBends(md) {
  const cards = [];
  const re = /<div class="bend-item">([\s\S]*?)\n<\/div>/g;
  for (let m; (m = re.exec(md)); ) {
    const body = m[1];
    const when = /<p class="bend-when">([\s\S]*?)<\/p>/.exec(body);
    const says = /<p class="bend-says">([\s\S]*?)<\/p>/.exec(body);
    const at = m.index + (says ? m[0].indexOf(says[0]) : 0);
    cards.push({
      when: when ? text(when[1]) : "",
      body: text(body),
      quotes: says ? [...says[1].matchAll(/<code>([\s\S]*?)<\/code>/g)].map((q) => squash(decode(q[1]))) : [],
      line: md.slice(0, at).split("\n").length,
    });
  }
  return cards;
}

const found = (src, pattern) =>
  pattern instanceof RegExp ? pattern.test(src) : squash(src.replace(/\\?`/g, "")).includes(squash(pattern));

export function bendQuoteFailures(root, read = (p) => readFileSync(join(root, p), "utf8")) {
  const fails = [];
  let md;
  try {
    md = read(GUIDE);
  } catch {
    return [`${GUIDE} missing`];
  }
  const cards = parseBends(md);
  const rows = new Map(BENDS.map((b) => [b.when, b]));
  for (const card of cards) {
    if (!rows.has(card.when)) fails.push(`${GUIDE}:${card.line}: card "${card.when}" has no source row in scripts/docs-quotes.mjs`);
  }
  for (const row of BENDS) {
    const card = cards.find((c) => c.when === row.when);
    if (!card) {
      fails.push(`${GUIDE}: card "${row.when}" is gone — update BENDS in scripts/docs-quotes.mjs`);
      continue;
    }
    const at = `${GUIDE}:${card.line}: card "${row.when}"`;
    for (const [phrase, file, patterns = phrase] of row.checks) {
      if (!card.body.includes(phrase)) {
        fails.push(`${at}: expected phrase "${phrase}" is no longer in the card — update the card or its row`);
        continue;
      }
      let src;
      try {
        src = read(file);
      } catch {
        fails.push(`${at}: source ${file} is missing`);
        continue;
      }
      for (const p of [patterns].flat()) {
        if (!found(src, p)) fails.push(`${at}: "${phrase}" no longer matches ${file} (looked for ${p}) — the script changed; re-quote it`);
      }
    }
    for (const q of card.quotes) {
      // A command named inside a checked sentence is covered by that sentence.
      if (row.checks.some(([phrase]) => phrase.includes(q))) continue;
      let rest = q;
      for (const [phrase] of row.checks) rest = rest.split(phrase).join(" ");
      for (const ex of row.examples ?? []) rest = rest.split(ex).join(" ");
      rest = rest.replace(/<[^>]+>|…/g, " ");
      if (!/^[\s.,:;—()'"]*$/.test(rest)) fails.push(`${at}: quote "${q}" has unchecked text "${squash(rest)}" — add it to the card's row`);
    }
  }
  return fails;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const root = join(dirname(fileURLToPath(import.meta.url)), "..");
  const fails = bendQuoteFailures(root);
  if (fails.length) {
    console.error("docs-quotes FAILED:");
    for (const f of fails) console.error("  - " + f);
    process.exit(1);
  }
  console.log(`docs-quotes ok: ${BENDS.length} dev-flow cards match their sources`);
}
