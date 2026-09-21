# 2026-09-20 — unified-providers: one Providers pane for all nine CLIs (ADR-0169)

Shipped: `CliCredentials.jsx` serves every CLI (pi's editor and its
`Providers.jsx` in both apps are deleted); the roster carries pi's providers
from the catalog (native + custom) and each row's cached `usage`, plus `add`,
`custom` and `verify` fields; the pane renders pi's grid columns for everybody
(`Provider · Account · Identity · Usage · 7d spend · actions`), the Usage cell
via the frozen `QuotaStrip` with a one-call **Check**, 7d spend from
`/api/sessions/stats` via `spendByProvider`, and the union of doors gated on
the roster's own flags (Add provider + Custom provider + Edit provider for pi;
Sign in, native import, per-provider Add, row Verify for guests). No new
route: the per-row Check is `GET /api/providers/{p}/accounts/{id}/usage`.

Verified: Go — `TestPiRosterProvidersComeFromTheCatalog`,
`TestPiRowUsageComesFromTheCacheAlone`, `TestPiRosterActivatesAndPausesARow`
(Use writes `~/.pi/agent/auth.json` through the vault route — proved by test,
not assumed), `TestGuestRosterCarriesThePaneFieldsOnly`; JS — 940 tests green
incl. the updated capability list (`pi` joins the nine) and the dialog-policy /
boundary suites; `make web` builds both apps. Scratch QA at 1560×900 and
414×896: pi pane, row menu (Use · Rename · Verify with Pi · Pause · Sign out),
Add-provider dialog, custom page, Codex pane; `__picodeOverlayAudit()` ok:true
on all four overlays, 36px rows aligned. Blind spot: `pi auth check`'s verdict
and a real vendor OAuth were not exercised (no live subscription in the scratch).

visual-review: PASS (six captures read by a subagent; see the verdict line)
Not done / debts: replacing a signed-in provider's key is now Sign out → Add
provider (pi's editor had a replace path the dialog does not carry); nothing
else was lost — see the pane report's not_carried list in the branch.

## Next up

- A "Replace" affordance on a signed-in provider's row (pi's old editor had
  one), if the two-step detour proves annoying in use.
