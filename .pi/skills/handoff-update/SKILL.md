---
name: handoff-update
description: End-of-session note — one short file in docs/handoff/ plus a true docs/handoff.md (≤ 100 lines), written from `make close-summary`, never from a full context. Use before ending any session that changed repo state.
---

# Handoff update (ADR-0086)

The handoff used to be one 485-line file that every session read, rewrote
and hand-merged (103 of 181 merges). Now: **one note per session** in
`docs/handoff/`, and a **capped** `docs/handoff.md` that only says what is
true right now. If nothing changed, say so — no empty ritual.

## Steps

0. **Verify you are home.** `git branch --show-current` and `pwd` must be
   your branch and your worktree. Never write these files from another
   session's path.
1. **Run `make close-summary`** (or `make close`, which ends with it). It
   prints the commits, the diff, the scope and the docs still owed. Write
   from that output. Long session? Do this step in a fresh, small context
   — the summary is all it needs.
2. **Write `docs/handoff/<date>-<branch>.md`**, ≤ 25 lines:

   ```markdown
   # <date> — <branch>: <one-line title>
   Shipped: what exists now (ADR numbers, routes, files).
   Verified: gates run, what was tested where (scratch instance, real tmux…).
   visual-review: PASS | FAIL | UNVERIFIED | n/a
   Not done / debts: what remains, honestly.
   Merge: fast-forward ready | merged as <sha>.
   ```

3. **Keep `docs/handoff.md` true** — edit in place, never append:
   - *Current state*: replace the line your work makes stale.
   - *In flight*: only branches that are not merged.
   - *Next up*: ordered, concrete; remove what shipped.
   - *Known debts*: one line each, with the owner when it is not us.
   The pre-commit hook refuses more than 100 lines. Move nothing to the
   archive: history is `git log` and `docs/handoff/`.
4. **Changelog**: user-visible change without an `[Unreleased]` entry → add
   one line now.
5. **Commit** the note with the code, or as the `docs:` commit right after.

## Verdict

```
handoff-update: DONE (docs/handoff/2026-09-06-x.md; handoff.md 92 lines; changelog +1)
handoff-update: NO-OP (session was read-only, nothing changed)
```
