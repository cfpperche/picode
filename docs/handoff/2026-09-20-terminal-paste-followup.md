# 2026-09-20 — feat/terminal-paste-followup: paste-door unification, keydown claim, functional staging
Shipped: `promptDoorFor` (one door predicate = context-menu rule) passed as
prop from App/Canvas Panel instead of recomputed in TermSurface;
`termPasteClaim.js` suppresses the keydown path's late text when a
files-paste is claimed; `addFiles` appends functionally under the cap;
accompanying clipboard text seeds the attach message. Registered the new
shared module in `web/shared/package.json` (boundary gate).
Verified: `make ci-scoped` PASS (after registering the export — the
boundary gate caught the miss); node 19/19 incl. new claim tests; `make
web` ok; scratch `paste-followup` live: files+text paste on a plain shell
→ prevented + toast, no caption leak in the terminal, overlayAudit ok.
Blind spots: bar-open on a live CLI pane still not run (no authenticated
CLI in scratch); real keydown+granted-permission double path reasoned,
not exercised.
visual-review: PASS (paste-followup-toast.png read; card 5/5; audit ok)
Not done: none. Mobile paperclip flow unchanged (out of scope).
Merge: fast-forward ready.
