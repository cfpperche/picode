# 2026-09-11 — feat/pkg-config-c3: compact joins the GUI-configurable set

Shipped (C3 of `docs/plans/package-config-manifest.md`): `pi-compact`
descriptor — `.pi/compact.json`, workspace scope, fields for enabled,
token/percent triggers, floor, cooldown, summarizer model, thinking and
instructions. `ConfigField` gained numeric `min`/`max` (validated on both
sides), the engine resolves workspace-scope files through the store, and the
generic form grew a tri-state checkbox: untouched = unset = the key is
omitted from the file (pi-compact reads "missing key = default"). `fallback`
arrays stay hand-edited — descriptor v1 has no list type. Owner directive
applied: config files are never hand-created; the GUI flow is the acceptance
test. Sweep recorded: inbox/checklist `server.json` is PiCode-provisioned
identity; mcp-adapter, diff, browser-capture, byteplus, agent-browser-native
have no user config file — no Configure button, honestly.

Verified: go decision-table rows (workspace scope, min/max, unset omission),
`make ci-scoped` (go[10], test-js, build), and a scratch session configuring
web-search AND compact end-to-end through the GUI — save preserved
`fallback`/`atTokens` from the raw file; min/max refusal captured; the
owner's `.pi/compact.json` restored byte-identical afterwards.

visual-review: PASS (compact form read: real pre-populated values, tri-state
checkbox, min/max inline error, saved state; screenshots in
`var/screenshots/compact-*.png`).

Not done / debts: upstream `picode.config` proposal; docs-site entry for the
Packages view; descriptor list type (arrays like `fallback`).

Merge: fast-forward ready.
