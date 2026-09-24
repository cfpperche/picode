# Missions

Plan: docs/plans/missions.md
Verification: docs/plans/missions-verification.md
Integration boundary: [Delivery flow](delivery-flow.md) owns the queue and its provider/local modes.

Status: approved M0–M3 implemented under ADRs 0199–0200; one controlled Codex → Claude Code native pilot exercised transfer and review. The owner reported deployment; the public Missions guide now covers the lifecycle and controls.

## Next

- Pilot 1/5 completed: `mission_WUFG3EEGTSC3LAAM54DRTWPGJF` used a dedicated Omp agent and manual context paste to fix the dev-flow outline; a real stale-main review was rejected, refreshed and accepted. Run four more owner-workflow missions covering cross-CLI transfer, interruption and phone answer; record outcomes and compare observed context-recovery time with five baseline tasks.
- M4 (separate reviewer) requires separate approval. For M5 (dependent stages and unattended execution), the owner must decide in a new ADR whether its integration step consumes Delivery's declared queue/provider boundary under ADR-0186 instead of adding a second integration executor; check current runner capability evidence first.

## Debts

- [ ] Accepted `kind: file` evidence is only a relative path and digest, not a retained artifact. In pilot 1, six screenshots lived under `.worktrees/mission-dev-flow-outline/var/screenshots/`; after normal worktree cleanup, `GET /api/missions/mission_WUFG3EEGTSC3LAAM54DRTWPGJF` kept the completed history but returned observation `available: false` / `the working folder is unavailable`, and the original evidence paths no longer resolved. Verified copies and the accepted record are under `var/qa/missions-pilot/mission_WUFG3EEGTSC3LAAM54DRTWPGJF/`. Decide a durable evidence policy (snapshot bytes or an explicitly managed artifact location) and its retention/security boundary before claiming file evidence remains inspectable after cleanup.
- [ ] Validate the complete Missions flow on a physical phone and in the Windows native shell; browser emulation does not establish either environment.
- [ ] Exercise native submission crash points beyond the current durable-receipt/no-retry fixtures and completed-mission daemon restart; the controlled pilot did not inject crashes at every external-effect boundary.
- [ ] Measure live provider behavior beyond Codex 0.156.1 and Claude Code 2.1.280, including Pi managed/interactive paths; common adapter and fixture coverage is not provider acceptance.
