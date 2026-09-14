# Communication, inbox and remote

The owner kept Activate now gated on a recorded session; remote mode stays out of v1 (`docs/plans/communication-activation-accept.md`).

## Next

- Owner: the 2026-09-14 native matrix connected 6/6 onboarding flows, but check correlation failed on model id-invention (Claude/Opus) and grok's withheld attention Enter under load — decide if either deserves product follow-up (`docs/plans/communication-onboarding.md`).

## Debts
- Communication: mobile/non-Linux recovery and PTY check-to-write remain unverified; uncertain attempts never retry.
- Onboarding: initial and resumed Codex/Hermes/OpenCode need a first prompt. The flow is natively connected for all six (2026-09-14); run-test correlation needs a quiet machine — matrix section of `docs/plans/communication-onboarding.md`. Six-CLI transport: `docs/plans/communication-native-finish.md`.
- Inbox: unmanaged Pi items lack an address until session update. Feed events can be missed across reconnects (ADR-0048); webhooks are at-least-once, receivers dedupe.
- Pi has one active credential slot; per-agent OAuth is the owner's.
