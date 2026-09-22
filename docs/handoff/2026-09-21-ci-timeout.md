# 2026-09-21 — feat/ci-timeout: sharding was the wrong fix, and the revert that "restored" tmux proves it

One commit. Two wrong turns worth writing down, because both came from the same mistake.

**The fact that holds:** `internal/server` and `internal/store` each pass `go test`'s 10-minute default in one process under `-race`. Store alone measures 557s unsharded here — 43 seconds under the default, on a machine faster than any hosted runner. That is a timeout problem.

**Wrong turn one:** sharding. It looked right locally (415s and 484s longest shards) because this machine has cores to spare; a hosted runner has two. The run after it landed had *all eight* shards hit exactly 600s — the same wall, reached by contention instead of by length.

**Wrong turn two:** `internal/tmux` went red on ubuntu the same run, and I blamed a namespace change in `feat/ci-env-tests` and reverted it. The revert did not fix it: `TestPaneCwdIsRightFromTheFirstInstant` failed again with `no current target`. The real cause was the sharding — every extra concurrent process starts its own tmux server, and on two cores they stop coming up. The revert was still the right change on its own merits (it removed a fix production never needed), but the reasoning I gave for it was wrong.

CI now runs one process per package with `-timeout 30m`; `make ci` still shards locally.

The lesson, in one line: **a local measurement is evidence about the local machine.** Both errors were confident extrapolations from a 16-core box to a 2-core runner, and both cost a full matrix run to discover.

Verified: `make close` green, full scope. visual-review: n/a. Nothing deployed.

## Next up

- The matrix has not passed yet on any commit. The next run tells whether the timeout is the last blocker or only the last one *I* found; `TestGuestLaneCommandsMatchTheEngineByteForByte` (another session's) was failing on ubuntu too.
