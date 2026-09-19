# 2026-09-19 — mobile-term-row-menu: phone Work ⋯ matches desktop lifecycle

Shipped: `termRowMenu` lives in `web/shared/domain/termRowMenu.js`. Phone
Work ⋯ (`surface: "phone"`) is Rename, Continue in… when pinned, Start or
Restart/Stop, Remove — no Launch or Terminal settings. Rename uses a
MobileSheet prompt (`web/mobile/src/lib/prompt.js`).

Verified: `node --test` on the shared menu (11 cases, phone surface
included); `make ci-scoped` PASS (fmt,vet,hooks,test-js,build,docs).
visual-review: UNVERIFIED (menu shape by unit test; no scratch screenshot,
not on a phone).

Not done: Agent CLIs Terminals ⋯ on the phone is still hand-built
(`docs/handoff/open/terminal.md`).

Merge: fast-forward ready.
