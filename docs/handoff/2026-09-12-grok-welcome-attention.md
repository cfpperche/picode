# 2026-09-12 — grok-welcome-attention: recognize the empty Grok welcome composer

Changed: first-message attention recognizes Grok's exact empty `[stable]` welcome footer; post-paste submission still requires the native Enter footer and exact pointer.
Cause: the CognixSE message was stored, but the previously unsupported welcome frame prevented recipient notification; earlier warmed-up tests missed this state.
Verified: focused `TestPeerGrok*`, `make ci-scoped`, `make close` and adversarial code review passed; gate log: `/tmp/picode-grok-welcome-close.log`.
Native: fresh Pi-to-Grok test `check_7A43BKRBORLDW67SLH26CUV6WK` passed without a preparatory prompt in either terminal; reply identity, `reply_to` and both ACKs verified.
visual-review: PASS on scratch desktop/mobile; screenshots read by the visual reviewer and overlay audit passed.
Evidence: `var/qa/grok-welcome-attention/native-result.json`; `var/screenshots/grok-welcome-attention/` (local, uncommitted).
Not deployed; production CognixSE sessions remain untouched.
Cleanup: exact task-owned fixtures and processes removed; `var/qa/grok-welcome-attention/cleanup.json` records no residuals.
Merge: implementation `2af37430`; latest main merged, fast-forward ready. Root integration runs full main CI.
Plan: `docs/plans/grok-welcome-attention.md`.
