# 2026-09-25 — feat/legacy-api: legacy API group 2 retired (/api/pi-keys, GET /api/packages, workspaces[].agent)
Owner asked for "grupo 2" of docs/handoff/open/legacy-compat.md.
Shipped: /api/pi-keys removed (UI already used /api/cli-keys?cli=pi); pi_keys_test.go
ported to /api/cli-keys; piKeysSnapshot/piKeysPlatform kept (cli-keys wraps them);
qa-cli-settings(-recovery) moved — recovery read a `.path` the report never had, now `.file`.
GET /api/packages route removed: 4 Go tests + qa-cli-packages.mjs read
/api/packages/report?cli=pi (rows[].configKind/source). loadPackageReport and
Report.Legacy stay: 5 internal readers and the Pi mutation responses (POST/DELETE
/api/packages, POST /api/packages/update) still answer in pipkg shape.
GET /api/packages/updates kept: the live badge read for all nine CLIs; LegacyUpdates only
names its (single) response shape — reclassified as not legacy.
workspaces[].agent removed: workspace removal now closes every agent's shell/panel (was
first agent only — small bug fix); palette Run/Stop reads agents[0].mode; agentsOf /
workspaceAgents fallbacks dropped; Go tests use wk.Agents[0]; docs-video-stills uses agents.
openapi.json regenerated. Commits: e853ea663, 3c6e03166, 239d041b2, d0f5363b0, 9646f01f9.
Verified: `make ci-scoped` PASS (first run hit an unrelated attach-interrupt timing flake),
`make close` green. Scratch: /api/packages and /api/pi-keys 404; cli-keys, packages/report,
packages/updates 200; /api/workspaces has no `agent` key; Pi Packages pane (no alerts), Pi
Keyboard pane (90 actions), palette shows "Run QA". Blind spot: no older external client was run.
visual-review: PASS (lapi-packages.png, lapi-keyboard.png, lapi-palette.png)
Merge: fast-forward ready.

## Debts

- TestAttachInterruptOnTmux flake and remaining legacy items: docs/handoff/open/legacy-compat.md
