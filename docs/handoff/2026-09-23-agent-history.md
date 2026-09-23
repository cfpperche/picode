# 2026-09-23 — feat/agent-history: bring back a removed agent

Shipped: ADR-0205 — the `agent_exits` row is the tombstone a removed agent comes back from.
- **API.** `GET /api/agent-history`, `POST /api/agent-history/{id}/restore` and `/forget` (`internal/server/agent_history.go`); migration 072 adds `forgotten_at`.
- **Lookup.** Exits now record `sessions.cwd`, with a lazy Pi session fallback; `internal/clisession/locate.go` (`Locator`) finds each CLI's conversation on disk.
- **Restore.** Same setup, same conversation (`--resume`/equivalent per CLI), into its workspace or another, in its folder. Pi's private session folder moves to the new agent id — found by visual QA (chat was empty after restore).
- **UI.** Desktop page `#/history` (user menu ▸ Agent history, palette); `web/shared/domain/agentHistory.js`.

Verified: `go test` for store, clisession and server history tests, including a Claude Code restore that launches with `--resume cc-1`; `make close` green.
visual-review: PASS after two FAIL rounds (390px name truncation, dead folder select when the folder is missing, empty Pi chat after restore — all fixed); shots `var/screenshots/agent-history-r3-*.png`, not committed.
Blind spot: one scratch instance in a browser; not run inside the Windows desktop shell or on a real phone.
Not done: debts below, owned by `docs/handoff/open/agent-exits.md`. Not deployed (owner's call).
Merge: fast-forward ready.

## Next up

- Phone screen for Agent history (More ▸ Agent history).

## Debts

- 8-second Undo re-points a Pi `sessionPath` without moving the private folder; route Undo through restore (open/agent-exits.md).
- Agent history has no phone screen (open/agent-exits.md).
- Restore mints a new agent id: automations, pins and Canvas edges on the old id stay broken (open/agent-exits.md).
- Exits written before ADR-0205 have no `cwd`; their lookup reads the CLI's whole store, slow on a large Codex tree (open/agent-exits.md).
