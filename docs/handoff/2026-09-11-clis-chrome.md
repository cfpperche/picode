# 2026-09-11 — feat/clis-chrome: Agent CLIs from the sidebar header

Shipped: desktop Agent CLIs is the last sidebar-header control (button,
not a rail tab) — Lucide Boxes, `go("clis")` → `#/clis`. The current rail
tab stays selected; `aria-current="page"` when the hash is clis. The
package-update dot moved onto that icon. Agent CLIs left desktop user-menu
Tools (Tools = automations, llama, integrations); search still finds it.
Ctrl+K and mobile More are unchanged. Launch Customize is back on the
CLIDefaults “Launch settings” heading (desktop + mobile), off the
Launch/Terminals/Sessions/Providers tab row. Fragment
`docs/changelog.d/clis-chrome.md`. Docs-site `agent-clis.md`,
`packages.md`, `terminal-status.md` and `docs/architecture/routes.md`.

Verified: `make close` PASS (ci-scoped fmt, vet, hooks, test-js, build,
docs). Visual-review PASS on scratch http://localhost:8472 session
clis-chrome: sidebar header, click icon → `#/clis` with Workspaces still
selected and Customize on Launch settings, user-menu Tools without Agent
CLIs, `__picodeOverlayAudit()` ok. Screenshots `var/screenshots/clis-chrome/`
(not committed).
visual-review: PASS

Not done / debts: Agent CLIs still not in docs-shots SURFACE_PROFILES
(existing Canvas/CLIs gap). Mobile More still lists Agent CLIs (intentional).

Merge: fast-forward ready (main merged in as 40e6e70c).
