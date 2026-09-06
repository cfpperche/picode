# 2026-09-06 — feat/ephemeral-loopback-sessions: loopback browser sessions end when the access ends

Shipped: `RevokeStaleLoopbackSessions` + `StaleLoopbackBrowserSessions`
(internal/store/auth.go); a minute sweep in `cmd/picode/main.go` revokes
auto-minted loopback browser sessions (kind=browser, no device id,
ip=127.0.0.1) whose `last_seen_at` is 10 min old, via `RevokeSession`
(one `session.revoked` event per row); daily prune unchanged. ADR-0049
amendment 2026-09-06, architecture `#/devices` row, CHANGELOG [Unreleased].

Verified: `make ci-scoped` PASS (fmt, vet, hooks, 15 Go packages). Decision
table in `TestRevokeStaleLoopbackSessions` (idle/fresh/paired/token/revoked/
expired/unparsable, idempotence, survivor tab); events row in
`TestEveryMutationAppendsAnEvent`.
visual-review: n/a (no UI change — Devices already reacts to `session.revoked`).
Not done / debts: live validation on the deployed daemon (the zombie pile
should self-clear on the first sweep after upgrade); `presence.Label`
headless detection untouched.
Merge: fast-forward ready.
