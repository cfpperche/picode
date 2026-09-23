# 2026-09-23 — feat/cli-ws-name-fix: literal `$` workspace names in CLI switchers
Shipped: `namedScope` fix in web/shared/domain/cliNative.js (replacer `() => name`); regression asserts in cliNative.test.js; fragment docs/changelog.d/cli-ws-name-fix.md.
Follow-up to feat/cli-ws-name (landed): string replacement interpreted `$` patterns, so `$&` left Settings/Models/Memory generic while Packages/doctor showed true name; `A$'B` mangled (reproduced in node pre-fix).
Verified: node --test domain suites 21/21 pass; `make ci-scoped` PASS (fmt, vet, hooks, test-js, build, living-docs; 3 paths).
visual-review: n/a (no visual change for ordinary names; prior scratch evidence in var/screenshots/clis-ws-name/ stands).
Not done / debts: none.
Merge: fast-forward ready.

## Next up

- none — no follow-up; fix complete.

## Debts

- none.
