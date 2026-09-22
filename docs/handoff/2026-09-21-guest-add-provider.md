# 2026-09-21 — feat/guest-add-provider: guest Add provider dialog with guided OAuth sign-in

Shipped: guest Add provider dialog (roster label changed from Add API key — all guests; vocabulary
parity with pi) with Guided sign-in inside it for providers carrying an oauth kind plus a declared
sign-in: one click closes the dialog and starts the ADR-0168 strip (CLI's own login in a terminal;
vendor OAuth never passes through PiCode); Check now files the account. Server test pin updated.
Files: internal/server/{credentials.go,credentials_pi_test.go}, web/{browser,mobile}/CliCredentials.jsx,
docs (cli-providers.md, guide, fragment).
Verified: commit d5b9a4c7, ci-scoped green (fmt, vet, hooks, go[4], test-js, build, docs).
Not done / debts: none. Merge: done — landed at `e0bd79ab` and serving in `0.4.0+ac671b8`.
visual-review: PASS (3 stills: bar label, dialog with guided button + help line, strip after click).
