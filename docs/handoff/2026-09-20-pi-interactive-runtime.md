# 2026-09-20 — pi-interactive-runtime: shared Pi terminal integration

Implemented: ADR-0162 binds Pi interactive agents to the shared CLI launcher, activity events and native session identity. Agent settings, Ask, session ownership and opt-in communication retain the agent identity; RPC keeps its own state.
Compatibility: existing legacy Pi panes stay live until explicit stop/restart; failed preparation preserves their address. Lifecycle locks and verified shutdown receipts fence replacement, RPC transitions and removal.
Reviewed: independent lifecycle review covered process exit, legacy receipt migration, native identity publication and delete/start races.
Verified: `make ci-scoped` passed on d1f82cbb (fmt, vet, hooks, Go, JS, embedded build and docs); `make close` reused that green tree. Isolated Linux tmux tests cover both mode transitions, shutdown receipts, native session fences, Inbox deduplication and launch rollback. Darwin arm64/Windows amd64 cross-builds passed; hosted runtime behavior was not exercised.
visual-review: PASS, reported by the independent scratch screenshot reviewer for desktop/mobile states and overlays.
Limits: the scratch provider credential was invalid, so no successful live LLM completion is claimed. Durable validation gaps are tracked in `docs/handoff/open/pi-interactive-runtime.md`.
Deployment: not performed; deployment remains the owner's action.
