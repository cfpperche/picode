# 2026-09-24 — feat/attach-delivery-fixes: Hermes accepted receipt, Omp Ctrl+Q and quiet stop
Shipped (467fd945f, ADR-0206 second 2026-09-24 amendment): Hermes `/queue`
renders nothing until the turn ends (re-measured live on Hermes 0.21.4), so its
follow-up receipt is a new value `accepted` (input row held the pasted command,
then let it go after Enter); the composer shows an info toast "Queued. The CLI
shows it when the current turn ends." (TermAttachBar + TermAttachSheet via
`deliveryInfo` in `web/shared/domain/deliveryModes.js`). Omp follow-up is
Ctrl+Q instead of `/queue` (measured on Omp 18.2.11: a 3-line numbered list
queued as one entry, newlines kept, ran as one follow-up); trade: it depends on
Omp's default keybinding. Stop and send on Omp stopped before first output
counts the `esc …` working row gone for two reads in a row as the stop (row
vanishes within 0.7 s, no stop line). Files: `internal/server/term_delivery.go`;
docs: ADR-0206 + index row, study, `docs/architecture/cli-terminal-launch.md`,
docs-site `guide/agent-clis.md`, changelog fragment; three debts paid in
`docs/handoff/open/attach-delivery.md`.
Verified: new rows in TestAttachDeliveryMidTurnOnTmux (hermes accepted /
unconfirmed) and TestAttachInterruptOnTmux (omp working row gone → sends;
control row without a busy pattern → not-stopped); make ci-scoped PASS.
Blind spot: measured by probe outside PiCode plus fake-pane tests; none of the
three changes was run through PiCode against a real CLI.
visual-review: UNVERIFIED (info toast copy not screenshot-reviewed; it reuses
the existing toast.info, no new component).
Not done / debts: in `docs/handoff/open/attach-delivery.md` (owner live-check,
Grok steer via ui.follow_up_behavior, older Pi receiver ignoring deliverAs,
Pi receiver interrupt untested live, no live run of the door).
Merge: fast-forward ready.
