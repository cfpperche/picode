# 2026-09-20 — pi-interactive-runtime: shared Pi terminal integration

Implemented: ADR-0162 binds Pi interactive agents to the shared CLI launcher, activity events and native session identity. Agent settings, Ask, session ownership and opt-in communication retain the agent identity; RPC keeps its own state.
Compatibility: existing legacy Pi panes stay live until explicit stop/restart; failed preparation preserves their address. Lifecycle locks and verified shutdown receipts fence replacement, RPC transitions and removal.
Reviewed: independent lifecycle review covered process exit, legacy receipt migration, native identity publication and delete/start races.
Verified: focused Linux runtime tests exercised isolated tmux processes; an earlier `make ci-scoped` passed as reported by the working agent. The final local gate rerun was not available when this note was prepared. Darwin arm64/Windows amd64 cross-builds passed; hosted runtime behavior was not exercised.
visual-review: PASS, reported by the independent scratch screenshot reviewer for desktop/mobile states and overlays.
Limits: the scratch provider credential was invalid, so no successful live LLM completion is claimed. Durable validation gaps are tracked in `docs/handoff/open/pi-interactive-runtime.md`.
Deployment: not performed; deployment remains the owner's action.
