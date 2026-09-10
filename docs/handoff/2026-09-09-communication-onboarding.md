# 2026-09-09 — communication-onboarding: guided workspace connections

Implemented: ADR-0110; workspace participation, per-conversation setup, native diagnostics,
desktop/mobile Communication, Activity and Advanced repair controls.
Verified: final `make ci-scoped` PASS (fmt, vet, hooks, Go [18 packages], JS, build, docs);
evidence: `var/qa/communication-onboarding/quality-final.log` (52 paths; Vale clean).
Native scratch QA: managed Pi pair and Pi terminal to managed Pi each sent a message,
received a correlated reply and acknowledged both; both connection checks passed.
Pi terminal automatic resume preserved its exact native conversation.
Evidence: `native-managed-{result,history}.json` and `native-tui-{result,history}.json`
under `var/qa/communication-onboarding/`; decision coverage is in the work plan.
visual-review: PASS — desktop/mobile, light/dark, empty/blocked/error, activity and test modal;
final Advanced and hover captures reviewed; overlay audits ok. Visual card: yes/yes/yes/no/yes.
Not done / debts: Codex SessionStart before a first turn unverified; its waiting state
does not prove startup readiness, and the manual fresh launch did not prove resume.
Claude/OpenCode/Grok/Hermes onboarding matrix, physical mobile and non-Linux recovery remain.
Partial decision rows 5, 9 and 12 remain in `docs/handoff.md`: moved-owner consent,
missing-adapter repair and native stubborn-child timeout.
Merge: main moved; current-main merge, `make close`, fast-forward and main CI pending. No deployment.
