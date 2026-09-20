# 2026-09-18 — pre-commit-conflict-markers

Owner-approved adaptation of the tachyon secrets-guard lesson to PiCode's
own flow, after two same-day incidents of a conflicted merge resolved with
`git add -A` shipping markers (one on main, breaking the web build).

## Done
- `.githooks/pre-commit`: refuses ADDED lines starting with
  `<<<<<<<`/`>>>>>>>`, placed BEFORE the merge-in-progress early exit — a
  conflict resolution is exactly where markers ride, and that early exit
  skips the pre-existing `git diff --check` refusal. No escape token: the
  bare `=======` separator stays with the whitespace check (pre-existing
  refusal outside merges).
- Selftest rows (scripts/hooks-selftest.sh): start/end marker refused,
  bare separator refused by the whitespace check, mid-line quote allowed,
  context-only marker allowed, and the incident replay — a bad conflict
  RESOLUTION commit refused while a clean one passes.
- AGENTS.md §5: the never-`git add -A`/`git add .` rule (stage explicit
  paths), with the incident cited.

## Debts
- The bare `=======` separator can still ride a merge resolution (git
  --check skips merges too); unambiguous start/end markers cannot.
