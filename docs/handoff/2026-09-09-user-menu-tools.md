# 2026-09-09 — user-menu-tools: fold Integrations into Tools

Shipped: desktop user menu and mobile More drop the leftover *Agents and
connections* heading. Integrations sits in Tools with Agent CLIs and
Automations (mobile also Pins, Apps, llama.cpp). Search-only Providers/
Packages/Settings stay under Agent CLIs. Model + tests:
`web/desktop/src/lib/userMenuModel.js`, `web/mobile/src/lib/moreMenuModel.js`.
Verified: model tests 13/13; scratch :8473 — click Integrations (desktop
`#/integrations`, mobile `#/more/integrations`); search match + empty
state; overlayAudit ok (open + empty).
visual-review: PASS (desktop open light/dark, empty search, search
Integrations; mobile More + empty + Integrations; card 5/5)
Not done / debts: Agent CLIs subtitle still truncates in the desktop
menu (pre-existing).
Merge: fast-forward ready.
