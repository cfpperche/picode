# Communication, inbox and remote

The owner kept Activate now gated on a recorded session; remote mode stays out of v1 (`docs/plans/communication-activation-accept.md`).

## Debts
- Two `internal/server` process tests are flaky under load and can fail `make ci`: `TestPeerStopStubbornChildStaysPending` ("stubborn child reported stopped: <nil>") and `TestPaneRootSurvivesSIGHUP` ("trapped pane root died on SIGHUP"). Each passed 2 of 5 isolated reruns on 2026-09-13; both predate the browser work (`d9bc2dd9`).
- Communication: mobile/non-Linux recovery and PTY check-to-write remain unverified; uncertain attempts never retry.
- Onboarding: initial and resumed Codex/Hermes/OpenCode needed a first prompt in matrix validation. Six-CLI transport: `docs/plans/communication-native-finish.md`.
- Inbox: unmanaged Pi items lack an address until session update. Feed events can be missed across reconnects (ADR-0048); webhooks are at-least-once, receivers dedupe.
- Pi has one active credential slot; per-agent OAuth is the owner's.
