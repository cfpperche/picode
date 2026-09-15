# 2026-09-15 — activity-bench: Orca/Happy activity for muse and agy
Research slice at owner request: how Orca and other benchmarks do activity
for the muse and agy CLIs, input to Fatia 5. Delivered
`docs/benchmarks/2026-09-15-agy-title-statusline-muse-msp.md`.
Orca: agy allowlisted with argv plus window-title patterns and tui-idle on
OSC title transitions; NO Muse matcher (unlisted agents invisible by
design). Happy agy backend derives phase from its own `--print` child plus
SSE hints, no CLI channel. Vibe Kanban lists neither.
Measured locally: neither TUI ever sets the terminal title (frozen, zero
OSC across boot/turns/idle) — Orca tier-1 has nothing to read; the
dynamic-Antigravity-titles claim does not hold for agy 1.2.3 defaults.
Found live: agy title/statusLine settings blocks fire a command per state
change with agent_state idle/thinking/working/tool_use/initializing
(observed authenticating/idle/working/idle); payload transcript_path is
stale (IDE dir), resolve brain from conversation_id.
Muse: zero push surface in-binary; leads are muse serve MSP
(turn/started/completed, session/statusChanged, view/subscribe per offline
schema) and session-message ingress (currently closed). Six tiny agy turns
spent on probes, all fake conversations plus cache entries removed,
settings.json restored byte-identical, probe tmux server killed by exact
session, owner agy untouched.
visual-review: N/A (docs-only; verification is live PTY measurement)
