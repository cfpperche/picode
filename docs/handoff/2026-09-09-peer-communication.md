# 2026-09-09 — feat/peer-communication: direct session messages

Implemented: ADR-0104, embedded Go MCP with four tools, two SQLite tables,
per-conversation opt-in/revoke, same-workspace contacts, durable retry receipts,
non-destructive inbox reads and atomic acknowledgments; no task/runtime changes.
Desktop/mobile Agent CLIs → Messages manages setup and history.
Dependency: official MCP Go SDK v1.7.0 replaces handwritten protocol handling.
Verified: full ci-scoped, race checks, store decision table, HTTP MCP/auth,
installed pi-mcp-adapter
2.32.1 discovery/send/retry/read/ack/reply on owned fixtures (no model turns).
Browser matrix: empty, blocked, setup, retained-history failures, confirm,
revocation, delayed credential after owner change, history pagination; both apps
on isolated :8473. A real adapter message refreshed the browser through SSE.
visual-review: PASS; screenshots in var/screenshots/peer-*; settled overlays ok.
Validation close/main CI is recorded in the final session result.
Later increments: launch-time wiring, automatic notification, guest runtime
compatibility matrix, real model turns and physical mobile acceptance.
Credentials identify the selected recorded session, not native-process proof.
Merge: fast-forward delivery; deploy remains the owner's batch.
