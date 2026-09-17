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
// ADR-0145: the board is a *bounded view*. Its sources are append-only (one
// note per session, one bullet per item), so a fixed byte cap could only be
// met by deleting another agent's prose — measured 2026-09-16: four prune
// rounds in one session, one "paid" line already false. The view now truncates
// by design (per-topic bullets, a short note window) and the targets below are
// a warning, not a gate.
const MAX_LINES = 120;
const MAX_BYTES = 12 * 1024;
// A note is a handoff to the next session, not a ledger: its bullets ride the
// board for a week. Durable items belong in docs/handoff/open/<topic>.md.
const NOTE_FRESH_DAYS = 7;
// Per topic, before the pointer line: enough to see what is next, not the
// whole list. The file has the rest.
const TOPIC_NEXT_MAX = 2;
// Bullets a topic's Debts section may contribute as a *count* line is always
// one line; the quota is what keeps the biggest topics from dominating.
const NOTE_BULLET_MAX = 8;

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
  const { open: debts, paid } = scanDebts(markdown);
  return { stem, title, next, debts, paid, plan: planPath(markdown), bullets: countBullets(markdown), visible: next.length + debts.length + paid.length };
}

// A debt bullet carries its state when it wants to: `- [x]` is paid, a plain
// `- ` bullet (and `- [ ]`) is open. Paid ones leave the board and stay in the
// file — the ledger keeps its history, the view keeps its size (ADR-0145).
function scanDebts(markdown) {
  const open = [];
  const paid = [];
  let inDebts = false;
  for (const raw of markdown.split("\n")) {
    const heading = /^##\s+(.*)$/.exec(raw);
    if (heading) {
      inDebts = DEBT_HEADINGS.includes(normalizeHeading(heading[1]));
      continue;
    }
    if (/^#\s+/.test(raw)) {
      inDebts = false;
      continue;
    }
    if (!inDebts) continue;
    const item = /^[-*]\s+(?:\[([ xX])\]\s*)?(.*)$/.exec(raw);
    if (!item) continue;
    const done = (item[1] ?? "").toLowerCase() === "x";
    (done ? paid : open).push(item[2].trim());
  }
  return { open, paid };
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
  const { open: debts } = scanDebts(markdown);
  return { stem, date, branch: stem.replace(/^\d{4}-\d{2}-\d{2}-/, ""), next: pick(sections, NEXT_HEADINGS), debts };
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
  // ADR-0145: every section is bounded by construction, so the view always
  // renders and the targets below can only warn. A note is a handoff to the
  // next session: its bullets ride along for NOTE_FRESH_DAYS, newest first, and
  // the ones that do not fit are named by a pointer rather than dropped in
  // silence.
  const fresh = notes.filter((n) => ageInDays(n.date, now) <= NOTE_FRESH_DAYS).sort((a, b) => (a.date < b.date ? 1 : -1));
  const next = [];
  let nextHidden = 0;
  for (const t of topics) {
    next.push(...attribute(t.next.slice(0, TOPIC_NEXT_MAX), t.stem));
    if (t.next.length > TOPIC_NEXT_MAX) {
      next.push(`- …+${t.next.length - TOPIC_NEXT_MAX} more in \`docs/handoff/open/${t.stem}.md\``);
      nextHidden += t.next.length - TOPIC_NEXT_MAX;
    }
  }
  const noteNext = fresh.flatMap((n) => attribute(n.next.slice(0, 1), n.stem));
  next.push(...noteNext.slice(0, NOTE_BULLET_MAX));
  if (noteNext.length > NOTE_BULLET_MAX) {
    next.push(`- …+${noteNext.length - NOTE_BULLET_MAX} more from recent session notes in \`docs/handoff/\``);
    nextHidden += noteNext.length - NOTE_BULLET_MAX;
  }
  // Debts: a topic is one line (its count, its file, its plan — ADR-0131 C); a
  // fresh note's open debt is a line of its own until it ages out or moves into
  // the topic that owns it.
  const debts = [
    ...[...topics]
      .filter((t) => t.debts.length)
      .sort((a, b) => b.debts.length - a.debts.length || a.stem.localeCompare(b.stem))
      .map((t) => {
        const paid = t.paid.length ? `, ${t.paid.length} paid` : "";
        return `- ${t.debts.length} open debt(s)${paid} — *${t.stem}* — open \`docs/handoff/open/${t.stem}.md\`${t.plan ? ` — Plan: \`${t.plan}\`` : ""}`;
      }),
  ];
  const noteDebts = fresh.flatMap((n) => attribute(n.debts.slice(0, 1), n.stem));
  debts.push(...noteDebts.slice(0, NOTE_BULLET_MAX));
  if (noteDebts.length > NOTE_BULLET_MAX) {
    debts.push(`- …+${noteDebts.length - NOTE_BULLET_MAX} more from recent session notes in \`docs/handoff/\``);
  }

  const lines = [
    "# Handoff — living project state",
    "",
    "> Generated by `make handoff` (ADR-0123): **do not edit this file**, it is not in git.",
    "> *In flight* is git state. *Next up* is a **view**: two bullets per topic plus the",
    `> newest session notes (≤ ${NOTE_FRESH_DAYS} days), then a pointer — the rest lives in \`docs/handoff/open/<topic>.md\`.`,
    "> *Debts* counts each topic's **open** ones (`- [ ]`, or a plain bullet); `- [x] paid` stays in the file.",
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

  // Who is over quota: named so the *sources* can be trimmed, never the view.
  const over = [];
  const bytes = text.length;
  if (lines.length > MAX_LINES || bytes > MAX_BYTES) {
    const weigh = (list) =>
      list
        .map((entry) => ({ stem: entry.stem, lines: entry.next.length + entry.debts.length + (entry.paid?.length ?? 0) }))
        .sort((a, b) => b.lines - a.lines);
    over.push(...weigh([...topics, ...notes]).slice(0, 8));
  }
  return { text, over, stats: { lines: lines.length, bytes, hidden: nextHidden } };
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
    // ADR-0145: over the target is a warning, never a gate. The view is bounded
    // by construction, so this points at the *sources* to trim — topics first,
    // since a note ages out on its own.
    console.error(
      `handoff: over target (${stats.lines} lines / ${stats.bytes} bytes; the targets are ${MAX_LINES} lines and ${MAX_BYTES} bytes).\n` +
        "The view still rendered; trim the largest sources if the board reads badly:",
    );
    for (const o of over) console.error(`  - ${o.stem}: ${o.lines} bullet(s)`);
    if (stats.hidden) console.error(`  (${stats.hidden} bullet(s) behind pointers)`);
  }
  writeFileSync(outPath, text);
  console.log(`handoff: wrote docs/handoff.md (${stats.lines} lines, ${stats.bytes} bytes${stats.hidden ? `, ${stats.hidden} behind pointers` : ""})`);
  return 0;
}

const invoked = process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1];
if (invoked) process.exit(main());
