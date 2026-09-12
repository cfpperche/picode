# 2026-09-12 — feat/settings-ux: Pi Settings pane refinement proposal

Shipped: docs only — `docs/plans/cli-settings-ux.md` (proposal: one layer at a
time with a labelled switcher and the layer's file named, provenance per row
from the API's `has` map plus a **Use inherited** reset, Keys as a `pkg-tabs`
sub-tab, the glued `settings-ctx` line removed, `layer`/`view` on the route;
phases P0–P3, decision table, verification, three owner questions) and the
study `docs/benchmarks/2026-09-12-cli-settings-ux.md`. No code changed.

Verified: `make ci-scoped`. Current pane measured on scratch `sxux` (desktop
1600×1000, trusted workspace, agent in the route): **6014 px** of scroll in a
1000 px viewport; Global 414 px / 6 rows, Workspace **431 px / the same six
labels**, Agent 202 px / 3 rows, **Keys 4533 px / 89 rows in 12 groups (75 %
of the page)**; no provenance in the DOM although `layer.has` is in the API
response; `settings-ctx` ("Atlas") 0 px under the tab rule. Captures read:
`var/screenshots/sxux/01-current-top.png` (Global + Workspace duplicated).

visual-review: n/a (no UI shipped; the capture records the current pane)

Not done / debts: implementation not started — P0 (sub-tabs + layer switcher +
ctx line) needs no backend change and takes the page to ~600 px; P1's
**Use inherited** needs the one API field (`reset`) the plan flags as the
owner's call. The pane writes to the project layer through the repo's own
`.pi/settings.json` in a fixture, so any browser script for this must clean up
after itself. `docs/handoff.md` is at its 8 KB cap (8147 B), so nothing was
added there; this note carries it.

Merge: fast-forward ready.
