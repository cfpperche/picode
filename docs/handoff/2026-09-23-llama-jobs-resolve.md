# 2026-09-23 — llama-jobs-resolve: unknown jobs can end (ADR-0083/0090 amended)

Shipped (branch 3 of today's adversarial llama.cpp review, owner-approved amendments): `abandoned` state + Abandon
(Activity, desktop + mobile, confirm; releases the model, sends nothing); jobs on PiCode's own stopped service end
`interrupted` (llamaservice.Stopped → llamajob; Preview settles them before its guard, outside its lock); restarted
unload still loaded → interrupted; ObserveDownload hashes outside the lock and re-checks size/mtime; start-time sweep
of `.install-*` and incomplete unrecorded `release-*` (complete ones preserved); a failed baseline save no longer
answers 500 for a running download; catalog `fresh` no longer joins an older probe.
Verified: `make ci-scoped` PASS; new tests pass under `-race` (abandon, stopped-service interruption, restarted
unload, the review's blocked-start reproduction, sweep, abandon route, fresh); visual-review PASS on scratch
(buttons aligned, confirm dialog overlayAudit ok, abandoned card on desktop and 390px).

## Debts

- Cleanup preview/execute still hash selected cache files under the service lock (`service.go` cleanup path)
- Supervisor kill semantics after the leader is reaped (PID reuse window, router children on SIGKILL) — unverified
