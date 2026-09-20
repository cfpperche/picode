# 2026-09-19 — workspace-cli: create an Agent CLI from the workspace
Shipped: workspace **New → Agent CLI** (desktop + mobile) and Ctrl+K
**New Agent CLI · workspace** open a catalog picker; POST `/principals`
then start. Uninstalled CLIs hidden. `#/clis/new` stays for a free
terminal.
Verified: `make close` PASS; scratch overlay `workspace-cli-picker.png`
+ `__picodeOverlayAudit` ok; Cancel closes. Blind spot: empty/error
states not captured — this PATH had every catalog CLI installed.
visual-review: PASS (happy overlay); empty/error UNVERIFIED
Not done: Fatia 4 (automations / Inspector via prompt door). Topic:
`open/managed-principals.md`.
Merge: fast-forward ready.

## Next up

- Fatia 4: automations and Inspector ask through the prompt door
