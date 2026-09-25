# Managed principals → agents (ADR-0160)

Plan: `docs/plans/cli-as-agents.md`

## Next

- Owner: accept or amend ADR-0217 (written as proposed).
- One needs-you → Inbox → answer round trip on a guest `start` run (Grok and Hermes ran live 2026-09-25: 2/2 each).

- [x] Fatia F landed 2026-09-20 (`284eaac7e`): message automations and the pane/graph ask reach CLI agents through the prompt door with receipts. Start runs on guest CLIs followed on 2026-09-25 (ADR-0217, `feat/cli-automation-start`).
- ∞ (deferred): managed mode per CLI, until ADR-0091 is re-measured.

## Debts
- [ ] An unattended Claude Code `start` run fails as unrecognized if Claude's "Teach auto mode about your environment?" menu is on screen at start (ADR-0217).
- [x] `start` runs on guest CLIs do not read the final message back; the Inbox result only points at the session (ADR-0217). — paid 2026-09-25 (`feat/cli-run-final-message`, `cliRunFinal`).
- [x] OpenCode, Grok and Hermes `start` runs have no measured cost (`climetrics.Metered`), so the cost cap does not stop them; the editor and the runs table say so. OpenCode keeps sessions in SQLite, Grok and Hermes have no priced session file yet. — paid 2026-09-25 (`feat/cli-cost-meters`, `climetrics.MeterSession`).
- [ ] A turn recorded at $0 on a model the price table lacks (GLM via OpenCode/Hermes on the owner's Z.AI plan) is `unpriced` and never trips a `start` run's cost cap; pricing GLM needs a price source for it (feat/cli-cost-meters).
- [ ] Antigravity (composer unmeasured) and Muse (hooks not attributable to a terminal, no end-of-turn signal) cannot take `start` runs (ADR-0217).
- [ ] Omp's composer reader is scoped to unattended senders (`unattendedReaderCLI`). Moving it into `doorReaderCLI` would change Omp's fork (task pasted after start instead of a launch argument), mission dispatch and attach delivery; each needs a live measurement first.
