#!/usr/bin/env node
// handoff-board — generate docs/handoff.md (ADR-0123).
//
// The board is a view, not a document: the file that every session used to
// rewrite (406 commits in ten days, 99.8% of its byte cap) is now assembled
// from what sessions actually write — one topic file per open subject under
// docs/handoff/open/, the `## Next up` / `## Debts` sections of session notes,
// and live git state for what is in flight. It is git-ignored; the hook
// refuses a staged copy.
//
//   make handoff                 # write docs/handoff.md
//   node scripts/handoff-board.mjs --stdout    # render without writing
//   node scripts/handoff-board.mjs --check     # exit 1 if the file is stale
//
// The budget is a generator rule, not a writer's discipline: over 120 lines or
// 12 KB, the generator fails and names the files that overflow.

import { readFileSync, readdirSync, writeFileSync, existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { flightLines, readWorktrees } from "./worktree-status.mjs";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
// ADR-0123: the caps survive as a pruning pressure, not as a writer's rule.
// ADR-0105's 8 KB was measured against a hand-edited file gamed with 581-byte
// lines; this view is generated from a fixed source set and read once per
// session, so it carries the ledger at 12 KB and still fails over budget.
const MAX_LINES = 120;
const MAX_BYTES = 12 * 1024;
const NOTE_NEXT_DAYS = 30;

const NEXT_HEADINGS = ["next", "next up", "next steps"];
const DEBT_HEADINGS = ["debts", "debt", "debts / open questions", "open questions", "known debts"];

function normalizeHeading(text) {
  return text.replace(/[#*_`]/g, "").replace(/\s+/g, " ").trim().toLowerCase();
}

// A bullet is a `- ` line plus any indented continuation lines beneath it.
export function parseSections(markdown) {
  const out = {};
  let current = null;
  let bullet = null;
  for (const raw of markdown.split("\n")) {
    const heading = /^##\s+(.*)$/.exec(raw);
    if (heading) {
      current = normalizeHeading(heading[1]);
      bullet = null;
      continue;
    }
    if (/^#\s+/.test(raw)) {
      current = null;
      continue;
    }
    if (!current) continue;
    const item = /^[-*]\s+(.*)$/.exec(raw);
    if (item) {
      bullet = item[1].trim();
      (out[current] ??= []).push(bullet);
      continue;
    }
    if (bullet && /^\s+\S/.test(raw)) {
      const list = out[current];
      list[list.length - 1] = `${list[list.length - 1]} ${raw.trim()}`;
      bullet = list[list.length - 1];
      continue;
    }
    if (raw.trim() === "") bullet = null;
  }
  return out;
}

function pick(sections, names) {
  const found = [];
  for (const [heading, bullets] of Object.entries(sections)) {
    if (names.includes(heading)) found.push(...bullets);
  }
  return found;
}

// A topic file also declares where its detail lives: "Plan: docs/plans/x.md"
// anywhere in the file (prose or bullet). ADR-0131 renders that path next to
// the topic's counts, so a reader can go from the board to the detail in one
// hop instead of finding it by hand.
function planPath(markdown) {
  const m = /Plan:\s*`?([^`\s)]+)/i.exec(markdown);
  return m ? m[1].replace(/[.,;:]+$/, "") : "";
}

// Every bullet in the file, including the ones under headings the board does
// not render (`Traps`, `Slice 3`). A file with bullets and none under a
// recognized heading is invisible in full — the defect ADR-0131 refuses.
function countBullets(markdown) {
  let total = 0;
  for (const line of markdown.split("\n")) if (/^[-*]\s+/.test(line)) total++;
  return total;
}

