# 2026-09-21 — delivery-page: Git Delivery reads as a page

Shipped: the Git tab's Delivery view draws the route page frame
(`.settings-wrap` + `.settings-head` + `.settings-card`, the Agent CLIs
geometry of ADR-0103) instead of a full-bleed grey list — target, filter,
change list, detail and the empty, blocked, error and loading states live in
the card, with one page scrollbar (`web/browser/src/components/Delivery.jsx`,
`styles/delivery.css`). The two selects carry the house control look and the
stacked row separates its two verdicts. History, the tab's strip and the mobile
section are untouched. Rule amended in `docs/benchmarks.md` (§ One page width),
surface paragraph in `docs/architecture/delivery.md`, guard
`web/tools/delivery-surface.test.mjs`.

Verified: `make ci-scoped` PASS (fmt, vet, hooks, test-js, build) in this
worktree, then `make close`; scratch `qa-scratch delivery-page` on :8473 with
three fixtures (the repo, a repo with a target and no changes, a repo with no
default target) plus injected fetch failure; single scroll container, body and
`.gg-surface` do not scroll, and overlayAudit `ok` with the toolbar row at
36/36/36. Screenshots of populated, detail, empty, no-target, error, loading,
dark, 560px, `/browser/` and `/desktop/` (identical geometry) read by an
independent reviewer: PASS, History re-captured as an unchanged canvas. Method
blind spot: Linux headless Chromium — no physical Windows-shell or phone
acceptance is claimed, and the partial-read box (needs >32 checkouts or a ref
flip mid-collection) was not forced live.

visual-review: PASS (card geometry in every state; History unchanged)
Merge: fast-forward ready after merging main.
