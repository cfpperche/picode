# 2026-09-12 — communication-native-finish: native transport acceptance

Implemented: OpenCode editor-boundary detection accepts its sidebar and wrapped path without relaxing draft, footer, pointer-fit or post-paste safeguards.
Verified: parser regression cases, pointer boundaries and post-paste edits; `make ci-scoped` and `make close` passed.
Native: Pi → OpenCode, Claude → Hermes, Grok → Codex and post-restart OpenCode → Pi passed with message, reply and both ACKs in six real scratch TUIs; all four receipt chains were independently checked.
Configuration: OpenCode `zai-coding-plan/glm-5.3-flash`; Pi `xai/grok-4.6` with its installed adapter loaded through native package settings. Earlier missing adapter configuration was a scratch-fixture error.
Recovery: abrupt scratch-daemon death preserved all six native processes/identities; all returned Connected/confirmed/Idle within approximately six seconds. OpenCode and Codex unsent drafts stayed byte-identical; the pending check did not overwrite or submit them.
visual-review: PASS — screenshots read by a subagent; desktop/mobile actual connected, activity and empty states, native selector and overlay audit passed. Failure-state simulation and physical-device acceptance were not repeated.
Evidence: `var/qa/communication-native-finish-20260912/`; decision table: `docs/plans/communication-native-finish.md`.
Cleanup: six exact fixture terminals, the scratch daemon and verified task-owned child processes removed; no owned live processes remain. Existing platform, onboarding and PTY race debts remain in `open/communication.md`.
Merge: fast-forward ready at close; root integration CI is the final gate. No deploy.
