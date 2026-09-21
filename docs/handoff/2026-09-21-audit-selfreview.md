# 2026-09-21 — feat/audit-selfreview: the gates from feat/audit-fixes were passing work they claimed to check

One commit (7c6adb48), written after an adversarial pass over the branch that landed hours earlier. Both new gates had the failure mode they exist to stop.

The ADR status check in `scripts/docs-check.mjs` read `- **Status**: x` but not `- **Status:** x` — the spelling ten of the 171 decision records use — and a file it could not parse was skipped, not failed. It was checking 161 of 171 and reporting success. Asterisks are now dropped before matching, both spellings read, and an unreadable status line fails.

`TestEveryExportedMutationAnnouncesOrIsListed` looked for SQL in each method's own body, so any exported mutator that delegates its write was invisible: `AddAgent` through `AddAgentWithCLI`, `CreateTerminal`, `EnablePeer`, `ReplaceFrom` and thirteen more — seventeen writes unchecked. Fourteen of them did announce, so the gate had been right by luck rather than by construction. The SQL signal now follows calls on the receiver the way the event signal already did; the three that announce nothing are listed with reasons (`AppendEvent`/`AppendEventTx` *are* announcing; `ReplaceFrom` swaps the whole database). Bodies are read with line comments stripped — "we deliberately do not AppendEvent here" would have satisfied the old check.

Both holes have probes that fail without the fix. What survived the pass unchanged: `swapTree`/`swapRegular` (no sibling hazard — `sweepPinDirs` reads inside `pins/`, snapshots copy by exact name, nothing globs the data-dir root), the route-coverage test (no filesystem side effects from the reflection-filled `Deps`), and the three new guides, whose numbers were re-checked against `MaxPinFiles`/`MaxPinImageSize`/`PIN_LIMITS` rather than against the architecture doc they were written from.

Verified: `make close` green, scoped. visual-review: n/a — no UI diff. Nothing deployed.

## Debts

- ~~The mutation gate reads source text, not types: a write assembled from a package-level constant, or a table name with a digit, would still evade it.~~ **Paid the same day by `feat/gate-union` (51e19c3c):** a write is now either SQL the test can read or a call that runs one, which between them have no gap. Filing this was an unmeasured guess about cost — the fix was three lines.
