# Communication, inbox and remote

## Next

- Run explicit activation acceptance on the currently available test terminals, recording provider/model behavior and one-turn cost.
- Remote-mode (a PiCode server off the host) is the owner's infrastructure — decide whether it is in scope at all.

## Debts

- Communication: mobile/non-Linux recovery, custom Codex resume args/`--`, and the PTY check-to-write race remain unverified; uncertain attempts never auto-retry and deleted owners can leave private setup files.
- Onboarding: initial Codex/Hermes/OpenCode and resumed Codex/Hermes required a first prompt in real matrix validation; fresh Claude waits for a saved first conversation (see `docs/plans/cli-attention-matrix.md`); rows 5/9/12 of `docs/plans/communication-onboarding.md` (consent, adapter repair, child timeout) remain partial. Six native CLIs passed paired transport acceptance with correct providers and loaded adapters; see `docs/plans/communication-native-finish.md`.
- Inbox: unmanaged Pi items lack an address until session update; daemon death between park and JSONL row is accepted. Feed events can be missed across reconnects (ADR-0048); webhooks are at-least-once and receivers dedupe by id.
- Pi has one active credential slot; per-agent OAuth is the owner's.
