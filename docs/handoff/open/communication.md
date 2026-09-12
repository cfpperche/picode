# Communication, inbox and remote

## Next

- Remote-mode (a PiCode server off the host) is the owner's infrastructure — decide whether it is in scope at all.

## Debts

- Communication: physical-mobile/non-Linux recovery and custom Codex resume args/`--` unverified; the PTY check-to-write race stands and an uncertain attempt never auto-retries; a deleted owner can leave private setup files.
- Onboarding: initial Codex/Hermes/OpenCode and resumed Codex/Hermes required a first prompt in real matrix validation; fresh Claude waits for a saved first conversation (see `docs/plans/cli-attention-matrix.md`); rows 5/9/12 of `docs/plans/communication-onboarding.md` (consent, adapter repair, child timeout) remain partial. Six native CLIs passed paired transport acceptance with correct providers and loaded adapters; see `docs/plans/communication-native-finish.md`.
- Inbox terminal replies: `pi (unmanaged)` items have no address until each pi session updates; a daemon death between park and JSONL row is accepted.
- Feed: ephemeral events can be missed across reconnects (ADR-0048). Webhooks are at-least-once within event retention; receivers dedupe by id.
- Pi has one active credential slot; per-agent OAuth is the owner's.
- Pre-rewrite Inbox rows still read as live in the audit list.
