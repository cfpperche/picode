# 2026-09-06 — feat/opencode-cli: OpenCode terminals and activity plugin

PR1: catalog id `opencode`; sessions from `opencode.db` read-only;
`--session <id>` resume; presence wrapper; no data-dir overlay.

PR2: session plugin via `OPENCODE_CONFIG` (merge, relative
`./picode-activity.js`). User `opencode.jsonc` is not written. Map:
`session.status` busy/retry → working; idle → idle; permission/question
asked → needs-you; replied → working; question rejected → idle.
Maintenance skips the plugin. `OPENCODE_CONFIG` is launcher-reserved.

Verified PR1: `make test` / `test-js` / `fmt-check` / `vet`; visual-review
PASS scratch :8471. PR2: same gates; `opencode debug info` lists the
plugin; jsonc mtime unchanged; visual-review PASS scratch :8471 with
Activity on (overlayAudit ok).
visual-review: PASS
Not done: live Working/Needs you unproven until deploy; no Reader/Writer.
Merge: merged as ba87bf2c.
