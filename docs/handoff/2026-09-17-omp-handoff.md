# 2026-09-17 — omp-handoff: omp joins handoff as source and native target

Shipped: `OmpSource.Read` (pi-family schema-v3 projection — live-chain
replay, title records outside the chain, provider-qualified model_change,
thinking and custom records dropped to the manifest), `Write` at the
measured-minimal native shape into the cwd bucket with the read-back
round-trip, and `PromptArgs` = the positional prompt (omp cannot
pre-assign ids). Capabilities: list/read/write/prompt — omp is a full
handoff source and native+brief target. The Write spike ran live before
implementation: a full clone with a new id and a header+one-message
minimum both resumed on omp 18.2.4, the model reading the cloned history.
Verified: make close green; scratch `ompd` — an omp session's ⋯ menu now
offers Continue in Pi/Claude/Codex/Grok/Hermes/OpenCode/Muse/Antigravity,
overlayAudit ok.
visual-review: PASS (omp-continue-source.png; card 5/5)
Not done / debts: none on this branch; Fatia 5b (lifecycle install-method
matrix: bun-global, curl-native layout, vendor channel, Install lane) is
the next omp slice.
Merge: fast-forward ready.

## Next up

- Omp Fatia 5b (lifecycle matrix): probe bun-global in an isolated BUN_INSTALL_DIR and the curl installer's target layout, then decide vendorOnNpm vs npmUpdateOnly and the LatestFrom source. Plan: `docs/plans/omp-cli.md`.

## Debts

- Live omp TUI activity acceptance (Ready/Working in a real PiCode terminal) is external until the owner runs omp authenticated inside PiCode.
