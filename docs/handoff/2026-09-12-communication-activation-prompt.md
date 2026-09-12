# 2026-09-12 — communication-activation-prompt: add explicit CLI activation

Shipped: Agent CLIs Messages now offers “Activate now” for an enabled, open terminal whose native identity is still unobserved. The endpoint reuses the guarded native input path, requires an idle empty composer and current session, sends once, and reports that one model turn may be consumed.

Verified: Go workspace participation test, shared participant tests, browser build and `make ci-scoped` passed. Scratch browser fixture covered the activation row and confirmation dialog; overlay audit was ok.

visual-review: PASS (activation row and confirmation dialog; card 5/5).

Not done / debts: real six-CLI activation cost and provider behavior remain an owner-run acceptance; the existing non-Linux and native first-event limits remain documented in `docs/handoff/open/communication.md`.

Merge: fast-forward ready as `b2b92600`.

## Next up

- Owner fast-forwards this branch to main, runs full main CI, then decides whether to deploy.

## Debts

- No automatic activation or retry was added.
