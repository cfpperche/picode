# Missions

Plan: docs/plans/missions.md
Verification: docs/plans/missions-verification.md

Status: approved M0–M3 implemented under ADRs 0199–0200; one controlled Codex → Claude Code native pilot exercised transfer and review. The owner reported deployment; the public Missions guide now covers the lifecycle and controls.

## Next

- Run five real owner-workflow missions, including a cross-CLI transfer, interruption, rejected review and phone answer; record outcomes and compare time spent recovering context with five baseline tasks.
- M4 (separate reviewer) and M5 (dependent stages and unattended execution) require separate approval and current runner capability evidence before implementation.

## Debts

- [ ] Validate the complete Missions flow on a physical phone and in the Windows native shell; browser emulation does not establish either environment.
- [ ] Exercise native submission crash points beyond the current durable-receipt/no-retry fixtures and completed-mission daemon restart; the controlled pilot did not inject crashes at every external-effect boundary.
- [ ] Measure live provider behavior beyond Codex 0.156.1 and Claude Code 2.1.280, including Pi managed/interactive paths; common adapter and fixture coverage is not provider acceptance.
