# 2026-09-23 — feat/attach-delivery: Steer / Follow-up in the attach composer while the CLI works

Shipped: ADR-0206 (accepted; amends ADR-0089 and ADR-0160 Fatia F). The attach composer (terminal message bar,
desktop + phone sheet) offers Steer / Follow-up mid-turn through per-CLI adapters in `internal/server/term_delivery.go`,
measured live in all nine CLIs (Pi Enter/Alt+Enter, Omp Enter//queue, Hermes /steer//queue, Muse Enter/Alt+Enter,
Codex Enter/Tab; Claude Code and OpenCode steer only; Antigravity and Grok follow-up only). `needs-you` and drafts still
refused; automations stay prompt-only; receipt `queued` only when the pane renders the message, else `unconfirmed`.
Pi agents pass the mode to the receiver as `deliverAs`. New `GET /api/terminals/{id}/prompt` and
`/api/agents/{id}/prompt`. SearchCombo: opt-in `closeFocus`, a list without search takes focus on open, the current
value is barred (not tinted) — touches every desktop SearchCombo.
Verified: Go tests incl. a real isolated-tmux mid-turn test (queued vs unconfirmed); JS domain tests; make ci-scoped
green. Blind spot: no real working CLI behind the finished door — the receipt was proven on a fake pane only.
visual-review: PASS (desktop + mobile, scratch instance, inert Codex terminal; fixed head alignment, dark selected
marker, focus return, list focus ring).
Not done / debts: see docs/handoff/open/attach-delivery.md. Merge: fast-forward ready after merging main.

## Next up

- Owner live-check after deploy: send Steer and Follow-up into a working Claude Code, Codex and Hermes

## Debts

- No live run of the finished door against a real working CLI (docs/handoff/open/attach-delivery.md)
- Hermes /queue receipt reads unconfirmed; Grok steer waits on ui.follow_up_behavior (docs/handoff/open/attach-delivery.md)
- Old Pi receiver ignores deliverAs; Omp /queue payload flattened to one line (docs/handoff/open/attach-delivery.md)
