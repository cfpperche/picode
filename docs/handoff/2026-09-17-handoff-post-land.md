# 2026-09-17 — handoff-post-land: a post-merge fact has a door (ADR-0149)

The owner asked how to close the loop after the field confirmation of the
wordmark fix had nowhere to go. Two halves, both shipped here.

**A — a note never states a pending state.** `/skill:handoff-update` now
requires "Verified: method + blind spot" ("not run inside the Windows shell"),
never "the owner's click is the last check" — a sentence that expires the hour
after it is written, and costs the next session the verification again.

**B — one door on `main` (ADR-0149, amends ADR-0105).** `docs/handoff/`
files that already landed can be amended, and `docs/handoff/open/<topic>.md`
can be added, by one commit in the root checkout: no worktree, branch or gate
run, because the fact (a field confirmation, a debt paid by observation)
cannot exist before the merge. Fragments, *new* session notes and the board
stay refused on `main`. `docs/decisions/README.md` records the amendment in
both index rows; AGENTS.md §1/§5, CONTRIBUTING.md, the skill and `make land`'s
closing hint carry it.

Verified: `make hooks-check` — 37 rows, including three new ones (amending a
landed note allowed, new topic file allowed, new session note still refused);
`make ci-scoped` PASS (full). The door goes live when this lands, because
`core.hooksPath` points at the root checkout's `.githooks`; the demonstration
is the commit that corrects today's `2026-09-17-shell-brand-click.md` — made
on `main`, with no branch.

No changelog fragment: this is the contributor process, not user-visible
behavior.

visual-review: n/a (no UI surface touched)

## Debts

- The door's real-world use is measured by whether `docs/handoff/` commits on
  `main` stay corrections instead of turning into the 65-commit ritual
  ADR-0105 banned; if they drift, the hook condition comes back.