export function parseTopicFile(markdown, stem) {
  const sections = parseSections(markdown);
  const title = (/^#\s+(.+)$/m.exec(markdown)?.[1] ?? stem).trim();
  const next = pick(sections, NEXT_HEADINGS);
  const debts = pick(sections, DEBT_HEADINGS);
  return { stem, title, next, debts, plan: planPath(markdown), bullets: countBullets(markdown), visible: next.length + debts.length };
}

// ADR-0131 (A): a topic file that opens with prose and lists its debts anyway
// used to contribute nothing at all, silently. Name it instead.
export function invisibleTopics(topics) {
  return topics.filter((t) => t.bullets > 0 && t.visible === 0).map((t) => t.stem);
}

export function parseNote(markdown, fileName) {
  const stem = fileName.replace(/\.md$/, "");
  const date = (stem.match(/^(\d{4}-\d{2}-\d{2})/) ?? [])[1] ?? "";
  const sections = parseSections(markdown);
  return { stem, date, branch: stem.replace(/^\d{4}-\d{2}-\d{2}-/, ""), next: pick(sections, NEXT_HEADINGS), debts: pick(sections, DEBT_HEADINGS) };
}

export function ageInDays(date, now) {
  if (!date) return Infinity;
  const ms = now - Date.parse(`${date}T00:00:00Z`);
  return Number.isNaN(ms) ? Infinity : ms / 86_400_000;
}

function attribute(bullets, source) {
  return bullets.map((b) => `- ${b} — *${source}*`);
}

export function render({ worktrees, topics, notes, now = Date.now(), idleHours = 24 }) {
  const fresh = notes.filter((n) => ageInDays(n.date, now) <= NOTE_NEXT_DAYS);
  const next = [
    ...topics.flatMap((t) => attribute(t.next, t.stem)),
    ...fresh.flatMap((n) => attribute(n.next, n.stem)),
  ];
  // Debts are the unbounded side: they accumulate per topic and only matter
  // when that topic is being worked on. The board carries the count and the
  // plan; the bullets stay in the file (ADR-0131 C).
  const debts = [
    ...[...topics]
      .filter((t) => t.debts.length)
      .sort((a, b) => b.debts.length - a.debts.length || a.stem.localeCompare(b.stem))
      .map((t) => `- ${t.debts.length} debt(s) — *${t.stem}* — open \`docs/handoff/open/${t.stem}.md\`${t.plan ? ` — Plan: \`${t.plan}\`` : ""}`),
    ...notes.flatMap((n) => attribute(n.debts, n.stem)),
  ];

  const lines = [
    "# Handoff — living project state",
    "",
    "> Generated by `make handoff` (ADR-0123): **do not edit this file**, it is not in git.",
    "> *In flight* is git state. *Next up* carries the one-line next steps; *debts* list each",
    "> topic's count with its plan, and the bullets live in `docs/handoff/open/<topic>.md`",
    "> (notes contribute their `## Next up` / `## Debts` bullets directly).",
    "> To change something here: edit that topic file or your session note, then run `make handoff`.",
    "",
    `_${new Date(now).toISOString().replace(/\.\d+Z$/, "Z")} — ${worktrees.length - 1} worktree(s), ${topics.length} open topic(s), ${notes.length} note(s)_`,
    "",
    "## In flight",
    "",
    ...flightLines(worktrees, now, idleHours),
    "",
    "## Next up",
    "",
    ...(next.length ? next : ["Nothing declared. Add a `## Next` bullet to `docs/handoff/open/<topic>.md`."]),
    "",
    "## Debts / open questions",
    "",
    ...(debts.length ? debts : ["Nothing declared."]),
    "",
  ];
  const text = lines.join("\n");

  // Who overflows, if anyone: the generator names the files, the writer prunes.
  const over = [];
  const notesBytes = text.length;
  if (lines.length > MAX_LINES || notesBytes > MAX_BYTES) {
    const weigh = (list) =>
      list.map((entry) => ({ stem: entry.stem, lines: entry.next.length + entry.debts.length })).sort((a, b) => b.lines - a.lines);
    over.push(...weigh([...topics, ...notes]).slice(0, 8));
  }
  return { text, over, stats: { lines: lines.length, bytes: notesBytes } };
}

function readTopics() {
  const dir = join(root, "docs", "handoff", "open");
  if (!existsSync(dir)) return [];
  return readdirSync(dir)
    .filter((f) => f.endsWith(".md") && f !== "README.md")
    .sort()
    .map((f) => parseTopicFile(readFileSync(join(dir, f), "utf8"), f.replace(/\.md$/, "")));
}

function readNotes() {
  const dir = join(root, "docs", "handoff");
  if (!existsSync(dir)) return [];
  return readdirSync(dir)
    .filter((f) => f.endsWith(".md"))
    .sort()
    .map((f) => parseNote(readFileSync(join(dir, f), "utf8"), f));
}

function main() {
  const argv = process.argv.slice(2);
  const idleIdx = argv.indexOf("--idle-hours");
  const idleHours = idleIdx >= 0 ? Number(argv[idleIdx + 1]) || 24 : 24;
  const now = Date.now();
  const { text, over, stats } = render({ worktrees: readWorktrees(), topics: readTopics(), notes: readNotes(), now, idleHours });
  const outPath = join(root, "docs", "handoff.md");

  if (argv.includes("--stdout")) {
    console.log(text);
    return 0;
  }
  if (argv.includes("--check")) {
    const current = existsSync(outPath) ? readFileSync(outPath, "utf8") : "";
    // The render is deterministic for a given instant: reuse the instant the
    // file records, so the check tests the SOURCES, not the clock.
    const recorded = Date.parse((current.match(/^_(\d{4}-\d\d-\d\dT[0-9:]+Z)/m) ?? [])[1] ?? "");
    if (!Number.isNaN(recorded)) {
      const again = render({ worktrees: readWorktrees(), topics: readTopics(), notes: readNotes(), now: recorded, idleHours });
      if (current !== again.text) {
        console.error("handoff-check FAILED: docs/handoff.md is stale — run `make handoff`");
        return 1;
      }
      console.log(`handoff-check ok (${again.stats.lines} lines, ${again.stats.bytes} bytes)`);
      return 0;
    }
    if (current !== text) {
      console.error("handoff-check FAILED: docs/handoff.md is stale — run `make handoff`");
      return 1;
    }
    console.log(`handoff-check ok (${stats.lines} lines, ${stats.bytes} bytes)`);
    return 0;
  }
  const invisible = invisibleTopics(readTopics());
  if (invisible.length) {
    console.error(
      "handoff: a topic file is invisible — it lists bullets but none under a `## Next` or `## Debts` heading,\n" +
        "so the board would show nothing from it (ADR-0131 A):",
    );
    for (const stem of invisible) console.error(`  - docs/handoff/open/${stem}.md`);
    console.error("Fix: add the heading, or delete the bullets if they are not open items.");
    return 1;
  }
  if (over.length) {
    console.error(
      `handoff: over budget (${stats.lines} lines / ${stats.bytes} bytes; the caps are ${MAX_LINES} lines and ${MAX_BYTES} bytes).\n` +
        "Prune the largest sources (bullets under `## Next` / `## Debts`):",
    );
    for (const o of over) console.error(`  - ${o.stem}: ${o.lines} bullet(s)`);
    return 1;
  }
  writeFileSync(outPath, text);
  console.log(`handoff: wrote docs/handoff.md (${stats.lines} lines, ${stats.bytes} bytes)`);
  return 0;
}

const invoked = process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1];
if (invoked) process.exit(main());
