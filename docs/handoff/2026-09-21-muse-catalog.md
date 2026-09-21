# 2026-09-21 — feat/muse-catalog: Muse's plugin surface, measured through the CLI and its TUI
Shipped (ADR-0167): a catalog row composes the install spec `name@marketplace` from the row's
`name`+`marketplace` halves and keeps the resolved path as the row's path, so Install runs the
right command; spec flag `catalogNeedsRoster` + `markInstalled` join the catalog with the CLI's own
roster where declared (only Muse declares it), so a plugin already there reads Installed, not
Install. Non-empty fixtures `testdata/muse.available.json` + `testdata/muse.marketplaces.json`; the
live harness writes a marketplace in Claude's catalog format (which Muse's store reader probes),
adds it, installs from it and asserts the installed row and the catalog join — the provenance word
differs by route (`native-local` vs `marketplace-user-added`), so the assertion is on the fact and
the fixture pins the word. `docs/handoff/open/packages.md` now says the catalog was measured;
fragment `docs/changelog.d/muse-catalog.md`.
Verified: `go test ./internal/clipkgs/...`; the gated live suite end to end in a fresh sandbox;
`make ci-scoped` PASS (FULL scope, METADATA=1); `make close` PASS, ff-ready, reusing the green stamp
after the merge with `main`. Pane read against a real Muse in a scratch (`--provider echo`, no Meta
sign-in, no credential copied): roster `picode-spare 0.2.0 user-local`, catalog `installed: true`
with `source: picode-spare@picode-probe-live`, `__picodeOverlayAudit()` ok. Blind spot: the TUI was
driven over the scratch's own tmux socket (`send-keys`/`capture-pane`), never in the owner's live pane.
visual-review: PASS
Not done / debts: Muse's TUI `/plugins` panel lists a built-in first-party plugin (`TBH Reminders`,
built-in/product) that no CLI command reports; PiCode manages what the vendor's store manages.
Merge: fast-forward ready.

## Debts

- Muse's built-in plugins stay invisible: docs/handoff/open/packages.md.
