# 2026-09-09 — peer-launch: Private setup on conversation resume

Shipped: ADR-0106 adds private communication setup for recorded Pi, Claude Code, Codex and OpenCode resumes; desktop/mobile explain setup and recovery.
Verified: scoped CI passed; launch tests cover exact resume, fresh/fork exclusion, revoked/stale credentials, setup rollback and secret-free diagnostics.
Verified: owned managed Pi → resumed Pi terminal → managed Pi exchanged two messages with both acknowledgements.
Verified: Claude/Codex native model list_contacts calls passed over HTTP and verified HTTPS; OpenCode handshake passed on both.
Verified: managed Pi called list_contacts over HTTPS using its generated child CA bundle.
visual-review: PASS — subagent read all 16 browser-tls PNGs; empty, blocked, setup, confirmation, retained-history error, missing adapter and ready light/dark; all overlay audits ok.
Evidence: var/screenshots/peer-launch-20260909/; runtime versions and limitations in docs/plans/peer-communication.md.
Not done / debts: Claude/Codex probes used a Pi fixture capability, not full recorded native conversation resumes; OpenCode model turns and physical-device acceptance remain untested.
Not done / debts: Grok/Hermes automatic setup unavailable; automatic recipient wake is not implemented; native session discovery remains best effort.
Merge: fast-forward ready; final close and main CI pending.
Deploy: not performed; owner-controlled.
