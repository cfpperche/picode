# 2026-09-20 — cli-agents-f: the door gains receipts; automations reach CLI agents
Fatia F (ADR-0160, owner-approved with tachyon prior art reviewed). The
ADR-0089 door no longer pastes blind: for CLIs with a measured input
reader (peer_attention.go — pi, claude-code, codex, grok, hermes,
opencode) delivery is gated (sensor working/needs-you refused; composer
occupied refused) and verified (row reads empty again after Enter; a
lost Enter retried once; never a second paste). The response carries a
`delivery` receipt — verified / unconfirmed(staged|unreadable) /
unverified (no reader, blind paste as before). Automations aimed at a
CLI agent deliver through the door on the bound terminal and the receipt
is the run's outcome (mapDoorOutcome): verified → done, refused →
skipped (named), unconfirmed/unavailable → failed. Graph/pane ask on
non-pi CLI terminals rides the door (fire and forget, no reply file,
`terminal_ask_delivered` provenance); plain shells stay refused
(ADR-0062). ADR-0089 gained the dated amendment; attach surfaces warn on
`unconfirmed`.
Note: F0 (porting tachyon composer profiles) was dropped — PiCode's own
peer_attention.go already carries the measured per-CLI input rules, so
the slice reuses them instead of a second measured table.
Verified: unit table (mapDoorOutcome, no-terminal, not-running), full
server package, scratch visual on the earlier slice; ci-scoped PASS.
Blind spot: verified-receipt on a live claude pane is scratch-e2e only
(peak: needs a real TUI in CI).

## Next up

- Nothing queued in this plan; ∞ (managed mode per CLI) stays deferred
  until ADR-0091 is re-measured.
