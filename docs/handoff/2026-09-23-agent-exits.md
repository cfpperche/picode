# 2026-09-23 — feat/agent-exits: removing an agent keeps an exit record

Shipped: ADR-0194, option A of `docs/benchmarks/2026-09-23-agent-exit-feedback.md`.
- **Exit record.** Every removal a person makes — `DELETE /api/agents/{id}`, a workspace's removal, removing an agent's terminal — writes an `agent_exits` row (migration 069) in the same transaction as the delete; the agent route answers 200 with the exit.
- **Question.** The removal dialog, desktop and phone, asks an optional outcome and, when it went badly, the reasons (`web/shared/domain/agentExit.js`, `web/shared/styles/agent-exit.css`).
- **Turns.** `agents` counts turns from three edges: managed Pi `agent_start`, the Pi TUI busy flip, a guest CLI working after idle.
- **Reading.** `/api/agent-exits` (list, summary, export, prefs, label, delete, undo); the `#/outcomes` page (user menu ▸ Outcomes) and Agent outcomes on the dashboard. Docs: `docs/architecture/agent-exits.md`, `docs-site/guide/outcomes.md`.

Verified: `make ci-scoped` and `make close` PASS (fmt, vet, hooks, go[22], test-js, build, docs), exit 0. New tests: `internal/store/agent_exits_test.go`, `internal/server/agent_exits_test.go`, `internal/rpc/turns_test.go`, `web/shared/domain/agentExit.test.js`.
Visual review on a scratch instance (`qa-scratch exits`). First pass FAIL: Outcomes had no column headers, raw CLI ids and a raw `changes` value, and the dialog grew both ways; measuring found the desktop entry not importing the shared exit stylesheet and the exit width beating the bottom sheet's full width on the phone and narrow windows. All fixed before the second pass.
visual-review: PASS (second pass; overlayAudit ok on every dialog; card 5/5; shots in `var/screenshots/agent-exits/`, not committed)
Blind spot: a browser against one scratch instance at 1440, 700 and 390 px wide; not run inside the Windows desktop shell or on a real phone.
Seen in review, pre-existing (not this branch): the removal toast's close button sits against Undo; the dark-theme Cancel border is nearly invisible; the Pi icon renders as a white square in dark mode; the phone's Remove text is about 4.3:1 on its light background.
Not done / debts: option B next, C and D later, and five accepted debts (no phone Outcomes screen, no cost per agent, …) — all in `docs/handoff/open/agent-exits.md`, not repeated here.
Merge: fast-forward ready.
