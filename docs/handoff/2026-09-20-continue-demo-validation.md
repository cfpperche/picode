# 2026-09-20 — continue-demo-validation: live Continue menu measurements

Recorded: docs/reports/2026-09-20-continue-in-demo.md; no product code changes.
Verified: real installed CLIs on scratch built from main 1889244a, isolated home
and demo folder, READY prompt, 90-sample observation and actual screenshot review.
Continue appeared for Claude, Codex, Grok, Hermes, OpenCode and fresh Omp.
Pi replied but bound only after explicit session listing; Muse replied but its
native session index stayed empty, including after stop. Both debts are in
docs/handoff/open/sessions.md. OpenCode's provider returned 403; Antigravity
stayed at terms/data-use onboarding with consent not accepted.
visual-review: PASS for captured layout, hover and overlay audits; Pi automatic
binding and Muse menu availability FAIL, recorded without claiming a fix.
Cross-CLI handoff was not executed; excluded setup/auth failures are in the report.
Evidence: /home/goat/picode/var/screenshots/continue-demo-validation/.
No changelog: validation-only documentation; no deployment performed.
Merge: fast-forward ready at close-summary against main 1889244a.
