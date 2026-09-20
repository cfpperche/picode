# 2026-09-20 — pi-launch-settings-unify: unify agent launch settings

Shipped: Pi agent Launch settings opens the bound terminal editor; Settings keeps scoped agent configuration. Unbound Pi agents expose Settings only. Browser and mobile resolve workspace/free ownership, fix the bound CLI identity and hide reserved Pi quick flags while preserving standalone/default controls.

Verified: 54 targeted JS tests and targeted server launch tests passed; `make close` passed fmt, vet, hooks, JS/package tests, build and docs. Scratch checks covered menu routing, fixed CLI identity, reserved-flag rejection with draft retention and Settings-link discard guards. Save/reopen retained an environment override; runtime mode, terminal identity, start time and applied fingerprint stayed unchanged while launchPending became true.

visual-review: PASS — 20 screenshots read in a subagent: light/dark, narrow/mobile, bound/free Pi, other CLIs, standalone/default controls and missing/empty states; overlay audit passed.

Not done: no deployment, real-model or physical-device acceptance; UI verification used a scratch browser.

Merge: fast-forward ready after merging current main into the feature branch.
