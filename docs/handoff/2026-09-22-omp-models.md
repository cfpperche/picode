# 2026-09-22 — feat/omp-models: omp Models pane — what omp reaches, and which of it it may use

Shipped: `#/clis/omp/models` (ADR-0181 slice 2, accepted by the owner). The catalog comes from `/api/cli-models`, read in the workspace. It is grouped by provider and shows kind chips in words, a filter, and context and price columns under a header. The Allowed switch writes `enabledModels`; Hide provider / Show write `disabledProviders`; both work on both layers. The pane appears only for CLIs with a models reader (`MODELS_CLIS` = `climodels.Supported`, parity test). Both keys are settings `list` rows with `pane: "models"`, hidden from Settings. A toggle writes the layer's whole list, because arrays replace across layers. Empty and blocked states: a warning plus "Allow every model here" when every allowed entry is an exact selector that cannot be reached; an all-hidden state with "Show all"; a no-match state with "Clear filter"; a 0/0 price shows nothing. The catalog is cached by a fingerprint of both config layers plus models.yml, for at most 10 min. agent.db and models.db are left out of the fingerprint because omp rewrites them on every run (measured: the first version never hit the cache). Refresh sends `fresh=1`. A cold probe takes about 11 s; a cached one about 0 s.

Verified: go and JS tests, `make ci-scoped` PASS. On a qa-scratch (port 8490, own HOME) every write was driven through the UI and the file read back: allow on global; hide openai in the workspace wrote ["groq","openai"] and left global untouched; Show; Show all wrote []; Allow every model here wrote `enabledModels: []`. Cache timings measured with curl. Blind spot: never read by a live omp TUI.
visual-review: PASS after three fix rounds (desktop 1600×1000, mobile 390×844, dark; overlayAudit rows aligned; screenshots read in subagents). Blind spot: the mobile layer switcher is in no mobile frame. Its centring was measured (align-items center) and seen on desktop, never read on the phone.

Not done / debts: `qa-scratch.sh start ompmod` reported success while another session's scratch (packages-alias) held port 8471. This daemon failed to bind, and the health check was answered by the other instance. `qa-scratch.sh seed` then created workspace `qa-98f6bd`, agent `atlas-5883a2` and terminal `shell-e54886` in that instance. The permission classifier refused their removal; they remain for the owner or that session to delete. The flat dotted-key debt from omp-roles still stands (`docs/handoff/open/agent-clis-native.md`). The IQ column is still missing from `omp models --json`.
Merge: fast-forward ready once main is merged in and `make close` reruns.

## Next up

- Slice 4: resolved-config + doctor card from `omp config list --json`.

## Debts

- `qa-scratch.sh start` must check that the pid it launched is the one listening before it reports success. A port held by another scratch passed the check, and `seed` wrote into the wrong instance.
