# 2026-09-13 — feat/handoff-brief-413: Brief on too-large Continue

Shipped: sessions over 64 MB no longer 413 the preview. Native is dropped; Brief uses the last 64 MB of the JSONL. Continue enables. Native execute of an oversize source is still 413.
Verified: `make close` PASS (ci-scoped go[6], test-js, build, docs); `TestHandoffOversizeFallsBackToBrief`, `TestScanSessionTailReadsTheRecentEnd`.
visual-review: n/a (same dialog; warning list instead of a blocking 413)
Not done / debts: no live screenshot of the too-large Codex → Pi path after deploy.
Merge: fast-forward ready (`git merge --ff-only feat/handoff-brief-413 && make ci`). Deploy is the owner's call.
