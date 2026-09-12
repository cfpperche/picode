# 2026-09-11 — feat/packages-pane-context: selected agent on Agent CLIs Packages

Shipped: Agent CLIs Packages, Connectors and Settings take the selected sidebar agent and workspace onto the hash (`#/clis/pi/packages?workspaceId=…&agentId=…`) so This workspace / This agent radios appear next to This machine. Nested leftover “Back to agent / Machine packages” chrome removed. Fragment `docs/changelog.d/packages-pane-context.md`.

Verified: `make close` PASS (ci-scoped fmt,vet,hooks,test-js,build,docs). Scratch http://localhost:8473 session pkg-ctx: select Atlas, sidebar Agent CLIs icon, Packages tab → hash carries workspaceId and agentId; radios This machine / QA / This agent; context Atlas · QA. Screenshot `var/screenshots/packages-pane-context/02-packages-radios-clean.png` (not committed).

visual-review: PASS

Not done / debts: none on this branch. Agent CLIs is still absent from docs-shots SURFACE_PROFILES (pre-existing).

Merge: fast-forward ready (main merged in as ea5f502b).
