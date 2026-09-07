# 2026-09-06 — llama-service: owned CPU lifecycle

Implemented: ADR-0090 and migration 034; desktop/mobile `#/llama/service`.
Reviewed install/start/stop/restart/update/rollback, pinned official CPU releases,
durable jobs/feed, process-group supervisor, selected owned-cache cleanup,
conservative configured-agent guards and allowlisted diagnostics.
Light is two threads/4096 context; GPU and model autoload are off.
Verified: real b10809/b10826 scratch install, start, update, rollback, stop and
archive cleanup through API/UI; real failed-candidate recovery in Go.
Race acceptance passed for llamaservice (15.847s) and llamajob (5.239s).
Ownership matrix covers new/existing/untracked/outside/symlink/alias/failed files.
visual-review: PASS; desktop/mobile empty, blocked, error, dark/light and
confirmation captures read; overlayAudit ok after animation settling.
Evidence: `var/screenshots/llama-service{,-states}/`; reproducible QA script.
`make close` PASS; post-merge `make ci` PASS (Go, 966 JS/package tests,
build, docs and Vale). Public captures regenerated and read.
Debts: ARM64 hardware and real model-download ownership ledger acceptance;
GPU unsupported; old release directories retained; creation failure can leave
an empty unadopted folder. Full limits and decision matrix are in the plan.
Merge: fast-forward to `c694fb20`; worktree cleanup follows this record.
Deploy: guarded batch succeeded at 22:54 -03, `0.1.0+c694fb2`; systemd active,
version/API/desktop/mobile HTTP checks passed. No llama service auto-created.
