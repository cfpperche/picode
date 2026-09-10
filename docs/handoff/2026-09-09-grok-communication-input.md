# 2026-09-09 — grok-communication-input: recognize Grok's bordered composer

Implemented: exact Grok 1.0.25 composer recognition and pointer-width preflight before a durable delivery claim; legacy input recognition remains supported.
Verified: review found no safety blockers; 13 captured-frame rows, three native post-paste assertions and 10 width combinations cover drafts, mode, resize, continuation, permission/footer changes and exact pasted text.
Gates: final `make ci-scoped` PASS (fmt, vet, hooks, four Go packages, eight paths); `TestPeerGrok*` PASS; `make close` and integration `make ci` pending.
Native acceptance: PASS — managed Pi → Grok 1.0.25 automatic prompt, native read, correlated reply and both acknowledgments; `check_LCDSXBRMOWZ4XEDP5ULFVGZNSK` passed (`var/qa/grok-communication-input/native-result.json`, `native-history.json`).
visual-review: n/a — no UI changes.
Not done / debts: complete branch/integration gates; deployment remains owner-controlled. The first pre-fix native check expired and was not relabeled as passing.
Merge: uncommitted branch based on `b30d44c8`; fast-forward ancestry confirmed by `make close-summary`.
