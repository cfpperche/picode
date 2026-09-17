# 2026-09-17 — omp-lifecycle: the omp install-method matrix closes

Shipped: the three lifecycle unknowns measured and closed. bun-global omp
(realpath into node_modules) is honestly `MethodUnknown` — npm would miss
the bun copy and the vendor updater resolves by PATH (measured: run from
a bun install it updated an npm copy); scoped to `@oh-my-pi` so opencode's
installer-aware bun handling keeps classifying npm. The curl installer's
native binary at `~/.local/bin/omp` is `MethodVendor`, mutating through
`omp update` / `omp update --force` like grok accepts. `omp update
--check` is parsed (`ParseOmpCheck`, both measured shapes) as the vendor
LatestFrom for native installs; npm installs keep the npm registry. A
missing omp now installs through the npm lane (`ForMissing` derives it).
Verified: make close green (fmt,vet,go,docs); decision tables — For()
omp npm/vendor/unknown rows, DetectMethod bun/curl rows, ParseOmpCheck
shapes. Probes recorded in `docs/plans/omp-cli.md` (Fatia 5b outcome).
visual-review: n/a (no UI surface changed — the Install button for a
missing omp renders from existing capability chrome).
Merge: fast-forward ready.

## Next up

- Omp live acceptance: run omp authenticated inside a PiCode terminal once (Ready/Working) and exercise Continue in Omp end-to-end — both are owner-run, everything mechanical is green.

## Debts

- Live omp TUI activity acceptance (Ready/Working in a real PiCode terminal) is external until the owner runs omp authenticated inside PiCode.
- brew installs of omp are unobserved (no macOS here): DetectMethod has no brew pattern; they classify by whatever the symlink resolves to.
