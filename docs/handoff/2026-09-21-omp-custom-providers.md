# 2026-09-21 — feat/omp-custom-providers: custom providers for omp via its models.yml (ADR-0175)
Shipped: the custom-provider surface for omp (ADR-0175, amends ADR-0169). PiCode merges
definitions + apiKey into `~/.omp/agent/models.yml` (node-level YAML merge; unknown fields and
comments survive; written 0600). The roster gains the omp door and definition rows (definition +
keyed, never key material). `PUT`/`DELETE /api/providers/custom/{id}` take `cli`; verify and Load
models spend the saved key. The pane form offers omp's accepted subset (2 compat flags, 5 thinking
formats; no chat-template objects, no thinking levels). Definition lines carry
Edit/Verify/Remove, and the page is keyed per route, so a failed Add cannot leak into Edit.
Files: `internal/catalog/modelsyaml.go` (+test), `internal/server/{credentials,custom_models,
providers_usage,slash_ops}.go` (+`custom_provider_omp_test.go`), `web/{browser,mobile}`
CliCredentials+CustomEndpointPage, `web/shared` {modelLoad,schemas,cliProviders}.
Docs: `docs/decisions/0175`, `docs/architecture/cli-providers.md`,
`docs/architecture/credentials.md`, `docs-site/guide/providers.md`, changelog fragment.
Verified: commit `3b1c362c` — ci-scoped green (fmt, vet, hooks, go[7], test-js, build, docs);
`make close` green; reviewed on a scratch instance.
Not done / debts: none.
Merge: done — main carries `6396f58b`; shipped in the 2026-09-21 deploys (`0.4.0+b0f4ba4`
serves it). Worktree and branch removed.

visual-review: PASS (7 stills, overlayAudit ok, card 5/5) on scratch instance.
