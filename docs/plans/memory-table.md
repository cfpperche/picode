# Memory pane: from a list to a table that can audit

Status: **approved by the owner 2026-09-20 ("pode fazer as 3 fatias sem
dependência"); slices 1–3 implemented on `feat/memory-table`.** Slice 4 is
open and is the only one that would add a dependency. The subsystem is
`docs/architecture/cli-memory.md`; the decision behind the pane is ADR-0163.

- **Date:** 2026-09-20
- **Owner request:** replace the flat list with a better component; research
  the web for a data grid and for benchmarks of what a table can usefully do
  to these files.
- **Measured on this repository's own store**
  (`~/.claude/projects/-home-goat-picode/memory`, 2026-09-20), not inferred.

## What the files already carry

The list showed three things: title, kind chip, truncated description. The
files carry more, and all of it is on disk already:

| Field | Where | Example |
|---|---|---|
| `name` | frontmatter | `agent-browser-eval-scope-and-radix-clicks` |
| `description` | frontmatter | the one-liner the list already showed |
| `metadata.type` | frontmatter | feedback 30, project 22, reference 2 |
| `metadata.modified` | frontmatter, written by the CLI | `2026-09-05T22:14:45.912Z` (41 of 55 files carry one) |
| `metadata.originSessionId` | frontmatter | which session wrote it |
| `[[wikilinks]]` | body | a citation graph between memories |
| index row | `MEMORY.md` | whether the CLI loads it at session start |
| bytes, mtime | filesystem | |

## What a table finds that the list could not

Over the 54 memories in this repository's store:

| Finding | Count | Why it matters |
|---|---|---|
| Broken `[[link]]` | **1** — `no-typing-into-other-sessions` → `[[picode-parallel-sessions]]`, which does not exist | the memory points at nothing |
| Never cited by another memory | **23 of 54** | candidates to merge or drop |
| Most cited | `picode-scratch-dogfood` (10), `verify-by-exit-code-not-grep` (7), `deploy-merges-first` (6) | the load-bearing ones |
| Not in the index | 0 today | a memory absent from `MEMORY.md` is invisible to the CLI at session start |
| Index budget | 54 lines, 10.3 KB | **Claude Code loads only the first 200 lines or 25 KB of `MEMORY.md`**, whichever comes first; everything past that is dropped at the next load. At 41 % of the byte budget |

That last row is the one no other tool shows: a documented vendor limit the
owner cannot see from the CLI, and the pane already knows both numbers.

## The component

Verified 2026-09-20: `@tanstack/react-table@9.2.4`, MIT, 134 KB unpacked,
pulling `@tanstack/table-core` and `@tanstack/react-store` — three packages,
not one. V9 went stable on 2026-08-04 and made features opt-in
(`tableFeatures({ rowSortingFeature, … })`), so a table ships only the
features it registers. It is headless: it renders nothing, which is what would
let the repo keep its own tokens and markup.

| Option | Verdict |
|---|---|
| **TanStack Table v9, headless** | Worth adopting *if* the pane grows a cross-project view. Sorting, faceted filters, selection and grouping for three MIT packages, with PiCode owning every element |
| AG Grid | Refuse: its own chrome, its own theme, enterprise tiers. ADR-0008 and `docs/benchmarks.md` both point the other way |
| MUI DataGrid | Refuse: pulls MUI beside Tailwind and the token set |
| react-data-grid | Refuse: renders its own spreadsheet layout; the gain over hand-rolling is small and the fit is worse |
| **No library** (shipped) | Sorting and filtering 54 rows is about 40 lines of pure functions in `web/shared/domain/memoryTable.js`. The Providers roster next door is already a hand-rolled six-column CSS grid |

**The honest reading: the value is in the columns and the audit, not in the
grid library.** At 54 rows the library is the cheap part. It earns its place
only at the scale the cross-project view would bring — Claude Code alone has
**513 memory files across 37 projects** on this machine, and that needs
virtualization, which hand-rolling does not give for free.

## Benchmarks

| Source | Pattern taken |
|---|---|
| **Obsidian Bases** (core plugin, 2026) | Database views over files whose data stays in the markdown; frontmatter properties are the columns. Exactly our shape |
| **OpenMemory dashboard** (mem0) | Filter by category and date. Taken. Refused: memory *states* (active/paused/archived) and the per-app access switch — these are files, and a state PiCode invents is a second store, which ADR-0163 already refused |
| **Knowledge-base audit practice** | Every audited item gets one action: Keep, Update, Merge, Delete. Inventory lists last-updated and category to find duplicates and gaps |
| **claude-mem viewer** | Delete one item with a confirm, stats refresh, live sync across tabs (ours is the ADR-0048 feed) |
| **agentmemory** | Consolidation on session stop. Refused: a retention score PiCode computes for itself — the CLI owns what it keeps |

## What shipped

Columns, all derived from what the server already reads:

1. **Memory** — title, kind chip, description
2. **Cited by** — inbound `[[links]]`; the zero is the finding, so it is
   printed rather than hidden behind a chip
3. **Size** — bytes
4. **Modified** — frontmatter date where the CLI wrote one, mtime otherwise,
   with a tooltip saying which
5. **Health** — chips for: not in the index, broken link out

Above the table: the index budget meter and the kind facets with counts.
Selection and bulk delete below them, with a confirm that names the blast
radius. Read-only tiers get the same columns with the actions absent, not
disabled.

**Session (`originSessionId`) was left out.** The field is in the payload, but
PiCode has no route that opens a Claude Code session from its id, and a column
of opaque ids is not a column.

| # | Slice | State |
|---|---|---|
| 1 | Derive the extra fields server-side and add them to `GET /api/cli-memory` | shipped |
| 2 | Sortable columns + faceted kind filter + the index budget meter, hand-rolled | shipped |
| 3 | Selection and bulk delete | shipped |
| 4 | *(open)* cross-project view for Claude Code's 37 projects, and with it TanStack v9 + virtualization | the owner's call; the only slice that adds a dependency |

## Open questions

- Should the table read only the current store, or offer the cross-project
  view? The second is where the library decision lives.
- Deleting a memory leaves its row in `MEMORY.md`. The pane says so and the
  dangling row shows up one line up on the next load; whether PiCode should
  edit the index itself is a separate decision — it is the CLI's file, and
  ADR-0163's line is that PiCode never writes what it does not own.
