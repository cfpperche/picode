# 2026-09-21 — feat/ci-env-tests: GitHub CI had been red for three days, and none of it was a product fault

Three commits (01994907..aab617fc). CI on `main` last passed on 09-18; the v0.4.0 release was cut over it. Five failing groups, four fixed with a local reproduction each, one deliberately left alone.

**tmux (all of macOS).** `error connecting to … (File name too long)` reads like tmux and is path length: TMPDIR on a macOS runner is `/var/folders/<2>/<28>/T/` and tmux binds `<dir>/tmux-<uid>/default` past the kernel's `sun_path` limit. The namespace `tmuxtest.Main` creates is the root cause — every test in the package rides it — and `socketDir`/`socketPath` cover fixtures that build their own, failing loudly rather than returning something unbindable. Two `PaneCwd` tests failed for the other half: macOS `/var` is a symlink to `/private/var`, so the pane reports the resolved path and the fixture asserted its own spelling; `samePath` resolves both sides, including two comparisons inside polling loops that would have surfaced as timeouts.

**`TestInboxFlagParsing` (all three runners).** A real product bug, not a fixture one: `picode inbox notify` dialled the daemon before validating flags, so a missing `--title` answered "PiCode is not running" with exit 1. `ask` already had the order right. It passed on every developer machine because one is running — or because the shell sits inside a PiCode terminal and `PICODE_TERM_URL` is set.

**Fixture data race and host arch (macOS).** `resolve()` probes ports sixteen at a time by design; the stubs counted those probes with a plain `calls++` from inside them. Atomic now. And an install fixture served `picode-linux-amd64` to a path that asks for `LinuxAssetName(runtime.GOARCH)`.

Every fix reproduces on Linux before it is applied: a long TMPDIR, a symlinked TMPDIR, a scrubbed PiCode environment, the old counter under `-race`. The four groups are green under all of those at once.

**Not fixed, on purpose:** `TestStopIdleFencesConversationAndCommands` on ubuntu. 36 rounds under three parallel loops are green here, and no background holder of `commandMu` exists to explain it — see `docs/handoff/open/process.md` for what was ruled out. A bounded retry would have turned the gate green without anyone understanding it.

Verified: `make close` green, scoped. visual-review: n/a — no UI diff. Nothing deployed, nothing pushed.

## Next up

- The next push to `main` is the real verdict: four groups should clear, and `TestStopIdleFencesConversationAndCommands` may still fail on ubuntu.
