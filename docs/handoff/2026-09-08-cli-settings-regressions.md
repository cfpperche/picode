# 2026-09-08 — cli-settings-regressions: recover native settings safely

Shipped: fixes for the three regressions found in `3b49ceaa` (ADR-0101).
- Desktop restart failures propagate as partial success; a failed stop prevents start.
- Background read failures retain the editor/draft and block writes until retry succeeds.
- Missing identities remove the editor; a blocked key capture cannot write through its listener.
- Mobile agent controls and global keys remain available with unreadable native defaults.
- Desktop Tools/Checklist placement belongs to Radix; wrapped control rows retain alignment checks.

Verified: `make ci-scoped` PASS; native and quick-sheet browser regression scripts PASS.
`qa-cli-settings-recovery.mjs` passes 13 scenarios, including all eight desktop
restart outcomes (managed/interactive × PATCH/stop/start failure or success).
New missing-context and wrapped-alignment unit cases pass. Runtime responses
are injected; all fixture agents remain stopped and no model turn is sent.
visual-review: PASS — screenshots read at verified 1365×1000, 560×1000 and
390×844 viewports; Tools/Checklist and quick-sheet overlay audits are ok.
Not done: physical-device/PWA/IME and real-process restart acceptance.
Evidence: `var/screenshots/cli-settings-regressions/` (logs, screenshots, results).
Merge: fast-forward integration uses `make ci` on main; no manual deployment.
