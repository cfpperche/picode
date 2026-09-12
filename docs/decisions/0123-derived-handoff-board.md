# ADR-0123: The handoff board is derived, not written

- **Status**: accepted
- **Date**: 2026-09-12
- **Boundary**: process — how project state is recorded, who is allowed to write it, and what enforces it
- **Supersedes**: the board clauses of ADR-0086 (a hand-edited `docs/handoff.md` of ≤ 100 lines / 8 KB read first by every session) and of ADR-0105 §1 ("every session keeps `docs/handoff.md` true")

## Context

Measured 2026-09-12 over the ten days that followed ADR-0105 — 1167 commits,
184 session notes, 5–6 worktrees in flight at any moment:

| Metric | Value |
|---|---|
| Non-merge commits touching `docs/handoff.md` | **406** — ~35% of all commits, ~2.2 per session |
| Size | **8172 of 8192 bytes** (99.8%) while a session must still add its lines |
| Lines added / removed | **+5162 / −6766** — the board is *net losing* text every day |
| Hand-resolved merges | **1 in 12** recomputed with `git merge-tree` — ADR-0105's fragment fix worked; the board is the remaining hot file |
| In-flight accuracy | the board named `feat/desktop-v2-mgmt`, removed days earlier, and never listed `feat/term-attach-sketch`, worktree and dirty tree on disk |

One cause explains both failures. The board is the only **shared** file every
session must write, and at the same time the only place that reports
**in-flight state — which is git state, not prose**. ADR-0105 solved exactly
this class for the changelog by assembling it from per-branch fragments; the
board stayed the single-writer exception and saturated: a session writing three
lines into a file at 99.8% has to delete someone else's text to fit. This
session could not record anything at all (`docs/handoff.md is at its 8 KB cap`)
and left its pointer in its own note instead.

## Decision

`docs/handoff.md` becomes a **generated, git-ignored view** — `make handoff`,
also run by `make close`: *in flight* from `git worktree list` plus per-branch
state (ahead/behind `main`, dirty files, last commit and last green gate age),
*next up* and *debts* from `docs/handoff/open/<topic>.md` and from the
`## Next up` / `## Debts` sections of session notes, *plans* from the status
lines of `docs/plans/*.md`. No session writes it; `.githooks/pre-commit`
refuses a staged copy (an ignored file forced with `git add -f`); `make
handoff` regenerates it before anything reads it.

Durable items live in one file per topic under `docs/handoff/open/`, the shape
the board's two sections already had by hand; a paid item is deleted from its
topic file in the branch that paid it. Session-local follow-ups stay in the
session note (`docs/handoff/<date>-<branch>.md`).

## Consequences

- The last shared singleton leaves the merge surface: 406 writes in ten days
  become 0, and "in flight" cannot go stale because it is computed from disk.
- The cap stops being a rule a human must respect and becomes a generator
  budget: over budget, `make handoff` fails and names the files that overflow.
  It moves from 100 lines / 8 KB to 120 lines / 12 KB, because the measurement
  above is about writers, not bytes: the ledger is 8.1 KB of prose today, the
  per-topic split makes its structure visible and costs ~10.5 KB to render, and
  ADR-0105's 8 KB was measured against a hand-edited file gamed with 581-byte
  lines — not against a generated view of a fixed source set. The budget stays
  because it is the only thing that makes anyone prune.
- A debt now has an owner: a debt without a topic file is a debt nobody owns.
- Cost: the board is not in the repository, so a fresh clone reads nothing
  until `make handoff` runs — `README.md`, `AGENTS.md` and `docs/guidelines.md`
  say so, and the target needs no network.
- If this is wrong (a generated board reads worse than a curated one), the
  inputs are untouched: notes and topic files still hold every word, and one
  commit removing the ignore rule and the target restores a hand-written board.

## Alternatives considered

- **Keep the board and raise the cap.** Size is not the problem; the number of
  writers is. At ~18 sessions a day any cap is either evicted or large enough
  to be a token tax at the top of every context (ADR-0086's original finding).
- **Front-matter status keys in every session note** (the first shape of this
  plan, `docs/plans/cli-settings-ux.md` §P0's sibling proposal). A note reaches
  `main` with its branch, so a board assembled from notes shows *nothing* in
  flight for every unmerged branch — the state it exists to report. It is git
  state; derive it.
- **Generate the board but keep it committed.** Every branch would produce a
  different version of the same file and conflict on every merge — the exact
  cost being removed.
- **One long-lived `docs/handoff/debts.md`.** A shared file again, with the
  same saturation and conflict behaviour as the board it replaces.
