# 2026-09-09 — unified-messages: one mailbox across native conversations

Implemented: ADR-0107; `picode messages contacts|send|read|ack`, native identity reports,
Grok/Hermes hooks/plugins, durable attention outcomes and desktop/mobile Messages history.
Fixed: Pi receiver registry bootstrap, guarded multiline input and Codex resume hook options.
Verified: `make ci-scoped` PASS; final close and main CI pending.
Native scratch QA: two simultaneous Grok/Hermes TUIs exchanged and acknowledged messages;
`/new` refused old setup, resume/restart recovered history, and multiline drafts survived.
Pi terminal and managed Pi receiver/MCP/ack PASS. Codex contacts and automatic pointer/read
PASS; prompted send/reply/ack PASS with native Allow once approvals; resume hooks recovered.
visual-review: PASS — independent desktop/mobile screenshots, scrolled history and overlays.
Not done / debts: Claude/OpenCode full model roundtrips await native quota/balance;
physical mobile, non-Linux process recovery and custom Codex resume global args/`--` unverified.
PTY rechecks cannot eliminate the check-to-write race; uncertain attempts never auto-retry.
Evidence and decision table: `docs/plans/unified-native-messages.md`; `var/qa/unified-native/`.
Merge: pending current-main merge, final close, fast-forward and full main CI. No deployment.
