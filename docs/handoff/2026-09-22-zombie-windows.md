# 2026-09-22 — feat/zombie-windows: I broke Windows fixing macOS, and three more macOS fixtures

Two commits, from the matrix that measured `feat/macos-tests`.

**My own regression, first.** That branch put `processZombie`'s portable form in `agent_stop_other.go`, which is `//go:build !linux` — and "not Linux" includes Windows, where `syscall.Kill` does not exist. `go vet ./...` on the Windows job died on `undefined: syscall.Kill`, and a job that had just gone green went red. I ran `GOOS=darwin go vet` and not `GOOS=windows`, having used that exact command earlier the same day for this exact class of mistake. Three files now, one per answer: `/proc` on Linux, signal 0 on other Unix, and on Windows — where the daemon does not run — a process it cannot inspect is reported as not-a-zombie rather than declared dead.

**Three more macOS fixtures.** Two build their own tmux sockets under `t.TempDir()` and failed with `File name too long`, the same ceiling `internal/tmux` already has a helper for; `internal/server` now has its own. The third asserts the Python recorder's writes succeed, and the recorder refuses on any host without `/proc/sys/kernel/random/boot_id` by its own first two lines — it does not use `observationFixture`, so it never inherited that skip.

Verified: `go vet ./...` clean for linux, darwin and windows; the two tmux tests green under a TMPDIR that is both long and a symlink; `make close` green.

The pattern across this whole run, worth naming: every macOS failure was either a real product bug that Linux hid, or a fixture asserting something its platform refuses to do. None was macOS being difficult.

## Next up

- macOS has still never been measured green. The verdict needs `gh workflow run CI --ref main` — a push only runs ubuntu.
