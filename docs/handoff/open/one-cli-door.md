# One door for agent CLIs

Decision: ADR-0184. Plan: `docs/plans/one-cli-door.md`. All four branches landed on 2026-09-22; what remains is below.

## Next

## Debts

- [ ] Migration 067 moves a Pi terminal's pinned conversation or a lone `--session` onto the agent, but other reserved Pi flags in its args (`--model`, `--provider`, …) still make the agent's start refuse ("Configure … through this Pi agent's settings"). None existed on the owner's instance (2026-09-22).
- [ ] The owner's intent is that a Pi agent runs chat and terminal at once; the runtime still refuses chat while its terminal is live ("Close this agent's terminal before starting chat", `startAgentRPC`). Unchanged by ADR-0184; the owner's call whether to lift it.
- [ ] A migrated Omp agent starts with a private `--session-dir`, so its `/resume` picker no longer lists the conversations it had as a bare terminal (explicit Resume still works). No such row on the owner's instance.
- [ ] Sign-in terminals' `terminal.updated` / `terminal.launch` events hit an id the app does not hold, so each one makes the fleet refetch (harmless churn).
- [ ] A CLI that execs into a binary of another name is invisible to the tmux fallback, so Make agent never appears for it (pre-existing; the PiCode PATH wrapper announces real CLIs).
- [ ] The sign-in "CLI exits" and "idle past 15 min" rows are unit-tested (script ends with `exit`; reaper verdicts), not against a live login; the live-session reap test skips without tmux.
