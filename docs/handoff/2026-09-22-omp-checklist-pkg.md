# 2026-09-22 — feat/omp-checklist-pkg: the omp checklist mirror becomes an installable package
Shipped: the injected `-e omp-checklist.ts` from feat/omp-checklist is reverted; the mirror is now
`packages/omp-checklist`, an installable omp package (package.json with the `omp.extensions`
manifest) the user installs at whatever scope they want, exactly like `pi-checklist` for pi:
Global (Omp's user layer / plugin store — Packages pane rows with native verbs), This workspace
(`.omp/settings.json`, the `pi-browser` pattern) or an agent's package list (each entry launches
as `-e`, ADR-0176 slice 4 — `agentOmpScopeFlags`). Not installed means no checklist line (ADR-0092
silence). The server keeps the checklist routes, feed and sidebar; the wrapper injects only the
terminal-state extension again, and `uninstallIntercept` keeps its omp case.
Verified: the 12-case decision table moved into the package as node:test
(packages/omp-checklist/test/mirror.test.ts) — positive cases await the recorder's request event,
silence cases run the harness in a child process whose clean exit proves nothing was published.
Wired into `make test-js`. Live on production: `omp plugin install <local package dir>` copied it
into Omp's plugin store, the Packages pane showed the row under Global, and a fresh omp terminal
(checklist-mirror-94bf68, workspace picode-5fd7eb) mirrored todo init/start exactly once after the
redundant user-layer entry was removed. Workspace scope rides the proven settings-file path; agent
scope rides the shipped `-e` flags machinery.
visual-review: n/a (server-side; the checklist UI already exists)
Merge: fast-forward ready.

## Debts

- Todo plans over 50 tasks clip to the store cap (maxChecklistItems=50, internal/store/checklists.go);
  the TUI keeps full fidelity.
- Agent scope validated by its shipped machinery and tests, not a live managed-agent run.
