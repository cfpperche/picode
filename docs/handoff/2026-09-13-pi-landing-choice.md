# 2026-09-13 — feat/pi-landing-choice: Continue in Pi asks where it opens

Shipped: **landing choice** for the handoff target that is also a managed
agent (pi only). `GET /api/clis` advertises `sessions.agent`; the handoff
request takes `landing: agent | terminal` (omitted = shipped default).
The web derives a nested menu level (Pi agent · in the app / Pi CLI · in
a terminal) on session rows, terminal ⋯ menus and the pane menu, plus a
**Where** section in the dialog.

Verified: server decision-table tests (terminal+native → cwd bucket +
`--session`; agent+brief → queued first prompt; agent on other CLI →
400; defaults unchanged), web unit tests, `make ci-scoped` PASS. Visual
review on scratch `pi-landing` with a seeded Claude session: nested
submenu, dialog Where section, real E2E to a stopped "Race fix" agent
(`revealAgent`), brief→agent queued prompt, terminal landing cleaned up.
`overlayAudit` ok. Shots: `var/screenshots/pi-landing-*.png`.
Docs: architecture, agent-clis guide, changelog fragment.

Debts: terminal rows still need a live pin for Continue (unchanged); the
nested submenu click needed keyboard/eval in headless QA (Radix portal
timing), human mouse flow is the standard Radix one.
