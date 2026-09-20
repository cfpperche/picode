# 2026-09-20 — ci: the shell gets compile gates (desktop-shell-gate)

Branch stacked on feat/browser-capture-tests (5565af2e); commit a27d023f
"process: the shell gets compile gates (desktop scope in ci)" + main merged.

- What landed: `desktop-shell/**` was an unknown ci-scope path — nothing on
  main compiled the shell, so 8117fd50's pub(crate) `permissions::site_of`
  break (E0603) sat unnoticed until the capture branch's cross-build. Now:
  pathScope `desktop-shell/**` → "desktop"; a shell-only diff runs
  `make desktop-test` (pure half host tests: `rustc --test src/lib.rs` +
  `src/annotate.rs`) plus `make desktop-shell` (cargo xwin cross-build) in
  ci-scoped; mixed diffs fail safe to full.
- `desktop-test` joined ci-gates (`make ci` always tests the pure crate);
  ci-scope-reuse covers the desktop-shell/ subtree (ADR-0124); GH ci.yml
  gains a desktop job (host tests only, condition full||desktop); AGENTS.md
  documents desktop-test.
- Tests extended in ci-scope.test.mjs and ci-scope-reuse.test.mjs (33 pass).
- Residual debt: docs/handoff/open/desktop-shell-gates.md — remote CI never
  compiles the COM half (xwin needs the Windows SDK; runs locally and at
  release, release.yml pattern).
- Evidence: `make desktop-test` green; 33 scope/reuse tests green; `make
  close` green (full matrix).

## Next up
- Land feat/browser-capture-tests first (or with) — this branch is stacked on it.
