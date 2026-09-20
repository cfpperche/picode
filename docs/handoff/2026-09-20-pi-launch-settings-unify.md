# 2026-09-20 — pi-launch-settings-unify: unify agent launch settings

Shipped: Pi agent Launch settings opens the bound terminal editor; Settings keeps scoped agent configuration. Unbound Pi agents expose Settings only. Browser and mobile resolve workspace/free ownership, fix the bound CLI identity and hide reserved Pi quick flags while preserving standalone/default controls.

Verified: 54 targeted JS tests and targeted server launch tests passed; `make ci-scoped` and `make ci-docs` passed. Scratch browser checks covered menu routing, fixed CLI identity, reserved-flag rejection with draft retention, and the Settings-link discard guard.

visual-review: PASS — 20 screenshots read in a subagent: light/dark, narrow/mobile, bound/free Pi, other CLIs, standalone/default controls and missing/empty states; overlay audit passed.

Not done: no deployment, real-model or physical-device acceptance; UI verification used a scratch browser.

Merge: prepared for fast-forward after branch closing verification.
