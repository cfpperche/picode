# 2026-09-25 — feat/mobile-tab-overflow: phone tab bars overflow like the editor tabs
Shipped: the phone app (web/mobile) gets the editor-strip overflow chrome for
tab bars that scrolled with a thin scrollbar: Preferences (7 tabs), CliPaneTabs
(9 CLI sections), CliTabs (CLIs/Messages/Settings), llama.cpp sections. Phone
copies of OverflowTabs.jsx and useTabStrip.js (ADR-0072); the pure tabStrip.js
(+ test) moved to web/shared/domain, exported in web/shared/package.json, and
desktop imports follow. Arrows are 36px tap targets; swipe still scrolls.
PageFrame's reveal/hide-partial-tab effect removed (OverflowTabs reveals the
selected tab). Two-tab Packages/Connectors/Skills bars unchanged. Phone gained a
`.tab-list .um-label` style ("Out of view" heading was unstyled).
Merge with main: 7fb8ca5cb fixed the Preferences double divider its own way
(kept; this branch's override dropped) and patched PageFrame's tab mask — this
branch removes that effect, since OverflowTabs reveals and fades instead.
Verified: `make ci-scoped` PASS. Scratch instance at 390×844, native scrollbars
shown: no strip scrollbar, no page overflow, selected tab in view on deep links
(Backup via list, Codex › Connectors); overlayAudit ok on both lists. CLIs bar
and llama nav do not overflow at 390px, so their arrows were not seen there.
visual-review: PASS (pass 1 FAIL, doubled rule on Backup, fixed; pass 2 PASS on
phone and desktop Preferences)
Noted, not fixed (pre-existing): Backup folder placeholder truncated beside
Browse/Reveal on the phone; llama nav's selected style is a pill while the
other bars underline.
Not done / debts: none.
Merge: fast-forward ready.
