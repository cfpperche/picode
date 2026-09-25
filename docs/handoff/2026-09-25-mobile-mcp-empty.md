# 2026-09-25 — feat/mobile-mcp-empty: empty Connectors list gets a centred line and an action
Shipped: owner asked to fix the phone Connectors empty state after Packages' was centred.
Desktop had no centred state either; the rule breaker was Codex (any CLI-scoped connectors
pane) with nothing configured: a lone grey `pkg-fine` "No connectors yet." with no action.
Both apps' Mcps.jsx now render the Packages shape there: `.pkg-empty`, centred bold
"No connectors yet." + "Open the Marketplace" (btn-sm, switches to the Marketplace tab).
Left as is: `.mcp-empty` states (Pi adapter + Open packages, not installed + Check again,
load/catalog errors + Retry) already carry line + action; their phone left-stacked
override stays. Claude's "in development — coming soon" pane is unaffected.
Verified: `make ci-scoped` PASS; scratch Codex on phone (title 600, centre x=195, button
36px) and desktop; the button selects the Marketplace tab. visual-review: PASS (both).
Not done / debts: none. Merge: fast-forward ready.
