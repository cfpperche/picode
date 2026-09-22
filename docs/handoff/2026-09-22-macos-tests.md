# 2026-09-22 — feat/macos-tests: two of the six macOS failures were product bugs, not fixtures

Three commits. The owner froze every other session so a matrix could measure a still `main`; with ubuntu and Windows green, six macOS failures in `internal/server` were the whole remaining gap. A seventh turned up on the way.

**`processZombie` called every live process dead off Linux.** It read `/proc/<pid>/stat` and treated an unreadable file as "gone" — right on Linux, where `/proc` is always mounted, and wrong everywhere else, where it never exists. `devServerProcessGone` feeds Stop and the hide sweep, so on macOS a running dev server answered `stopped: true` and the Servers panel's hidden list emptied itself on the next read — the "N hidden" line lying in exactly the way ADR-0151 set out to prevent. Three tests had said so for three releases and were read as macOS being macOS. Now split by platform, the way `platformProcessStartToken` beside it already was; signal 0 is the portable liveness test.

**`previewWithin` compared a canonical file against a raw root.** With a symlink above the root the two are never comparable, so every file resolved "outside" the project and a preview answered 404 for its own pages — every path on macOS (`/var` is `/private/var`), and any linked mount or home on Linux. Both sides resolve now; the existing escape cases still refuse. This one was not in the six: it surfaced from running the package under a symlinked TMPDIR, which is the cheapest way to be a macOS runner for twenty minutes.

**Two fixtures assumed their host.** The gitwatch alias test had a third, accidental alias from TMPDIR itself; the native-observation tests never asked `nativeObservationSupported()`, which already reports false off Linux — hence a refused recovery and then a nil-map panic that read like a product fault.

Both product bugs now have the test that was missing: `TestThisProcessIsNotReportedGone` (the running process is alive on every platform) and `TestPreviewRootReachedThroughASymlink`, each proven to fail without its fix.

Verified: `make close` green; `GOOS=darwin go vet ./...` clean on both arches; the package green under a symlinked TMPDIR. visual-review: n/a. Nothing deployed.

## Next up

- macOS has still never been measured green: the verdict on these six (plus the preview one) needs `gh workflow run CI --ref main`, since a push only runs ubuntu.
