# Terminal activity: hooks and the process tree

## Next

- Watch the live "Running" status for a week of real use (ADR-0212). If a helper that detaches itself stays Running forever, add a row to the chain or launcher tables in `internal/server/term_command.go`.

## Debts

- [x] Pane titles re-measured and `docs/architecture/direct-session-communication.md` corrected (feat/terminal-activity-2).
- [ ] Muse and Antigravity have no runtime lease, so a `!` command in either still shows Open (ADR-0212). The pane walk would need a root without a lease.
- [ ] Omp runs shell builtins (`!sleep 30`) inside its own process, so the tree walk cannot see them. Its `user_bash` event only marks the start: omp 18.2 emits no end event and runs the command after the handler returns. Reporting working there would leave the agent stuck. External commands (`make`, `npm`) are already seen. Revisit if omp adds an end event.
- [x] The 280px sidebar now keeps the spinner on busy pills and drops only the age (feat/terminal-activity-2).
