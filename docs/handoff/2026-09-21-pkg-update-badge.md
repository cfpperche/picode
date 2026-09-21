# 2026-09-21 — feat/pkg-update-badge: the guest-CLI update badge, on a signal that exists
Shipped (ADR-0167 amendment): `clipkgs.CheckUpdates(ctx, cli, p, scope, fresh)` reads the CLI's own
roster and its own catalog (both cached; `fresh` bypasses), pairs rows by id then name, and marks a row
only on `pipkg.Newer(catalogVersion, installedVersion)`, so `Row.Latest` is set only when the badge is
and a non-semver pair is never a claim. A catalog that cannot be read returns the roster unmarked with
the reason in `Note` ("Could not read … catalog, so nothing was compared"), never "up to date"; no
update verb → `ErrVerbAbsent` → 400. New `Row.Latest`, `Row.UpdateAvailable`, `Report.CheckedAt`; route
`GET /api/cli-packages/updates?cli=&workspace=&scope=&refresh=1` (a read, `checkedAt` in the view);
shared `rowUpdateState` → `update | current | unknown | none`, run once per mount (a ref guards it)
where `caps.update`; the pane shows `→ 0.3.0`, gates **Update** on it, and offers "Check again" /
"Check for updates" with "Everything is up to date." when current, both apps; docs `cli-packages.md` + fragment.
Verified: the live harness end to end — Muse at 0.2.0, its snapshot refreshed through the vendor's own
`muse plugins marketplace update` to 0.3.0, the row back `BEHIND latest=0.3.0`; new `TestMarkUpdates`,
`TestCheckUpdatesAgainstAStubVendor` (stub vendors on PATH), `TestCLIPackageUpdatesMarksWhatIsBehind`,
the JS `rowUpdateState`/`updates` paths (19 JS pass); `make ci-scoped` PASS (15 paths), `make close`
PASS twice, ff-ready after merging a fast-moving `main`. `close-summary`: 16 files, +476/−32, scope
`FULL= GO=1 WEB=1 PACKAGES= DOCS=1 METADATA=1`, 3 ahead / 0 behind, clean; main ff to `a8c159c6`.
Visual: a scratch instance (not the owner's live window), desktop and 390px — `picode-probe 0.1.0 → 0.2.0`
with **Update**, `steady 1.0.0` without; `__picodeOverlayAudit()` `ok: true` on both after two CSS fixes.
visual-review: PASS
Merge: fast-forward ready.

## Debts

- Muse's built-in plugins have no update signal: docs/handoff/open/packages.md.
