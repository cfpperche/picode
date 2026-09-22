# 2026-09-22 — feat/npm-user-prefix

Origin: the adversarial review of ADR-0179 found two gaps left by feat/runtime-without-pi. First, the Windows installer now installs no CLI, while Agent CLIs' Install runs `npm install -g` as the account. A NodeSource or distro npm has a root-owned `/usr` prefix, so that install fails with EACCES. The deleted pi branch had worked around this. Second, a node older than 22 was no longer upgraded, and Pi needs >=22.19.

Fix: `internal/desktop/stages.go` adds a probe line, `npm config get prefix | grep -q '^/usr' && echo missing:npm-user-prefix` (it has no `$`). ParseProbe also adds `NodeUpgrade` ("node-22") when node is older than 22, and a new `NpmUserPrefixScript` sets the prefix to `~/.local`. `cmd/picode-desktop/stages.go` upgrades node and runs the prefix script as the target account. The adopted-distro confirmation lists both steps, and a prefix-only run is allowed on any distro family.

Tests: `TestRunInstallRuntimeUpgradesAnOldNode` replaces the test that asserted the opposite. The new tests are `TestRunInstallRuntimeSetsTheUserPrefixWithNode` (runs with `-u <account>`), `TestRunInstallRuntimePrefixOnly` and `TestParseProbeNamesOldNodeAndSystemPrefix`. The existing no-`$` script test also checks the new script.

Verified: `go test` of both packages is green, and `make ci-scoped` PASS. The real probe script was also run locally. With nvm node 24 it reported nothing missing. With a fake npm that answers `/usr` plus node v18 it reported `[npm-user-prefix node-22]`.

Not verified: nothing ran on Windows or WSL. That stays the owner's `picode-test` VM debt. Member containers still get the distro npm, recorded as a new debt in `docs/handoff/open/multi-cli-ade.md`.
