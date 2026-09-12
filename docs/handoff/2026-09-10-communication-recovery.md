# 2026-09-10 — communication-recovery: native restart recovery
Shipped: ADR-0112 private native observations, exact process/pane recovery, independent activity/connection/test states, retained participant choices, Pi receiver renewal, Hermes root-turn context and native Grok empty suggestions.
Verified: `make ci-scoped` passed after both adapter fixes; adversarial review PASS; 26 decision-table rows have regression coverage.
Verified: six real TUIs recovered Idle/Connected in 6.045s; the later backend restart recovered six identities/connections in 0.809s while delivering a pending reply. Native PID/run/session stayed unchanged; Codex draft bytes and pending mail survived the earlier restart.
Verified: eight native message/reply exchanges with both ACKs; the final fresh Grok → Pi check passed automatically. Four expired checks and one uncertain post-paste refusal remain recorded.
Verified: scratch desktop 1440×1000 light/dark and mobile 390×844 empty, blocked, error, reconnect, connected and confirmation states; screenshots read by the visual-review subagent, overlay audits clean.
visual-review: PASS
Evidence: root `var/qa/communication-recovery-20260910/`; screenshots in `var/screenshots/communication-recovery/`; decision table and detailed acceptance in `docs/plans/communication-recovery.md`.
Not done / debts: long wrapped OpenCode sidebars/footers, physical mobile, non-Linux recovery and intermittent post-paste refusal; old Pi receivers need reload and the Hermes adapter update needs one native stop/resume. OpenCode used xAI after a Z.AI balance failure.
Cleanup: six task-owned scratch terminals and daemon 8473 stopped after all six settled Idle; evidence retained.
Merge: fast-forward ready; main CI follows. Deployment remains the owner's call.
