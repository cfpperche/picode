# 2026-09-12 — feat/remove-session-adopt: remove the "From a Pi session" adoption path (ADR-0126)
Shipped: adoption path gone — agents are born only from new sessions (ADR-0126, superseding ADR-0021).
Removed: create-dialog session mode (web/browser + web/mobile CreateForm.jsx, CreateSheet.jsx, App.jsx props/state, session-picker CSS blocks), POST /api/clis/{cli}/sessions/adopt route + handler (internal/server/cli_sessions.go), session.CopyFile (internal/session), TestAdoptPiSessionDecisionTable (replaced by TestAdoptRouteIsGone + writeSessionFixture rename), pi-only adopt guard rows (cli_sessions_test.go).
Kept on purpose: GET /api/clis/{cli}/sessions (sessions-management surface uses it), adoptHome (CLI handoff, ADR-0088), ADR-0053 spawn-time pending-session heal (pointer heal, not a birth path). Already-adopted agents keep working; session files were never modified.
Docs: ADR-0126 accepted; ADR-0021 status line + index row → superseded; docs/design/adopt-pi-session.md marked superseded; docs/architecture/routes.md lost a stale duplicated fragment + the adopt sentence; docs/architecture/agent-manager.md updated; changelog fragment docs/changelog.d/remove-session-adopt.md; OpenAPI regenerated.
Verified: make ci-scoped PASS (fmt,vet,hooks,go,test-js,build); main merged into the branch cleanly, go build OK.
Verified: visual review on qa-scratch instance rsadopt — desktop agent dialog, desktop workspace dialog, mobile bottom sheet all captured and read; __picodeOverlayAudit ok:true everywhere; no "From a Pi session" anywhere.
visual-review: PASS (rsadopt-agent-create.png + rsadopt-ws-create.png + rsadopt-mobile-create.png, overlayAudit ok, card 5/5)
Merge: fast-forward ready.
