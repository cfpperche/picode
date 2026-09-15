# 2026-09-15 — feat/custom-endpoint-page: custom endpoint form moved from dialog to page
Shipped: CustomEndpointPage.jsx twins (browser+mobile, 332 lines each); custom form removed from both Providers.jsx (-321 each); routes extended (cliLaunch custom new/edit parse, cliProvidersHash custom, go providers-custom both apps); tests (cliLaunch/cliProviders/routes/application-routes); providers.css page layout; docs (cli-providers.md, routes.md, providers.md name rules); changelog fragment. Base 3e9a9c01 (main), head 6933303d (3c1d5f08 + main merge).
Verified: JS suites green, make web + ci-scoped PASS; QA scratch end-to-end (created qatest endpoint, roster shows it, edit prefill, unknown-id empty state, narrow single-column confirmed at 600px, mobile surface renders page); overlayAudit ok:true both apps.
visual-review: PASS (7/7 shots read in subagent; card 5/5)
Not done / debts: see Debts.
Merge: fast-forward ready.

## Debts

- duplicate-name error renders at form bottom while Name is at top (inline-at-field would be better)
- pre-existing tab-strip truncation at narrow widths
