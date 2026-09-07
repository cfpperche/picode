# 2026-09-07 — llama-ledger-validation: real download and cache cleanup

Found: b10809 omits model paths; a real successful download had no ownership.
Fixed: correlate complete download evidence with new HF snapshot/blob entries,
verify content hashes and retain existing, shared, changed or referenced files.
Cleanup removes the recorded snapshot link with its owned blob.
Verified: real Qwen3-0.6B Q4_0 download, 428,970,080 bytes; model never loaded.
Independent SHA-256 and read-only SQLite checks passed. Running/referenced,
unowned and same-size changed files refused cleanup; exact owned pair deleted.
Regression: 12 HF cache matrix rows; race tests passed (10.584s).
Evidence: reproducible scripts/qa-llama-ledger.py and docs/plans/llama-manager.md;
local JSON/logs in var/qa/llama-ledger/ include the original failure.
visual-review: n/a (backend correction; no UI code changed).
Gates: focused race, make close and post-merge make ci PASS (Go/JS/build/docs).
Debts: ARM64 hardware; old release pruning; first-creation failure cleanup.
Merge: fast-forward to 4ca834fc; unrelated root .gitignore modification preserved.
Deployment: correction awaits the normal batch; installed 0.1.0+dfa9f7b at verification.
Cleanup: scratch routers stopped and disposable model data removed; evidence retained.
