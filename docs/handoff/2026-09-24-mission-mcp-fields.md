# Mission MCP mutation field validation

- Implemented actionable validation for missing `generation`, `expectedVersion`, and `requestId` on mutation actions before daemon writes.
- Preserved `show` and `context` reads, valid agent writes, stale-generation refusal, and existing idempotent retry behavior.
- Focused tests cover missing fields, valid agent writes, and stale-generation refusal.
- Updated Missions architecture, the open Missions debt, and `docs/changelog.d/mission-mcp-fields.md`.
- Implementation: `ad091c9b2`; main catch-up merge: `909b8a26a`.
- `make ci-scoped` PASS: fmt, vet, hooks, Go (5), living-docs.
- `make close` PASS after merging main; it reused the green scoped gates.

## Next up

- Owner accepted mission `mission_6S3Q5CSTFZIZ4R2MNPTSQSJZCE` at version 13 after stopping Codex; reviewed revision `b1a004710`.
- Landed `b1a004710` on `main`; full `make ci` PASS, including 81 Go packages. Branch and worktree removed.
- Accepted record retained under `var/qa/missions-pilot/`. After cleanup, the completed mission again reports `the working folder is unavailable`; durable evidence/history remains open.
- No deployment was performed; deployment remains the owner's action.
