# 2026-09-12 — feat/connectors-add: Connectors pane refinement

Shipped: `docs/plans/connectors-ux.md` marked implemented — the Connectors pane
in both apps is now a header action + a configured roster + one Add dialog
(`AddConnectorDialog` in `Mcps.jsx`): search, labelled **Save to** wired to the
route scope, service rows with Add/Added, and *Custom server…* / *Import a
file…* as secondary entries. Rows carry state (only while an agent runs — with
it stopped the pane says so once), the scope tag (title = config file), the
target, a Radix switch and an overflow menu (Sign out, Remove). The glued
`settings-ctx` line is gone; the unreachable non-embedded "MCPs" branch is
deleted; `Mcps` imports its own stylesheet (mobile had been rendering the pane
unstyled because only the lazy webhooks route imported it).

Verified: `make ci-scoped` PASS (10 files). New `scripts/qa-cli-connectors.mjs`
against scratch `cxadd` — 14 scenario groups, desktop + mobile: adapter absent,
empty, configured (disabled row, sign-in row, stopped-agent line), dialog
search + Added state, Add payload `scope=user`, target switched to the
workspace (route hash + POST body + reload), removal confirmed by file name and
sent as one DELETE, 1024 px and mobile-sheet geometry. `__picodeOverlayAudit()`
ok in every capture; row controls measured 36 px.

visual-review: PASS (desktop-configured, desktop-empty, desktop-add-dialog,
desktop-1024, mobile-sheet read; card 5/5 below)

Not done / debts: `Live` and `Failed` could not be exercised — no running agent
in the fixture produces the adapter's `mcp.updated` payload; those words are
unchanged code paths plus Go tests. `docs/handoff.md` sits at 8147/8192 bytes,
so no Next-up pointer fitted; this note carries it.

Merge: fast-forward ready.
