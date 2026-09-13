# Communication, inbox and remote

## Next

- Owner: Activate now cold-start vs recorded session. Remote stays out of v1.

## Debts

- Mobile/non-Linux recovery and PTY check-to-write remain unverified; uncertain attempts never retry; deleted owners can leave setup files.
- Onboarding: initial Codex/Hermes/OpenCode and resumed Codex/Hermes needed a first prompt in matrix validation; rows 9/12 (adapter repair, child timeout) partial. Six native CLIs passed transport acceptance — `docs/plans/communication-native-finish.md`.
- Inbox: unmanaged Pi items lack an address until session update. Feed events can be missed across reconnects (ADR-0048); webhooks are at-least-once, receivers dedupe.
- Pi has one active credential slot; per-agent OAuth is the owner's.
