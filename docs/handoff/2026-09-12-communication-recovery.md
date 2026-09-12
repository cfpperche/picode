# 2026-09-12 — communication-recovery: harden native recovery
Implemented: preserve live hooks without Linux process metadata; use nonblocking shared fence locks to distinguish active writes from persistent recorder failures.
Implemented: failed recording offers Terminal controls on desktop/mobile; uncertain delivery logs bounded reasons without message or terminal content; no automatic retry.
Implemented: native launchers override inherited CLI markers; missing own identities cannot borrow a parent Codex connection.
Reviewed: retain current-main native identity, Pi reply routing, consent audit and native approval safeguards; no adversarial blockers reported.
Verified: `make ci-scoped` PASS (full); recorder/platform/lock/API/attention and shared state regressions included.
Verified: scratch desktop light/dark and mobile screenshot review PASS; both confirmation overlay audits clean.
visual-review: PASS — root `var/screenshots/communication-recovery-20260912/`.
Native rerun: in progress; current Z.AI OpenCode reported insufficient balance and Pi OAuth refresh failed. Neither is a transport pass.
Historical evidence: September 10 eight ACK-complete diagnostics and six-CLI restart recovery remain historical, not a current-build rerun.
Not done / debts: finish current native rerun, `make close`, main integration/full CI and exact-fixture cleanup; physical-mobile/non-Linux acceptance remains external.
Merge: pending current-main update and closing gates. No deployment performed.
