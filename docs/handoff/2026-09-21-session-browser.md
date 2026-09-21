# 2026-09-21 — feat/session-browser: an agent drives the browser split bound to its own session

Shipped: ADR-0172 — an identified caller (agent id, else `term:<id>`) opens the work
browser's split beside its own session and drives it: `open`, `navigate`, `click`, `type`,
`press` (plus `evaluate`, `snapshot`, `screenshot`, `events`) at act on any http(s) URL,
with no stored grant. `open` asks the desktop page to launch the split and select it; commands
never fall through to another tab. Closing the split ends the binding — the next call opens
a new one. A caller with no identity stays read-only on the tab on screen; headless tools
are separate. `internal/browser/session.go` is the new seam; the per-principal grant in
Settings now covers raw CDP only (Developer mode + Full tier unchanged).
`.omp/settings.json` loads the same `packages/pi-browser` extension for every omp in this
workspace, with no `-e`.
Verified: `ci-scoped: PASS (full)`; `TestSessionDriveDecisionTable` and the `Prepare`
refusals in `internal/browser/session_test.go`; `sessionBrowser.test.js` for the host-tab
binding; and measured here, omp's tool list contains `browser` and a call reached the daemon —
its legacy-pi shims resolve `@earendil-works/pi-coding-agent` and `pi-tui`, so the same
extension loads under omp. Blind spot: never run inside the Windows desktop shell, so no live
tool call opening the split was observed — that run is the owner's.
visual-review: n/a
Merge: not fast-forward — main has moved one commit ahead of the branch (fe346362).

## Debts

- A live Windows acceptance of a tool call opening the split is pending; the durable item is
  owned by `docs/handoff/open/work-browser-tabs.md`.
