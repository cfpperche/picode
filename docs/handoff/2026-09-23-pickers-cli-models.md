# 2026-09-23 — pickers-cli-models: Pi's rows get thinking levels; the picker switch is dropped

Shipped: `/api/cli-models?cli=pi` fills each row's `thinking` from models-store.json (`catalog.FillPiThinking`,
copying the kept rows), so Pi answers in omp's shape. Verified: `make ci-scoped` PASS; API and copy tests.
Decided with the owner (not done): Pi's model pickers keep reading `/api/catalog`. Mapped every consumer: all
pickers are Pi-only; `/api/catalog` already reads through the same climodels reader/cache and composes sign-in,
custom providers and llama.cpp models, which `/api/cli-models` would have to rebuild. The one-source goal pays off
as the other seven CLIs get readers — next.

## Next up

- Measure the other seven CLIs for a read-only model-listing command (and whether it depends on the folder)
