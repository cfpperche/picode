# 2026-09-07 — llama-install-cleanup: installation cleanup and setup recovery

Implemented: reviewed older-installation removal on desktop/mobile; current,
rollback and unknown/changed directories retained. Durable creation intent,
atomic no-replace publication, empty-owned-attempt recovery after failure/restart.
Dependency: existing x/sys promoted to direct for Linux RENAME_NOREPLACE;
no new dependency or version. Boundary remains ADR-0090.
Verified: close PASS; llamaservice race suite PASS (17.429s), creation failure/crash and
cleanup decision matrices, partial-deletion restart without replay. Unreadable
proof regression reproduced, fixed and covered; recovery intent is retained.
Real scratch: b10809 -> b10826 -> b10809; desktop cancel preserves selection;
mobile cleanup removes only old installation and clears selection. No model loaded.
visual-review: PASS; empty/error/blocked/review/result, desktop/mobile dark/light;
screenshots read, overlayAudit ok, no browser errors.
Evidence: var/screenshots/llama-installations-final/ (ignored).
Not done: ARM64 hardware acceptance; unrecorded/partially damaged directories
remain protected for inspection. Final main integration gate pending.
Merge: feature merged as 44a548a1; permission follow-up pending integration.
Deployment uses the normal batch.
