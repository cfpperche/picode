# One door for agent CLIs

Decision: ADR-0184. Plan: `docs/plans/one-cli-door.md`. All four branches landed on 2026-09-22; what remains is below.

## Next

## Debts

- [ ] Migration 067 turns an unbound Pi terminal into a Pi agent as is: launch args with a flag a Pi agent reserves (`--session`, `--model`, …) make its next start refuse with "Configure … through this Pi agent's settings". None existed on the owner's instance (2026-09-22); fix by editing the agent's launch settings.
