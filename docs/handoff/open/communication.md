# Communication, inbox and remote

The owner kept Activate now gated on a recorded session; remote mode stays out of v1 (`docs/plans/communication-activation-accept.md`).

## Debts
- Communication: mobile/non-Linux recovery and PTY check-to-write remain unverified; uncertain attempts never retry.
- Onboarding: claude → grok correlation blocked by the Claude account's exhausted weekly quota (resets ~12:10am; 2026-09-17 attempt created zero messages — the limit-notice footer is correctly refused as an unknown composer). Retry after reset; owner's. The pair's transport was accepted 2026-09-12 and grok-as-recipient correlation passed repeatedly under load. Six-CLI transport: `docs/plans/communication-native-finish.md`; attempts: matrix section of `docs/plans/communication-onboarding.md`.
- Inbox: `system`-sourced items (pi launched outside the launcher) keep the honest reply refusal, and daemon death between park and JSONL row stays accepted (`docs/plans/inbox-terminal-address.md`).
- Pi has one active credential slot; per-agent OAuth is the owner's.
