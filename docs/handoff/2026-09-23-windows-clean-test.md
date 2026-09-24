# 2026-09-23 — feat/windows-clean-test: the layer behind the compile error

The full-matrix run that verified the Windows compile fix got one step further and failed in `cmd/picode`: `TestToolPathWidensLikeTheTerminal` writes `#!/bin/sh` fakes and drives systemd drop-ins, unit files and `/mnt/c` PATH shapes — none of which exist on Windows — while the step runs that package there for its host-boundary parts.
It now skips on Windows with that reason, rather than being made platform-shaped: the fixtures *are* the platform, and the CLI's Windows story is WSL, where those sources do exist.
`git log` puts it with `clean: find cleanup tools the way the person's terminal would` and `wsl: pay the three Management debts` — recent work whose shape no Windows leg had run since it landed (the previous runs died at vet, one layer earlier).
Verified: `go test ./cmd/picode` green on Linux, `GOOS=windows go vet ./cmd/picode` clean, `make ci-scoped` PASS (1 path).
Blind spot: only a Windows runner executes it, so the skip is proved by the next dispatch — which is also the run that can turn the Windows leg green for the first time.
