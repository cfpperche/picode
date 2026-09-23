# Attach delivery modes (steer / follow-up into a working CLI)

Decision: ADR-0206. Study: `docs/benchmarks/2026-09-23-attach-delivery-modes.md`. Architecture: `docs/architecture/cli-terminal-launch.md`.

## Next

- Interrupt-then-send is not offered (the ADR's rejected-for-now alternative); build it if the owner asks for a "stop and send" mode.

## Debts

- [ ] No live run of the finished door against a real working CLI: the adapters rest on the probe measurement and the `queued`/`unconfirmed` receipt heuristics were tested only on a fake pane (real isolated tmux, inert program).
- [ ] Hermes `/queue` renders the message only at turn end, so its follow-up receipt reads `unconfirmed` even when it worked.
- [ ] Grok's steer is not offered until PiCode reads `ui.follow_up_behavior` (the setting that decides what Grok's mid-turn Enter does).
- [ ] An older Pi receiver ignores `deliverAs` (a steer lands as a follow-up) until the Pi process reloads the receiver.
- [ ] Omp `/queue` splits a numbered list into several follow-ups, so the payload is flattened to one line for it.
