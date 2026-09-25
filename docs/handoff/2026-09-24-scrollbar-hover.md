# 2026-09-24 — feat/scrollbar-hover: every scrollbar hides until its area is hovered
Shipped (27f9070f0): the sidebar's hide-until-hover thumb is now the default in
web/shared/styles/scrollbars.css for every surface (browser, desktop, mobile
shells); the sidebar-only rules left web/browser/src/styles/app.css and
web/mobile/src/styles/app.css. Gated on `@media (hover: hover) and (pointer:
fine)` — touch keeps thumbs drawn. Reveal on :hover, :focus-within, :active.
The Instructions table (`.instr-scroll`) keeps its always-visible sideways bar
via its own rule. benchmarks.md bullet + changelog.d fragment updated.
Finding: Chromium does not restyle `::-webkit-scrollbar-thumb` when the
scrolling box's own :hover changes (`.x:hover::-webkit-scrollbar-thumb` stays
stale until relayout); it does when an ancestor's :hover changes. So the WebKit
rule is parent-keyed (`:hover > *::-webkit-scrollbar-thumb`); the Firefox path
keys on the box itself.
Verified: `make ci-scoped` PASS, `make close` ran. Scratch instance; headless
Chromium reports hover:none and agent-browser hides scrollbars in screenshots
by default (`--hide-scrollbars false`), so the (hover:hover) rules were lifted
out of their @media in-page to exercise the shipped selectors. Rest/hover ×
sidebar/content: 6 cases pass by pixel check; overlayAudit ok. Blind spot:
Firefox not rendered; no real mouse in the Windows desktop shell (WebView2).
visual-review: PASS (sb-rest, sb-hover, sb-sidebar-hover, sb-sidebar-rest; card 5/5)
Merge: fast-forward ready.

## Debts

- Scrollbar hide-until-hover unverified in Firefox and with a real mouse in the Windows shell (WebView2).
