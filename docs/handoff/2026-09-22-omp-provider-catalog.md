# 2026-09-22 — feat/omp-provider-catalog: Omp's pane offers Omp's whole /login

Shipped: ADR-0183 (owner-approved, amends ADR-0169 for omp). `internal/clicreds/omp_catalog.go` reads `@oh-my-pi/pi-catalog/src/compat/rules.json` next to the `omp` on PATH (`PICODE_OMP_RULES` overrides it). It appends every `/login` provider the declaration lacks, 70 on Omp 18.2.9, for 81 in total. Each row gets Omp's name (the new `name` field on the row), the first key variable (injected at launch through `ompCredentialEnv`), and `oauth` plus a note where Omp signs in. Aliases of declared ids are skipped. `providerName(id, name)` falls back to that name. The Add select is alphabetical. A provider without `api_key` gets Guided sign-in as the primary action, with no key field and no Save: before this, a key saved there made an unreadable row. Both test suites point `PICODE_OMP_RULES` at nothing, so a developer's installed omp does not change their rosters.

Verified: `make ci-scoped` PASS. New tests cover the parser table, `For`/`Declarations`/`EnvVar` with and without the file, the npm-layout locator, and the server roster plus launch env for a catalog key. Scratch instance with the real omp: API 81 providers, select sorted with 81 options, keyless/key/both dialog states at 1440px and 390px, overlay audit ok. Pi and OpenCode unchanged. The first QA pass was FAIL: keyless providers showed a key field. It was fixed and rechecked.
visual-review: PASS (ompcat2-keyless-zai.png, ompcat2-mobile-keyless.png, ompcat2-cerebras.png; overlayAudit ok; card 5/5)

Not done: a sign-in-only catalog provider that `oauth.Supports` does not know falls back to Omp's `/login` terminal. Its login lands in Omp's SQLite, so PiCode shows no row for it. In the Add dialog, the "Saved to this machine's vault." subtitle stays even for keyless providers. Long Omp names truncate in the table's provider column.

## Debts

- Omp's rules.json path is Omp internals: when an Omp release moves it, the pane falls back to 11 providers silently (ADR-0183)

Merge: fast-forward ready.
