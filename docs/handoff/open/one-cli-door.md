# One door for agent CLIs

Decision: ADR-0184. Plan: `docs/plans/one-cli-door.md` (four branches, in order).

## Next

- `feat/cli-door-close`: remove `POST /api/clis/{cli}/terminals` (move its 17 test fixtures to agents), sign-in terminals internal (`kind`), visible on their card, reaped on credential/exit/idle/boot.
- `feat/cli-door-adopt`: Make agent from a PiCode shell running a catalog CLI (the bind endpoint exists: `POST /api/workspaces/{id}/principals` with `terminalId`).

## Debts

- [ ] Migration 067 turns an unbound Pi terminal into a Pi agent as is: launch args with a flag a Pi agent reserves (`--session`, `--model`, …) make its next start refuse with "Configure … through this Pi agent's settings". None existed on the owner's instance (2026-09-22); fix by editing the agent's launch settings.
