# 2026-09-22 — feat/omp-doctor: a Checks card for omp's Settings pane
Shipped: slice 4 of `docs/benchmarks/2026-09-22-omp-helpers-placement.md`. A read-only Checks card at the top of omp's Settings pane, fed by `GET /api/cli-doctor?cli=omp&workspace=` (`internal/clidoctor`; a DOCTOR_CLIS parity test keeps the lists in step). The resolved list is omp's own `omp config list --json` run in the workspace (499 keys, about 0.6 s, no file writes, measured), with a source for each value: This workspace / Global / Not in either file. Credentials are masked by omp's `redacted` flag plus a last-segment rule, pinned against omp's 8 credential keys and harmless look-alikes.
Findings: a quarantined `.broken-*` copy, a file that does not parse, unknown keys (worked out from omp's own listing, so there is no table of retired keys), a key written both flat and nested, a key written only flat, a legacy `.omp/settings.json`, `.pi/` without `.omp/`, a committed `.env`, no approval mode set ("Change it" goes to a new declared Settings row, `tools.approvalMode`), and N omp terminals running (from the presence registry). The card re-reads on the `cli.settings` feed.
Verified: Go and JS tests (decision table in `clidoctor_test.go`); `make ci-scoped` PASS, with OpenAPI regenerated. On qa-scratch with a fixture that triggers every finding: every finding showed; "Change it" focused the approval select; setting `write` on the workspace layer cleared that finding live, and the file read back `tools.approvalMode: write`; the resolved list showed the right sources. Blind spots: the card's mobile layout was not captured, and a live omp TUI was never observed.
visual-review: PASS after two rounds (desktop 1600×1000, dark; screenshots read in subagents).
Not done / debts: two things seen on scratch and not investigated, listed below. The flat dotted-key debt from omp-roles still stands (`docs/handoff/open/agent-clis-native.md`).
Merge: fast-forward ready once `main` is merged in and `make close` passes again (26 behind at close-summary).

## Next up

- The omp line of the study is done through slice 4. What is left on the study menu: the Project pane (needs the owner's yes and an ADR) and presets.

## Debts

- An omp agent created through `POST /api/workspaces/{id}/agents` with cli=omp was gone after the scratch daemon restarted: the API listed the workspace with no agents. Not investigated.
- The guest Settings pane takes its workspace only from the sidebar selection. A `?agentId=` link does not resolve the agent's workspace, so the workspace layer and the folder checks are missing when the pane is opened that way.
