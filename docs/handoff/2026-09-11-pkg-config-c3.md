# 2026-09-11 — feat/pkg-config-c3: compact joins the GUI-configurable set

Shipped (C3 of `docs/plans/package-config-manifest.md`): `pi-compact` added
to the descriptor catalog — `.pi/compact.json`, workspace scope, with
enabled/atTokens/atPercent/floorTokens/cooldownTurns/model/thinking/
instructions. `ConfigField` gained numeric `min`/`max` (bounds validated
server-side and client-side), the engine gained workspace-scope resolution
(GET/PUT/DELETE take the workspace folder from the store), and the generic
form grew a tri-state checkbox: untouched = unset = the key is omitted from
the file, which is how pi-compact reads "missing key = default". `fallback`
(array) stays hand-edited — descriptor v1 has no list type.

Owner directive applied: config files are never created by hand — the GUI
flow itself is the acceptance test. The plan's stopgap section was replaced
by that rule.

Sweep result (in the plan + ADR): pi-inbox/pi-checklist `server.json` is
connection identity PiCode provisions (not exposed); pi-mcp-adapter,
pi-diff, pi-browser-capture, pi-byteplus-modelark, pi-agent-browser-native
have no user config file — no Configure button, honestly.

Verified: go (decision-table rows for workspace scope, min/max, unset
omission), `make ci-scoped` (go[10], test-js, build), and a scratch session
configuring BOTH pi-web-search and pi-compact end-to-end through the GUI —
save preserved `fallback`/`atTokens` from the raw file; min/max refusal
captured; the owner's `.pi/compact.json` was restored byte-identical after
the scratch exercised it.

visual-review: PASS (compact form read: pre-populated real values, tri-state
checkbox, min/max inline error, saved state; screenshots in
`var/screenshots/compact-*.png`).

Not done / debts: upstream `picode.config` proposal; docs-site entry for the
Packages view; descriptor list type (arrays like `fallback`).

Merge: fast-forward ready.
