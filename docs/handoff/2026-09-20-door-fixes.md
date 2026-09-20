# 2026-09-20 — door-fixes: adversarial review of Fatia F
Owner asked for an adversarial review. Findings, all fixed on this branch:
1. BUG: the door's settle loop pressed Enter on every 150ms poll (up to
   ~10) and twice on a lost-Enter observation — a burst that can dismiss
   dialogs or submit someone's draft. Now: paste once, at most one extra
   Enter on a holds-tail observation, diverged/unreadable stop the
   sequence with a named receipt.
2. The occupied/working gate was bypassed on a transient capture failure
   (degraded straight to blind paste). Now the pre-paste capture retries
   (2×150ms) before degrading, and the degrade is documented in the ADR
   table.
3. An unconfirmed automation delivery finished DONE with a note — a done
   that lies (the work has not started). Now failed with the reason.
4. removeAgent matched "not found" in the message text; now keys off the
   404 status (err.status).
Verified: full server package, ci-scoped PASS. ADR-0089 table corrected.
