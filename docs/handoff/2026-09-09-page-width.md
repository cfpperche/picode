# 2026-09-09 — feat/page-width: one panel width for every desktop page

Shipped: every `PageFrame` route shares the Agent CLIs geometry (ADR-0103):
`.settings-wrap` fluid to 1240px, centred, shared gutters. The `wide` prop
and the 680/1080 variants are gone; `.cli-page` keeps only card-padding
rules. Routes that changed: system, integrations, mcps, devices,
preferences (+sections), pins (were 680px); automations, llama/*, termset
(were 1080px). Workspace canvases (`#/`, term/file/git/tree/app) keep their
own layout; mobile is full-width by design (ADR-0072). Rule + route map
documented in docs/benchmarks.md (UI/UX); CHANGELOG [Unreleased] entry.

Verified: ci-scoped PASS (fmt, vet, hooks, test-js, build); qa-scratch
`page-width` :8471 — screenshots of all 10 routes at 1706px plus system at
1280px read (var/screenshots/pw-*.png); `__picodeOverlayAudit()` ok.
visual-review: PASS
Not done / debts: none known; pins editor and Preferences rows judged only
at 1706/1280 widths.
Merge: fast-forward ready.
