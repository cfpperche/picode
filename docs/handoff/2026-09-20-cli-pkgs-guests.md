# 2026-09-20 — cli-pkgs-guests: native packages for the eight guest agent CLIs

Shipped: ADR-0167. `internal/clipkgs` holds one declaration per CLI (argv per verb,
roster, scopes, notes where a verb is absent; capabilities derive from the declaration).
New `/api/cli-packages*`: installs, removals and marketplace fetches are durable jobs
in the ADR-0087 lane carrying a payload. A guest pane in browser and mobile; the
OpenCode config-array splice is surgical and atomic, 409 on an unknown shape. Pi's own
pane is untouched. Touched: `internal/clijob` (payload through Resolve/Start) and
`internal/store/cli_jobs.go` (Payload field, wider action vocabulary); `internal/connectors`
exports LenientJSON and the OpenCode config paths. `docs/changelog.d/cli-pkgs-guests.md`
travelled with the code. At close: 53 files, +8292/-41; scope FULL (GO=1 WEB=1 DOCS=1 METADATA=1).
Verified: `make ci-scoped` and two `make close` runs PASS, after merging `main` twice.
`internal/clipkgs/live_test.go` (`PICODE_PKGS_LIVE=1`) drives every vendor's argv against
the installed binaries (17 fixtures) — headless, sandboxed HOME, no vendor TUI entered.
visual-review: PASS on a scratch instance — desktop `#/clis/{omp,muse,codex,grok}/packages`
(installed rows, empty, vendor's blocked text, marketplace tab, remove confirm) and mobile
390px; `window.__picodeOverlayAudit()` ok on all four desktop routes and on mobile after two
CSS fixes found in the pass (scope-row label/radio alignment; the mobile filter toolbar had
no height recipe). Card 5/5.
Not done: no Update action or availability badge for a guest's plugins; Muse's roster is read
tolerantly because its build refuses every plugin verb; a consent refusal shows the vendor's
words but no copy button; a hand-written `plugin` string cannot be spliced — all in
`docs/handoff/open/packages.md`, and OpenCode writing `<cwd>/.opencode/opencode.json`
(unnamed by the shared MCP path rule) in `docs/handoff/open/connectors-parity.md`.
Merge: 3 ahead, 4 behind, clean at close — merge `main`, rerun `make close`, then fast-forward from the root.
