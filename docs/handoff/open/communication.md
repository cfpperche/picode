# Communication, inbox and remote

The owner kept Activate now gated on a recorded session; remote mode stays out of v1 (`docs/plans/communication-activation-accept.md`).

## Debts

- Communication: mobile/non-Linux recovery and PTY check-to-write remain unverified; uncertain attempts never retry.
- Onboarding: decision rows covered; initial Codex/Hermes/OpenCode and resumed Codex/Hermes need a first prompt, fresh Claude a saved first conversation (`docs/plans/cli-attention-matrix.md`). Six-CLI paired transport: `docs/plans/communication-native-finish.md`.
- Inbox: unmanaged Pi items lack an address until session update; daemon death between park and JSONL row is accepted. Feed events can be missed across reconnects (ADR-0048); webhooks are at-least-once and receivers dedupe by id.
- Pi has one active credential slot; per-agent OAuth is the owner's.
