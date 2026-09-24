# 2026-09-24 — attach-interrupt: Stop and send

Shipped: a fourth attach mode, **Stop and send** (`delivery: "interrupt"`),
for all nine CLIs (ADR-0206 amendment 2026-09-24). The door presses the
measured stop key (Esc; OpenCode Esc Esc; Grok and Hermes one Ctrl+C), waits
≤ 3 s for the CLI's stop line or state leaving `working`, then runs the
verified prompt path; unseen stop → 409 `not-stopped` (nothing pasted);
field refilled by the CLI (Muse retract) → 409 `restored`. Never the
composer's default. Pi agents: receiver `ctx.abort()`, wait for idle. The
composer shows "Stopping the agent…" while it waits; popover labels never wrap.
Files: `internal/server/term_delivery.go`, `web/shared/domain/deliveryModes.js`,
`TermAttachBar.jsx`, `TermAttachSheet.jsx`, docs-site `guide/agent-clis.md`.
Verified: Go tests incl. an isolated-tmux end-to-end (stops → sent; never
stops → 409 `not-stopped`, nothing pasted); stop keys measured live on the
nine CLIs by probe. Blind spot: no Stop and send through PiCode against a
real working CLI. `TestCLIRestartPreparationFailureAndWorkspaceCleanup`
flaked once in a `make close` shard (known, see open/terminal.md); rerun green.
visual-review: PASS (desktop + mobile, after fixing label wrap, the in-flight
status and a stale error on retry).
Not done / debts: in `docs/handoff/open/attach-delivery.md`.
Merge: fast-forward ready.

## Next up

- Owner live-check after deploy: Stop and send on Claude Code, Codex, Hermes.
