# 2026-09-12 — grok-welcome-attention: recognize the empty Grok welcome composer

Changed: first-message attention recognizes Grok's exact empty `[stable]` welcome footer; post-paste submission still requires the native Enter footer and exact pointer.
Cause: the CognixSE message was stored, but the previously unsupported welcome frame prevented recipient notification; earlier warmed-up tests missed this state.
Verified: focused `TestPeerGrok*`, `make ci-scoped` and adversarial code review passed.
Native: fresh Pi-to-Grok test `check_7A43BKRBORLDW67SLH26CUV6WK` passed without a preparatory prompt in either terminal; reply identity, `reply_to` and both ACKs verified.
visual-review: PASS on scratch desktop/mobile; screenshots read by the visual reviewer and overlay audit passed.
Evidence: `var/qa/grok-welcome-attention/native-result.json`; `var/screenshots/grok-welcome-attention/` (local, uncommitted).
Not deployed; production CognixSE sessions remain untouched.
Merge: pending `make close`, fast-forward and full main CI.
Plan: `docs/plans/grok-welcome-attention.md`.
