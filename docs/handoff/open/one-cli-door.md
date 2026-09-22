# One door for agent CLIs

Decision: ADR-0184. Plan: `docs/plans/one-cli-door.md` (four branches, in order).

## Next

- `feat/cli-door-agents`: every user-facing launch (New terminal, profile, palette, Sessions resume, handoff) creates an agent.
- `feat/cli-door-migrate`, then `feat/cli-door-close` (route removed, sign-in terminals internal and reaped), then `feat/cli-door-adopt` (Make agent from a PiCode shell).

## Debts
