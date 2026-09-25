# 2026-09-25 — feat/qa-scripts-fix: the QA scripts pass end to end again
Owner: fix the QA scripts first. Commits: 4d4311b01, 07cfabee4, fce9f3ac0, 2c36dbca1, 8556fe814, f3110d269, b672b8271, d6ed0177a.
Scripts (all run live, all PASS): qa-cli-settings (37 checks), qa-cli-settings-recovery (13), qa-cli-packages (22),
qa-cli-providers (rewritten as a smoke test of the one-pane Providers, desktop+mobile), qa-llama (desktop+mobile;
captures now go to var/screenshots/llama, not frozen docs/screenshots), qa-mobile-settings, qa-cli-connectors.
Drift fixed: "This machine"→"Global", scope= words, #w-steer→#w-steeringMode (two absence checks passed vacuously),
untrusted layer is a notice, pane follows the selected terminal's workspace (ADR-0179), key-row density ceiling
96→112 (two-chord row with a warn note is 106px), .settings-layer-body, .key-act, .pi-settings-fields, /browser/
desktop layout (ADR-0122), removed CLI picker, Marketplace tab replacing the Add connector dialog, remount races.
Product fixes: CliPackages keyed the pane by scope, so a package config page (pi-roles) lost its unsaved per-layer
draft on a layer switch (desktop+mobile); mobile Connectors Marketplace toolbar had a 44px layer switcher beside a
36px button (More-page padding) — CSS fix.
How to run: `go build -o bin/picode-docs-fixture ./cmd/picode-docs-fixture && ./bin/picode-docs-fixture -addr
127.0.0.1:18861`, restarted between scripts (they leave state); qa-cli-connectors uses `scripts/qa-scratch.sh
start/seed`; qa-llama needs PICODE_PLAYWRIGHT_MODULE=<playwright>/index.mjs and PICODE_QA_CHROME=/opt/google/chrome/chrome.
Verified: `make ci-scoped` PASS; scripts on the fixture / scratch only. Blind spot: not run inside the Windows shell.
visual-review: PASS (desktop-roles-draft.png, desktop-roles-agent.png, qafix-mobile-marketplace-row.png; 36/36, overlay audit ok).
Observed, not changed (owner's call): the unified Packages pane no longer blocks a machine-scope install on a missing or
mismatched agentId; a deleted free agent's agent-scope install is refused by the server (404), nothing lands elsewhere.
Merge: fast-forward ready.

## Debts

- Packages agentId guard, PackageTarget moved-location QA, Providers account-flow QA: docs/handoff/open/legacy-compat.md
