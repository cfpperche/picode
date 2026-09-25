# Attach delivery modes (steer / follow-up into a working CLI)

Decision: ADR-0206. Study: `docs/benchmarks/2026-09-23-attach-delivery-modes.md`. Architecture: `docs/architecture/cli-terminal-launch.md`.

## Next

- Backlog (owner, 2026-09-24: everything below waits for a future session; the Hermes/Omp fixes of 1a2c56ab0 are deployed). Owner live-check through PiCode of Steer, Follow-up and Stop and send on a working Claude Code, Codex and Hermes, plus a multi-line Omp follow-up, an Omp stopped right after send, a Pi agent's Stop and send, and the Hermes "Queued…" toast copy (not screenshot-reviewed).
- Backlog: Grok steer (read `ui.follow_up_behavior`), a UI notice for an older Pi receiver that ignores `deliverAs`, and an icon-only Delivery chip on narrow windows (the chip takes 91–151 px of the field).

## Debts

- [ ] No live run of the finished door against a real working CLI: the adapters rest on the probe measurement and the `queued`/`unconfirmed` receipt heuristics were tested only on a fake pane (real isolated tmux, inert program).
- [x] Hermes `/queue` renders the message only at turn end, so its follow-up receipt reads `unconfirmed` even when it worked. Paid 2026-09-24: receipt `accepted` from the input row (ADR-0206 amendment).
- [ ] Grok's steer is not offered until PiCode reads `ui.follow_up_behavior` (the setting that decides what Grok's mid-turn Enter does).
- [ ] An older Pi receiver ignores `deliverAs` (a steer lands as a follow-up) until the Pi process reloads the receiver.
- [x] Omp `/queue` splits a numbered list into several follow-ups, so the payload is flattened to one line for it. Paid 2026-09-24: follow-up is Ctrl+Q, newlines kept.
- [ ] Stop and send has no live run against a real working CLI through PiCode: the stop keys and stop lines rest on the probe measurement (2026-09-24); the end-to-end test drives an inert program in isolated tmux.
- [x] Omp stopped before its first output prints no stop line, so Stop and send answers `not-stopped` there (the user retries). Paid 2026-09-24: the `esc …` working row gone for two reads counts as stopped.
- [ ] Pi's receiver path for Stop and send (`ctx.abort()`, wait for idle, then `sendUserMessage`) is untested live.
