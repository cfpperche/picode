# 2026-09-14 — feat/sidebar-scroll-pin

Sidebar tabs scrolled their whole section: the WORKSPACES / Agents /
Terminals / Apps / Pins title row (and the Pins/Apps search bar) travelled
out of view with the list. Each `.side-section` is now a pinned column —
`display: flex; overflow: hidden` — and a new `.side-scroll` wrapper around
the list is the only scroller (`.pins-head` / `.pins-search` are
`flex: none`). The hover-to-reveal scrollbar rules were retargeted from
`.side-section` to `.side-scroll`.

- Gates: ci-scoped PASS (fmt, vet, hooks, test-js, build) at 41bc548c.
- Visual: all five tabs verified on a scratch instance with 15 agents /
  13 terminals / 10 pins seeded; screenshots read (ws-top, ws-bottom,
  terms-bottom, agents-bottom, pins-bottom, apps-tab) — headers pinned,
  single scrollbar, `__picodeOverlayAudit` ok. Card 5/5.
- No ADR: no protocol/persistence/security/process boundary crossed.
