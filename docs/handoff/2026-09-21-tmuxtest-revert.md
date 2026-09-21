# 2026-09-21 — feat/tmuxtest-revert: the gate was dying for a reason no test fix could touch

One commit (39fd83b4), from what the runners said about `feat/ci-env-tests` rather than what its local reproductions predicted.

**A regression of my own.** `internal/tmux` had been green on ubuntu and went red. That branch moved `tmuxtest`'s private namespace to `/tmp` unconditionally, justified by an 80-character TMPDIR — more hostile than any real runner. macOS TMPDIR is 49 characters and the namespace socket lands at 92 of the 104 the kernel binds: it always fit. What overflowed on macOS were the deeper sockets the package's own fixtures build, and those were already fixed in the fixtures. The namespace stays on TMPDIR now and falls back only when a socket under it genuinely would not fit — verified against an ordinary TMPDIR, one the length of macOS's, and a pathological one. The lesson is narrow and worth keeping: a synthetic reproduction that is *harsher* than production justifies a fix production does not need, and the fix can cost more than the bug.

**The real blocker.** `go test -race ./...` runs `internal/server` in one process, where the race detector puts it past the 10-minute default timeout: `panic: test timed out after 10m0s`, every run, ubuntu and macOS alike, failing the job whatever else passed. It had been happening since before any of this work and was invisible to a `--- FAIL:` grep — a panic is not a FAIL line. It reproduces locally in exactly 600s. CI now runs the same sharded script `make ci` uses, with `-race` passed through: four shards, longest 448s.

Scoreboard from the verified matrix run before this branch: Windows green, macOS 16 failures down to 5, ubuntu holding `TestStopIdleFencesConversationAndCommands` (still unexplained, `docs/handoff/open/process.md`) plus the two tmux tests this branch restores.

Verified: `make close` green, full scope. visual-review: n/a — no UI diff. Nothing deployed.

## Next up

- The next full matrix (`gh workflow run CI --ref main`; a push only runs ubuntu) is the verdict on the remaining three macOS devserver failures, which no one has diagnosed yet.
