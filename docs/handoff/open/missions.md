# Missions

Plan: docs/plans/missions.md
Verification: docs/plans/missions-verification.md
Integration boundary: [Delivery flow](delivery-flow.md) owns the queue and its provider/local modes.

Status: approved M0–M3 implemented under ADRs 0199–0200; one controlled Codex → Claude Code native pilot exercised transfer and review. The owner reported deployment; the public Missions guide now covers the lifecycle and controls.

## Next

- Pilots 1–2/5 completed: `mission_WUFG3EEGTSC3LAAM54DRTWPGJF` used dedicated Omp/manual context paste and a stale-main review; `mission_N7K4X3IA26T6AQC66EEW27LWSO` used Codex → Claude Code transfer and a copy correction after review. Both were accepted, landed and passed full CI. Run three more real missions covering interruption and a physical-phone answer; compare observed context-recovery time with five baseline tasks before claiming a gain. Accepted records are under `var/qa/missions-pilot/`.
- M4 (separate reviewer) requires separate approval. For M5 (dependent stages and unattended execution), the owner must decide in a new ADR whether its integration step consumes Delivery's declared queue/provider boundary under ADR-0186 instead of adding a second integration executor; check current runner capability evidence first.

## Debts

- [ ] Accepted `kind: file` evidence is only a relative path and digest, not a retained artifact. In pilot 1, six screenshots lived under `.worktrees/mission-dev-flow-outline/var/screenshots/`; after normal cleanup, the completed history remained but the original paths no longer resolved. The completed note-only pilot 2 also returned observation `available: false` / `the working folder is unavailable` after cleanup. Verified copies of pilot 1 screenshots and both accepted records are under `var/qa/missions-pilot/`. Decide a durable evidence policy (snapshot bytes or a managed artifact location) and show completed history without an alarming missing-worktree observation.
- [ ] Mission MCP writes without `generation` pass the tool schema but fail with the generic `this action is not allowed for this assignment`. In pilot 2, Codex acknowledged generation 1, then its `report` omitted that field and was rejected; retry with `generation: 1` succeeded. Require or explicitly validate `generation`, `expectedVersion` and `requestId` for writes, and name the missing field in the response; preserve stale-generation refusal semantics.
- [ ] Validate the complete Missions flow on a physical phone and in the Windows native shell; browser emulation does not establish either environment.
- [ ] Exercise native submission crash points beyond the current durable-receipt/no-retry fixtures and completed-mission daemon restart; the controlled pilot did not inject crashes at every external-effect boundary.
- [ ] Measure live provider behavior beyond Codex 0.156.1 and Claude Code 2.1.280, including Pi managed/interactive paths; common adapter and fixture coverage is not provider acceptance.
