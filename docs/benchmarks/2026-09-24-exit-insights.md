# Study: computed insights over the exit catalog (option B)

- **Date**: 2026-09-24
- **Follows**: [2026-09-23 — feedback when an agent is removed](2026-09-23-agent-exit-feedback.md),
  whose option B the owner approved as "next"; ADR-0194 (the exit record) and
  its amendments (cost, instruction revisions).
- **Question**: which insights can PiCode compute from its exit records, with
  what minimum evidence, and where does each one lead the person?

## What the catalog holds today (production, read-only, 2026-09-24)

| Fact | Value |
|---|---|
| Exits | 20 (17 not undone) over ~22 hours, all in one workspace |
| Question shown / answered | 11 asked, 8 answered (73%); 6 skipped as `idle` |
| Outcomes | 8 × `resolved`; no `partial`, `unresolved` or `trial`; no reasons |
| By CLI | Omp 6, Claude Code 5, Codex 4, Pi 2 |
| Model recorded | 2 of 17 (Pi only). Every terminal CLI's exit has `model: ""` |
| Cost measured | 14 of 17 |
| Worked | 9 with turns, 6 with none, 2 not measured |
| Instruction revisions | 3 (recorded since today) |

Three consequences shape the design:

1. **Samples are small and one-sided.** Any comparison drawn today would be
   noise; the insight surface must say "not enough yet" gracefully and show
   n everywhere.
2. **The model is missing for terminal CLIs.** "Which model works best" is
   unanswerable until the exit records it. The session file the cost meter
   already reads carries a model per turn (`climetrics.cliEntry.model`), so
   the fix is in the exit, not a new source.
3. **Failures carry the learning, and there are none yet.** The reason-driven
   insights are designed now and light up as answers arrive.

## The insights

Each insight is a rule over the non-undone exits of one workspace (and the
same rule across all of them for the global view). None calls a model.

| Id | Question it answers | Rule | Shown when | Leads to |
|---|---|---|---|---|
| `outcome-by-cli` | Which CLI resolves more here? | Resolved share among answered exits, per CLI, with a 95% Wilson interval | ≥ 5 answered exits for a CLI; a difference is called only when the intervals do not overlap | New agent: the CLI picker line |
| `cost-per-resolved` | What does a resolved agent cost, per CLI (and model, after B0)? | Median cost of resolved exits with a measured cost | ≥ 3 resolved exits with cost | The CLI's page; Model roles |
| `recurring-reason` | What keeps going wrong? | A reason on ≥ 3 exits of the same CLI in 30 days | ≥ 3 | The reason's door (table below) |
| `idle-removals` | Are agents created and dropped without working? | Share of exits with 0 turns, per CLI | ≥ 5 exits for the CLI and share ≥ 50% | The CLI's Check (a launch that fails looks like this) |
| `instructions-before-after` | Did the last edit to the instruction files help? | Exits grouped by their instruction hashes: the newest revision vs the one before, resolved share of each | ≥ 5 answered exits on each side | The Instructions page |
| `needed-you` | Which setup interrupts the person most? | Inbox items that needed the person, per exit, per CLI | ≥ 5 exits for the CLI | CLI Settings (permission mode) |

Doors per reason (the study's "each reason opens the door that already
exists"):

| Reason | Door |
|---|---|
| `setup` — wrong setup | The CLI's Settings, Connectors |
| `misunderstood` | The Instructions page (`#/instructions/<workspace>`), Snippets |
| `stuck` — looped | Checklist level `always`; Model roles |
| `false_done` | Delivery review; the workspace's checks |
| `slow_costly` | Model roles (a cheaper model); the CLI's cost on its page |
| `switched` | `outcome-by-cli` for the CLI switched to |

## Honesty rules

- **n on every number**, and the evidence: each insight lists the exit ids
  it was computed from, and the catalog filters to them in one click.
- **No ranking below the minimum.** Under it the section shows one line:
  "Not enough answers yet — 3 of the 5 needed." (an empty state, with the
  count toward the threshold).
- **Intervals, not bare percentages**, where two groups are compared; "about
  the same" is a valid verdict.
- **Only answered exits count for outcomes**; unanswered ones still count
  for cost, turns and idle removals.
- **Undone exits never count** (ADR-0194).
- **Survivorship is named**, not hidden: the section says the numbers cover
  removed agents only (agents still running are not in them).

## Where they show

| Place | What |
|---|---|
| Outcomes ▸ a new **Insights** section above the catalog | The insights for the workspace filter already on the page (all workspaces = the global view) |
| New agent dialog | One line for the picked CLI, beside the instructions line: "Here Codex resolved 4 of 4 answered (n small)." Only above the minimum |
| The door an insight names | Nothing new: the insight links there |

## What does not belong in B

- **Applying a change for the person.** B links to the door; a one-click
  apply for configuration PiCode owns, with "applied at" recorded for a
  before/after, needs persistence (which insight, when, what changed) — a
  later slice behind its own ADR.
- **Reading transcripts.** That is option C (a model review) and D.
- **Insights across people.** Option E stays parked.

## Slices

| Slice | What | Boundary |
|---|---|---|
| **B0** | The exit records the model(s) of a terminal CLI's sessions, from the session file the meter already reads (dominant model by turns, and the list) | Persistence content (the `model` column, filled) — a line in ADR-0194's amendment list |
| **B1** | `GET /api/agent-exits/insights?workspace=` computed on read; Outcomes ▸ Insights with `outcome-by-cli`, `cost-per-resolved`, `idle-removals`, `recurring-reason`; the minimum/empty states; evidence filter | Protocol: one read route; no table |
| **B2** | `instructions-before-after`, `needed-you`; the New agent line | None new |
| **B3** | Apply for PiCode-owned configuration, "applied at", before/after | ADR (persistence) |

## Decisions for the owner

1. The thresholds above (5 answered per group; 3 for a recurring reason and
   for cost) — conservative on purpose for a catalog of 17.
2. Whether the New agent line shows a comparison at all while n is small
   (recommended: only above the minimum, never a ranking).
3. B0 before B1 (recommended): without the model, half of B1 has nothing
   to group by.
