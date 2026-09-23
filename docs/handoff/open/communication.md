# Communication, inbox and remote

The owner kept Activate now gated on a recorded session; remote mode stays out of v1 (`docs/plans/communication-activation-accept.md`).

## Debts
- Communication: mobile/non-Linux recovery and PTY check-to-write remain unverified; uncertain attempts never retry.
- Onboarding: every matrix correlation has now passed natively at least once (finale: claude → grok, 2026-09-17). What remains is vendor behavior — first prompts, model discretion, weekly quotas — see `docs/plans/cli-attention-matrix.md` and the matrix section of `docs/plans/communication-onboarding.md`. Six-CLI transport: `docs/plans/communication-native-finish.md`.
- [x] Inbox: `system`-sourced items (pi launched outside the launcher) kept an honest reply refusal — **paid 2026-09-21**: the answer is recorded on the item instead, so the human's Reply always closes it, and the note/toast name who still must be told another way (a plain `ask`, an unmanaged pi — the former policy, kept in the copy). Daemon death between park and JSONL row stays accepted (`docs/plans/inbox-terminal-address.md`).
- Pi has one active credential slot; per-agent OAuth is the owner's.
- [ ] Inbox → non-Pi agents: Omp's receiver submit (`pi.sendUserMessage`) not exercised live — the 18.2.9 probe stopped before a model call (2026-09-22, `internal/server/agent_answer.go`).
- [ ] Inbox → non-Pi agents: the paste door for a non-Pi CLI has no JSONL row proof; success is the verified composer submit only (2026-09-22).
