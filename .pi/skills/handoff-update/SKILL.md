---
name: handoff-update
description: End-of-session ritual — update docs/handoff.md and CHANGELOG.md so the next agent or human picks up exactly where this session stopped. Use before ending any session that changed repo state.
---

# Handoff update

The handoff is PiCode's heartbeat (see `AGENTS.md` non-negotiable #1 and #4:
documentation evolves with the project; honesty over polish). Run this
**before ending any session that changed state**. If nothing changed,
say so — don't perform empty ritual updates.

## Steps

1. **Read** `docs/handoff.md` fully. It is short; keep it that way.

2. **Update "Current state"** — rewrite it so a brand-new session understands
   what exists *right now*. Delete stale lines; don't append.

3. **Update "In flight"** — list anything half-done:
   - What works, what doesn't, what's untested.
   - The tree must still compile and pass gates (AGENTS.md rule #2);
     if you couldn't finish, say exactly what remains.

4. **Update "Next up"** — ordered, concrete next steps. Add criteria for
   decisions that need ADRs (e.g. "ADR-0004: frontend framework").

5. **Update "Known debts / open questions"** — anything owed, with owner
   context ("needs owner action on GitHub settings").

6. **Prepend to "Recent activity"** — one line: date + what changed.
   UI sessions include the `visual-review:` verdict (PASS/FAIL/UNVERIFIED).

7. **Housekeeping**: if the file exceeds ~150 lines, move the oldest
   "Recent activity" blocks to `docs/handoff-archive.md` (create if missing).

8. **Changelog**: if user-visible changes were made and not yet logged,
   add them under `[Unreleased]` in `CHANGELOG.md` now.

9. **Commit** docs together with the code they describe (same commit or
   an immediately following `docs: handoff` commit).

## Verdict

```
handoff-update: DONE (handoff +2/-9 lines, changelog +1, committed as 1a2b3c)
handoff-update: NO-OP (session was read-only, nothing changed)
```
