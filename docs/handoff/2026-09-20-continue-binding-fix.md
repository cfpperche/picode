# 2026-09-20 — continue-binding-fix: bounded session binding retries

Implemented: a 10-second background pending-session resolver after interactive
Pi startup; a 2-second Muse latest-session retry when its index returns no row.
Architecture and changelog describe the changes; no protocol boundary changed.
Observed: scratch menu.png visibly exposes Continue in for Pi retry; the same
capture shows three failed turns and zero output tokens, not successful replies.
Visual card: viewport fit yes, readable yes, trigger usable yes, no observed
clipping/double-scroll; next click clear. Hover was not captured.
visual-review: PASS for screenshot geometry and capture-time overlay audit
(ok true, four hits, no uncovered overlays or clipping); hover unverified.
Verified: ci-scoped PASS, reported by the working agent.
Existing Pi/Muse debts remain open in docs/handoff/open/sessions.md: the capture
does not prove late first-prompt binding or recovery from Muse's empty index.
No cross-CLI transfer or successful model reply is asserted by this note.
Evidence: var/screenshots/continue-binding-fix/menu.png (gitignored).
Merge: fast-forward ready at close-summary against main 3f15548c.
