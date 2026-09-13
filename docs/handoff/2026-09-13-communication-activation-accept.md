# 2026-09-13 — communication-activation-accept: measure Activate now on six CLIs

Shipped: acceptance table in `docs/plans/communication-activation-accept.md`. Six scratch TUIs; no Activate now button; every POST 409; no model turn billed by activation. Pi/Grok connected without it; Claude waits for a saved first message; Codex/Hermes/OpenCode have no sessionKey on welcome.
Verified: API snapshots and pane captures in `var/qa/activation-accept/`; Messages screenshot read; six exact scratch terminals removed via `qa-scratch stop`.
visual-review: n/a (measurement, no UI change)
Not done / debts: cold-start widen is the owner's call; remote mode not exercised.
Merge: fast-forward ready after this close.
